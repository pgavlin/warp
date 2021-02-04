package wax

import (
	"math"
	"math/bits"

	"github.com/pgavlin/warp/exec"
	"github.com/pgavlin/warp/wasm/code"
)

type values []uint64

func (vs values) U8(i int) uint8 {
	return uint8(vs[i])
}

func (vs values) U16(i int) uint16 {
	return uint16(vs[i])
}

func (vs values) U32(i int) uint32 {
	return uint32(vs[i])
}

func (vs values) U64(i int) uint64 {
	return vs[i]
}

func (vs values) I(i int) int {
	return int(vs[i])
}

func (vs values) I8(i int) int8 {
	return int8(vs[i])
}

func (vs values) I16(i int) int16 {
	return int16(vs[i])
}

func (vs values) I32(i int) int32 {
	return int32(vs[i])
}

func (vs values) I64(i int) int64 {
	return int64(vs[i])
}

func (vs values) F32(i int) float32 {
	return math.Float32frombits(vs.U32(i))
}

func (vs values) F64(i int) float64 {
	return math.Float64frombits(vs.U64(i))
}

func i32Bool(v bool) int32 {
	if v {
		return 1
	}
	return 0
}

func evaluate(x *Expression) (result uint64, ok bool) {
	defer func() {
		if x := recover(); x != nil {
			result, ok = 0, false
		}
	}()

	// We can only evaluate pure expressions.
	if x.Flags&^FlagsMayTrap != 0 {
		return 0, false
	}

	args := make(values, len(x.Uses))
	for i, u := range x.Uses {
		if u.IsTemp() {
			return 0, false
		}
		v, ok := evaluate(u.X)
		if !ok {
			return 0, false
		}
		args[i] = v
	}

	instr := x.Instr
	if x.IsPseudo() {
		switch instr.Opcode {
		case PseudoI32ConvertBool:
			return uint64(instr.I32()), true
		}
		return 0, false
	}

	switch instr.Opcode {
	case code.OpI32Const:
		return uint64(instr.I32()), true
	case code.OpI64Const:
		return uint64(instr.I64()), true
	case code.OpF32Const:
		return uint64(math.Float32bits(instr.F32())), true
	case code.OpF64Const:
		return uint64(math.Float64bits(instr.F64())), true

	case code.OpI32Eqz:
		return uint64(i32Bool(args.I32(0) == 0)), true
	case code.OpI32Eq:
		return uint64(i32Bool(args.I32(0) == args.I32(1))), true
	case code.OpI32Ne:
		return uint64(i32Bool(args.I32(0) != args.I32(1))), true
	case code.OpI32LtS:
		v2, v1 := args.I32(1), args.I32(0)
		return uint64(i32Bool(v1 < v2)), true
	case code.OpI32LtU:
		v2, v1 := args.U32(1), args.U32(0)
		return uint64(i32Bool(v1 < v2)), true
	case code.OpI32GtS:
		v2, v1 := args.I32(1), args.I32(0)
		return uint64(i32Bool(v1 > v2)), true
	case code.OpI32GtU:
		v2, v1 := args.U32(1), args.U32(0)
		return uint64(i32Bool(v1 > v2)), true
	case code.OpI32LeS:
		v2, v1 := args.I32(1), args.I32(0)
		return uint64(i32Bool(v1 <= v2)), true
	case code.OpI32LeU:
		v2, v1 := args.U32(1), args.U32(0)
		return uint64(i32Bool(v1 <= v2)), true
	case code.OpI32GeS:
		v2, v1 := args.I32(1), args.I32(0)
		return uint64(i32Bool(v1 >= v2)), true
	case code.OpI32GeU:
		v2, v1 := args.U32(1), args.U32(0)
		return uint64(i32Bool(v1 >= v2)), true

	case code.OpI64Eqz:
		return uint64(i32Bool(args.I64(0) == 0)), true
	case code.OpI64Eq:
		return uint64(i32Bool(args.I64(0) == args.I64(1))), true
	case code.OpI64Ne:
		return uint64(i32Bool(args.I64(0) != args.I64(1))), true
	case code.OpI64LtS:
		v2, v1 := args.I64(1), args.I64(0)
		return uint64(i32Bool(v1 < v2)), true
	case code.OpI64LtU:
		v2, v1 := args.U64(1), args.U64(0)
		return uint64(i32Bool(v1 < v2)), true
	case code.OpI64GtS:
		v2, v1 := args.I64(1), args.I64(0)
		return uint64(i32Bool(v1 > v2)), true
	case code.OpI64GtU:
		v2, v1 := args.U64(1), args.U64(0)
		return uint64(i32Bool(v1 > v2)), true
	case code.OpI64LeS:
		v2, v1 := args.I64(1), args.I64(0)
		return uint64(i32Bool(v1 <= v2)), true
	case code.OpI64LeU:
		v2, v1 := args.U64(1), args.U64(0)
		return uint64(i32Bool(v1 <= v2)), true
	case code.OpI64GeS:
		v2, v1 := args.I64(1), args.I64(0)
		return uint64(i32Bool(v1 >= v2)), true
	case code.OpI64GeU:
		v2, v1 := args.U64(1), args.U64(0)
		return uint64(i32Bool(v1 >= v2)), true

	case code.OpF32Eq:
		return uint64(i32Bool(args.F32(0) == args.F32(1))), true
	case code.OpF32Ne:
		return uint64(i32Bool(args.F32(0) != args.F32(1))), true
	case code.OpF32Lt:
		v2, v1 := args.F32(1), args.F32(0)
		return uint64(i32Bool(v1 < v2)), true
	case code.OpF32Gt:
		v2, v1 := args.F32(1), args.F32(0)
		return uint64(i32Bool(v1 > v2)), true
	case code.OpF32Le:
		v2, v1 := args.F32(1), args.F32(0)
		return uint64(i32Bool(v1 <= v2)), true
	case code.OpF32Ge:
		v2, v1 := args.F32(1), args.F32(0)
		return uint64(i32Bool(v1 >= v2)), true

	case code.OpF64Eq:
		return uint64(i32Bool(args.F64(0) == args.F64(1))), true
	case code.OpF64Ne:
		return uint64(i32Bool(args.F64(0) != args.F64(1))), true
	case code.OpF64Lt:
		v2, v1 := args.F64(1), args.F64(0)
		return uint64(i32Bool(v1 < v2)), true
	case code.OpF64Gt:
		v2, v1 := args.F64(1), args.F64(0)
		return uint64(i32Bool(v1 > v2)), true
	case code.OpF64Le:
		v2, v1 := args.F64(1), args.F64(0)
		return uint64(i32Bool(v1 <= v2)), true
	case code.OpF64Ge:
		v2, v1 := args.F64(1), args.F64(0)
		return uint64(i32Bool(v1 >= v2)), true

	case code.OpI32Clz:
		return uint64(bits.LeadingZeros32(args.U32(0))), true
	case code.OpI32Ctz:
		return uint64(bits.TrailingZeros32(args.U32(0))), true
	case code.OpI32Popcnt:
		return uint64(bits.OnesCount32(args.U32(0))), true
	case code.OpI32Add:
		return uint64(args.I32(0) + args.I32(1)), true
	case code.OpI32Sub:
		v2, v1 := args.I32(1), args.I32(0)
		return uint64(v1 - v2), true
	case code.OpI32Mul:
		return uint64(args.I32(0) * args.I32(1)), true
	case code.OpI32DivS:
		v2, v1 := args.I32(1), args.I32(0)
		return uint64(exec.I32DivS(v1, v2)), true
	case code.OpI32DivU:
		v2, v1 := args.U32(1), args.U32(0)
		return uint64(v1 / v2), true
	case code.OpI32RemS:
		v2, v1 := args.I32(1), args.I32(0)
		return uint64(v1 % v2), true
	case code.OpI32RemU:
		v2, v1 := args.U32(1), args.U32(0)
		return uint64(v1 % v2), true
	case code.OpI32And:
		return uint64(args.I32(0) & args.I32(1)), true
	case code.OpI32Or:
		return uint64(args.I32(0) | args.I32(1)), true
	case code.OpI32Xor:
		return uint64(args.I32(0) ^ args.I32(1)), true
	case code.OpI32Shl:
		v2, v1 := args.I32(1), args.I32(0)
		return uint64(v1 << (v2 & 31)), true
	case code.OpI32ShrS:
		v2, v1 := args.I32(1), args.I32(0)
		return uint64(v1 >> (v2 & 31)), true
	case code.OpI32ShrU:
		v2, v1 := args.U32(1), args.U32(0)
		return uint64(v1 >> (v2 & 31)), true
	case code.OpI32Rotl:
		v2, v1 := args.I(1), args.U32(0)
		return uint64(bits.RotateLeft32(v1, v2)), true
	case code.OpI32Rotr:
		v2, v1 := args.I(1), args.U32(0)
		return uint64(bits.RotateLeft32(v1, -v2)), true

	case code.OpI64Clz:
		return uint64(bits.LeadingZeros64(args.U64(0))), true
	case code.OpI64Ctz:
		return uint64(bits.TrailingZeros64(args.U64(0))), true
	case code.OpI64Popcnt:
		return uint64(bits.OnesCount64(args.U64(0))), true
	case code.OpI64Add:
		return uint64(args.I64(0) + args.I64(1)), true
	case code.OpI64Sub:
		v2, v1 := args.I64(1), args.I64(0)
		return uint64(v1 - v2), true
	case code.OpI64Mul:
		return uint64(args.I64(0) * args.I64(1)), true
	case code.OpI64DivS:
		v2, v1 := args.I64(1), args.I64(0)
		return uint64(exec.I64DivS(v1, v2)), true
	case code.OpI64DivU:
		v2, v1 := args.U64(1), args.U64(0)
		return uint64(v1 / v2), true
	case code.OpI64RemS:
		v2, v1 := args.I64(1), args.I64(0)
		return uint64(v1 % v2), true
	case code.OpI64RemU:
		v2, v1 := args.U64(1), args.U64(0)
		return uint64(v1 % v2), true
	case code.OpI64And:
		return uint64(args.I64(0) & args.I64(1)), true
	case code.OpI64Or:
		return uint64(args.I64(0) | args.I64(1)), true
	case code.OpI64Xor:
		return uint64(args.I64(0) ^ args.I64(1)), true
	case code.OpI64Shl:
		v2, v1 := args.I64(1), args.I64(0)
		return uint64(v1 << (v2 & 63)), true
	case code.OpI64ShrS:
		v2, v1 := args.I64(1), args.I64(0)
		return uint64(v1 >> (v2 & 63)), true
	case code.OpI64ShrU:
		v2, v1 := args.U64(1), args.U64(0)
		return uint64(v1 >> (v2 & 63)), true
	case code.OpI64Rotl:
		v2, v1 := args.I(1), args.U64(0)
		return uint64(bits.RotateLeft64(v1, v2)), true
	case code.OpI64Rotr:
		v2, v1 := args.I(1), args.U64(0)
		return uint64(bits.RotateLeft64(v1, -v2)), true

	case code.OpF32Abs:
		return uint64(math.Float32bits(float32(math.Abs(float64(args.F32(0)))))), true
	case code.OpF32Neg:
		return uint64(math.Float32bits(-args.F32(0))), true
	case code.OpF32Ceil:
		return uint64(math.Float32bits(float32(math.Ceil(float64(args.F32(0)))))), true
	case code.OpF32Floor:
		return uint64(math.Float32bits(float32(math.Floor(float64(args.F32(0)))))), true
	case code.OpF32Trunc:
		return uint64(math.Float32bits(float32(math.Trunc(float64(args.F32(0)))))), true
	case code.OpF32Nearest:
		return uint64(math.Float32bits(float32(math.RoundToEven(float64(args.F32(0)))))), true
	case code.OpF32Sqrt:
		return uint64(math.Float32bits(float32(math.Sqrt(float64(args.F32(0)))))), true
	case code.OpF32Add:
		return uint64(math.Float32bits(args.F32(0) + args.F32(1))), true
	case code.OpF32Sub:
		v2, v1 := args.F32(1), args.F32(0)
		return uint64(math.Float32bits(v1 - v2)), true
	case code.OpF32Mul:
		return uint64(math.Float32bits(args.F32(0) * args.F32(1))), true
	case code.OpF32Div:
		v2, v1 := args.F32(1), args.F32(0)
		return uint64(math.Float32bits(v1 / v2)), true
	case code.OpF32Min:
		return uint64(math.Float32bits(float32(exec.Fmin(float64(args.F32(0)), float64(args.F32(1)))))), true
	case code.OpF32Max:
		return uint64(math.Float32bits(float32(exec.Fmax(float64(args.F32(0)), float64(args.F32(1)))))), true
	case code.OpF32Copysign:
		v2, v1 := args.F32(1), args.F32(0)
		return uint64(math.Float32bits(float32(math.Copysign(float64(v1), float64(v2))))), true

	case code.OpF64Abs:
		return uint64(math.Float64bits(math.Abs(args.F64(0)))), true
	case code.OpF64Neg:
		return uint64(math.Float64bits(-args.F64(0))), true
	case code.OpF64Ceil:
		return uint64(math.Float64bits(math.Ceil(args.F64(0)))), true
	case code.OpF64Floor:
		return uint64(math.Float64bits(math.Floor(args.F64(0)))), true
	case code.OpF64Trunc:
		return uint64(math.Float64bits(math.Trunc(args.F64(0)))), true
	case code.OpF64Nearest:
		return uint64(math.Float64bits(math.RoundToEven(args.F64(0)))), true
	case code.OpF64Sqrt:
		return uint64(math.Float64bits(math.Sqrt(args.F64(0)))), true
	case code.OpF64Add:
		return uint64(math.Float64bits(args.F64(0) + args.F64(1))), true
	case code.OpF64Sub:
		v2, v1 := args.F64(1), args.F64(0)
		return uint64(math.Float64bits(v1 - v2)), true
	case code.OpF64Mul:
		return uint64(math.Float64bits(args.F64(0) * args.F64(1))), true
	case code.OpF64Div:
		v2, v1 := args.F64(1), args.F64(0)
		return uint64(math.Float64bits(v1 / v2)), true
	case code.OpF64Min:
		return uint64(math.Float64bits(exec.Fmin(args.F64(0), args.F64(1)))), true
	case code.OpF64Max:
		return uint64(math.Float64bits(exec.Fmax(args.F64(0), args.F64(1)))), true
	case code.OpF64Copysign:
		v2, v1 := args.F64(1), args.F64(0)
		return uint64(math.Float64bits(math.Copysign(v1, v2))), true

	case code.OpI32WrapI64:
		return uint64(int32(args.I64(0))), true
	case code.OpI32TruncF32S:
		return uint64(exec.I32TruncS(float64(args.F32(0)))), true
	case code.OpI32TruncF32U:
		return uint64(exec.I32TruncU(float64(args.F32(0)))), true
	case code.OpI32TruncF64S:
		return uint64(exec.I32TruncS(args.F64(0))), true
	case code.OpI32TruncF64U:
		return uint64(exec.I32TruncU(args.F64(0))), true

	case code.OpI64ExtendI32S:
		return uint64(int64(args.I32(0))), true
	case code.OpI64ExtendI32U:
		return uint64(int64(args.U32(0))), true
	case code.OpI64TruncF32S:
		return uint64(exec.I64TruncS(float64(args.F32(0)))), true
	case code.OpI64TruncF32U:
		return uint64(exec.I64TruncU(float64(args.F32(0)))), true
	case code.OpI64TruncF64S:
		return uint64(exec.I64TruncS(args.F64(0))), true
	case code.OpI64TruncF64U:
		return uint64(exec.I64TruncU(args.F64(0))), true

	case code.OpF32ConvertI32S:
		return uint64(math.Float32bits(float32(args.I32(0)))), true
	case code.OpF32ConvertI32U:
		return uint64(math.Float32bits(float32(args.U32(0)))), true
	case code.OpF32ConvertI64S:
		return uint64(math.Float32bits(float32(args.I64(0)))), true
	case code.OpF32ConvertI64U:
		return uint64(math.Float32bits(float32(args.U64(0)))), true
	case code.OpF32DemoteF64:
		return uint64(math.Float32bits(float32(args.F64(0)))), true

	case code.OpF64ConvertI32S:
		return uint64(math.Float64bits(float64(args.I32(0)))), true
	case code.OpF64ConvertI32U:
		return uint64(math.Float64bits(float64(args.U32(0)))), true
	case code.OpF64ConvertI64S:
		return uint64(math.Float64bits(float64(args.I64(0)))), true
	case code.OpF64ConvertI64U:
		return uint64(math.Float64bits(float64(args.U64(0)))), true
	case code.OpF64PromoteF32:
		return uint64(math.Float64bits(float64(args.F32(0)))), true

	case code.OpI32ReinterpretF32:
		return uint64(math.Float32bits(args.F32(0))), true
	case code.OpI64ReinterpretF64:
		return uint64(math.Float64bits(args.F64(0))), true
	case code.OpF32ReinterpretI32:
		return uint64(args.U32(0)), true
	case code.OpF64ReinterpretI64:
		return uint64(args.U64(0)), true

	case code.OpI32Extend8S:
		return uint64(int32(int8(args.I32(0)))), true
	case code.OpI32Extend16S:
		return uint64(int32(int16(args.I32(0)))), true
	case code.OpI64Extend8S:
		return uint64(int64(int8(args.I64(0)))), true
	case code.OpI64Extend16S:
		return uint64(int64(int16(args.I64(0)))), true
	case code.OpI64Extend32S:
		return uint64(int64(int32(args.I64(0)))), true

	case code.OpPrefix:
		switch instr.Immediate {
		case code.OpI32TruncSatF32S:
			return uint64(exec.I32TruncSatS(float64(args.F32(0)))), true
		case code.OpI32TruncSatF32U:
			return uint64(exec.I32TruncSatU(float64(args.F32(0)))), true
		case code.OpI32TruncSatF64S:
			return uint64(exec.I32TruncSatS(args.F64(0))), true
		case code.OpI32TruncSatF64U:
			return uint64(exec.I32TruncSatU(args.F64(0))), true
		case code.OpI64TruncSatF32S:
			return uint64(exec.I64TruncSatS(float64(args.F32(0)))), true
		case code.OpI64TruncSatF32U:
			return uint64(exec.I64TruncSatU(float64(args.F32(0)))), true
		case code.OpI64TruncSatF64S:
			return uint64(exec.I64TruncSatS(args.F64(0))), true
		case code.OpI64TruncSatF64U:
			return uint64(exec.I64TruncSatU(args.F64(0))), true
		}
	}

	return 0, false
}
