package code

import "math"

func Unreachable() Instruction {
	return Instruction{Opcode: OpUnreachable}
}

func Nop() Instruction {
	return Instruction{Opcode: OpNop}
}

func Block(blockType ...uint64) Instruction {
	typ := uint64(BlockTypeEmpty)
	if len(blockType) != 0 {
		typ = blockType[0]
	}
	return Instruction{Opcode: OpBlock, Immediate: typ}
}

func Loop(blockType ...uint64) Instruction {
	typ := uint64(BlockTypeEmpty)
	if len(blockType) != 0 {
		typ = blockType[0]
	}
	return Instruction{Opcode: OpLoop, Immediate: typ}
}

func If(blockType ...uint64) Instruction {
	typ := uint64(BlockTypeEmpty)
	if len(blockType) != 0 {
		typ = blockType[0]
	}
	return Instruction{Opcode: OpIf, Immediate: typ}
}

func Else() Instruction {
	return Instruction{Opcode: OpElse}
}

func End() Instruction {
	return Instruction{Opcode: OpEnd}
}

func Br(labelidx int) Instruction {
	return Instruction{Opcode: OpBr, Immediate: uint64(labelidx)}
}

func BrIf(labelidx int) Instruction {
	return Instruction{Opcode: OpBrIf, Immediate: uint64(labelidx)}
}

func BrTable(labelidx int, labelidxN ...int) Instruction {
	labels := make([]int, len(labelidxN))
	if len(labelidxN) > 0 {
		labels[0], labelidx = labelidx, labelidxN[len(labelidxN)-1]
		copy(labels[1:], labelidxN[:len(labelidxN)-1])
	}

	return Instruction{Opcode: OpBrTable, Immediate: uint64(labelidx), Labels: labels}
}

func Return() Instruction {
	return Instruction{Opcode: OpReturn}
}

func Call(funcidx uint32) Instruction {
	return Instruction{Opcode: OpCall, Immediate: uint64(funcidx)}
}

func CallIndirect(tableidx uint32) Instruction {
	return Instruction{Opcode: OpCallIndirect, Immediate: uint64(tableidx)}
}

func Drop() Instruction {
	return Instruction{Opcode: OpDrop}
}

func Select() Instruction {
	return Instruction{Opcode: OpSelect}
}

func LocalGet(localidx uint32) Instruction {
	return Instruction{Opcode: OpLocalGet, Immediate: uint64(localidx)}
}

func LocalSet(localidx uint32) Instruction {
	return Instruction{Opcode: OpLocalSet, Immediate: uint64(localidx)}
}

func LocalTee(localidx uint32) Instruction {
	return Instruction{Opcode: OpLocalTee, Immediate: uint64(localidx)}
}

func GlobalGet(globalidx uint32) Instruction {
	return Instruction{Opcode: OpGlobalGet, Immediate: uint64(globalidx)}
}

func GlobalSet(globalidx uint32) Instruction {
	return Instruction{Opcode: OpGlobalSet, Immediate: uint64(globalidx)}
}

func I32Load(offset, align uint32) Instruction {
	return Instruction{Opcode: OpI32Load, Immediate: memarg(offset, align)}
}

func I64Load(offset, align uint32) Instruction {
	return Instruction{Opcode: OpI64Load, Immediate: memarg(offset, align)}
}

func F32Load(offset, align uint32) Instruction {
	return Instruction{Opcode: OpF32Load, Immediate: memarg(offset, align)}
}

func F64Load(offset, align uint32) Instruction {
	return Instruction{Opcode: OpF64Load, Immediate: memarg(offset, align)}
}

func I32Load8S(offset, align uint32) Instruction {
	return Instruction{Opcode: OpI32Load8S, Immediate: memarg(offset, align)}
}

func I32Load8U(offset, align uint32) Instruction {
	return Instruction{Opcode: OpI32Load8U, Immediate: memarg(offset, align)}
}

func I32Load16S(offset, align uint32) Instruction {
	return Instruction{Opcode: OpI32Load16S, Immediate: memarg(offset, align)}
}

