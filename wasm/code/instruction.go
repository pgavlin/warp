package code

import (
	"fmt"
	"io"
	"math"
	"strings"

	"github.com/pgavlin/warp/wasm"
)

type Instruction struct {
	Opcode    byte   `json:"opcode"`
	Immediate uint64 `json:"immediate"`
	Labels    []int  `json:"labels"`
}

func (i *Instruction) Continuation() int {
	return i.Labels[0]
}

func (i *Instruction) Else() int {
	return i.Labels[1]
}

func (i *Instruction) StackHeight() int {
	return int((i.Immediate & StackHeightMask) >> 32)
}

func (i *Instruction) Default() int {
	return int(i.Immediate)
}

func (i *Instruction) Labelidx() int {
	return int(i.Immediate)
}

func (i *Instruction) Funcidx() uint32 {
	return uint32(i.Immediate)
}

func (i *Instruction) Localidx() uint32 {
	return uint32(i.Immediate)
}

func (i *Instruction) Globalidx() uint32 {
	return uint32(i.Immediate)
}

func (i *Instruction) Typeidx() uint32 {
	return uint32(i.Immediate)
}

func (i *Instruction) Memarg() (offset uint32, align uint32) {
	return uint32(i.Immediate), uint32(i.Immediate >> 32)
}

func (i *Instruction) Offset() uint32 {
	return uint32(i.Immediate)
}

func (i *Instruction) I32() int32 {
	return int32(i.Immediate)
}

func (i *Instruction) I64() int64 {
	return int64(i.Immediate)
}

func (i *Instruction) F32() float32 {
	return math.Float32frombits(uint32(i.Immediate))
}

func (i *Instruction) F64() float64 {
	return math.Float64frombits(uint64(i.Immediate))
}

func (i *Instruction) BlockType(scope Scope) (in, out []wasm.ValueType, ok bool) {
	switch i.Immediate & BlockTypeMask {
	case BlockTypeEmpty:
		return nil, nil, true
	case BlockTypeI32:
		return nil, []wasm.ValueType{wasm.ValueTypeI32}, true
	case BlockTypeI64:
		return nil, []wasm.ValueType{wasm.ValueTypeI64}, true
	case BlockTypeF32:
		return nil, []wasm.ValueType{wasm.ValueTypeF32}, true
	case BlockTypeF64:
		return nil, []wasm.ValueType{wasm.ValueTypeF64}, true
	default:
		sig, ok := scope.GetType(i.Typeidx())
		if !ok {
			return nil, nil, false
		}
		return sig.ParamTypes, sig.ReturnTypes, true
	}
}

