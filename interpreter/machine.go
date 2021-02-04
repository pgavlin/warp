package interpreter

import (
	"io"

	"github.com/pgavlin/warp/exec"
	"github.com/pgavlin/warp/wasm"
	"github.com/pgavlin/warp/wasm/code"
	"github.com/pgavlin/warp/wasm/trace"
)

type frame struct {
	m *machine

	module *module
	params int
	fp     int
	locals []uint64
	blocks []uint64
	stack  []uint64
}

type machine struct {
	thread *exec.Thread

	stack  []uint64
	frames []frame
}

type scope struct {
	module *module
	locals []wasm.ValueType
}

func (s *scope) GetLocalType(localidx uint32) (wasm.ValueType, bool) {
	if localidx >= uint32(len(s.locals)) {
		return 0, false
	}
	return s.locals[int(localidx)], true
}

func (s *scope) GetGlobalType(globalidx uint32) (wasm.GlobalVar, bool) {
	global, ok := s.module.getGlobal(globalidx)
	if !ok {
		return wasm.GlobalVar{}, false
	}
	return global.Type(), true
}

func (s *scope) GetFunctionSignature(funcidx uint32) (wasm.FunctionSig, bool) {
	func_, ok := s.module.getFunction(funcidx)
	if !ok {
		return wasm.FunctionSig{}, false
	}
	return func_.GetSignature(), true
}

func (s *scope) GetType(typeidx uint32) (wasm.FunctionSig, bool) {
	if typeidx >= uint32(len(s.module.types)) {
		return wasm.FunctionSig{}, false
	}
	return s.module.types[int(typeidx)], true
}

func (s *scope) HasTable(tableidx uint32) bool {
	return tableidx == 0 && s.module.table0 != nil
}

func (s *scope) HasMemory(memoryidx uint32) bool {
	return memoryidx == 0 && s.module.mem0 != nil
}

func (m *machine) init(t *exec.Thread) {
	m.thread = t
	m.stack = make([]uint64, 0, 1024)
	m.frames = make([]frame, 0, 128)
}

func (m *machine) zero64(s []uint64) {
	for i := range s {
		s[i] = 0
	}
}

