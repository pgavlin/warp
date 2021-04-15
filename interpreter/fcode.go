package interpreter

import (
	"math"

	"github.com/pgavlin/warp/wasm/code"
)

// fcode is a simple bytecode designed to compactly represent WASM method bodies. A single
// fcode instruction may represent multiple WASM instructions. For example, the sequence
//
//     loop
//     local.get 0
//     i64.const -1
//     i64.add
//     local.tee 0
//     br.if 0
//     end
//
// would be represented by the following fcode:
//
//     l0:
//         v0 = i64.add v0, -1
//         br.if.l l0
//
// The fcode representation is ~1/4 the size of the WASM representation with a
// proportional decrease in execution time.
//
// Analysis of several WASM programs compiled from Rust, C, and Go indicates that ~55% of
// instructions are local.get/local.set, constants, compares, branches, or block
// instructions. Fcode has been designed to take advantage of those statistics by folding
// most local.get and constant instructions into the consuming instruction, combining
// many compares and branches, and removing most block instructions.
//
// An fcode function body is composed of a list of labels, a list of switch tables, and a
// sequence of instructions. An fcode function _invocation_ has an associated frame
// that holds the function's arguments, locals, and temporary values. The frame is
// statically-sized. The translation from WASM to fcode takes advantage of the fact that
// numbering the WASM stack entries from 0 (for the bottom of the stack) to N (for the top
// of the stack) produces a legal allocation of values to frame slots: each value's frame
// slot is the number of its stack entry plus the number of locals.
const (
	vfFrame        = 1 << iota // this value is a frame address
	vfConst                    // this value is a constant
	vfMaterialized             // this value has been materialized (and therefore must be copied to its frame address)

	ifDefLocal  = 1 << iota // this instruction defines a local
	ifSrc1Frame             // this instruction's first operand is on the frame
	ifSrc2Frame             // this instruction's second operand is on the frame
)

// An fvalue records information about a value on the abstract machine's stack.
//
// There are two kinds of values:
// 1. Frame addresses
// 2. Constants
//
// Frame addresses refer directly to slots in the frame. These values are suitable for
// use as frame sources and destinations for instructions. Constants are self-explanatory.
//
// A value may need to be materialized if it must be on the frame e.g. because it is
// passed to or returned from a block or call, or refers to a frame slot that is
// reassigned before the value is used (e.g. a local.get that is not used prior to a
// local.set that targets the same local).
//
// Each value-producing instruction (e.g. local.get, i32.add, f32.const) is assigned a
// destination frame slot at the time it is encountered. The frame slot is selected by
// numbering the WASM stack as described in the documentation at the top of this file.
// If an unmaterialized value is consumed by a local.set and if it is legal to move the
// set backwards to the point of the instruction that produced the value, the local.set
// may be eliminated by replacing the producing instruction's frame slot with the slot
// for the destination local.
type fvalue struct {
	flags     uint32 // information about this value (is it a frame address or a constant)
	ip        int    // the ip of the instruction that produced this value
	addr      uint32 // the frame slot that holds this value
	immediate uint64 // the constant value or frame slot for this value if it is a constant or unmaterialized frame value, respectively
}

type opcode uint16

// An finstruction holds a single fcode instruction.
//
// Each instruction is composed of an opcode, flags, a destination, and up to three
// sources. The set of instructions is divided into the following classes, each of which
// assigns different meanings to the destination and source fields:
//
// - If
// - Else
// - Branch instructions
// - Conditional branch instructions
// - Return
// - Call instructions
// - Local instructions
// - Select
// - Global instructions
// - Load instructions
// - Store instructions
// - Memory instructions
// - Constant instructions
// - Binary numeric operators
// - Unary numeric operators
//
type finstruction struct {
	opcode opcode
	flags  uint16

	dest uint32
	src1 uint32
	src2 uint64
}

func (i *finstruction) Labelidx() int {
	return int(int32(i.dest))
}

func (i *finstruction) Switchidx() int {
	return int(i.src2 >> 32)
}

func (i *finstruction) StackHeight() int {
	return int(int32(i.src2))
}

func (i *finstruction) Globalidx() uint32 {
	return uint32(i.src2)
}

func (i *finstruction) Typeidx() uint32 {
	return uint32(i.src2 >> 32)
}

func (i *finstruction) Funcidx() uint32 {
	return uint32(i.src2 >> 32)
}

func (i *finstruction) Offset() uint32 {
	return uint32(i.src2)
}

func (i *finstruction) Src2() uint32 {
	return uint32(i.src2)
}

func (i *finstruction) Src3() uint32 {
	return uint32(i.src2 >> 32)
}

func (i *finstruction) I32() int32 {
	return int32(i.src2)
}