func (i *Instruction) Stack(scope Scope) (pop, push int) {
	switch i.Opcode {
	case OpIf, OpBrIf, OpBrTable, OpDrop, OpLocalSet, OpGlobalSet:
		return 1, 0

	case OpI32Store, OpI64Store, OpF32Store, OpF64Store, OpI32Store8, OpI32Store16, OpI64Store8, OpI64Store16, OpI64Store32:
		return 2, 0

	case OpI32Load, OpI64Load, OpF32Load, OpF64Load,
		OpI32Load8S, OpI32Load8U, OpI32Load16S, OpI32Load16U,
		OpI64Load8S, OpI64Load8U, OpI64Load16S, OpI64Load16U, OpI64Load32S, OpI64Load32U:
		return 1, 1

	case OpLocalGet, OpGlobalGet, OpMemorySize, OpI32Const, OpI64Const, OpF32Const, OpF64Const:
		return 0, 1

	case OpLocalTee, OpMemoryGrow, OpI32Eqz, OpI64Eqz, OpI32Clz, OpI32Ctz, OpI32Popcnt, OpI64Clz, OpI64Ctz, OpI64Popcnt,
		OpF32Abs, OpF32Neg, OpF32Ceil, OpF32Floor, OpF32Trunc, OpF32Nearest, OpF32Sqrt,
		OpF64Abs, OpF64Neg, OpF64Ceil, OpF64Floor, OpF64Trunc, OpF64Nearest, OpF64Sqrt,
		OpI32WrapI64, OpI32TruncF32S, OpI32TruncF32U, OpI32TruncF64S, OpI32TruncF64U,
		OpI64ExtendI32S, OpI64ExtendI32U, OpI64TruncF32S, OpI64TruncF32U, OpI64TruncF64S, OpI64TruncF64U,
		OpF32ConvertI32S, OpF32ConvertI32U, OpF32ConvertI64S, OpF32ConvertI64U, OpF32DemoteF64,
		OpF64ConvertI32S, OpF64ConvertI32U, OpF64ConvertI64S, OpF64ConvertI64U, OpF64PromoteF32,
		OpI32ReinterpretF32, OpI64ReinterpretF64, OpF32ReinterpretI32, OpF64ReinterpretI64,
		OpI32Extend8S, OpI32Extend16S, OpI64Extend8S, OpI64Extend16S, OpI64Extend32S:
		return 1, 1

	case OpI32Eq, OpI32Ne, OpI32LtS, OpI32LtU, OpI32GtS, OpI32GtU, OpI32LeS, OpI32LeU, OpI32GeS, OpI32GeU,
		OpI64Eq, OpI64Ne, OpI64LtS, OpI64LtU, OpI64GtS, OpI64GtU, OpI64LeS, OpI64LeU, OpI64GeS, OpI64GeU,
		OpF32Eq, OpF32Ne, OpF32Lt, OpF32Gt, OpF32Le, OpF32Ge, OpF64Eq, OpF64Ne, OpF64Lt, OpF64Gt, OpF64Le, OpF64Ge,
		OpI32Add, OpI32Sub, OpI32Mul, OpI32DivS, OpI32DivU, OpI32RemS, OpI32RemU, OpI32And, OpI32Or, OpI32Xor, OpI32Shl, OpI32ShrS, OpI32ShrU, OpI32Rotl, OpI32Rotr,
		OpI64Add, OpI64Sub, OpI64Mul, OpI64DivS, OpI64DivU, OpI64RemS, OpI64RemU, OpI64And, OpI64Or, OpI64Xor, OpI64Shl, OpI64ShrS, OpI64ShrU, OpI64Rotl, OpI64Rotr,
		OpF32Add, OpF32Sub, OpF32Mul, OpF32Div, OpF32Min, OpF32Max, OpF32Copysign,
		OpF64Add, OpF64Sub, OpF64Mul, OpF64Div, OpF64Min, OpF64Max, OpF64Copysign:
		return 2, 1

	case OpSelect:
		return 3, 1

	case OpCall:
		sig, _ := scope.GetFunctionSignature(i.Funcidx())
		return len(sig.ParamTypes), len(sig.ReturnTypes)

	case OpCallIndirect:
		sig, _ := scope.GetType(i.Typeidx())
		return len(sig.ParamTypes) + 1, len(sig.ReturnTypes)

	case OpPrefix:
		switch i.Immediate {
		case OpI32TruncSatF32S, OpI32TruncSatF32U, OpI32TruncSatF64S, OpI32TruncSatF64U, OpI64TruncSatF32S, OpI64TruncSatF32U, OpI64TruncSatF64S, OpI64TruncSatF64U:
			return 1, 1
		}
	}

	return 0, 0
}

