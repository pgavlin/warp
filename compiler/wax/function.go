package wax

import (
	"math"

	"github.com/pgavlin/warp/wasm"
	"github.com/pgavlin/warp/wasm/code"
	"github.com/willf/bitset"
)

// - form expression trees by stacking instructions
// - spill stack to temps at side-effecting instructions
// - control flow compiles to labeled for loops; branches compile to breaks
//     - avoids the need to predeclare + initialize temps
//     - if/else nests an if/else inside the for loop
//     - blocks that are never targeted omit the for loop + label

// block entry:
// - uses on block consume initial inputs
// - defs on block are multiple-def temps
// - output temps are allocated

// block exit:
// - uses on exit consume output values
// - defs on exit produce block outputs
// - output temps are def'd

// else is tough b/c it is both an exit and an entry!
// - uses on else consume output values
// - defs on else produce block inputs

// branches:
// - to loop continuation:
//     - uses are block inputs
// - to block/if continuation:
//     - uses are block outputs
//
// - for br_if, inputs are teed

type Function struct {
	Formatter Formatter

	Signature wasm.FunctionSig

	Locals     []wasm.ValueType
	UsedLocals []bool
	Stack      []*Use
	Temps      int

	Blocks []*Block
	Labels int

	Body []*Def

	basicBlocks []*basicBlock // only used for spill placement during import
}

type FunctionScope struct {
	code.Scope
	f *Function
}

func (s *FunctionScope) GetLocalType(localidx uint32) (wasm.ValueType, bool) {
	if localidx >= uint32(len(s.f.Locals)) {
		return 0, false
	}
	return s.f.Locals[int(localidx)], true
}

func (f *Function) Scope(module code.Scope) *FunctionScope {
	return &FunctionScope{Scope: module, f: f}
}

func ImportFunction(typeIndex uint32, signature wasm.FunctionSig, body wasm.FunctionBody, scope code.Scope, formatter Formatter) Function {
	f := NewFunction(typeIndex, signature, body, scope, formatter)
	s := f.Scope(scope)

	decoded, err := code.Decode(body.Code, s, signature.ReturnTypes)
	if err != nil {
		panic(err)
	}

	// Compile the function body into expression trees.
	for ip, instr := range decoded.Instructions {
		f.ImportInstruction(ip, instr, s)
	}

	return f
}

func NewFunction(typeIndex uint32, signature wasm.FunctionSig, body wasm.FunctionBody, scope code.Scope, formatter Formatter) Function {
	f := Function{Formatter: formatter, Signature: signature}

	// Expand locals.
	for _, t := range signature.ParamTypes {
		f.Locals = append(f.Locals, t)
	}
	for _, l := range body.Locals {
		for i := 0; i < int(l.Count); i++ {
			f.Locals = append(f.Locals, l.Type)
		}
	}
	f.UsedLocals = make([]bool, len(f.Locals))

	// Push the initial basic block.
	f.basicBlocks = append(f.basicBlocks, &basicBlock{})

	// Push the initial block.
	f.ImportInstruction(-1, code.Block(uint64(typeIndex)), f.Scope(scope))

	return f
}

func (f *Function) FinishImport() []*Def {
	// the terminal basic block must be empty.
	for _, bb := range f.basicBlocks[:len(f.basicBlocks)-1] {
		f.Body = append(f.Body, bb.body...)
		f.Body = append(f.Body, bb.terminator)
	}
	f.basicBlocks = nil
	return f.Body
}

func (f *Function) LabelTypes(idx int) []wasm.ValueType {
	dest := f.Blocks[len(f.Blocks)-idx-1]
	if dest.Entry.Instr.Opcode == code.OpLoop {
		return dest.Ins
	}
	return dest.Outs
}

func (f *Function) Unreachable() bool {
	return len(f.Blocks) > 0 && f.Blocks[len(f.Blocks)-1].Unreachable
}

func (f *Function) DropStack(until int) {
	bb := f.basicBlocks[len(f.basicBlocks)-1]
	for _, u := range f.Stack[until:] {
		bb.body = append(bb.body, &Def{
			Expression: &Expression{
				Function: f,
				Instr:    code.Drop(),
				Uses:     []*Use{u},
			},
		})
	}
	f.Stack = f.Stack[:until]
}