func I32Load16U(offset, align uint32) Instruction {
	return Instruction{Opcode: OpI32Load16U, Immediate: memarg(offset, align)}
}

func I64Load8S(offset, align uint32) Instruction {
	return Instruction{Opcode: OpI64Load8S, Immediate: memarg(offset, align)}
}

func I64Load8U(offset, align uint32) Instruction {
	return Instruction{Opcode: OpI64Load8U, Immediate: memarg(offset, align)}
}

func I64Load16S(offset, align uint32) Instruction {
	return Instruction{Opcode: OpI64Load16S, Immediate: memarg(offset, align)}
}

func I64Load16U(offset, align uint32) Instruction {
	return Instruction{Opcode: OpI64Load16U, Immediate: memarg(offset, align)}
}

func I64Load32S(offset, align uint32) Instruction {
	return Instruction{Opcode: OpI64Load32S, Immediate: memarg(offset, align)}
}

func I64Load32U(offset, align uint32) Instruction {
	return Instruction{Opcode: OpI64Load32U, Immediate: memarg(offset, align)}
}

func I32Store(offset, align uint32) Instruction {
	return Instruction{Opcode: OpI32Store, Immediate: memarg(offset, align)}
}

func I64Store(offset, align uint32) Instruction {
	return Instruction{Opcode: OpI64Store, Immediate: memarg(offset, align)}
}

func F32Store(offset, align uint32) Instruction {
	return Instruction{Opcode: OpF32Store, Immediate: memarg(offset, align)}
}

func F64Store(offset, align uint32) Instruction {
	return Instruction{Opcode: OpF64Store, Immediate: memarg(offset, align)}
}

func I32Store8(offset, align uint32) Instruction {
	return Instruction{Opcode: OpI32Store8, Immediate: memarg(offset, align)}
}

func I32Store16(offset, align uint32) Instruction {
	return Instruction{Opcode: OpI32Store16, Immediate: memarg(offset, align)}
}

func I64Store8(offset, align uint32) Instruction {
	return Instruction{Opcode: OpI64Store8, Immediate: memarg(offset, align)}
}

func I64Store16(offset, align uint32) Instruction {
	return Instruction{Opcode: OpI64Store16, Immediate: memarg(offset, align)}
}

func I64Store32(offset, align uint32) Instruction {
	return Instruction{Opcode: OpI64Store32, Immediate: memarg(offset, align)}
}

func MemorySize() Instruction {
	return Instruction{Opcode: OpMemorySize}
}

func MemoryGrow() Instruction {
	return Instruction{Opcode: OpMemoryGrow}
}

func I32Const(v int32) Instruction {
	return Instruction{Opcode: OpI32Const, Immediate: uint64(v)}
}

func I64Const(v int64) Instruction {
	return Instruction{Opcode: OpI64Const, Immediate: uint64(v)}
}

func F32Const(v float32) Instruction {
	return Instruction{Opcode: OpF32Const, Immediate: uint64(math.Float32bits(v))}
}

func F64Const(v float64) Instruction {
	return Instruction{Opcode: OpF64Const, Immediate: math.Float64bits(v)}
}

func I32Eqz() Instruction {
	return Instruction{Opcode: OpI32Eqz}
}

func I32Eq() Instruction {
	return Instruction{Opcode: OpI32Eq}
}

func I32Ne() Instruction {
	return Instruction{Opcode: OpI32Ne}
}

func I32LtS() Instruction {
	return Instruction{Opcode: OpI32LtS}
}

func I32LtU() Instruction {
	return Instruction{Opcode: OpI32LtU}
}

func I32GtS() Instruction {
	return Instruction{Opcode: OpI32GtS}
}

func I32GtU() Instruction {
	return Instruction{Opcode: OpI32GtU}
}

func I32LeS() Instruction {
	return Instruction{Opcode: OpI32LeS}
}

func I32LeU() Instruction {
	return Instruction{Opcode: OpI32LeU}
}

func I32GeS() Instruction {
	return Instruction{Opcode: OpI32GeS}
}

func I32GeU() Instruction {
	return Instruction{Opcode: OpI32GeU}
}