func (i *Instruction) Types(scope Scope) (pop, push []wasm.ValueType) {
	type Pop = []wasm.ValueType
	type Push = []wasm.ValueType

	const (
		I32 = wasm.ValueTypeI32
		I64 = wasm.ValueTypeI64
		F32 = wasm.ValueTypeF32
		F64 = wasm.ValueTypeF64
	)

	switch i.Opcode {
	case OpIf, OpBrIf, OpBrTable:
		return Pop{I32}, nil

	case OpReturn:
		return nil, nil

	case OpCall:
		sig, _ := scope.GetFunctionSignature(i.Funcidx())
		return sig.ParamTypes, sig.ReturnTypes
	case OpCallIndirect:
		sig, _ := scope.GetType(i.Typeidx())
		return sig.ParamTypes, sig.ReturnTypes

	case OpDrop:
		return Pop{wasm.ValueTypeT}, nil

	case OpSelect:
		return Pop{wasm.ValueTypeT, wasm.ValueTypeT, wasm.ValueTypeI32}, Push{wasm.ValueTypeT}

	case OpLocalGet:
		type_, _ := scope.GetLocalType(i.Localidx())
		return nil, Push{type_}
	case OpLocalSet:
		type_, _ := scope.GetLocalType(i.Localidx())
		return Pop{type_}, nil
	case OpLocalTee:
		type_, _ := scope.GetLocalType(i.Localidx())
		return Pop{type_}, Push{type_}

	case OpGlobalGet:
		type_, _ := scope.GetGlobalType(i.Globalidx())
		return nil, Push{type_.Type}
	case OpGlobalSet:
		type_, _ := scope.GetGlobalType(i.Globalidx())
		return Pop{type_.Type}, nil

	case OpI32Load:
		return Pop{I32}, Push{I32}
	case OpI64Load:
		return Pop{I32}, Push{I64}
	case OpF32Load:
		return Pop{I32}, Push{F32}
	case OpF64Load:
		return Pop{I32}, Push{F64}

	case OpI32Load8S, OpI32Load8U, OpI32Load16S, OpI32Load16U:
		return Pop{I32}, Push{I32}

	case OpI64Load8S, OpI64Load8U, OpI64Load16S, OpI64Load16U, OpI64Load32S, OpI64Load32U:
		return Pop{I32}, Push{I64}

	case OpI32Store:
		return Pop{I32, I32}, nil
	case OpI64Store:
		return Pop{I32, I64}, nil
	case OpF32Store:
		return Pop{I32, F32}, nil
	case OpF64Store:
		return Pop{I32, F64}, nil

	case OpI32Store8, OpI32Store16:
		return Pop{I32, I32}, nil

	case OpI64Store8, OpI64Store16, OpI64Store32:
		return Pop{I32, I64}, nil

	case OpMemorySize:
		return nil, Push{I32}
	case OpMemoryGrow:
		return Pop{I32}, Push{I32}

	case OpI32Const:
		return nil, Push{I32}
	case OpI64Const:
		return nil, Push{I64}
	case OpF32Const:
		return nil, Push{F32}
	case OpF64Const:
		return nil, Push{F64}

	case OpI32Eqz:
		return Pop{I32}, Push{I32}
	case OpI32Eq, OpI32Ne, OpI32LtS, OpI32LtU, OpI32GtS, OpI32GtU, OpI32LeS, OpI32LeU, OpI32GeS, OpI32GeU:
		return Pop{I32, I32}, Push{I32}

	case OpI64Eqz:
		return Pop{I64}, Push{I32}
	case OpI64Eq, OpI64Ne, OpI64LtS, OpI64LtU, OpI64GtS, OpI64GtU, OpI64LeS, OpI64LeU, OpI64GeS, OpI64GeU:
		return Pop{I64, I64}, Push{I32}

	case OpF32Eq, OpF32Ne, OpF32Lt, OpF32Gt, OpF32Le, OpF32Ge:
		return Pop{F32, F32}, Push{I32}

	case OpF64Eq, OpF64Ne, OpF64Lt, OpF64Gt, OpF64Le, OpF64Ge:
		return Pop{F64, F64}, Push{I32}

	case OpI32Clz, OpI32Ctz, OpI32Popcnt:
		return Pop{I32}, Push{I32}
	case OpI32Add, OpI32Sub, OpI32Mul, OpI32DivS, OpI32DivU, OpI32RemS, OpI32RemU, OpI32And, OpI32Or, OpI32Xor, OpI32Shl, OpI32ShrS, OpI32ShrU:
		return Pop{I32, I32}, Push{I32}
	case OpI32Rotl, OpI32Rotr:
		return Pop{I32, I32}, Push{I32}

	case OpI64Clz, OpI64Ctz, OpI64Popcnt:
		return Pop{I64}, Push{I64}
	case OpI64Add, OpI64Sub, OpI64Mul, OpI64DivS, OpI64DivU, OpI64RemS, OpI64RemU, OpI64And, OpI64Or, OpI64Xor, OpI64Shl, OpI64ShrS, OpI64ShrU:
		return Pop{I64, I64}, Push{I64}
	case OpI64Rotl, OpI64Rotr:
		return Pop{I64, I64}, Push{I64}

	case OpF32Neg:
		return Pop{F32}, Push{F32}
	case OpF32Abs, OpF32Ceil, OpF32Floor, OpF32Trunc, OpF32Nearest, OpF32Sqrt:
		return Pop{F32}, Push{F32}
	case OpF32Add, OpF32Sub, OpF32Mul, OpF32Div:
		return Pop{F32, F32}, Push{F32}
	case OpF32Min, OpF32Max, OpF32Copysign:
		return Pop{F32, F32}, Push{F32}

	case OpF64Neg:
		return Pop{F64}, Push{F64}
	case OpF64Abs, OpF64Ceil, OpF64Floor, OpF64Trunc, OpF64Nearest, OpF64Sqrt:
		return Pop{F64}, Push{F64}
	case OpF64Add, OpF64Sub, OpF64Mul, OpF64Div:
		return Pop{F64, F64}, Push{F64}
	case OpF64Min, OpF64Max, OpF64Copysign:
		return Pop{F64, F64}, Push{F64}

	case OpI32WrapI64:
		return Pop{I64}, Push{I32}
	case OpI32TruncF32S, OpI32TruncF32U:
		return Pop{F32}, Push{I32}
	case OpI32TruncF64S, OpI32TruncF64U:
		return Pop{F64}, Push{I32}

	case OpI64ExtendI32S, OpI64ExtendI32U:
		return Pop{I32}, Push{I64}
	case OpI64TruncF32S, OpI64TruncF32U:
		return Pop{F32}, Push{I64}
	case OpI64TruncF64S, OpI64TruncF64U:
		return Pop{F64}, Push{I64}

	case OpF32ConvertI32S, OpF32ConvertI32U:
		return Pop{I32}, Push{F32}
	case OpF32ConvertI64S, OpF32ConvertI64U:
		return Pop{I64}, Push{F32}
	case OpF32DemoteF64:
		return Pop{F64}, Push{F32}

	case OpF64ConvertI32S, OpF64ConvertI32U:
		return Pop{I32}, Push{F64}
	case OpF64ConvertI64S, OpF64ConvertI64U:
		return Pop{I64}, Push{F64}
	case OpF64PromoteF32:
		return Pop{F32}, Push{F64}

	case OpI32ReinterpretF32:
		return Pop{F32}, Push{I32}
	case OpI64ReinterpretF64:
		return Pop{F64}, Push{I64}
	case OpF32ReinterpretI32:
		return Pop{I32}, Push{F32}
	case OpF64ReinterpretI64:
		return Pop{I64}, Push{F64}

	case OpI32Extend8S, OpI32Extend16S:
		return Pop{I32}, Push{I32}
	case OpI64Extend8S, OpI64Extend16S, OpI64Extend32S:
		return Pop{I64}, Push{I64}

	case OpPrefix:
		switch i.Immediate {
		case OpI32TruncSatF32S, OpI32TruncSatF32U:
			return Pop{F32}, Push{I32}
		case OpI32TruncSatF64S, OpI32TruncSatF64U:
			return Pop{F64}, Push{I32}
		case OpI64TruncSatF32S, OpI64TruncSatF32U:
			return Pop{F32}, Push{I64}
		case OpI64TruncSatF64S, OpI64TruncSatF64U:
			return Pop{F64}, Push{I64}
		}
	}

	return
}