func (f *Function) ImportInstruction(ip int, instr code.Instruction, scope *FunctionScope) {
	type VT = wasm.ValueType

	const (
		Bool = ValueTypeBool
		I32  = wasm.ValueTypeI32
		I64  = wasm.ValueTypeI64
		F32  = wasm.ValueTypeF32
		F64  = wasm.ValueTypeF64
	)

	stackUses, stackDefs := []VT(nil), []VT(nil)
	labels := []int(nil)
	isOrdered, isBlock, isBranch, isElse, isEnd, isSelect, isUnreachable := false, false, false, false, false, false, false
	flags := Flags(0)
	usedLocals, storedLocals := bitset.BitSet{}, bitset.BitSet{}
	x := &Expression{Function: f, IP: ip, Instr: instr, basicBlock: f.basicBlocks[len(f.basicBlocks)-1]}

	switch instr.Opcode {
	case code.OpUnreachable:
		isOrdered, isUnreachable = true, true

	case code.OpNop:
		// no-op

	case code.OpBlock, code.OpLoop:
		if len(f.Blocks) > 0 {
			stackDefs, _, _ = instr.BlockType(scope)
			stackUses = stackDefs
		}
		isOrdered, isBlock = true, true
	case code.OpIf:
		stackDefs, _, _ = instr.BlockType(scope)
		stackUses = append([]VT(nil), stackDefs...)
		stackUses = append(stackUses, Bool)
		isOrdered, isBlock = true, true
	case code.OpElse:
		b := f.Blocks[len(f.Blocks)-1]
		stackDefs, stackUses = b.Ins, b.Outs
		isOrdered, isElse = true, true
	case code.OpEnd:
		b := f.Blocks[len(f.Blocks)-1]
		stackUses, stackDefs = b.Outs, b.Outs
		isOrdered, isEnd = true, true

	case code.OpBr:
		stackUses = f.LabelTypes(instr.Labelidx())
		labels = []int{int(instr.Labelidx())}
		isOrdered, isBranch, isUnreachable = true, true, true
	case code.OpBrIf:
		types := f.LabelTypes(instr.Labelidx())
		stackDefs = types
		stackUses = append([]VT(nil), types...)
		stackUses = append(stackUses, Bool)
		labels = []int{int(instr.Labelidx())}
		isOrdered, isBranch = true, true
	case code.OpBrTable:
		types := f.LabelTypes(instr.Default())
		stackUses = append([]VT(nil), types...)
		stackUses = append(stackUses, I32)
		labels = append([]int(nil), instr.Labels...)
		labels = append(labels, instr.Default())
		isOrdered, isBranch, isUnreachable = true, true, true

	case code.OpReturn:
		stackUses = f.Signature.ReturnTypes
		isOrdered, isUnreachable = true, true

	case code.OpCall:
		sig, _ := scope.GetFunctionSignature(x.Instr.Funcidx())
		stackUses, stackDefs = sig.ParamTypes, sig.ReturnTypes
		isOrdered, flags = true, FlagsLoadMem|FlagsLoadGlobal|FlagsStoreMem|FlagsStoreGlobal
	case code.OpCallIndirect:
		sig, _ := scope.GetType(x.Instr.Typeidx())
		stackUses = append([]VT(nil), sig.ParamTypes...)
		stackUses = append(stackUses, I32)
		stackDefs = sig.ReturnTypes
		isOrdered, flags = true, FlagsLoadMem|FlagsLoadGlobal|FlagsStoreMem|FlagsStoreGlobal

	case code.OpDrop:
		if !f.Unreachable() {
			stackUses = []VT{f.Stack[len(f.Stack)-1].Type}
		}
		isOrdered = true

	case code.OpSelect:
		if !f.Unreachable() {
			operandType := f.Stack[len(f.Stack)-2].Type
			stackUses = []VT{operandType, operandType, Bool}
			stackDefs = []VT{operandType}
		}
		isOrdered, isSelect = true, true

	case code.OpLocalGet:
		localidx := int(instr.Localidx())
		f.UsedLocals[localidx] = !f.Unreachable()
		stackDefs = f.Locals[localidx : localidx+1]
		usedLocals.Set(uint(localidx))
		flags = FlagsLoadLocal
	case code.OpLocalSet:
		localidx := int(instr.Localidx())
		stackUses = f.Locals[localidx : localidx+1]
		storedLocals.Set(uint(localidx))
		isOrdered, flags = true, FlagsStoreLocal
	case code.OpLocalTee:
		// Decompose tee into a set followed by a get to avoid extra temps and allow for code motion.
		f.ImportInstruction(ip, code.LocalSet(instr.Localidx()), scope)
		f.ImportInstruction(ip, code.LocalGet(instr.Localidx()), scope)
		return

	case code.OpGlobalGet:
		t, _ := scope.GetGlobalType(instr.Globalidx())
		stackDefs = []VT{t.Type}
		flags = FlagsLoadGlobal
	case code.OpGlobalSet:
		t, _ := scope.GetGlobalType(instr.Globalidx())
		stackUses = []VT{t.Type}
		isOrdered, flags = true, FlagsStoreGlobal

	case code.OpI32Load:
		stackUses = []VT{I32}
		stackDefs = []VT{I32}
		flags = FlagsLoadMem
	case code.OpI64Load:
		stackUses = []VT{I32}
		stackDefs = []VT{I64}
		flags = FlagsLoadMem
	case code.OpF32Load:
		stackUses = []VT{I32}
		stackDefs = []VT{F32}
		flags = FlagsLoadMem
	case code.OpF64Load:
		stackUses = []VT{I32}
		stackDefs = []VT{F64}
		flags = FlagsLoadMem

	case code.OpI32Load8S, code.OpI32Load8U, code.OpI32Load16S, code.OpI32Load16U:
		stackUses = []VT{I32}
		stackDefs = []VT{I32}
		flags = FlagsLoadMem

	case code.OpI64Load8S, code.OpI64Load8U, code.OpI64Load16S, code.OpI64Load16U, code.OpI64Load32S, code.OpI64Load32U:
		stackUses = []VT{I32}
		stackDefs = []VT{I64}
		flags = FlagsLoadMem

	case code.OpI32Store:
		stackUses = []VT{I32, I32}
		isOrdered, flags = true, FlagsStoreMem
	case code.OpI64Store:
		stackUses = []VT{I64, I32}
		isOrdered, flags = true, FlagsStoreMem
	case code.OpF32Store:
		stackUses = []VT{F32, I32}
		isOrdered, flags = true, FlagsStoreMem
	case code.OpF64Store:
		stackUses = []VT{F64, I32}
		isOrdered, flags = true, FlagsStoreMem

	case code.OpI32Store8, code.OpI32Store16:
		stackUses = []VT{I32, I32}
		isOrdered, flags = true, FlagsStoreMem

	case code.OpI64Store8, code.OpI64Store16, code.OpI64Store32:
		stackUses = []VT{I64, I32}
		isOrdered, flags = true, FlagsStoreMem

	case code.OpMemorySize:
		stackDefs = []VT{I32}
		flags = FlagsLoadMem
	case code.OpMemoryGrow:
		stackUses = []VT{I32}
		stackDefs = []VT{I32}
		isOrdered, flags = true, FlagsStoreMem

	case code.OpI32Const:
		stackDefs = []VT{I32}
	case code.OpI64Const:
		stackDefs = []VT{I64}
	case code.OpF32Const:
		stackDefs = []VT{F32}
	case code.OpF64Const:
		stackDefs = []VT{F64}

	case code.OpI32Eqz:
		stackUses = []VT{I32}
		stackDefs = []VT{Bool}
	case code.OpI32Eq, code.OpI32Ne, code.OpI32LtS, code.OpI32LtU, code.OpI32GtS, code.OpI32GtU, code.OpI32LeS, code.OpI32LeU, code.OpI32GeS, code.OpI32GeU:
		stackUses = []VT{I32, I32}
		stackDefs = []VT{Bool}

	case code.OpI64Eqz:
		stackUses = []VT{I64}
		stackDefs = []VT{Bool}
	case code.OpI64Eq, code.OpI64Ne, code.OpI64LtS, code.OpI64LtU, code.OpI64GtS, code.OpI64GtU, code.OpI64LeS, code.OpI64LeU, code.OpI64GeS, code.OpI64GeU:
		stackUses = []VT{I64, I64}
		stackDefs = []VT{Bool}

	case code.OpF32Eq, code.OpF32Ne, code.OpF32Lt, code.OpF32Gt, code.OpF32Le, code.OpF32Ge:
		stackUses = []VT{F32, F32}
		stackDefs = []VT{Bool}

	case code.OpF64Eq, code.OpF64Ne, code.OpF64Lt, code.OpF64Gt, code.OpF64Le, code.OpF64Ge:
		stackUses = []VT{F64, F64}
		stackDefs = []VT{Bool}

	case code.OpI32Clz, code.OpI32Ctz, code.OpI32Popcnt:
		stackUses = []VT{I32}
		stackDefs = []VT{I32}
	case code.OpI32Add, code.OpI32Sub, code.OpI32Mul, code.OpI32DivS, code.OpI32DivU, code.OpI32RemS, code.OpI32RemU, code.OpI32And, code.OpI32Or, code.OpI32Xor, code.OpI32Shl, code.OpI32ShrS, code.OpI32ShrU:
		stackUses = []VT{I32, I32}
		stackDefs = []VT{I32}
	case code.OpI32Rotl, code.OpI32Rotr:
		stackUses = []VT{I32, I32}
		stackDefs = []VT{I32}

	case code.OpI64Clz, code.OpI64Ctz, code.OpI64Popcnt:
		stackUses = []VT{I64}
		stackDefs = []VT{I64}
	case code.OpI64Add, code.OpI64Sub, code.OpI64Mul, code.OpI64DivS, code.OpI64DivU, code.OpI64RemS, code.OpI64RemU, code.OpI64And, code.OpI64Or, code.OpI64Xor, code.OpI64Shl, code.OpI64ShrS, code.OpI64ShrU:
		stackUses = []VT{I64, I64}
		stackDefs = []VT{I64}
	case code.OpI64Rotl, code.OpI64Rotr:
		stackUses = []VT{I64, I64}
		stackDefs = []VT{I64}

	case code.OpF32Neg:
		stackUses = []VT{F32}
		stackDefs = []VT{F32}
	case code.OpF32Abs, code.OpF32Ceil, code.OpF32Floor, code.OpF32Trunc, code.OpF32Nearest, code.OpF32Sqrt:
		stackUses = []VT{F32}
		stackDefs = []VT{F32}
	case code.OpF32Add, code.OpF32Sub, code.OpF32Mul, code.OpF32Div:
		stackUses = []VT{F32, F32}
		stackDefs = []VT{F32}
	case code.OpF32Min, code.OpF32Max, code.OpF32Copysign:
		stackUses = []VT{F32, F32}
		stackDefs = []VT{F32}

	case code.OpF64Neg:
		stackUses = []VT{F64}
		stackDefs = []VT{F64}
	case code.OpF64Abs, code.OpF64Ceil, code.OpF64Floor, code.OpF64Trunc, code.OpF64Nearest, code.OpF64Sqrt:
		stackUses = []VT{F64}
		stackDefs = []VT{F64}
	case code.OpF64Add, code.OpF64Sub, code.OpF64Mul, code.OpF64Div:
		stackUses = []VT{F64, F64}
		stackDefs = []VT{F64}
	case code.OpF64Min, code.OpF64Max, code.OpF64Copysign:
		stackUses = []VT{F64, F64}
		stackDefs = []VT{F64}

	case code.OpI32WrapI64:
		stackUses = []VT{I64}
		stackDefs = []VT{I32}
	case code.OpI32TruncF32S, code.OpI32TruncF32U:
		stackUses = []VT{F32}
		stackDefs = []VT{I32}
	case code.OpI32TruncF64S, code.OpI32TruncF64U:
		stackUses = []VT{F64}
		stackDefs = []VT{I32}

	case code.OpI64ExtendI32S, code.OpI64ExtendI32U:
		stackUses = []VT{I32}
		stackDefs = []VT{I64}
	case code.OpI64TruncF32S, code.OpI64TruncF32U:
		stackUses = []VT{F32}
		stackDefs = []VT{I64}
	case code.OpI64TruncF64S, code.OpI64TruncF64U:
		stackUses = []VT{F64}
		stackDefs = []VT{I64}

	case code.OpF32ConvertI32S, code.OpF32ConvertI32U:
		stackUses = []VT{I32}
		stackDefs = []VT{F32}
	case code.OpF32ConvertI64S, code.OpF32ConvertI64U:
		stackUses = []VT{I64}
		stackDefs = []VT{F32}
	case code.OpF32DemoteF64:
		stackUses = []VT{F64}
		stackDefs = []VT{F32}

	case code.OpF64ConvertI32S, code.OpF64ConvertI32U:
		stackUses = []VT{I32}
		stackDefs = []VT{F64}
	case code.OpF64ConvertI64S, code.OpF64ConvertI64U:
		stackUses = []VT{I64}
		stackDefs = []VT{F64}
	case code.OpF64PromoteF32:
		stackUses = []VT{F32}
		stackDefs = []VT{F64}

	case code.OpI32ReinterpretF32:
		stackUses = []VT{F32}
		stackDefs = []VT{I32}
	case code.OpI64ReinterpretF64:
		stackUses = []VT{F64}
		stackDefs = []VT{I64}
	case code.OpF32ReinterpretI32:
		stackUses = []VT{I32}
		stackDefs = []VT{F32}
	case code.OpF64ReinterpretI64:
		stackUses = []VT{I64}
		stackDefs = []VT{F64}

	case code.OpI32Extend8S, code.OpI32Extend16S:
		stackUses = []VT{I32}
		stackDefs = []VT{I32}
	case code.OpI64Extend8S, code.OpI64Extend16S, code.OpI64Extend32S:
		stackUses = []VT{I64}
		stackDefs = []VT{I64}

	case code.OpPrefix:
		switch x.Instr.Immediate {
		case code.OpI32TruncSatF32S, code.OpI32TruncSatF32U:
			stackUses = []VT{F32}
			stackDefs = []VT{I32}
		case code.OpI32TruncSatF64S, code.OpI32TruncSatF64U:
			stackUses = []VT{F64}
			stackDefs = []VT{I32}
		case code.OpI64TruncSatF32S, code.OpI64TruncSatF32U:
			stackUses = []VT{F32}
			stackDefs = []VT{I64}
		case code.OpI64TruncSatF64S, code.OpI64TruncSatF64U:
			stackUses = []VT{F64}
			stackDefs = []VT{I64}
		}
	}

	if f.Unreachable() {
		stackUses = nil

		switch {
		case isBlock:
			f.Blocks = append(f.Blocks, &Block{NeverReachable: true, Unreachable: true})
			return
		case isElse:
			b := f.Blocks[len(f.Blocks)-1]
			if b.NeverReachable {
				return
			}
			b.Unreachable = false
		case isEnd:
			b := f.Blocks[len(f.Blocks)-1]
			if b.NeverReachable {
				f.Blocks = f.Blocks[:len(f.Blocks)-1]
				return
			}

			// OK
		default:
			return
		}
	}

	// Store expression flags.
	x.Flags = flags

	// Pop uses.
	if len(stackUses) > 0 {
		firstUse := len(f.Stack) - len(stackUses)
		x.Uses = make(Uses, len(stackUses))
		for i, u := range f.Stack[firstUse:] {
			want := stackUses[i]
			switch {
			case want == Bool && u.Type != Bool:
				u = boolConvertI32(u)
			case u.Type == Bool && want != Bool:
				u = i32ConvertBool(u)
			}
			x.Uses[i] = u
			flags |= u.AllFlags
			usedLocals.InPlaceUnion(&u.Locals)
		}
		f.Stack = f.Stack[:firstUse]

		// Evaluate constant expressions.
		if v, ok := evaluate(x); ok {
			switch stackDefs[0] {
			case ValueTypeBool:
				x.Instr.Opcode, x.Instr.Immediate, x.Flags = PseudoBoolConst, v, x.Flags|FlagsPseudo
			case wasm.ValueTypeI32:
				x.Instr = code.I32Const(int32(v))
			case wasm.ValueTypeI64:
				x.Instr = code.I64Const(int64(v))
			case wasm.ValueTypeF32:
				x.Instr = code.F32Const(math.Float32frombits(uint32(v)))
			case wasm.ValueTypeF64:
				x.Instr = code.F64Const(math.Float64frombits(v))
			}
		}
	}

	if isUnreachable && len(f.Blocks) > 0 {
		// clear the stack
		b := f.Blocks[len(f.Blocks)-1]
		b.Unreachable = true
		f.DropStack(b.StackHeight)
	}

	// If this expression is ordered, spill the stack to temps and append the expression to
	// the function body.
	var d *Def
	if isOrdered {
		if !isSelect {
			// spill the stack
			for _, u := range f.Stack {
				if !u.CanMoveAfter(flags, storedLocals) {
					d := &Def{
						Expression: u.X,
						Types:      []wasm.ValueType{u.Type},
					}
					d.Temp, f.Temps = len(f.Locals)+f.Temps, f.Temps+1
					u.X.basicBlock.body = append(u.X.basicBlock.body, d)

					u.X, u.Temp = nil, d.Temp
				}
			}
		}

		d = &Def{Expression: x, Types: stackDefs}
		if len(stackDefs) != 0 && !isElse && !isEnd {
			d.Temp, f.Temps = len(f.Locals)+f.Temps, f.Temps+len(stackDefs)
		}

		// if this instruction terminates a basic block, update the block's terminator and push a new block
		if isBlock || isElse || isEnd || isBranch || x.Instr.Opcode == code.OpReturn {
			x.basicBlock.terminator = d
			f.basicBlocks = append(f.basicBlocks, &basicBlock{})
		} else {
			x.basicBlock.body = append(x.basicBlock.body, d)
		}
	}

	// Update blocks and branch targets.
	switch {
	case isBlock:
		ins, outs, _ := instr.BlockType(scope)

		b := &Block{
			Entry:       x,
			Label:       f.Labels,
			StackHeight: len(f.Stack),
			Ins:         ins,
			Outs:        outs,
			InTemp:      d.Temp,
		}
		if len(outs) != 0 {
			b.OutTemp, f.Temps = len(f.Locals)+f.Temps, f.Temps+len(outs)
		}

		d.Block, f.Blocks, f.Labels = b, append(f.Blocks, b), f.Labels+1
	case isBranch:
		for _, l := range labels {
			dest := f.Blocks[len(f.Blocks)-l-1]
			dest.BranchTarget = true
			d.BranchTargets = append(d.BranchTargets, dest)
		}
	case isElse:
		b := f.Blocks[len(f.Blocks)-1]
		b.Else = x
		d.Block = b
		d.Temp = b.InTemp
	case isEnd:
		b := f.Blocks[len(f.Blocks)-1]
		b.End = x
		d.Block = b
		d.Temp = b.OutTemp
		f.Blocks = f.Blocks[:len(f.Blocks)-1]
	}

	// Push defs.
	switch len(stackDefs) {
	case 0:
		// OK
	case 1:
		if d == nil {
			f.Stack = append(f.Stack, &Use{Function: f, Type: stackDefs[0], X: x, AllFlags: flags, Locals: usedLocals})
		} else {
			f.Stack = append(f.Stack, &Use{Function: f, Type: stackDefs[0], Temp: d.Temp})
		}
	default:
		if d == nil {
			d = &Def{Expression: x, Types: stackDefs}
			d.Temp, f.Temps = len(f.Locals)+f.Temps, f.Temps+len(stackDefs)
			x.basicBlock.body = append(x.basicBlock.body, d)
		}
		for i, t := range stackDefs {
			f.Stack = append(f.Stack, &Use{Function: f, Type: t, Temp: d.Temp + i})
		}
	}
}

func boolConvertI32(u *Use) *Use {
	zero := UseExpression(wasm.ValueTypeI32, &Expression{
		Function: u.Function,
		Instr:    code.I32Const(0),
	})
	return UseExpression(ValueTypeBool, &Expression{
		Function: u.Function,
		Instr:    code.I32Ne(),
		Uses:     Uses{u, zero},
	})
}

func i32ConvertBool(u *Use) *Use {
	return UseExpression(wasm.ValueTypeI32, Pseudo(u.Function, PseudoI32ConvertBool, 0, 0, u))
}
