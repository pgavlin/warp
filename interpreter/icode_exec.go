package interpreter

import (
	"math"
	"math/bits"

	"github.com/pgavlin/warp/exec"
	"github.com/pgavlin/warp/wasm/code"
)

func (f *frame) get(index uint32) uint64 {
	return f.locals[int(index)]
}

func (f *frame) set(index uint32, v uint64) {
	f.locals[int(index)] = v
}

func (f *frame) getI32(index uint32) int32 {
	return int32(f.locals[int(index)])
}

func (f *frame) getI64(index uint32) int64 {
	return int64(f.locals[int(index)])
}

func (f *frame) getF32(index uint32) float32 {
	return math.Float32frombits(uint32(f.locals[int(index)]))
}

func (f *frame) getF64(index uint32) float64 {
	return math.Float64frombits(f.locals[int(index)])
}

func (f *frame) setI32(index uint32, value int32) {
	f.locals[int(index)] = uint64(value)
}

func (f *frame) setI64(index uint32, value int64) {
	f.locals[int(index)] = uint64(value)
}

func (f *frame) setF32(index uint32, value float32) {
	f.locals[int(index)] = uint64(math.Float32bits(value))
}

func (f *frame) setF64(index uint32, value float64) {
	f.locals[int(index)] = math.Float64bits(value)
}

func (f *frame) pushContinuation(instr *code.Instruction, isLoop bool) {
	f.blocks = f.blocks[:len(f.blocks)+2]
	f.blocks[len(f.blocks)-2] = uint64(instr.Continuation())
	f.blocks[len(f.blocks)-1] = uint64(instr.StackHeight())<<32 | uint64(uint32(f.module.blockArity(instr, isLoop)))
}

func (f *frame) popContinuation() {
	f.blocks = f.blocks[:len(f.blocks)-2]
}

func (f *frame) dropn(n int) {
	f.stack = f.stack[:len(f.stack)-n]
}

func (f *frame) push(v uint64) {
	f.stack = f.stack[:len(f.stack)+1]
	f.stack[len(f.stack)-1] = v
}

func (f *frame) pushn(src []uint64) {
	f.stack = f.stack[:len(f.stack)+len(src)]
	copy(f.stack[len(f.stack)-len(src):], src)
}

func (f *frame) pop() uint64 {
	v := f.stack[len(f.stack)-1]
	f.stack = f.stack[:len(f.stack)-1]
	return v
}

func (f *frame) popn(dest []uint64) {
	copy(dest, f.stack[len(f.stack)-len(dest):])
	f.stack = f.stack[:len(f.stack)-len(dest)]
}

func (f *frame) pop2() (v2, v1 uint64) {
	v1, v2 = f.stack[len(f.stack)-2], f.stack[len(f.stack)-1]
	f.stack = f.stack[:len(f.stack)-2]
	return v2, v1
}

func (f *frame) pushI(v int) {
	f.push(uint64(v))
}

func (f *frame) pushU32(v uint32) {
	f.push(uint64(v))
}

func (f *frame) pushU64(v uint64) {
	f.push(v)
}

func (f *frame) pushI32(v int32) {
	f.push(uint64(v))
}

func (f *frame) pushI64(v int64) {
	f.push(uint64(v))
}

func (f *frame) pushF32(v float32) {
	f.push(uint64(math.Float32bits(v)))
}

func (f *frame) pushF64(v float64) {
	f.push(math.Float64bits(v))
}

func (f *frame) pushBool(v bool) {
	i := 0
	if v {
		i = 1
	}
	f.pushI32(int32(i))
}

func (f *frame) popI() int {
	return int(f.pop())
}

func (f *frame) popU32() uint32 {
	return uint32(f.pop())
}

func (f *frame) popBase() uint32 {
	i := f.popI32()
	if i < 0 {
		f.trap(exec.TrapOutOfBoundsMemoryAccess)
	}
	return uint32(i)
}

func (f *frame) popU64() uint64 {
	return uint64(f.pop())
}

func (f *frame) popI32() int32 {
	return int32(f.pop())
}

func (f *frame) popI64() int64 {
	return int64(f.pop())
}

func (f *frame) popF32() float32 {
	return math.Float32frombits(uint32(f.pop()))
}

func (f *frame) popF64() float64 {
	return math.Float64frombits(f.pop())
}

func (f *frame) popBool() bool {
	return f.popI32() != 0
}

func (f *frame) pop2U32() (v1, v2 uint32) {
	u1, u2 := f.pop2()
	return uint32(u1), uint32(u2)
}

func (f *frame) pop2U64() (v1, v2 uint64) {
	return f.pop2()
}

func (f *frame) pop2I32() (v1, v2 int32) {
	u1, u2 := f.pop2()
	return int32(u1), int32(u2)
}

func (f *frame) pop2I64() (v1, v2 int64) {
	u1, u2 := f.pop2()
	return int64(u1), int64(u2)
}

func (f *frame) pop2F32() (v1, v2 float32) {
	u1, u2 := f.pop2U32()
	return math.Float32frombits(u1), math.Float32frombits(u2)
}

func (f *frame) pop2F64() (v1, v2 float64) {
	u1, u2 := f.pop2()
	return math.Float64frombits(u1), math.Float64frombits(u2)
}

func (f *frame) branch(l int) int {
	dest := f.blocks[len(f.blocks)-l*2-2]

	blockDesc := f.blocks[len(f.blocks)-l*2-1]
	stackHeight, arity := int(blockDesc>>32), int(uint32(blockDesc))

	copy(f.stack[stackHeight:], f.stack[len(f.stack)-arity:])
	f.stack = f.stack[:stackHeight+arity]

	f.blocks = f.blocks[:len(f.blocks)-l*2-2]
	return int(dest)
}