func (i *Instruction) Encode(w io.Writer) error {
	return encodeInstruction(w, *i)
}

func (i *Instruction) Decode(r io.Reader) error {
	instr, err := decodeSingleInstruction(r)
	if err != nil {
		return err
	}
	*i = instr
	return nil
}

func memarg(offset, align uint32) uint64 {
	return uint64(align)<<32 | uint64(offset)
}

func (i *Instruction) blockString(op string) string {
	switch i.Immediate {
	case BlockTypeEmpty:
		return op
	case BlockTypeI32:
		return fmt.Sprintf("%s (result i32)", op)
	case BlockTypeI64:
		return fmt.Sprintf("%s (result i64)", op)
	case BlockTypeF32:
		return fmt.Sprintf("%s (result f32)", op)
	case BlockTypeF64:
		return fmt.Sprintf("%s (result f64)", op)
	default:
		return fmt.Sprintf("%s (type %v)", op, i.Typeidx())
	}
}

func (i *Instruction) memString(op string) string {
	var b strings.Builder
	b.WriteString(op)
	offset, align := i.Memarg()
	if offset != 0 {
		fmt.Fprintf(&b, " offset=%v", offset)
	}
	if align != 0 {
		fmt.Fprintf(&b, " align=%v", align)
	}
	return b.String()
}