func I64Eqz() Instruction {
	return Instruction{Opcode: OpI64Eqz}
}

func I64Eq() Instruction {
	return Instruction{Opcode: OpI64Eq}
}

func I64Ne() Instruction {
	return Instruction{Opcode: OpI64Ne}
}

func I64LtS() Instruction {
	return Instruction{Opcode: OpI64LtS}
}

func I64LtU() Instruction {
	return Instruction{Opcode: OpI64LtU}
}

func I64GtS() Instruction {
	return Instruction{Opcode: OpI64GtS}
}

func I64GtU() Instruction {
	return Instruction{Opcode: OpI64GtU}
}

func I64LeS() Instruction {
	return Instruction{Opcode: OpI64LeS}
}

func I64LeU() Instruction {
	return Instruction{Opcode: OpI64LeU}
}

func I64GeS() Instruction {
	return Instruction{Opcode: OpI64GeS}
}

func I64GeU() Instruction {
	return Instruction{Opcode: OpI64GeU}
}

func F32Eq() Instruction {
	return Instruction{Opcode: OpF32Eq}
}

func F32Ne() Instruction {
	return Instruction{Opcode: OpF32Ne}
}

func F32Lt() Instruction {
	return Instruction{Opcode: OpF32Lt}
}

func F32Gt() Instruction {
	return Instruction{Opcode: OpF32Gt}
}

func F32Le() Instruction {
	return Instruction{Opcode: OpF32Le}
}

func F32Ge() Instruction {
	return Instruction{Opcode: OpF32Ge}
}

func F64Eq() Instruction {
	return Instruction{Opcode: OpF64Eq}
}

func F64Ne() Instruction {
	return Instruction{Opcode: OpF64Ne}
}

func F64Lt() Instruction {
	return Instruction{Opcode: OpF64Lt}
}

func F64Gt() Instruction {
	return Instruction{Opcode: OpF64Gt}
}

func F64Le() Instruction {
	return Instruction{Opcode: OpF64Le}
}

func F64Ge() Instruction {
	return Instruction{Opcode: OpF64Ge}
}

func I32Clz() Instruction {
	return Instruction{Opcode: OpI32Clz}
}

func I32Ctz() Instruction {
	return Instruction{Opcode: OpI32Ctz}
}

func I32Popcnt() Instruction {
	return Instruction{Opcode: OpI32Popcnt}
}

func I32Add() Instruction {
	return Instruction{Opcode: OpI32Add}
}

func I32Sub() Instruction {
	return Instruction{Opcode: OpI32Sub}
}

func I32Mul() Instruction {
	return Instruction{Opcode: OpI32Mul}
}

func I32DivS() Instruction {
	return Instruction{Opcode: OpI32DivS}
}

func I32DivU() Instruction {
	return Instruction{Opcode: OpI32DivU}
}

func I32RemS() Instruction {
	return Instruction{Opcode: OpI32RemS}
}

func I32RemU() Instruction {
	return Instruction{Opcode: OpI32RemU}
}

func I32And() Instruction {
	return Instruction{Opcode: OpI32And}
}

func I32Or() Instruction {
	return Instruction{Opcode: OpI32Or}
}

func I32Xor() Instruction {
	return Instruction{Opcode: OpI32Xor}
}

func I32Shl() Instruction {
	return Instruction{Opcode: OpI32Shl}
}

func I32ShrS() Instruction {
	return Instruction{Opcode: OpI32ShrS}
}

func I32ShrU() Instruction {
	return Instruction{Opcode: OpI32ShrU}
}

func I32Rotl() Instruction {
	return Instruction{Opcode: OpI32Rotl}
}

func I32Rotr() Instruction {
	return Instruction{Opcode: OpI32Rotr}
}

func I64Clz() Instruction {
	return Instruction{Opcode: OpI64Clz}
}

func I64Ctz() Instruction {
	return Instruction{Opcode: OpI64Ctz}
}

func I64Popcnt() Instruction {
	return Instruction{Opcode: OpI64Popcnt}
}

func I64Add() Instruction {
	return Instruction{Opcode: OpI64Add}
}