func (f *frame) step(body []code.Instruction, ip int) int {
	instr := &body[ip]
	switch instr.Opcode {
	case code.OpUnreachable:
		f.trap(exec.TrapUnreachable)

	case code.OpNop:
		// no-op

	case code.OpBlock:
		f.pushContinuation(instr, false)
	case code.OpLoop:
		f.pushContinuation(instr, true)

	case code.OpIf:
		if !f.popBool() {
			if instr.Else() != 0 {
				f.pushContinuation(instr, false)
				return instr.Else() + 1
			}
			return instr.Continuation()
		}
		f.pushContinuation(instr, false)
	case code.OpElse:
		// This is the end of a taken if block.
		f.popContinuation()
		return instr.Continuation()
	case code.OpEnd:
		f.popContinuation()

	case code.OpBr:
		return f.branch(instr.Labelidx())
	case code.OpBrIf:
		if f.popBool() {
			return f.branch(instr.Labelidx())
		}
	case code.OpBrTable:
		if li := int(f.popI32()); li >= 0 && li < len(instr.Labels) {
			return f.branch(instr.Labels[li])
		}
		return f.branch(instr.Default())

	case code.OpReturn:
		// Check will happen in caller

	case code.OpCall:
		funcidx := instr.Funcidx()
		if funcidx < uint32(len(f.module.importedFunctions)) {
			f.invoke(f.module.importedFunctions[funcidx])
		} else {
			f.invokeDirect(&f.module.functions[funcidx-uint32(len(f.module.importedFunctions))])
		}
	case code.OpCallIndirect:
		table := f.module.table0.Entries()

		tableidx := f.popI32()
		if uint32(tableidx) >= uint32(len(table)) {
			f.trap(exec.TrapUndefinedElement)
		}

		function := table[int(tableidx)]
		if function == nil {
			f.trap(exec.TrapUninitializedElement)
		}

		expectedSig := f.module.types[int(instr.Typeidx())]
		actualSig := function.GetSignature()
		if !actualSig.Equals(expectedSig) {
			f.trap(exec.TrapIndirectCallTypeMismatch)
		}

		f.invoke(function)

	case code.OpDrop:
		f.pop()

	case code.OpSelect:
		condition, v2, v1 := f.popBool(), f.pop(), f.pop()
		if condition {
			f.push(v1)
		} else {
			f.push(v2)
		}

	case code.OpLocalGet:
		f.push(f.get(instr.Localidx()))
	case code.OpLocalSet:
		f.set(instr.Localidx(), f.pop())
	case code.OpLocalTee:
		v := f.pop()
		f.set(instr.Localidx(), v)
		f.push(v)

	case code.OpGlobalGet:
		global, _ := f.module.getGlobal(instr.Globalidx())
		f.push(global.Get())
	case code.OpGlobalSet:
		global, _ := f.module.getGlobal(instr.Globalidx())
		global.Set(f.pop())

	case code.OpI32Load:
		f.pushI32(int32(f.module.mem0.Uint32(f.popBase(), instr.Offset())))
	case code.OpI64Load:
		f.pushI64(int64(f.module.mem0.Uint64(f.popBase(), instr.Offset())))
	case code.OpF32Load:
		f.pushF32(f.module.mem0.Float32(f.popBase(), instr.Offset()))
	case code.OpF64Load:
		f.pushF64(f.module.mem0.Float64(f.popBase(), instr.Offset()))

	case code.OpI32Load8S:
		f.pushI32(int32(int8(f.module.mem0.Byte(f.popBase(), instr.Offset()))))
	case code.OpI32Load8U:
		f.pushI32(int32(f.module.mem0.Byte(f.popBase(), instr.Offset())))
	case code.OpI32Load16S:
		f.pushI32(int32(int16(f.module.mem0.Uint16(f.popBase(), instr.Offset()))))
	case code.OpI32Load16U:
		f.pushI32(int32(f.module.mem0.Uint16(f.popBase(), instr.Offset())))

	case code.OpI64Load8S:
		f.pushI64(int64(int8(f.module.mem0.Byte(f.popBase(), instr.Offset()))))
	case code.OpI64Load8U:
		f.pushI64(int64(f.module.mem0.Byte(f.popBase(), instr.Offset())))
	case code.OpI64Load16S:
		f.pushI64(int64(int16(f.module.mem0.Uint16(f.popBase(), instr.Offset()))))
	case code.OpI64Load16U:
		f.pushI64(int64(f.module.mem0.Uint16(f.popBase(), instr.Offset())))
	case code.OpI64Load32S:
		f.pushI64(int64(int32(f.module.mem0.Uint32(f.popBase(), instr.Offset()))))
	case code.OpI64Load32U:
		f.pushI64(int64(f.module.mem0.Uint32(f.popBase(), instr.Offset())))

	case code.OpI32Store:
		f.module.mem0.PutUint32(f.popU32(), f.popBase(), instr.Offset())
	case code.OpI64Store:
		f.module.mem0.PutUint64(f.popU64(), f.popBase(), instr.Offset())
	case code.OpF32Store:
		f.module.mem0.PutFloat32(f.popF32(), f.popBase(), instr.Offset())
	case code.OpF64Store:
		f.module.mem0.PutFloat64(f.popF64(), f.popBase(), instr.Offset())

	case code.OpI32Store8:
		f.module.mem0.PutByte(byte(f.popI32()), f.popBase(), instr.Offset())
	case code.OpI32Store16:
		f.module.mem0.PutUint16(uint16(f.popI32()), f.popBase(), instr.Offset())

	case code.OpI64Store8:
		f.module.mem0.PutByte(byte(f.popI64()), f.popBase(), instr.Offset())
	case code.OpI64Store16:
		f.module.mem0.PutUint16(uint16(f.popI64()), f.popBase(), instr.Offset())
	case code.OpI64Store32:
		f.module.mem0.PutUint32(uint32(f.popI64()), f.popBase(), instr.Offset())

	case code.OpMemorySize:
		f.pushI32(int32(f.module.mem0.Size()))
	case code.OpMemoryGrow:
		result, err := f.module.mem0.Grow(uint32(f.popI32()))
		if err != nil {
			f.pushI32(-1)
		} else {
			f.pushI32(int32(result))
		}

	case code.OpI32Const:
		f.pushI32(instr.I32())
	case code.OpI64Const:
		f.pushI64(instr.I64())
	case code.OpF32Const:
		f.pushF32(instr.F32())
	case code.OpF64Const:
		f.pushF64(instr.F64())

	case code.OpI32Eqz:
		f.pushBool(f.popI32() == 0)
	case code.OpI32Eq:
		v2, v1 := f.pop2I32()
		f.pushBool(v1 == v2)
	case code.OpI32Ne:
		v2, v1 := f.pop2I32()
		f.pushBool(v1 != v2)
	case code.OpI32LtS:
		v2, v1 := f.pop2I32()
		f.pushBool(v1 < v2)
	case code.OpI32LtU:
		v2, v1 := f.pop2U32()
		f.pushBool(v1 < v2)
	case code.OpI32GtS:
		v2, v1 := f.pop2I32()
		f.pushBool(v1 > v2)
	case code.OpI32GtU:
		v2, v1 := f.pop2U32()
		f.pushBool(v1 > v2)
	case code.OpI32LeS:
		v2, v1 := f.pop2I32()
		f.pushBool(v1 <= v2)
	case code.OpI32LeU:
		v2, v1 := f.pop2U32()
		f.pushBool(v1 <= v2)
	case code.OpI32GeS:
		v2, v1 := f.pop2I32()
		f.pushBool(v1 >= v2)
	case code.OpI32GeU:
		v2, v1 := f.pop2U32()
		f.pushBool(v1 >= v2)

	case code.OpI64Eqz:
		f.pushBool(f.popI64() == 0)
	case code.OpI64Eq:
		v2, v1 := f.pop2I64()
		f.pushBool(v1 == v2)
	case code.OpI64Ne:
		v2, v1 := f.pop2I64()
		f.pushBool(v1 != v2)
	case code.OpI64LtS:
		v2, v1 := f.pop2I64()
		f.pushBool(v1 < v2)
	case code.OpI64LtU:
		v2, v1 := f.pop2U64()
		f.pushBool(v1 < v2)
	case code.OpI64GtS:
		v2, v1 := f.pop2I64()
		f.pushBool(v1 > v2)
	case code.OpI64GtU:
		v2, v1 := f.pop2U64()
		f.pushBool(v1 > v2)
	case code.OpI64LeS:
		v2, v1 := f.pop2I64()
		f.pushBool(v1 <= v2)
	case code.OpI64LeU:
		v2, v1 := f.pop2U64()
		f.pushBool(v1 <= v2)
	case code.OpI64GeS:
		v2, v1 := f.pop2I64()
		f.pushBool(v1 >= v2)
	case code.OpI64GeU:
		v2, v1 := f.pop2U64()
		f.pushBool(v1 >= v2)

	case code.OpF32Eq:
		v2, v1 := f.pop2F32()
		f.pushBool(v1 == v2)
	case code.OpF32Ne:
		v2, v1 := f.pop2F32()
		f.pushBool(v1 != v2)
	case code.OpF32Lt:
		v2, v1 := f.pop2F32()
		f.pushBool(v1 < v2)
	case code.OpF32Gt:
		v2, v1 := f.pop2F32()
		f.pushBool(v1 > v2)
	case code.OpF32Le:
		v2, v1 := f.pop2F32()
		f.pushBool(v1 <= v2)
	case code.OpF32Ge:
		v2, v1 := f.pop2F32()
		f.pushBool(v1 >= v2)

	case code.OpF64Eq:
		v2, v1 := f.pop2F64()
		f.pushBool(v1 == v2)
	case code.OpF64Ne:
		v2, v1 := f.pop2F64()
		f.pushBool(v1 != v2)
	case code.OpF64Lt:
		v2, v1 := f.pop2F64()
		f.pushBool(v1 < v2)
	case code.OpF64Gt:
		v2, v1 := f.pop2F64()
		f.pushBool(v1 > v2)
	case code.OpF64Le:
		v2, v1 := f.pop2F64()
		f.pushBool(v1 <= v2)
	case code.OpF64Ge:
		v2, v1 := f.pop2F64()
		f.pushBool(v1 >= v2)

	case code.OpI32Clz:
		f.pushI(bits.LeadingZeros32(f.popU32()))
	case code.OpI32Ctz:
		f.pushI(bits.TrailingZeros32(f.popU32()))
	case code.OpI32Popcnt:
		f.pushI(bits.OnesCount32(f.popU32()))
	case code.OpI32Add:
		v2, v1 := f.pop2I32()
		f.pushI32(v1 + v2)
	case code.OpI32Sub:
		v2, v1 := f.pop2I32()
		f.pushI32(v1 - v2)
	case code.OpI32Mul:
		v2, v1 := f.pop2I32()
		f.pushI32(v1 * v2)
	case code.OpI32DivS:
		v2, v1 := f.pop2I32()
		f.pushI32(exec.I32DivS(v1, v2))
	case code.OpI32DivU:
		v2, v1 := f.pop2U32()
		f.pushU32(v1 / v2)
	case code.OpI32RemS:
		v2, v1 := f.pop2I32()
		f.pushI32(v1 % v2)
	case code.OpI32RemU:
		v2, v1 := f.pop2U32()
		f.pushU32(v1 % v2)
	case code.OpI32And:
		v2, v1 := f.pop2I32()
		f.pushI32(v1 & v2)
	case code.OpI32Or:
		v2, v1 := f.pop2I32()
		f.pushI32(v1 | v2)
	case code.OpI32Xor:
		v2, v1 := f.pop2I32()
		f.pushI32(v1 ^ v2)
	case code.OpI32Shl:
		v2, v1 := f.pop2I32()
		f.pushI32(v1 << (v2 & 31))
	case code.OpI32ShrS:
		v2, v1 := f.pop2I32()
		f.pushI32(v1 >> (v2 & 31))
	case code.OpI32ShrU:
		v2, v1 := f.pop2U32()
		f.pushU32(v1 >> (v2 & 31))
	case code.OpI32Rotl:
		v2, v1 := f.popI(), f.popU32()
		f.pushU32(bits.RotateLeft32(v1, v2))
	case code.OpI32Rotr:
		v2, v1 := f.popI(), f.popU32()
		f.pushU32(bits.RotateLeft32(v1, -v2))

	case code.OpI64Clz:
		f.pushI(bits.LeadingZeros64(f.popU64()))
	case code.OpI64Ctz:
		f.pushI(bits.TrailingZeros64(f.popU64()))
	case code.OpI64Popcnt:
		f.pushI(bits.OnesCount64(f.popU64()))
	case code.OpI64Add:
		v2, v1 := f.pop2I64()
		f.pushI64(v1 + v2)
	case code.OpI64Sub:
		v2, v1 := f.pop2I64()
		f.pushI64(v1 - v2)
	case code.OpI64Mul:
		v2, v1 := f.pop2I64()
		f.pushI64(v1 * v2)
	case code.OpI64DivS:
		v2, v1 := f.pop2I64()
		f.pushI64(exec.I64DivS(v1, v2))
	case code.OpI64DivU:
		v2, v1 := f.pop2U64()
		f.pushU64(v1 / v2)
	case code.OpI64RemS:
		v2, v1 := f.pop2I64()
		f.pushI64(v1 % v2)
	case code.OpI64RemU:
		v2, v1 := f.pop2U64()
		f.pushU64(v1 % v2)
	case code.OpI64And:
		v2, v1 := f.pop2I64()
		f.pushI64(v1 & v2)
	case code.OpI64Or:
		v2, v1 := f.pop2I64()
		f.pushI64(v1 | v2)
	case code.OpI64Xor:
		v2, v1 := f.pop2I64()
		f.pushI64(v1 ^ v2)
	case code.OpI64Shl:
		v2, v1 := f.pop2I64()
		f.pushI64(v1 << (v2 & 63))
	case code.OpI64ShrS:
		v2, v1 := f.pop2I64()
		f.pushI64(v1 >> (v2 & 63))
	case code.OpI64ShrU:
		v2, v1 := f.pop2U64()
		f.pushU64(v1 >> (v2 & 63))
	case code.OpI64Rotl:
		v2, v1 := f.popI(), f.popU64()
		f.pushU64(bits.RotateLeft64(v1, v2))
	case code.OpI64Rotr:
		v2, v1 := f.popI(), f.popU64()
		f.pushU64(bits.RotateLeft64(v1, -v2))

	case code.OpF32Abs:
		f.pushF32(float32(math.Abs(float64(f.popF32()))))
	case code.OpF32Neg:
		f.pushF32(-f.popF32())
	case code.OpF32Ceil:
		f.pushF32(float32(math.Ceil(float64(f.popF32()))))
	case code.OpF32Floor:
		f.pushF32(float32(math.Floor(float64(f.popF32()))))
	case code.OpF32Trunc:
		f.pushF32(float32(math.Trunc(float64(f.popF32()))))
	case code.OpF32Nearest:
		f.pushF32(float32(math.RoundToEven(float64(f.popF32()))))
	case code.OpF32Sqrt:
		f.pushF32(float32(math.Sqrt(float64(f.popF32()))))
	case code.OpF32Add:
		v2, v1 := f.pop2F32()
		f.pushF32(v1 + v2)
	case code.OpF32Sub:
		v2, v1 := f.pop2F32()
		f.pushF32(v1 - v2)
	case code.OpF32Mul:
		v2, v1 := f.pop2F32()
		f.pushF32(v1 * v2)
	case code.OpF32Div:
		v2, v1 := f.pop2F32()
		f.pushF32(v1 / v2)
	case code.OpF32Min:
		v2, v1 := f.pop2F32()
		f.pushF32(float32(exec.Fmin(float64(v1), float64(v2))))
	case code.OpF32Max:
		v2, v1 := f.pop2F32()
		f.pushF32(float32(exec.Fmax(float64(v1), float64(v2))))
	case code.OpF32Copysign:
		v2, v1 := f.pop2F32()
		f.pushF32(float32(math.Copysign(float64(v1), float64(v2))))

	case code.OpF64Abs:
		f.pushF64(math.Abs(f.popF64()))
	case code.OpF64Neg:
		f.pushF64(-f.popF64())
	case code.OpF64Ceil:
		f.pushF64(math.Ceil(f.popF64()))
	case code.OpF64Floor:
		f.pushF64(math.Floor(f.popF64()))
	case code.OpF64Trunc:
		f.pushF64(math.Trunc(f.popF64()))
	case code.OpF64Nearest:
		f.pushF64(math.RoundToEven(f.popF64()))
	case code.OpF64Sqrt:
		f.pushF64(math.Sqrt(f.popF64()))
	case code.OpF64Add:
		v2, v1 := f.pop2F64()
		f.pushF64(v1 + v2)
	case code.OpF64Sub:
		v2, v1 := f.pop2F64()
		f.pushF64(v1 - v2)
	case code.OpF64Mul:
		v2, v1 := f.pop2F64()
		f.pushF64(v1 * v2)
	case code.OpF64Div:
		v2, v1 := f.pop2F64()
		f.pushF64(v1 / v2)
	case code.OpF64Min:
		v2, v1 := f.pop2F64()
		f.pushF64(exec.Fmin(v1, v2))
	case code.OpF64Max:
		v2, v1 := f.pop2F64()
		f.pushF64(exec.Fmax(v1, v2))
	case code.OpF64Copysign:
		v2, v1 := f.pop2F64()
		f.pushF64(math.Copysign(v1, v2))

	case code.OpI32WrapI64:
		f.pushI32(int32(f.popI64()))
	case code.OpI32TruncF32S:
		f.pushI32(exec.I32TruncS(float64(f.popF32())))
	case code.OpI32TruncF32U:
		f.pushU32(exec.I32TruncU(float64(f.popF32())))
	case code.OpI32TruncF64S:
		f.pushI32(exec.I32TruncS(f.popF64()))
	case code.OpI32TruncF64U:
		f.pushU32(exec.I32TruncU(f.popF64()))

	case code.OpI64ExtendI32S:
		f.pushI64(int64(f.popI32()))
	case code.OpI64ExtendI32U:
		f.pushI64(int64(f.popU32()))
	case code.OpI64TruncF32S:
		f.pushI64(exec.I64TruncS(float64(f.popF32())))
	case code.OpI64TruncF32U:
		f.pushU64(exec.I64TruncU(float64(f.popF32())))
	case code.OpI64TruncF64S:
		f.pushI64(exec.I64TruncS(f.popF64()))
	case code.OpI64TruncF64U:
		f.pushU64(exec.I64TruncU(f.popF64()))

	case code.OpF32ConvertI32S:
		f.pushF32(float32(f.popI32()))
	case code.OpF32ConvertI32U:
		f.pushF32(float32(f.popU32()))
	case code.OpF32ConvertI64S:
		f.pushF32(float32(f.popI64()))
	case code.OpF32ConvertI64U:
		f.pushF32(float32(f.popU64()))
	case code.OpF32DemoteF64:
		f.pushF32(float32(f.popF64()))

	case code.OpF64ConvertI32S:
		f.pushF64(float64(f.popI32()))
	case code.OpF64ConvertI32U:
		f.pushF64(float64(f.popU32()))
	case code.OpF64ConvertI64S:
		f.pushF64(float64(f.popI64()))
	case code.OpF64ConvertI64U:
		f.pushF64(float64(f.popU64()))
	case code.OpF64PromoteF32:
		f.pushF64(float64(f.popF32()))

	case code.OpI32ReinterpretF32:
		f.pushU32(math.Float32bits(f.popF32()))
	case code.OpI64ReinterpretF64:
		f.pushU64(math.Float64bits(f.popF64()))
	case code.OpF32ReinterpretI32:
		f.pushF32(math.Float32frombits(f.popU32()))
	case code.OpF64ReinterpretI64:
		f.pushF64(math.Float64frombits(f.popU64()))

	case code.OpI32Extend8S:
		f.pushI32(int32(int8(f.popI32())))
	case code.OpI32Extend16S:
		f.pushI32(int32(int16(f.popI32())))
	case code.OpI64Extend8S:
		f.pushI64(int64(int8(f.popI64())))
	case code.OpI64Extend16S:
		f.pushI64(int64(int16(f.popI64())))
	case code.OpI64Extend32S:
		f.pushI64(int64(int32(f.popI64())))

	case code.OpPrefix:
		switch instr.Immediate {
		case code.OpI32TruncSatF32S:
			f.pushI32(exec.I32TruncSatS(float64(f.popF32())))
		case code.OpI32TruncSatF32U:
			f.pushU32(exec.I32TruncSatU(float64(f.popF32())))
		case code.OpI32TruncSatF64S:
			f.pushI32(exec.I32TruncSatS(f.popF64()))
		case code.OpI32TruncSatF64U:
			f.pushU32(exec.I32TruncSatU(f.popF64()))
		case code.OpI64TruncSatF32S:
			f.pushI64(exec.I64TruncSatS(float64(f.popF32())))
		case code.OpI64TruncSatF32U:
			f.pushU64(exec.I64TruncSatU(float64(f.popF32())))
		case code.OpI64TruncSatF64S:
			f.pushI64(exec.I64TruncSatS(f.popF64()))
		case code.OpI64TruncSatF64U:
			f.pushU64(exec.I64TruncSatU(f.popF64()))
		}
	}

	return ip + 1
}