func (i *Instruction) String() string {
	switch i.Opcode {
	case OpBlock, OpLoop, OpIf:
		return i.blockString(i.OpString())
	case OpBr, OpBrIf:
		return fmt.Sprintf("%s %d", i.OpString(), i.Labelidx())
	case OpBrTable:
		var b strings.Builder

		b.WriteString("br_table")
		for _, l := range i.Labels {
			fmt.Fprintf(&b, " %d", l)
		}
		fmt.Fprintf(&b, " %d", i.Labelidx())
		return b.String()
	case OpCall:
		return fmt.Sprintf("call %d", i.Funcidx())
	case OpCallIndirect:
		return fmt.Sprintf("call_indirect (type %v)", i.Typeidx())
	case OpLocalGet, OpLocalSet, OpLocalTee:
		return fmt.Sprintf("%s %v", i.OpString(), i.Localidx())
	case OpGlobalGet, OpGlobalSet:
		return fmt.Sprintf("%s %v", i.OpString(), i.Globalidx())
	case OpI32Load, OpI64Load, OpF32Load, OpF64Load, OpI32Load8S, OpI32Load8U, OpI32Load16S, OpI32Load16U, OpI64Load8S, OpI64Load8U, OpI64Load16S, OpI64Load16U, OpI64Load32S, OpI64Load32U, OpI32Store, OpI64Store, OpF32Store, OpF64Store, OpI32Store8, OpI32Store16, OpI64Store8, OpI64Store16, OpI64Store32:
		return i.memString(i.OpString())
	case OpI32Const:
		return fmt.Sprintf("i32.const %d", i.I32())
	case OpI64Const:
		return fmt.Sprintf("i64.const %d", i.I64())
	case OpF32Const:
		return fmt.Sprintf("f32.const %g", i.F32())
	case OpF64Const:
		return fmt.Sprintf("f64.const %g", i.F64())
	default:
		return i.OpString()
	}
}