func (m *machine) eq64(a, b []uint64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// frame layout:
//
// params (f.params)                      <-- f.locals starts here
// ---
// locals (len(f.locals) - f.params)      <-- fp points here
// blocks (cap(f.blocks))                 <-- f.blocks starts here
// stack  (len(f.stack))                  <-- f.stack starts here
//
// size of an active frame: len(f.stack) + cap(f.blocks) + len(f.locals) - f.params
func (m *machine) alloc(nparams, nlocals, maxStack, maxBlocks int) *frame {
	maxFrame := maxStack + nlocals + maxBlocks - nparams

	// Find the top of the stack. m.stack always ends after the maximum size of the currently-active frame, so we find
	// the top of the stack by finding the beginning of the active frame and adding the frame's current size.
	stack := m.stack
	if len(m.frames) != 0 {
		stack = m.stack[:m.frames[len(m.frames)-1].sp()]
	}

	// If we don't have enough room to allocate this frame, we need to grow the stack. Because this involves a copy, we
	// also need to crawl the call stack and update each frame's pointers.
	if cap(stack)-len(stack) < maxFrame {
		x := (maxFrame/1024 + 1) * 1024
		newStack := make([]uint64, len(stack), len(stack)+x)
		copy(newStack, stack)
		stack = newStack

		for i := range m.frames {
			f := &m.frames[i]

			frame := stack[f.fp-f.params:]
			f.locals, frame = frame[0:len(f.locals):len(f.locals)], frame[len(f.locals):]
			f.blocks, frame = frame[0:len(f.blocks):cap(f.blocks)], frame[cap(f.blocks):]
			f.stack = frame[0:len(f.stack):cap(f.stack)]
		}
	}

	if cap(m.frames)-len(m.frames) < 1 {
		frames := make([]frame, len(m.frames), cap(m.frames)+128)
		copy(frames, m.frames)
		m.frames = frames
	}
	m.frames = m.frames[:len(m.frames)+1]
	f := &m.frames[len(m.frames)-1]

	fp := len(stack)
	fr := stack[fp-nparams:]
	flocals := fr[0 : nlocals : nlocals+maxStack]
	fblocks := fr[nlocals : nlocals : nlocals+maxBlocks]
	fstack := fr[nlocals+maxBlocks : nlocals+maxBlocks : nlocals+maxBlocks+maxStack]
	m.zero64(fr[nparams:nlocals])

	m.stack = stack[:len(stack)+maxFrame]

	f.m = m
	f.params = nparams
	f.fp = fp
	f.locals = flocals
	f.blocks = fblocks
	f.stack = fstack
	return f
}

func (m *machine) free(sp int) {
	m.stack = m.stack[:sp]
	m.frames = m.frames[:len(m.frames)-1]
}

func (m *machine) push(fn *function) *frame {
	// Decode the function if necessary.
	switch fn.kind {
	case functionKindBytecode:
		locals := append([]wasm.ValueType(nil), fn.signature.ParamTypes...)
		for _, entry := range fn.localEntries {
			for i := 0; i < int(entry.Count); i++ {
				locals = append(locals, entry.Type)
			}
		}
		fn.numLocals = len(locals)

		body, err := code.Decode(fn.bytecode, &scope{
			module: fn.module,
			locals: locals,
		}, fn.signature.ReturnTypes)
		if err != nil {
			panic(err)
		}
		fn.icode, fn.metrics, fn.bytecode = body.Instructions, body.Metrics, nil

		switch {
		case fn.metrics.HasLoops:
			m.emitFcode(fn, fn.icode)
			fn.kind = functionKindFCode
		case len(fn.icode) >= 16:
			fn.kind = functionKindCountingICode
		default:
			fn.kind = functionKindICode
		}

	case functionKindCountingICode:
		fn.invokeCount++
		if fn.invokeCount > 1 {
			m.emitFcode(fn, fn.icode)
			fn.kind = functionKindFCode
		}
	}

	nblocks := 0
	if fn.kind != functionKindFCode || m.thread.Debug() {
		nblocks = fn.metrics.MaxNesting * 2
	}

	// Allocate space for the frame.
	nparams, nlocals, nstack := len(fn.signature.ParamTypes), fn.numLocals, fn.metrics.MaxStackDepth
	f := m.alloc(nparams, nlocals, nstack, nblocks)

	// Fill in the frame's details.
	f.module = fn.module

	return f
}

func (m *machine) pop(fn *function) {
	f := &m.frames[len(m.frames)-1]

	// Move results to the top of the stack.
	nresults := len(fn.signature.ReturnTypes)
	sp := f.fp - f.params
	copy(m.stack[sp:], f.stack[len(f.stack)-nresults:])

	// Return the frame's storage to the machine.
	m.free(sp + nresults)
}

func (f *frame) sp() int {
	return f.fp + len(f.stack) + cap(f.blocks) + len(f.locals) - f.params
}

func (f *frame) trap(t exec.Trap) {
	panic(t)
}

func (f *frame) runDebug(fn *function) {
	f.m.thread.EnterFrame(&exec.Frame{
		ModuleName:        f.module.name,
		FunctionIndex:     fn.index,
		FunctionSignature: fn.signature,
		Locals:            f.locals,
	})

	if trace, tracing := f.m.thread.Trace(); tracing {
		f.runTrace(trace, fn)
	} else {
		f.runICode(fn)
	}

	f.m.thread.LeaveFrame()
}

func (f *frame) runTrace(w io.Writer, fn *function) {
	// Push the first label.
	f.blocks = f.blocks[:2]
	f.blocks[0] = uint64(len(fn.icode) - 1)
	f.blocks[1] = uint64(len(fn.signature.ReturnTypes))

	locals := append([]wasm.ValueType(nil), fn.signature.ParamTypes...)
	for _, entry := range fn.localEntries {
		for i := 0; i < int(entry.Count); i++ {
			locals = append(locals, entry.Type)
		}
	}

	s := scope{module: f.module, locals: locals}
	ip := 0
	for {
		instr := &fn.icode[ip]
		popT, pushT := instr.Types(&s)
		pop, push := len(popT), len(pushT)
		traceEntry := trace.InstructionEntry{
			IP:          ip,
			Instruction: *instr,
			ArgTypes:    popT,
			ResultTypes: pushT,
			Args:        make([]uint64, pop),
			Results:     make([]uint64, push),
		}
		copy(traceEntry.Args, f.stack[len(f.stack)-pop:])

		ip = f.step(fn.icode, ip)

		copy(traceEntry.Results, f.stack[len(f.stack)-push:])
		traceEntry.Encode(w)

		if ip == len(fn.icode) {
			return
		}
		ip = ip
	}
}

func (f *frame) invoke(fn exec.Function) {
	if fn, ok := fn.(*function); ok {
		f.invokeDirect(fn)
		return
	}

	sig := fn.GetSignature()
	desc := function{
		signature: sig,
		metrics: code.Metrics{
			MaxStackDepth: len(sig.ReturnTypes),
			MaxNesting:    1,
		},
		kind:      functionKindVirtual,
		numLocals: len(sig.ParamTypes),
	}

	callee := f.m.push(&desc)
	callee.stack = callee.stack[:len(sig.ReturnTypes)]
	fn.UncheckedCall(callee.m.thread, callee.locals, callee.stack)
	callee.m.pop(&desc)

	f.stack = f.stack[:len(f.stack)-len(sig.ParamTypes)+len(sig.ReturnTypes)]
}

func (f *frame) invokeDirect(fn *function) {
	callee := f.m.push(fn)

	if f.m.thread.Debug() {
		callee.runDebug(fn)
	} else {
		callee.m.thread.Enter()
		if fn.kind == functionKindFCode {
			callee.runFCode(fn)
		} else {
			callee.runICode(fn)
		}
		callee.m.thread.Leave()
	}

	callee.m.pop(fn)

	f.stack = f.stack[:len(f.stack)-len(fn.signature.ParamTypes)+len(fn.signature.ReturnTypes)]
}