func (f *frame) runICode(fn *function) int {
	// Push the first label.
	f.blocks = f.blocks[:2]
	f.blocks[0] = uint64(len(fn.icode) - 1)
	f.blocks[1] = uint64(len(fn.signature.ReturnTypes))

	ip, body := 0, fn.icode
	for {
		instr := &body[ip]

		switch instr.Opcode {
		case code.OpUnreachable:
			f.trap(exec.TrapUnreachable)

		case code.OpNop:
			// no-op

		case code.OpBlock:
			f.pushContinuation(instr, false)
		case code.OpLoop:
			f.pushContinuation(instr, true)

		case code.OpIf:
			if !f.popBool() {
				if instr.Else() != 0 {
					f.pushContinuation(instr, false)
					ip = instr.Else() + 1
					continue
				}
				ip = instr.Continuation()
				continue
			}
			f.pushContinuation(instr, false)
		case code.OpElse:
			// This is the end of a taken if block.
			f.popContinuation()
			ip = instr.Continuation()
			continue
		case code.OpEnd:
			if ip == len(body)-1 {
				return ip
			}
			f.popContinuation()

		case code.OpBr:
			ip = f.branch(instr.Labelidx())
			continue
		case code.OpBrIf:
			if f.popBool() {
				ip = f.branch(instr.Labelidx())
				continue
			}
		case code.OpBrTable:
			if li := int(f.popI32()); li >= 0 && li < len(instr.Labels) {
				ip = f.branch(instr.Labels[li])
				continue
			}
			ip = f.branch(instr.Default())
			continue

		case code.OpReturn:
			return ip

		case code.OpCall:
			funcidx := instr.Funcidx()
			if funcidx < uint32(len(f.module.importedFunctions)) {
				f.invoke(f.module.importedFunctions[funcidx])
			} else {
				f.invokeDirect(&f.module.functions[funcidx-uint32(len(f.module.importedFunctions))])
			}
		case code.OpCallIndirect:
			table := f.module.table0.Entries()

			tableidx := f.popI32()
			if uint32(tableidx) >= uint32(len(table)) {
				f.trap(exec.TrapUndefinedElement)
			}

			function := table[int(tableidx)]
			if function == nil {
				f.trap(exec.TrapUninitializedElement)
			}

			expectedSig := f.module.types[int(instr.Typeidx())]
			actualSig := function.GetSignature()
			if !actualSig.Equals(expectedSig) {
				f.trap(exec.TrapIndirectCallTypeMismatch)
			}

			f.invoke(function)

		case code.OpDrop:
			f.pop()

		case code.OpSelect:
			condition, v2, v1 := f.popBool(), f.pop(), f.pop()
			if condition {
				f.push(v1)
			} else {
				f.push(v2)
			}

		case code.OpLocalGet:
			f.push(f.get(instr.Localidx()))
		case code.OpLocalSet:
			f.set(instr.Localidx(), f.pop())
		case code.OpLocalTee:
			v := f.pop()
			f.set(instr.Localidx(), v)
			f.push(v)

		case code.OpGlobalGet:
			global, _ := f.module.getGlobal(instr.Globalidx())
			f.push(global.Get())
		case code.OpGlobalSet:
			global, _ := f.module.getGlobal(instr.Globalidx())
			global.Set(f.pop())

		case code.OpI32Load:
			f.pushI32(int32(f.module.mem0.Uint32(f.popBase(), instr.Offset())))
		case code.OpI64Load:
			f.pushI64(int64(f.module.mem0.Uint64(f.popBase(), instr.Offset())))
		case code.OpF32Load:
			f.pushF32(f.module.mem0.Float32(f.popBase(), instr.Offset()))
		case code.OpF64Load:
			f.pushF64(f.module.mem0.Float64(f.popBase(), instr.Offset()))

		case code.OpI32Load8S:
			f.pushI32(int32(int8(f.module.mem0.Byte(f.popBase(), instr.Offset()))))
		case code.OpI32Load8U:
			f.pushI32(int32(f.module.mem0.Byte(f.popBase(), instr.Offset())))
		case code.OpI32Load16S:
			f.pushI32(int32(int16(f.module.mem0.Uint16(f.popBase(), instr.Offset()))))
		case code.OpI32Load16U:
			f.pushI32(int32(f.module.mem0.Uint16(f.popBase(), instr.Offset())))

		case code.OpI64Load8S:
			f.pushI64(int64(int8(f.module.mem0.Byte(f.popBase(), instr.Offset()))))
		case code.OpI64Load8U:
			f.pushI64(int64(f.module.mem0.Byte(f.popBase(), instr.Offset())))
		case code.OpI64Load16S:
			f.pushI64(int64(int16(f.module.mem0.Uint16(f.popBase(), instr.Offset()))))
		case code.OpI64Load16U:
			f.pushI64(int64(f.module.mem0.Uint16(f.popBase(), instr.Offset())))
		case code.OpI64Load32S:
			f.pushI64(int64(int32(f.module.mem0.Uint32(f.popBase(), instr.Offset()))))
		case code.OpI64Load32U:
			f.pushI64(int64(f.module.mem0.Uint32(f.popBase(), instr.Offset())))

		case code.OpI32Store:
			f.module.mem0.PutUint32(f.popU32(), f.popBase(), instr.Offset())
		case code.OpI64Store:
			f.module.mem0.PutUint64(f.popU64(), f.popBase(), instr.Offset())
		case code.OpF32Store:
			f.module.mem0.PutFloat32(f.popF32(), f.popBase(), instr.Offset())
		case code.OpF64Store:
			f.module.mem0.PutFloat64(f.popF64(), f.popBase(), instr.Offset())

		case code.OpI32Store8:
			f.module.mem0.PutByte(byte(f.popI32()), f.popBase(), instr.Offset())
		case code.OpI32Store16:
			f.module.mem0.PutUint16(uint16(f.popI32()), f.popBase(), instr.Offset())

		case code.OpI64Store8:
			f.module.mem0.PutByte(byte(f.popI64()), f.popBase(), instr.Offset())
		case code.OpI64Store16:
			f.module.mem0.PutUint16(uint16(f.popI64()), f.popBase(), instr.Offset())
		case code.OpI64Store32:
			f.module.mem0.PutUint32(uint32(f.popI64()), f.popBase(), instr.Offset())

		case code.OpMemorySize:
			f.pushI32(int32(f.module.mem0.Size()))
		case code.OpMemoryGrow:
			result, err := f.module.mem0.Grow(uint32(f.popI32()))
			if err != nil {
				f.pushI32(-1)
			} else {
				f.pushI32(int32(result))
			}

		case code.OpI32Const:
			f.pushI32(instr.I32())
		case code.OpI64Const:
			f.pushI64(instr.I64())
		case code.OpF32Const:
			f.pushF32(instr.F32())
		case code.OpF64Const:
			f.pushF64(instr.F64())

		case code.OpI32Eqz:
			f.pushBool(f.popI32() == 0)
		case code.OpI32Eq:
			v2, v1 := f.pop2I32()
			f.pushBool(v1 == v2)
		case code.OpI32Ne:
			v2, v1 := f.pop2I32()
			f.pushBool(v1 != v2)
		case code.OpI32LtS:
			v2, v1 := f.pop2I32()
			f.pushBool(v1 < v2)
		case code.OpI32LtU:
			v2, v1 := f.pop2U32()
			f.pushBool(v1 < v2)
		case code.OpI32GtS:
			v2, v1 := f.pop2I32()
			f.pushBool(v1 > v2)
		case code.OpI32GtU:
			v2, v1 := f.pop2U32()
			f.pushBool(v1 > v2)
		case code.OpI32LeS:
			v2, v1 := f.pop2I32()
			f.pushBool(v1 <= v2)
		case code.OpI32LeU:
			v2, v1 := f.pop2U32()
			f.pushBool(v1 <= v2)
		case code.OpI32GeS:
			v2, v1 := f.pop2I32()
			f.pushBool(v1 >= v2)
		case code.OpI32GeU:
			v2, v1 := f.pop2U32()
			f.pushBool(v1 >= v2)

		case code.OpI64Eqz:
			f.pushBool(f.popI64() == 0)
		case code.OpI64Eq:
			v2, v1 := f.pop2I64()
			f.pushBool(v1 == v2)
		case code.OpI64Ne:
			v2, v1 := f.pop2I64()
			f.pushBool(v1 != v2)
		case code.OpI64LtS:
			v2, v1 := f.pop2I64()
			f.pushBool(v1 < v2)
		case code.OpI64LtU:
			v2, v1 := f.pop2U64()
			f.pushBool(v1 < v2)
		case code.OpI64GtS:
			v2, v1 := f.pop2I64()
			f.pushBool(v1 > v2)
		case code.OpI64GtU:
			v2, v1 := f.pop2U64()
			f.pushBool(v1 > v2)
		case code.OpI64LeS:
			v2, v1 := f.pop2I64()
			f.pushBool(v1 <= v2)
		case code.OpI64LeU:
			v2, v1 := f.pop2U64()
			f.pushBool(v1 <= v2)
		case code.OpI64GeS:
			v2, v1 := f.pop2I64()
			f.pushBool(v1 >= v2)
		case code.OpI64GeU:
			v2, v1 := f.pop2U64()
			f.pushBool(v1 >= v2)

		case code.OpF32Eq:
			v2, v1 := f.pop2F32()
			f.pushBool(v1 == v2)
		case code.OpF32Ne:
			v2, v1 := f.pop2F32()
			f.pushBool(v1 != v2)
		case code.OpF32Lt:
			v2, v1 := f.pop2F32()
			f.pushBool(v1 < v2)
		case code.OpF32Gt:
			v2, v1 := f.pop2F32()
			f.pushBool(v1 > v2)
		case code.OpF32Le:
			v2, v1 := f.pop2F32()
			f.pushBool(v1 <= v2)
		case code.OpF32Ge:
			v2, v1 := f.pop2F32()
			f.pushBool(v1 >= v2)

		case code.OpF64Eq:
			v2, v1 := f.pop2F64()
			f.pushBool(v1 == v2)
		case code.OpF64Ne:
			v2, v1 := f.pop2F64()
			f.pushBool(v1 != v2)
		case code.OpF64Lt:
			v2, v1 := f.pop2F64()
			f.pushBool(v1 < v2)
		case code.OpF64Gt:
			v2, v1 := f.pop2F64()
			f.pushBool(v1 > v2)
		case code.OpF64Le:
			v2, v1 := f.pop2F64()
			f.pushBool(v1 <= v2)
		case code.OpF64Ge:
			v2, v1 := f.pop2F64()
			f.pushBool(v1 >= v2)

		case code.OpI32Clz:
			f.pushI(bits.LeadingZeros32(f.popU32()))
		case code.OpI32Ctz:
			f.pushI(bits.TrailingZeros32(f.popU32()))
		case code.OpI32Popcnt:
			f.pushI(bits.OnesCount32(f.popU32()))
		case code.OpI32Add:
			v2, v1 := f.pop2I32()
			f.pushI32(v1 + v2)
		case code.OpI32Sub:
			v2, v1 := f.pop2I32()
			f.pushI32(v1 - v2)
		case code.OpI32Mul:
			v2, v1 := f.pop2I32()
			f.pushI32(v1 * v2)
		case code.OpI32DivS:
			v2, v1 := f.pop2I32()
			f.pushI32(exec.I32DivS(v1, v2))
		case code.OpI32DivU:
			v2, v1 := f.pop2U32()
			f.pushU32(v1 / v2)
		case code.OpI32RemS:
			v2, v1 := f.pop2I32()
			f.pushI32(v1 % v2)
		case code.OpI32RemU:
			v2, v1 := f.pop2U32()
			f.pushU32(v1 % v2)
		case code.OpI32And:
			v2, v1 := f.pop2I32()
			f.pushI32(v1 & v2)
		case code.OpI32Or:
			v2, v1 := f.pop2I32()
			f.pushI32(v1 | v2)
		case code.OpI32Xor:
			v2, v1 := f.pop2I32()
			f.pushI32(v1 ^ v2)
		case code.OpI32Shl:
			v2, v1 := f.pop2I32()
			f.pushI32(v1 << (v2 & 31))
		case code.OpI32ShrS:
			v2, v1 := f.pop2I32()
			f.pushI32(v1 >> (v2 & 31))
		case code.OpI32ShrU:
			v2, v1 := f.pop2U32()
			f.pushU32(v1 >> (v2 & 31))
		case code.OpI32Rotl:
			v2, v1 := f.popI(), f.popU32()
			f.pushU32(bits.RotateLeft32(v1, v2))
		case code.OpI32Rotr:
			v2, v1 := f.popI(), f.popU32()
			f.pushU32(bits.RotateLeft32(v1, -v2))

		case code.OpI64Clz:
			f.pushI(bits.LeadingZeros64(f.popU64()))
		case code.OpI64Ctz:
			f.pushI(bits.TrailingZeros64(f.popU64()))
		case code.OpI64Popcnt:
			f.pushI(bits.OnesCount64(f.popU64()))
		case code.OpI64Add:
			v2, v1 := f.pop2I64()
			f.pushI64(v1 + v2)
		case code.OpI64Sub:
			v2, v1 := f.pop2I64()
			f.pushI64(v1 - v2)
		case code.OpI64Mul:
			v2, v1 := f.pop2I64()
			f.pushI64(v1 * v2)
		case code.OpI64DivS:
			v2, v1 := f.pop2I64()
			f.pushI64(exec.I64DivS(v1, v2))
		case code.OpI64DivU:
			v2, v1 := f.pop2U64()
			f.pushU64(v1 / v2)
		case code.OpI64RemS:
			v2, v1 := f.pop2I64()
			f.pushI64(v1 % v2)
		case code.OpI64RemU:
			v2, v1 := f.pop2U64()
			f.pushU64(v1 % v2)
		case code.OpI64And:
			v2, v1 := f.pop2I64()
			f.pushI64(v1 & v2)
		case code.OpI64Or:
			v2, v1 := f.pop2I64()
			f.pushI64(v1 | v2)
		case code.OpI64Xor:
			v2, v1 := f.pop2I64()
			f.pushI64(v1 ^ v2)
		case code.OpI64Shl:
			v2, v1 := f.pop2I64()
			f.pushI64(v1 << (v2 & 63))
		case code.OpI64ShrS:
			v2, v1 := f.pop2I64()
			f.pushI64(v1 >> (v2 & 63))
		case code.OpI64ShrU:
			v2, v1 := f.pop2U64()
			f.pushU64(v1 >> (v2 & 63))
		case code.OpI64Rotl:
			v2, v1 := f.popI(), f.popU64()
			f.pushU64(bits.RotateLeft64(v1, v2))
		case code.OpI64Rotr:
			v2, v1 := f.popI(), f.popU64()
			f.pushU64(bits.RotateLeft64(v1, -v2))

		case code.OpF32Abs:
			f.pushF32(float32(math.Abs(float64(f.popF32()))))
		case code.OpF32Neg:
			f.pushF32(-f.popF32())
		case code.OpF32Ceil:
			f.pushF32(float32(math.Ceil(float64(f.popF32()))))
		case code.OpF32Floor:
			f.pushF32(float32(math.Floor(float64(f.popF32()))))
		case code.OpF32Trunc:
			f.pushF32(float32(math.Trunc(float64(f.popF32()))))
		case code.OpF32Nearest:
			f.pushF32(float32(math.RoundToEven(float64(f.popF32()))))
		case code.OpF32Sqrt:
			f.pushF32(float32(math.Sqrt(float64(f.popF32()))))
		case code.OpF32Add:
			v2, v1 := f.pop2F32()
			f.pushF32(v1 + v2)
		case code.OpF32Sub:
			v2, v1 := f.pop2F32()
			f.pushF32(v1 - v2)
		case code.OpF32Mul:
			v2, v1 := f.pop2F32()
			f.pushF32(v1 * v2)
		case code.OpF32Div:
			v2, v1 := f.pop2F32()
			f.pushF32(v1 / v2)
		case code.OpF32Min:
			v2, v1 := f.pop2F32()
			f.pushF32(float32(exec.Fmin(float64(v1), float64(v2))))
		case code.OpF32Max:
			v2, v1 := f.pop2F32()
			f.pushF32(float32(exec.Fmax(float64(v1), float64(v2))))
		case code.OpF32Copysign:
			v2, v1 := f.pop2F32()
			f.pushF32(float32(math.Copysign(float64(v1), float64(v2))))

		case code.OpF64Abs:
			f.pushF64(math.Abs(f.popF64()))
		case code.OpF64Neg:
			f.pushF64(-f.popF64())
		case code.OpF64Ceil:
			f.pushF64(math.Ceil(f.popF64()))
		case code.OpF64Floor:
			f.pushF64(math.Floor(f.popF64()))
		case code.OpF64Trunc:
			f.pushF64(math.Trunc(f.popF64()))
		case code.OpF64Nearest:
			f.pushF64(math.RoundToEven(f.popF64()))
		case code.OpF64Sqrt:
			f.pushF64(math.Sqrt(f.popF64()))
		case code.OpF64Add:
			v2, v1 := f.pop2F64()
			f.pushF64(v1 + v2)
		case code.OpF64Sub:
			v2, v1 := f.pop2F64()
			f.pushF64(v1 - v2)
		case code.OpF64Mul:
			v2, v1 := f.pop2F64()
			f.pushF64(v1 * v2)
		case code.OpF64Div:
			v2, v1 := f.pop2F64()
			f.pushF64(v1 / v2)
		case code.OpF64Min:
			v2, v1 := f.pop2F64()
			f.pushF64(exec.Fmin(v1, v2))
		case code.OpF64Max:
			v2, v1 := f.pop2F64()
			f.pushF64(exec.Fmax(v1, v2))
		case code.OpF64Copysign:
			v2, v1 := f.pop2F64()
			f.pushF64(math.Copysign(v1, v2))

		case code.OpI32WrapI64:
			f.pushI32(int32(f.popI64()))
		case code.OpI32TruncF32S:
			f.pushI32(exec.I32TruncS(float64(f.popF32())))
		case code.OpI32TruncF32U:
			f.pushU32(exec.I32TruncU(float64(f.popF32())))
		case code.OpI32TruncF64S:
			f.pushI32(exec.I32TruncS(f.popF64()))
		case code.OpI32TruncF64U:
			f.pushU32(exec.I32TruncU(f.popF64()))

		case code.OpI64ExtendI32S:
			f.pushI64(int64(f.popI32()))
		case code.OpI64ExtendI32U:
			f.pushI64(int64(f.popU32()))
		case code.OpI64TruncF32S:
			f.pushI64(exec.I64TruncS(float64(f.popF32())))
		case code.OpI64TruncF32U:
			f.pushU64(exec.I64TruncU(float64(f.popF32())))
		case code.OpI64TruncF64S:
			f.pushI64(exec.I64TruncS(f.popF64()))
		case code.OpI64TruncF64U:
			f.pushU64(exec.I64TruncU(f.popF64()))

		case code.OpF32ConvertI32S:
			f.pushF32(float32(f.popI32()))
		case code.OpF32ConvertI32U:
			f.pushF32(float32(f.popU32()))
		case code.OpF32ConvertI64S:
			f.pushF32(float32(f.popI64()))
		case code.OpF32ConvertI64U:
			f.pushF32(float32(f.popU64()))
		case code.OpF32DemoteF64:
			f.pushF32(float32(f.popF64()))

		case code.OpF64ConvertI32S:
			f.pushF64(float64(f.popI32()))
		case code.OpF64ConvertI32U:
			f.pushF64(float64(f.popU32()))
		case code.OpF64ConvertI64S:
			f.pushF64(float64(f.popI64()))
		case code.OpF64ConvertI64U:
			f.pushF64(float64(f.popU64()))
		case code.OpF64PromoteF32:
			f.pushF64(float64(f.popF32()))

		case code.OpI32ReinterpretF32:
			f.pushU32(math.Float32bits(f.popF32()))
		case code.OpI64ReinterpretF64:
			f.pushU64(math.Float64bits(f.popF64()))
		case code.OpF32ReinterpretI32:
			f.pushF32(math.Float32frombits(f.popU32()))
		case code.OpF64ReinterpretI64:
			f.pushF64(math.Float64frombits(f.popU64()))

		case code.OpI32Extend8S:
			f.pushI32(int32(int8(f.popI32())))
		case code.OpI32Extend16S:
			f.pushI32(int32(int16(f.popI32())))
		case code.OpI64Extend8S:
			f.pushI64(int64(int8(f.popI64())))
		case code.OpI64Extend16S:
			f.pushI64(int64(int16(f.popI64())))
		case code.OpI64Extend32S:
			f.pushI64(int64(int32(f.popI64())))

		case code.OpPrefix:
			switch instr.Immediate {
			case code.OpI32TruncSatF32S:
				f.pushI32(exec.I32TruncSatS(float64(f.popF32())))
			case code.OpI32TruncSatF32U:
				f.pushU32(exec.I32TruncSatU(float64(f.popF32())))
			case code.OpI32TruncSatF64S:
				f.pushI32(exec.I32TruncSatS(f.popF64()))
			case code.OpI32TruncSatF64U:
				f.pushU32(exec.I32TruncSatU(f.popF64()))
			case code.OpI64TruncSatF32S:
				f.pushI64(exec.I64TruncSatS(float64(f.popF32())))
			case code.OpI64TruncSatF32U:
				f.pushU64(exec.I64TruncSatU(float64(f.popF32())))
			case code.OpI64TruncSatF64S:
				f.pushI64(exec.I64TruncSatS(f.popF64()))
			case code.OpI64TruncSatF64U:
				f.pushU64(exec.I64TruncSatU(f.popF64()))
			}
		}

		ip++
	}
}