func (i *Instruction) OpString() string {
	switch i.Opcode {
	case OpUnreachable:
		return "unreachable"
	case OpNop:
		return "nop"
	case OpBlock:
		return "block"
	case OpLoop:
		return "loop"
	case OpIf:
		return "if"
	case OpElse:
		return "else"
	case OpEnd:
		return "end"
	case OpBr:
		return "br"
	case OpBrIf:
		return "br_if"
	case OpBrTable:
		return "br_table"
	case OpReturn:
		return "return"
	case OpCall:
		return "call"
	case OpCallIndirect:
		return "call_indirect"
	case OpDrop:
		return "drop"
	case OpSelect:
		return "select"
	case OpLocalGet:
		return "local.get"
	case OpLocalSet:
		return "local.set"
	case OpLocalTee:
		return "local.tee"
	case OpGlobalGet:
		return "global.get"
	case OpGlobalSet:
		return "global.set"
	case OpI32Load:
		return "i32.load"
	case OpI64Load:
		return "i64.load"
	case OpF32Load:
		return "f32.load"
	case OpF64Load:
		return "f64.load"
	case OpI32Load8S:
		return "i32.load8_s"
	case OpI32Load8U:
		return "i32.load8_u"
	case OpI32Load16S:
		return "i32.load16_s"
	case OpI32Load16U:
		return "i32.load16_u"
	case OpI64Load8S:
		return "i64.load8_s"
	case OpI64Load8U:
		return "i64.load8_u"
	case OpI64Load16S:
		return "i64.load16_s"
	case OpI64Load16U:
		return "i64.load16_u"
	case OpI64Load32S:
		return "i64.load32_s"
	case OpI64Load32U:
		return "i64.load32_u"
	case OpI32Store:
		return "i32.store"
	case OpI64Store:
		return "i64.store"
	case OpF32Store:
		return "f32.store"
	case OpF64Store:
		return "f64.store"
	case OpI32Store8:
		return "i32.store8"
	case OpI32Store16:
		return "i32.store16"
	case OpI64Store8:
		return "i64.store8"
	case OpI64Store16:
		return "i64.store16"
	case OpI64Store32:
		return "i64.store32"
	case OpMemorySize:
		return "memory.size"
	case OpMemoryGrow:
		return "memory.grow"
	case OpI32Const:
		return "i32.const"
	case OpI64Const:
		return "i64.const"
	case OpF32Const:
		return "f32.const"
	case OpF64Const:
		return "f64.const"
	case OpI32Eqz:
		return "i32.eqz"
	case OpI32Eq:
		return "i32.eq"
	case OpI32Ne:
		return "i32.ne"
	case OpI32LtS:
		return "i32.lt_s"
	case OpI32LtU:
		return "i32.lt_u"
	case OpI32GtS:
		return "i32.gt_s"
	case OpI32GtU:
		return "i32.gt_u"
	case OpI32LeS:
		return "i32.le_s"
	case OpI32LeU:
		return "i32.le_u"
	case OpI32GeS:
		return "i32.ge_s"
	case OpI32GeU:
		return "i32.ge_u"
	case OpI64Eqz:
		return "i64.eqz"
	case OpI64Eq:
		return "i64.eq"
	case OpI64Ne:
		return "i64.ne"
	case OpI64LtS:
		return "i64.lt_s"
	case OpI64LtU:
		return "i64.lt_u"
	case OpI64GtS:
		return "i64.gt_s"
	case OpI64GtU:
		return "i64.gt_u"
	case OpI64LeS:
		return "i64.le_s"
	case OpI64LeU:
		return "i64.le_u"
	case OpI64GeS:
		return "i64.ge_s"
	case OpI64GeU:
		return "i64.ge_u"
	case OpF32Eq:
		return "f32.eq"
	case OpF32Ne:
		return "f32.ne"
	case OpF32Lt:
		return "f32.lt"
	case OpF32Gt:
		return "f32.gt"
	case OpF32Le:
		return "f32.le"
	case OpF32Ge:
		return "f32.ge"
	case OpF64Eq:
		return "f64.eq"
	case OpF64Ne:
		return "f64.ne"
	case OpF64Lt:
		return "f64.lt"
	case OpF64Gt:
		return "f64.gt"
	case OpF64Le:
		return "f64.le"
	case OpF64Ge:
		return "f64.ge"
	case OpI32Clz:
		return "i32.clz"
	case OpI32Ctz:
		return "i32.ctz"
	case OpI32Popcnt:
		return "i32.popcnt"
	case OpI32Add:
		return "i32.add"
	case OpI32Sub:
		return "i32.sub"
	case OpI32Mul:
		return "i32.mul"
	case OpI32DivS:
		return "i32.div_s"
	case OpI32DivU:
		return "i32.div_u"
	case OpI32RemS:
		return "i32.rem_s"
	case OpI32RemU:
		return "i32.rem_u"
	case OpI32And:
		return "i32.and"
	case OpI32Or:
		return "i32.or"
	case OpI32Xor:
		return "i32.xor"
	case OpI32Shl:
		return "i32.shl"
	case OpI32ShrS:
		return "i32.shr_s"
	case OpI32ShrU:
		return "i32.shr_u"
	case OpI32Rotl:
		return "i32.rotl"
	case OpI32Rotr:
		return "i32.rotr"
	case OpI64Clz:
		return "i64.clz"
	case OpI64Ctz:
		return "i64.ctz"
	case OpI64Popcnt:
		return "i64.popcnt"
	case OpI64Add:
		return "i64.add"
	case OpI64Sub:
		return "i64.sub"
	case OpI64Mul:
		return "i64.mul"
	case OpI64DivS:
		return "i64.div_s"
	case OpI64DivU:
		return "i64.div_u"
	case OpI64RemS:
		return "i64.rem_s"
	case OpI64RemU:
		return "i64.rem_u"
	case OpI64And:
		return "i64.and"
	case OpI64Or:
		return "i64.or"
	case OpI64Xor:
		return "i64.xor"
	case OpI64Shl:
		return "i64.shl"
	case OpI64ShrS:
		return "i64.shr_s"
	case OpI64ShrU:
		return "i64.shr_u"
	case OpI64Rotl:
		return "i64.rotl"
	case OpI64Rotr:
		return "i64.rotr"
	case OpF32Abs:
		return "f32.abs"
	case OpF32Neg:
		return "f32.neg"
	case OpF32Ceil:
		return "f32.ceil"
	case OpF32Floor:
		return "f32.floor"
	case OpF32Trunc:
		return "f32.trunc"
	case OpF32Nearest:
		return "f32.nearest"
	case OpF32Sqrt:
		return "f32.sqrt"
	case OpF32Add:
		return "f32.add"
	case OpF32Sub:
		return "f32.sub"
	case OpF32Mul:
		return "f32.mul"
	case OpF32Div:
		return "f32.div"
	case OpF32Min:
		return "f32.min"
	case OpF32Max:
		return "f32.max"
	case OpF32Copysign:
		return "f32.copysign"
	case OpF64Abs:
		return "f64.abs"
	case OpF64Neg:
		return "f64.neg"
	case OpF64Ceil:
		return "f64.ceil"
	case OpF64Floor:
		return "f64.floor"
	case OpF64Trunc:
		return "f64.trunc"
	case OpF64Nearest:
		return "f64.nearest"
	case OpF64Sqrt:
		return "f64.sqrt"
	case OpF64Add:
		return "f64.add"
	case OpF64Sub:
		return "f64.sub"
	case OpF64Mul:
		return "f64.mul"
	case OpF64Div:
		return "f64.div"
	case OpF64Min:
		return "f64.min"
	case OpF64Max:
		return "f64.max"
	case OpF64Copysign:
		return "f64.copysign"
	case OpI32WrapI64:
		return "i32.wrap_i64"
	case OpI32TruncF32S:
		return "i32.trunc_f32_s"
	case OpI32TruncF32U:
		return "i32.trunc_f32_u"
	case OpI32TruncF64S:
		return "i32.trunc_f64_s"
	case OpI32TruncF64U:
		return "i32.trunc_f64_u"
	case OpI64ExtendI32S:
		return "i64.extend_i32_s"
	case OpI64ExtendI32U:
		return "i64.extend_i32_u"
	case OpI64TruncF32S:
		return "i64.trunc_f32_s"
	case OpI64TruncF32U:
		return "i64.trunc_f32_u"
	case OpI64TruncF64S:
		return "i64.trunc_f64_s"
	case OpI64TruncF64U:
		return "i64.trunc_f64_u"
	case OpF32ConvertI32S:
		return "f32.convert_i32_s"
	case OpF32ConvertI32U:
		return "f32.convert_i32_u"
	case OpF32ConvertI64S:
		return "f32.convert_i64_s"
	case OpF32ConvertI64U:
		return "f32.convert_i64_u"
	case OpF32DemoteF64:
		return "f32.demote_f64"
	case OpF64ConvertI32S:
		return "f64.convert_i32_s"
	case OpF64ConvertI32U:
		return "f64.convert_i32_u"
	case OpF64ConvertI64S:
		return "f64.convert_i64_s"
	case OpF64ConvertI64U:
		return "f64.convert_i64_u"
	case OpF64PromoteF32:
		return "f64.promote_f32"
	case OpI32ReinterpretF32:
		return "i32.reinterpret_f32"
	case OpI64ReinterpretF64:
		return "i64.reinterpret_f64"
	case OpF32ReinterpretI32:
		return "f32.reinterpret_i32"
	case OpF64ReinterpretI64:
		return "f64.reinterpret_i64"
	case OpI32Extend8S:
		return "i32.extend8_s"
	case OpI32Extend16S:
		return "i32.extend16_s"
	case OpI64Extend8S:
		return "i64.extend8_s"
	case OpI64Extend16S:
		return "i64.extend16_s"
	case OpI64Extend32S:
		return "i64.extend32_s"
	case OpPrefix:
		switch i.Immediate {
		case OpI32TruncSatF32S:
			return "i32.trunc_sat_f32_s"
		case OpI32TruncSatF32U:
			return "i32.trunc_sat_f32_u"
		case OpI32TruncSatF64S:
			return "i32.trunc_sat_f64_s"
		case OpI32TruncSatF64U:
			return "i32.trunc_sat_f64_u"
		case OpI64TruncSatF32S:
			return "i64.trunc_sat_f32_s"
		case OpI64TruncSatF32U:
			return "i64.trunc_sat_f32_u"
		case OpI64TruncSatF64S:
			return "i64.trunc_sat_f64_s"
		case OpI64TruncSatF64U:
			return "i64.trunc_sat_f64_u"
		}
	}
	return "invalid"
}