func I64Sub() Instruction {
	return Instruction{Opcode: OpI64Sub}
}

func I64Mul() Instruction {
	return Instruction{Opcode: OpI64Mul}
}

func I64DivS() Instruction {
	return Instruction{Opcode: OpI64DivS}
}

func I64DivU() Instruction {
	return Instruction{Opcode: OpI64DivU}
}

func I64RemS() Instruction {
	return Instruction{Opcode: OpI64RemS}
}

func I64RemU() Instruction {
	return Instruction{Opcode: OpI64RemU}
}

func I64And() Instruction {
	return Instruction{Opcode: OpI64And}
}

func I64Or() Instruction {
	return Instruction{Opcode: OpI64Or}
}

func I64Xor() Instruction {
	return Instruction{Opcode: OpI64Xor}
}

func I64Shl() Instruction {
	return Instruction{Opcode: OpI64Shl}
}

func I64ShrS() Instruction {
	return Instruction{Opcode: OpI64ShrS}
}

func I64ShrU() Instruction {
	return Instruction{Opcode: OpI64ShrU}
}

func I64Rotl() Instruction {
	return Instruction{Opcode: OpI64Rotl}
}

func I64Rotr() Instruction {
	return Instruction{Opcode: OpI64Rotr}
}

func F32Abs() Instruction {
	return Instruction{Opcode: OpF32Abs}
}

func F32Neg() Instruction {
	return Instruction{Opcode: OpF32Neg}
}

func F32Ceil() Instruction {
	return Instruction{Opcode: OpF32Ceil}
}

func F32Floor() Instruction {
	return Instruction{Opcode: OpF32Floor}
}

func F32Trunc() Instruction {
	return Instruction{Opcode: OpF32Trunc}
}

func F32Nearest() Instruction {
	return Instruction{Opcode: OpF32Nearest}
}

func F32Sqrt() Instruction {
	return Instruction{Opcode: OpF32Sqrt}
}

func F32Add() Instruction {
	return Instruction{Opcode: OpF32Add}
}

func F32Sub() Instruction {
	return Instruction{Opcode: OpF32Sub}
}

func F32Mul() Instruction {
	return Instruction{Opcode: OpF32Mul}
}

func F32Div() Instruction {
	return Instruction{Opcode: OpF32Div}
}

func F32Min() Instruction {
	return Instruction{Opcode: OpF32Min}
}

func F32Max() Instruction {
	return Instruction{Opcode: OpF32Max}
}

func F32Copysign() Instruction {
	return Instruction{Opcode: OpF32Copysign}
}

func F64Abs() Instruction {
	return Instruction{Opcode: OpF64Abs}
}

func F64Neg() Instruction {
	return Instruction{Opcode: OpF64Neg}
}

func F64Ceil() Instruction {
	return Instruction{Opcode: OpF64Ceil}
}

func F64Floor() Instruction {
	return Instruction{Opcode: OpF64Floor}
}

func F64Trunc() Instruction {
	return Instruction{Opcode: OpF64Trunc}
}

func F64Nearest() Instruction {
	return Instruction{Opcode: OpF64Nearest}
}

func F64Sqrt() Instruction {
	return Instruction{Opcode: OpF64Sqrt}
}

func F64Add() Instruction {
	return Instruction{Opcode: OpF64Add}
}

func F64Sub() Instruction {
	return Instruction{Opcode: OpF64Sub}
}

func F64Mul() Instruction {
	return Instruction{Opcode: OpF64Mul}
}

func F64Div() Instruction {
	return Instruction{Opcode: OpF64Div}
}

func F64Min() Instruction {
	return Instruction{Opcode: OpF64Min}
}

func F64Max() Instruction {
	return Instruction{Opcode: OpF64Max}
}

func F64Copysign() Instruction {
	return Instruction{Opcode: OpF64Copysign}
}

func I32WrapI64() Instruction {
	return Instruction{Opcode: OpI32WrapI64}
}

func I32TruncF32S() Instruction {
	return Instruction{Opcode: OpI32TruncF32S}
}

func I32TruncF32U() Instruction {
	return Instruction{Opcode: OpI32TruncF32U}
}