func (i *finstruction) I64() int64 {
	return int64(i.src2)
}

func (i *finstruction) F32() float32 {
	return math.Float32frombits(uint32(i.src2))
}

func (i *finstruction) F64() float64 {
	return math.Float64frombits(i.src2)
}

type label struct {
	continuation [2]int
	stackHeight  int
	arity        int
}

func (l *label) Continuation() int {
	return l.continuation[0]
}

func (l *label) Else() int {
	return l.continuation[1]
}

type block struct {
	index            int
	ins, outs        int
	isLoop           bool
	entryUnreachable bool
	unreachable      bool
}

type switchTable struct {
	indices []int
}

type fimporter struct {
	fn     *function
	locals int

	labels   []label
	switches []switchTable
	body     []finstruction

	blocks []block
	stack  []fvalue
}

func (m *machine) emitFcode(fn *function, body []code.Instruction) {
	imp := fimporter{
		fn:     fn,
		locals: fn.numLocals,
		labels: make([]label, 1, fn.metrics.LabelCount),
		blocks: make([]block, 1, fn.metrics.MaxNesting),
		stack:  make([]fvalue, 0, fn.metrics.MaxStackDepth),
		body:   make([]finstruction, 0, len(body)/2),
	}

	imp.labels[0] = label{arity: len(fn.signature.ReturnTypes)}
	imp.blocks[0] = block{outs: len(fn.signature.ReturnTypes)}

	for i := range body {
		imp.emitInstruction(&body[i])
	}

	imp.labels[0].continuation[0] = len(imp.body)
	ret := code.Return()
	imp.emitInstruction(&ret)

	fn.labels = imp.labels
	fn.switches = imp.switches
	fn.fcode = imp.body
}

func (imp *fimporter) popMaterialized(n int) {
	imp.materialize(imp.stack[len(imp.stack)-n:])
	imp.stack = imp.stack[:len(imp.stack)-n]
}

func (imp *fimporter) popAddressable() uint32 {
	v := imp.stack[len(imp.stack)-1]
	imp.stack = imp.stack[:len(imp.stack)-1]
	imp.emitAddressable(&v)
	return v.addr
}

func (imp *fimporter) pushValues(ip int, stackHeight, flags uint32, n int) {
	for i := 0; i < n; i++ {
		imp.stack = append(imp.stack, fvalue{
			flags: vfFrame | flags,
			ip:    ip,
			addr:  uint32(imp.locals) + stackHeight + uint32(i),
		})
	}
}

func (imp *fimporter) emitUnreachable() {
	// Mark the rest of the block unreachable.
	stackHeight := 0
	if len(imp.blocks) > 0 {
		b := &imp.blocks[len(imp.blocks)-1]
		b.unreachable = true
		stackHeight = imp.labels[b.index].stackHeight
	}

	// Materialize all values on the stack up to the block's stack height.
	n := len(imp.stack) - stackHeight
	if n < 0 {
		n = len(imp.stack)
	}
	imp.popMaterialized(n)
}

func (imp *fimporter) emitAddressable(v *fvalue) {
	if v.flags&vfConst != 0 {
		v.flags = vfFrame
		v.ip = len(imp.body)
		imp.body = append(imp.body, finstruction{
			opcode: fopI64Const,
			dest:   v.addr,
			src2:   v.immediate,
		})
	}
}

func (imp *fimporter) emitMaterialize(v *fvalue) {
	if v.flags&vfMaterialized == 0 {
		imp.emitAddressable(v)

		if v.flags&vfFrame == 0 {
			imp.body = append(imp.body, finstruction{
				opcode: fopLocalGet,
				flags:  ifSrc1Frame,
				dest:   uint32(v.immediate),
				src1:   v.addr,
			})
			v.flags = vfFrame
			v.ip = len(imp.body)
			v.addr, v.immediate = uint32(v.immediate), 0
		}

		v.flags |= vfMaterialized
	}
}

func (imp *fimporter) materialize(vs []fvalue) {
	for i := range vs {
		imp.emitMaterialize(&vs[i])
	}
}

func (imp *fimporter) emit(fi *finstruction, nresults int) {
	imp.emitFlags(fi, 0, nresults)
}

func (imp *fimporter) emitFlags(fi *finstruction, flags uint32, nresults int) {
	ip := len(imp.body)
	imp.body = append(imp.body, *fi)
	imp.pushValues(ip, uint32(len(imp.stack)), flags, nresults)
}

func (imp *fimporter) emitBlock(ins, outs int, isLoop bool) {
	// Materialize values on the stack.
	lastStackheight := imp.labels[imp.blocks[len(imp.blocks)-1].index].stackHeight
	imp.materialize(imp.stack[lastStackheight:])

	imp.blocks = append(imp.blocks, block{
		index:  len(imp.labels),
		ins:    ins,
		outs:   outs,
		isLoop: isLoop,
	})

	label := label{stackHeight: len(imp.stack) - ins}
	if isLoop {
		label.continuation[0], label.arity = len(imp.body), ins
	} else {
		label.arity = outs
	}
	imp.labels = append(imp.labels, label)
}

func (imp *fimporter) emitElse() {
	b := &imp.blocks[len(imp.blocks)-1]

	if b.entryUnreachable {
		return
	}

	// Materialize block outputs if the end of the if block is reachable.
	if !b.unreachable {
		imp.popMaterialized(b.outs)
	} else {
		b.unreachable = false
	}

	// Emit the else instruction.
	imp.emitFlags(&finstruction{
		opcode: fopElse,
		dest:   uint32(b.index),
	}, vfMaterialized, b.ins)

	imp.labels[b.index].continuation[1] = len(imp.body)
}

func (imp *fimporter) emitEnd() {
	b := &imp.blocks[len(imp.blocks)-1]
	imp.blocks = imp.blocks[:len(imp.blocks)-1]

	if b.entryUnreachable {
		return
	}

	l := &imp.labels[b.index]

	// Materialize block outputs if the end is reachable.
	//
	// Otherwise, push the block's output slots.
	if !b.unreachable {
		imp.materialize(imp.stack[len(imp.stack)-b.outs:])
	} else {
		ip := len(imp.body)
		imp.pushValues(ip, uint32(l.stackHeight), vfMaterialized, b.outs)
	}

	// Set the continuation address. Do this after the spills as spilling can emit code.
	if !b.isLoop {
		ip := len(imp.body)
		l.continuation[0] = ip
	}
}

func (imp *fimporter) mergeConditionalBranch(conditionValue *fvalue, branch *finstruction) bool {
	if conditionValue.flags&vfFrame == 0 || conditionValue.ip != len(imp.body)-1 {
		return false
	}

	op := opcode(0)
	condition := &imp.body[conditionValue.ip]
	switch condition.opcode {
	case fopI32Eqz:
		op = fopBrIfI32Eqz
	case fopI32Eq:
		op = fopBrIfI32Eq
	case fopI32Ne:
		op = fopBrIfI32Ne
	case fopI32LtS:
		op = fopBrIfI32LtS
	case fopI32LtU:
		op = fopBrIfI32LtU
	case fopI32GtS:
		op = fopBrIfI32GtS
	case fopI32GtU:
		op = fopBrIfI32GtU
	case fopI32LeS:
		op = fopBrIfI32LeS
	case fopI32LeU:
		op = fopBrIfI32LeU
	case fopI32GeS:
		op = fopBrIfI32GeS
	case fopI32GeU:
		op = fopBrIfI32GeU

	case fopI64Eqz:
		op = fopBrIfI64Eqz
	case fopI64Eq:
		op = fopBrIfI64Eq
	case fopI64Ne:
		op = fopBrIfI64Ne
	case fopI64LtS:
		op = fopBrIfI64LtS
	case fopI64LtU:
		op = fopBrIfI64LtU
	case fopI64GtS:
		op = fopBrIfI64GtS
	case fopI64GtU:
		op = fopBrIfI64GtU
	case fopI64LeS:
		op = fopBrIfI64LeS
	case fopI64LeU:
		op = fopBrIfI64LeU
	case fopI64GeS:
		op = fopBrIfI64GeS
	case fopI64GeU:
		op = fopBrIfI64GeU

	case fopF32Eq:
		op = fopBrIfF32Eq
	case fopF32Ne:
		op = fopBrIfF32Ne
	case fopF32Lt:
		op = fopBrIfF32Lt
	case fopF32Gt:
		op = fopBrIfF32Gt
	case fopF32Le:
		op = fopBrIfF32Le
	case fopF32Ge:
		op = fopBrIfF32Ge

	case fopF64Eq:
		op = fopBrIfF64Eq
	case fopF64Ne:
		op = fopBrIfF64Ne
	case fopF64Lt:
		op = fopBrIfF64Lt
	case fopF64Gt:
		op = fopBrIfF64Gt
	case fopF64Le:
		op = fopBrIfF64Le
	case fopF64Ge:
		op = fopBrIfF64Ge

	case fopI32EqI:
		op = fopBrIfI32EqI
	case fopI32NeI:
		op = fopBrIfI32NeI
	case fopI32LtSI:
		op = fopBrIfI32LtSI
	case fopI32LtUI:
		op = fopBrIfI32LtUI
	case fopI32GtSI:
		op = fopBrIfI32GtSI
	case fopI32GtUI:
		op = fopBrIfI32GtUI
	case fopI32LeSI:
		op = fopBrIfI32LeSI
	case fopI32LeUI:
		op = fopBrIfI32LeUI
	case fopI32GeSI:
		op = fopBrIfI32GeSI
	case fopI32GeUI:
		op = fopBrIfI32GeUI

	case fopI64EqI:
		op = fopBrIfI64EqI
	case fopI64NeI:
		op = fopBrIfI64NeI
	case fopI64LtSI:
		op = fopBrIfI64LtSI
	case fopI64LtUI:
		op = fopBrIfI64LtUI
	case fopI64GtSI:
		op = fopBrIfI64GtSI
	case fopI64GtUI:
		op = fopBrIfI64GtUI
	case fopI64LeSI:
		op = fopBrIfI64LeSI
	case fopI64LeUI:
		op = fopBrIfI64LeUI
	case fopI64GeSI:
		op = fopBrIfI64GeSI
	case fopI64GeUI:
		op = fopBrIfI64GeUI

	case fopF32EqI:
		op = fopBrIfF32EqI
	case fopF32NeI:
		op = fopBrIfF32NeI
	case fopF32LtI:
		op = fopBrIfF32LtI
	case fopF32GtI:
		op = fopBrIfF32GtI
	case fopF32LeI:
		op = fopBrIfF32LeI
	case fopF32GeI:
		op = fopBrIfF32GeI

	case fopF64EqI:
		op = fopBrIfF64EqI
	case fopF64NeI:
		op = fopBrIfF64NeI
	case fopF64LtI:
		op = fopBrIfF64LtI
	case fopF64GtI:
		op = fopBrIfF64GtI
	case fopF64LeI:
		op = fopBrIfF64LeI
	case fopF64GeI:
		op = fopBrIfF64GeI

	default:
		return false
	}

	*branch = *condition
	branch.opcode = op
	return true
}