func I32TruncF64S() Instruction {
	return Instruction{Opcode: OpI32TruncF64S}
}

func I32TruncF64U() Instruction {
	return Instruction{Opcode: OpI32TruncF64U}
}

func I64ExtendI32S() Instruction {
	return Instruction{Opcode: OpI64ExtendI32S}
}

func I64ExtendI32U() Instruction {
	return Instruction{Opcode: OpI64ExtendI32U}
}

func I64TruncF32S() Instruction {
	return Instruction{Opcode: OpI64TruncF32S}
}

func I64TruncF32U() Instruction {
	return Instruction{Opcode: OpI64TruncF32U}
}

func I64TruncF64S() Instruction {
	return Instruction{Opcode: OpI64TruncF64S}
}

func I64TruncF64U() Instruction {
	return Instruction{Opcode: OpI64TruncF64U}
}

func F32ConvertI32S() Instruction {
	return Instruction{Opcode: OpF32ConvertI32S}
}

func F32ConvertI32U() Instruction {
	return Instruction{Opcode: OpF32ConvertI32U}
}

func F32ConvertI64S() Instruction {
	return Instruction{Opcode: OpF32ConvertI64S}
}

func F32ConvertI64U() Instruction {
	return Instruction{Opcode: OpF32ConvertI64U}
}

func F32DemoteF64() Instruction {
	return Instruction{Opcode: OpF32DemoteF64}
}

func F64ConvertI32S() Instruction {
	return Instruction{Opcode: OpF64ConvertI32S}
}

func F64ConvertI32U() Instruction {
	return Instruction{Opcode: OpF64ConvertI32U}
}

func F64ConvertI64S() Instruction {
	return Instruction{Opcode: OpF64ConvertI64S}
}

func F64ConvertI64U() Instruction {
	return Instruction{Opcode: OpF64ConvertI64U}
}

func F64PromoteF32() Instruction {
	return Instruction{Opcode: OpF64PromoteF32}
}

func I32ReinterpretF32() Instruction {
	return Instruction{Opcode: OpI32ReinterpretF32}
}

func I64ReinterpretF64() Instruction {
	return Instruction{Opcode: OpI64ReinterpretF64}
}

func F32ReinterpretI32() Instruction {
	return Instruction{Opcode: OpF32ReinterpretI32}
}

func F64ReinterpretI64() Instruction {
	return Instruction{Opcode: OpF64ReinterpretI64}
}

func I32Extend8S() Instruction {
	return Instruction{Opcode: OpI32Extend8S}
}

func I32Extend16S() Instruction {
	return Instruction{Opcode: OpI32Extend16S}
}

func I64Extend8S() Instruction {
	return Instruction{Opcode: OpI64Extend8S}
}

func I64Extend16S() Instruction {
	return Instruction{Opcode: OpI64Extend16S}
}

func I64Extend32S() Instruction {
	return Instruction{Opcode: OpI64Extend32S}
}

func I32TruncSatF32S() Instruction {
	return Instruction{Opcode: OpPrefix, Immediate: OpI32TruncSatF32S}
}

func I32TruncSatF32U() Instruction {
	return Instruction{Opcode: OpPrefix, Immediate: OpI32TruncSatF32U}
}

func I32TruncSatF64S() Instruction {
	return Instruction{Opcode: OpPrefix, Immediate: OpI32TruncSatF64S}
}

func I32TruncSatF64U() Instruction {
	return Instruction{Opcode: OpPrefix, Immediate: OpI32TruncSatF64U}
}

func I64TruncSatF32S() Instruction {
	return Instruction{Opcode: OpPrefix, Immediate: OpI64TruncSatF32S}
}

func I64TruncSatF32U() Instruction {
	return Instruction{Opcode: OpPrefix, Immediate: OpI64TruncSatF32U}
}

func I64TruncSatF64S() Instruction {
	return Instruction{Opcode: OpPrefix, Immediate: OpI64TruncSatF64S}
}

func I64TruncSatF64U() Instruction {
	return Instruction{Opcode: OpPrefix, Immediate: OpI64TruncSatF64U}
}