func (imp *fimporter) emitConditionalBranch(labelidx int) {
	condition := imp.stack[len(imp.stack)-1]
	imp.stack = imp.stack[:len(imp.stack)-1]

	destBlock := &imp.blocks[len(imp.blocks)-labelidx-1]
	destLabel := &imp.labels[destBlock.index]

	fi := finstruction{opcode: fopBrIfL}

	// attempt to merge the conditional branch and the condition
	if len(imp.stack) == destLabel.stackHeight && imp.mergeConditionalBranch(&condition, &fi) {
		imp.body = imp.body[:len(imp.body)-1]
	} else {
		// Ensure that the condition is addressable.
		imp.emitAddressable(&condition)

		fi.src1 = condition.addr

		// Record the stack height.
		fi.src2 = uint64(len(imp.stack))

		// If the stack height at the destination does not match the source stack height,
		// use a slow branch.
		if len(imp.stack) != destLabel.stackHeight {
			fi.opcode = fopBrIf
		}
	}

	fi.dest = uint32(destBlock.index)

	imp.materialize(imp.stack[len(imp.stack)-destLabel.arity:])

	imp.emit(&fi, 0)
}

func (imp *fimporter) emitBranch(fi *finstruction, labelidx int) {
	destBlock := &imp.blocks[len(imp.blocks)-labelidx-1]
	destLabel := &imp.labels[destBlock.index]

	// Record the stack height
	fi.src2 |= uint64(len(imp.stack))

	// If the stack height at the destination does not match the source stack height, use
	// a slow branch.
	if len(imp.stack) != destLabel.stackHeight {
		fi.opcode &^= 0x0100
	}

	fi.dest = uint32(destBlock.index)

	imp.popMaterialized(destLabel.arity)

	imp.emit(fi, 0)

	imp.emitUnreachable()
}

func (imp *fimporter) emitCall(fi *finstruction, nparams, nresults int) {
	// Record the stack height
	fi.src2 |= uint64(len(imp.stack))

	// Ensure all operands are materialized
	imp.popMaterialized(nparams)

	// Record the destination slot
	fi.dest = uint32(imp.locals + len(imp.stack))

	imp.emitFlags(fi, vfMaterialized, nresults)
}

func (imp *fimporter) pushLocalGet(localidx uint32) {
	imp.stack = append(imp.stack, fvalue{
		addr:      localidx,
		immediate: uint64(imp.locals + len(imp.stack)),
	})
}

func (imp *fimporter) emitLocalSet(localidx uint32) {
	// Materialize anything that interferes in the current block. Values that were on the stack
	// at block entry should already have been materialized.
	stackHeight := imp.labels[imp.blocks[len(imp.blocks)-1].index].stackHeight
	for i := range imp.stack[stackHeight:] {
		v := &imp.stack[stackHeight+i]
		if v.flags == 0 && v.addr == localidx {
			imp.emitMaterialize(v)
		}
	}

	v := imp.stack[len(imp.stack)-1]
	imp.stack = imp.stack[:len(imp.stack)-1]

	// If the operand is a constant, use the constant form.
	if v.flags&vfConst != 0 {
		imp.emit(&finstruction{
			opcode: fopLocalSetI,
			flags:  ifDefLocal,
			dest:   localidx,
			src2:   v.immediate,
		}, 0)
		return
	}

	// If this set's value was not materialized and refers to a stack slot, merge the set with
	// the instruction that produced the value.
	if v.flags&(vfMaterialized|vfFrame) == vfFrame && v.ip == len(imp.body)-1 {
		instr := &imp.body[v.ip]
		instr.flags |= ifDefLocal
		instr.dest = localidx
		return
	}

	imp.emit(&finstruction{
		opcode: fopLocalSet,
		flags:  ifDefLocal | ifSrc1Frame,
		dest:   localidx,
		src1:   v.addr,
	}, 0)
}

func (imp *fimporter) emitLoad(instr *code.Instruction) {
	address := imp.popAddressable()

	fi := finstruction{
		opcode: opcode(instr.Opcode),
		flags:  ifSrc1Frame,
		dest:   uint32(imp.locals + len(imp.stack)),
		src1:   address,
		src2:   uint64(instr.Offset()),
	}

	// If the offset is zero, use the at form.
	if fi.src2 == 0 {
		fi.opcode |= 0x0200
	}

	imp.emit(&fi, 1)
}

func (imp *fimporter) emitStore(instr *code.Instruction) {
	// 2 operands: the value to store and the store address
	value, address := imp.stack[len(imp.stack)-1], imp.stack[len(imp.stack)-2]
	imp.stack = imp.stack[:len(imp.stack)-2]

	fi := finstruction{
		opcode: opcode(instr.Opcode),
		flags:  ifSrc1Frame,
		dest:   address.addr,
		src1:   value.addr,
		src2:   uint64(instr.Offset()),
	}

	// If the address is a constant zero, replace it with the offset.
	if address.flags&(vfConst|vfMaterialized) == vfConst && address.immediate == 0 {
		address.immediate, fi.src2, fi.opcode = fi.src2, 0, fi.opcode|0x0200
	} else if fi.src2 == 0 {
		// If the offset is zero, use the at form.
		fi.opcode |= 0x0200
	}
	imp.emitAddressable(&address)

	// If the value is a constant 0, use the zero form.
	if value.flags&(vfConst|vfMaterialized) == vfConst && value.immediate == 0 {
		fi.opcode |= 0x100
	} else {
		imp.emitAddressable(&value)
	}

	imp.emit(&fi, 0)
}

func (imp *fimporter) pushConst(v uint64) {
	imp.stack = append(imp.stack, fvalue{
		flags:     vfConst,
		addr:      uint32(imp.locals + len(imp.stack)),
		immediate: v,
	})
}

func (imp *fimporter) emitBinOp(instr *code.Instruction, commutative bool) {
	// 2 operands: the RHS and the LHS
	rhs, lhs := imp.stack[len(imp.stack)-1], imp.stack[len(imp.stack)-2]
	imp.stack = imp.stack[:len(imp.stack)-2]

	// If the operation is commutative and the lhs is a constant, move it to the rhs
	if commutative && lhs.flags&vfConst != 0 {
		rhs, lhs = lhs, rhs
	}

	// Ensure the LHS is addressable.
	imp.emitAddressable(&lhs)

	fi := finstruction{
		opcode: opcode(instr.Opcode),
		flags:  ifSrc1Frame,
		dest:   uint32(imp.locals + len(imp.stack)),
		src1:   lhs.addr,
	}

	// If the RHS is a constant, use the operation's immediate form.
	if rhs.flags&vfConst != 0 {
		fi.src2 = rhs.immediate
		fi.opcode |= 0x0200
		imp.emit(&fi, 1)
		return
	}

	// Otherwise, use the operation's standard form.
	fi.src2 = uint64(rhs.addr)
	fi.flags |= ifSrc2Frame

	imp.emit(&fi, 1)
}

func (imp *fimporter) emitUnOp(instr *code.Instruction) {
	imp.emitUnOpF(&finstruction{opcode: opcode(instr.Opcode)})
}

func (imp *fimporter) emitUnOpF(fi *finstruction) {
	// 1 operand
	fi.src1 = imp.popAddressable()
	fi.dest = uint32(imp.locals + len(imp.stack))
	fi.flags = ifSrc1Frame

	imp.emit(fi, 1)
}

func (imp *fimporter) emitInstruction(instr *code.Instruction) {
	// Handle unreachable -> reachable transitions first.
	switch instr.Opcode {
	case code.OpElse:
		imp.emitElse()
		return
	case code.OpEnd:
		imp.emitEnd()
		return
	}

	// If this instruction is not reachable, ignore it.
	if len(imp.blocks) > 0 && imp.blocks[len(imp.blocks)-1].unreachable {
		// ...with the exception of control instructions, which we need to maintain the
		// nesting expected by else/end above.
		switch instr.Opcode {
		case code.OpBlock, code.OpLoop, code.OpIf:
			ins, outs := imp.fn.blockType(instr)
			imp.blocks = append(imp.blocks, block{
				index:            len(imp.labels) - 1,
				ins:              len(ins),
				outs:             len(outs),
				entryUnreachable: true,
				unreachable:      true,
			})
		}
		return
	}

	switch instr.Opcode {
	case code.OpUnreachable:
		imp.emitUnreachable()
		imp.emit(&finstruction{opcode: fopUnreachable}, 0)

	case code.OpNop:

	case code.OpBlock:
		ins, outs := imp.fn.blockType(instr)
		imp.emitBlock(len(ins), len(outs), false)

	case code.OpLoop:
		ins, outs := imp.fn.blockType(instr)
		imp.emitBlock(len(ins), len(outs), true)

	case code.OpIf:
		condition := imp.popAddressable()
		ins, outs := imp.fn.blockType(instr)
		imp.emitBlock(len(ins), len(outs), false)
		imp.body = append(imp.body, finstruction{
			opcode: fopIf,
			dest:   uint32(imp.blocks[len(imp.blocks)-1].index),
			src1:   condition,
		})

	case code.OpBr:
		imp.emitBranch(&finstruction{opcode: fopBrL}, instr.Labelidx())
	case code.OpBrIf:
		imp.emitConditionalBranch(instr.Labelidx())
	case code.OpBrTable:
		t := switchTable{indices: make([]int, len(instr.Labels))}
		for i, l := range instr.Labels {
			t.indices[i] = imp.blocks[len(imp.blocks)-l-1].index
		}
		switchIndex := len(imp.switches)
		imp.switches = append(imp.switches, t)

		imp.emitBranch(&finstruction{
			opcode: fopBrTableL,
			flags:  ifSrc1Frame,
			src1:   imp.popAddressable(),
			src2:   uint64(switchIndex) << 32,
		}, instr.Default())

	case code.OpReturn:
		// record the stack height
		stackHeight := len(imp.stack)
		imp.popMaterialized(len(imp.fn.signature.ReturnTypes))
		imp.emit(&finstruction{opcode: fopReturn, src2: uint64(stackHeight)}, 0)
		imp.emitUnreachable()

	case code.OpCall:
		funcidx, nparams, nresults := instr.Funcidx(), 0, 0
		if funcidx < uint32(len(imp.fn.module.importedFunctions)) {
			sig := imp.fn.module.importedFunctions[funcidx].GetSignature()
			nparams, nresults = len(sig.ParamTypes), len(sig.ReturnTypes)
		} else {
			callee := &imp.fn.module.functions[funcidx-uint32(len(imp.fn.module.importedFunctions))]
			nparams, nresults = len(callee.signature.ParamTypes), len(callee.signature.ReturnTypes)
		}
		imp.emitCall(&finstruction{
			opcode: fopCall,
			src2:   uint64(funcidx) << 32,
		}, nparams, nresults)
	case code.OpCallIndirect:
		sig := imp.fn.module.types[instr.Typeidx()]
		imp.emitCall(&finstruction{
			opcode: fopCallIndirect,
			flags:  ifSrc1Frame,
			src1:   imp.popAddressable(),
			src2:   uint64(instr.Typeidx()) << 32,
		}, len(sig.ParamTypes), len(sig.ReturnTypes))

	case code.OpDrop:
		// note that this can never remove side effects: we're either dropping a value
		// whose side effects are already in the function body or we're dropping a
		// side-effect-free value.
		imp.stack = imp.stack[:len(imp.stack)-1]

	case code.OpSelect:
		condition, v2, v1 := imp.popAddressable(), imp.popAddressable(), imp.popAddressable()
		imp.emit(&finstruction{
			opcode: fopSelect,
			flags:  ifSrc1Frame | ifSrc2Frame,
			dest:   uint32(imp.locals + len(imp.stack)),
			src1:   condition,
			src2:   uint64(v1) | uint64(v2)<<32,
		}, 1)

	case code.OpLocalGet:
		imp.pushLocalGet(instr.Localidx())
	case code.OpLocalSet:
		imp.emitLocalSet(instr.Localidx())
	case code.OpLocalTee:
		imp.emitLocalSet(instr.Localidx())
		imp.pushLocalGet(instr.Localidx())

	case code.OpGlobalGet:
		imp.emit(&finstruction{
			opcode: fopGlobalGet,
			dest:   uint32(imp.locals + len(imp.stack)),
			src2:   uint64(instr.Globalidx()),
		}, 1)
	case code.OpGlobalSet:
		v := imp.stack[len(imp.stack)-1]
		imp.stack = imp.stack[:len(imp.stack)-1]

		if v.flags&vfConst != 0 {
			imp.emit(&finstruction{
				opcode: fopGlobalSetI,
				dest:   instr.Globalidx(),
				src2:   v.immediate,
			}, 0)
			return
		}

		imp.emit(&finstruction{
			opcode: fopGlobalSet,
			flags:  ifSrc1Frame,
			dest:   instr.Globalidx(),
			src1:   v.addr,
		}, 0)

	case code.OpI32Load:
		imp.emitLoad(instr)
	case code.OpI64Load:
		imp.emitLoad(instr)
	case code.OpF32Load:
		imp.emitLoad(instr)
	case code.OpF64Load:
		imp.emitLoad(instr)
	case code.OpI32Load8S, code.OpI32Load8U, code.OpI32Load16S, code.OpI32Load16U:
		imp.emitLoad(instr)
	case code.OpI64Load8S, code.OpI64Load8U, code.OpI64Load16S, code.OpI64Load16U, code.OpI64Load32S, code.OpI64Load32U:
		imp.emitLoad(instr)

	case code.OpI32Store:
		imp.emitStore(instr)
	case code.OpI64Store:
		imp.emitStore(instr)
	case code.OpF32Store:
		imp.emitStore(instr)
	case code.OpF64Store:
		imp.emitStore(instr)
	case code.OpI32Store8, code.OpI32Store16:
		imp.emitStore(instr)
	case code.OpI64Store8, code.OpI64Store16, code.OpI64Store32:
		imp.emitStore(instr)

	case code.OpMemorySize:
		imp.emit(&finstruction{
			opcode: fopMemorySize,
			dest:   uint32(imp.locals + len(imp.stack)),
		}, 1)
	case code.OpMemoryGrow:
		imp.emitUnOp(instr)

	case code.OpI32Const:
		imp.pushConst(instr.Immediate)
	case code.OpI64Const:
		imp.pushConst(instr.Immediate)
	case code.OpF32Const:
		imp.pushConst(instr.Immediate)
	case code.OpF64Const:
		imp.pushConst(instr.Immediate)

	case code.OpI32Eqz:
		imp.emitUnOp(instr)
	case code.OpI32Eq, code.OpI32Ne:
		imp.emitBinOp(instr, true)
	case code.OpI32LtS, code.OpI32LtU, code.OpI32GtS, code.OpI32GtU, code.OpI32LeS, code.OpI32LeU, code.OpI32GeS, code.OpI32GeU:
		imp.emitBinOp(instr, false)

	case code.OpI64Eqz:
		imp.emitUnOp(instr)
	case code.OpI64Eq, code.OpI64Ne:
		imp.emitBinOp(instr, true)
	case code.OpI64LtS, code.OpI64LtU, code.OpI64GtS, code.OpI64GtU, code.OpI64LeS, code.OpI64LeU, code.OpI64GeS, code.OpI64GeU:
		imp.emitBinOp(instr, false)

	case code.OpF32Eq, code.OpF32Ne:
		imp.emitBinOp(instr, true)
	case code.OpF32Lt, code.OpF32Gt, code.OpF32Le, code.OpF32Ge:
		imp.emitBinOp(instr, false)

	case code.OpF64Eq, code.OpF64Ne:
		imp.emitBinOp(instr, true)
	case code.OpF64Lt, code.OpF64Gt, code.OpF64Le, code.OpF64Ge:
		imp.emitBinOp(instr, false)

	case code.OpI32Clz, code.OpI32Ctz, code.OpI32Popcnt:
		imp.emitUnOp(instr)
	case code.OpI32Add, code.OpI32Mul:
		imp.emitBinOp(instr, true)
	case code.OpI32Sub, code.OpI32DivS, code.OpI32DivU, code.OpI32RemS, code.OpI32RemU, code.OpI32And, code.OpI32Or, code.OpI32Xor, code.OpI32Shl, code.OpI32ShrS, code.OpI32ShrU:
		imp.emitBinOp(instr, false)
	case code.OpI32Rotl, code.OpI32Rotr:
		imp.emitBinOp(instr, false)

	case code.OpI64Clz, code.OpI64Ctz, code.OpI64Popcnt:
		imp.emitUnOp(instr)
	case code.OpI64Add, code.OpI64Mul:
		imp.emitBinOp(instr, true)
	case code.OpI64Sub, code.OpI64DivS, code.OpI64DivU, code.OpI64RemS, code.OpI64RemU, code.OpI64And, code.OpI64Or, code.OpI64Xor, code.OpI64Shl, code.OpI64ShrS, code.OpI64ShrU:
		imp.emitBinOp(instr, false)
	case code.OpI64Rotl, code.OpI64Rotr:
		imp.emitBinOp(instr, false)

	case code.OpF32Neg:
		imp.emitUnOp(instr)
	case code.OpF32Abs, code.OpF32Ceil, code.OpF32Floor, code.OpF32Trunc, code.OpF32Nearest, code.OpF32Sqrt:
		imp.emitUnOp(instr)
	case code.OpF32Add, code.OpF32Mul:
		imp.emitBinOp(instr, true)
	case code.OpF32Sub, code.OpF32Div:
		imp.emitBinOp(instr, false)
	case code.OpF32Min, code.OpF32Max:
		imp.emitBinOp(instr, true)
	case code.OpF32Copysign:
		imp.emitBinOp(instr, false)

	case code.OpF64Neg:
		imp.emitUnOp(instr)
	case code.OpF64Abs, code.OpF64Ceil, code.OpF64Floor, code.OpF64Trunc, code.OpF64Nearest, code.OpF64Sqrt:
		imp.emitUnOp(instr)
	case code.OpF64Add, code.OpF64Mul:
		imp.emitBinOp(instr, true)
	case code.OpF64Sub, code.OpF64Div:
		imp.emitBinOp(instr, false)
	case code.OpF64Min, code.OpF64Max:
		imp.emitBinOp(instr, true)
	case code.OpF64Copysign:
		imp.emitBinOp(instr, false)

	case code.OpI32WrapI64:
		imp.emitUnOp(instr)
	case code.OpI32TruncF32S, code.OpI32TruncF32U:
		imp.emitUnOp(instr)
	case code.OpI32TruncF64S, code.OpI32TruncF64U:
		imp.emitUnOp(instr)

	case code.OpI64ExtendI32S, code.OpI64ExtendI32U:
		imp.emitUnOp(instr)
	case code.OpI64TruncF32S, code.OpI64TruncF32U:
		imp.emitUnOp(instr)
	case code.OpI64TruncF64S, code.OpI64TruncF64U:
		imp.emitUnOp(instr)

	case code.OpF32ConvertI32S, code.OpF32ConvertI32U:
		imp.emitUnOp(instr)
	case code.OpF32ConvertI64S, code.OpF32ConvertI64U:
		imp.emitUnOp(instr)
	case code.OpF32DemoteF64:
		imp.emitUnOp(instr)

	case code.OpF64ConvertI32S, code.OpF64ConvertI32U:
		imp.emitUnOp(instr)
	case code.OpF64ConvertI64S, code.OpF64ConvertI64U:
		imp.emitUnOp(instr)
	case code.OpF64PromoteF32:
		imp.emitUnOp(instr)

	case code.OpI32ReinterpretF32:
		imp.emitUnOp(instr)
	case code.OpI64ReinterpretF64:
		imp.emitUnOp(instr)
	case code.OpF32ReinterpretI32:
		imp.emitUnOp(instr)
	case code.OpF64ReinterpretI64:
		imp.emitUnOp(instr)

	case code.OpI32Extend8S, code.OpI32Extend16S:
		imp.emitUnOp(instr)
	case code.OpI64Extend8S, code.OpI64Extend16S, code.OpI64Extend32S:
		imp.emitUnOp(instr)

	case code.OpPrefix:
		switch instr.Immediate {
		case code.OpI32TruncSatF32S:
			imp.emitUnOpF(&finstruction{opcode: fopI32TruncSatF32S})
		case code.OpI32TruncSatF32U:
			imp.emitUnOpF(&finstruction{opcode: fopI32TruncSatF32U})
		case code.OpI32TruncSatF64S:
			imp.emitUnOpF(&finstruction{opcode: fopI32TruncSatF64S})
		case code.OpI32TruncSatF64U:
			imp.emitUnOpF(&finstruction{opcode: fopI32TruncSatF64U})
		case code.OpI64TruncSatF32S:
			imp.emitUnOpF(&finstruction{opcode: fopI64TruncSatF32S})
		case code.OpI64TruncSatF32U:
			imp.emitUnOpF(&finstruction{opcode: fopI64TruncSatF32U})
		case code.OpI64TruncSatF64S:
			imp.emitUnOpF(&finstruction{opcode: fopI64TruncSatF64S})
		case code.OpI64TruncSatF64U:
			imp.emitUnOpF(&finstruction{opcode: fopI64TruncSatF64U})
		}
	}
}
