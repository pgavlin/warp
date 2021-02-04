package interpreter

import (
	"math"
	"math/bits"

	"github.com/pgavlin/warp/exec"
)

type lframe []uint64

func (f lframe) bool(offset uint32) bool {
	return f[offset] != 0
}

func (f lframe) setBool(v bool, offset uint32) {
	if v {
		f[offset] = 1
	} else {
		f[offset] = 0
	}
}

func (f *frame) branchF(dest *label) int {
	stackHeight, arity := dest.stackHeight, dest.arity

	copy(f.stack[stackHeight:], f.stack[len(f.stack)-arity:])
	f.stack = f.stack[:stackHeight+arity]

	return dest.Continuation()
}

func (f *frame) runFCode(fn *function) {
	labels := fn.labels
	switches := fn.switches
	body := fn.fcode

	// establish the frame
	frame := lframe(f.locals[:fn.numLocals+fn.metrics.MaxStackDepth])

	ip := 0
	for {
		instr := &body[ip]

		switch instr.opcode {
		case fopUnreachable:
			f.trap(exec.TrapUnreachable)

		case fopNop:
			// no-op

		case fopIf:
			if !frame.bool(instr.src1) {
				l := &labels[instr.Labelidx()]
				if l.Else() != 0 {
					ip = l.Else()
					continue
				}
				ip = l.Continuation()
				continue
			}
		case fopElse:
			// This is the end of a taken if block.
			l := &labels[instr.Labelidx()]
			ip = l.Continuation()
			continue

		case fopBr:
			f.stack = f.stack[:instr.StackHeight()]
			ip = f.branchF(&labels[instr.Labelidx()])
			continue
		case fopBrIf:
			if frame.bool(instr.src1) {
				f.stack = f.stack[:instr.StackHeight()]
				ip = f.branchF(&labels[instr.Labelidx()])
				continue
			}
		case fopBrTable:
			f.stack = f.stack[:instr.StackHeight()]

			table := switches[instr.Switchidx()]
			if li := int(frame[instr.src1]); li >= 0 && li < len(table.indices) {
				ip = f.branchF(&labels[table.indices[li]])
				continue
			}
			ip = f.branchF(&labels[instr.Labelidx()])
			continue

		case fopBrL:
			ip = labels[instr.Labelidx()].Continuation()
			continue
		case fopBrIfL:
			if frame.bool(instr.src1) {
				ip = labels[instr.Labelidx()].Continuation()
				continue
			}
		case fopBrTableL:
			table := switches[instr.Switchidx()]
			if li := int(frame[instr.src1]); li >= 0 && li < len(table.indices) {
				ip = labels[table.indices[li]].Continuation()
				continue
			}
			ip = labels[instr.Labelidx()].Continuation()
			continue

		case fopReturn:
			f.stack = f.stack[:instr.StackHeight()]
			return

		case fopCall:
			f.stack = f.stack[:instr.StackHeight()]

			funcidx := instr.Funcidx()
			if funcidx < uint32(len(f.module.importedFunctions)) {
				f.invoke(f.module.importedFunctions[funcidx])
			} else {
				f.invokeDirect(&f.module.functions[funcidx-uint32(len(f.module.importedFunctions))])
			}
		case fopCallIndirect:
			table := f.module.table0.Entries()

			tableidx := int32(frame[instr.src1])
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

			f.stack = f.stack[:instr.StackHeight()]
			f.invoke(function)

		case fopSelect:
			if frame.bool(instr.src1) {
				frame[instr.dest] = frame[instr.Src2()]
			} else {
				frame[instr.dest] = frame[instr.Src3()]
			}

		case fopLocalGet:
			frame[instr.dest] = frame[instr.src1]
		case fopLocalSet:
			frame[instr.dest] = frame[instr.src1]

		case fopGlobalGet:
			global, _ := f.module.getGlobal(instr.Globalidx())
			frame[instr.dest] = global.Get()
		case fopGlobalSet:
			global, _ := f.module.getGlobal(instr.dest)
			global.Set(frame[instr.src1])

		case fopI32Load, fopF32Load:
			frame[instr.dest] = uint64(f.module.mem0.Uint32(uint32(frame[instr.src1]), instr.Offset()))
		case fopI64Load, fopF64Load:
			frame[instr.dest] = f.module.mem0.Uint64(uint32(frame[instr.src1]), instr.Offset())

		case fopI32Load8S:
			frame[instr.dest] = uint64(int32(int8(f.module.mem0.Byte(uint32(frame[instr.src1]), instr.Offset()))))
		case fopI32Load8U:
			frame[instr.dest] = uint64(int32(f.module.mem0.Byte(uint32(frame[instr.src1]), instr.Offset())))
		case fopI32Load16S:
			frame[instr.dest] = uint64(int32(int16(f.module.mem0.Uint16(uint32(frame[instr.src1]), instr.Offset()))))
		case fopI32Load16U:
			frame[instr.dest] = uint64(int32(f.module.mem0.Uint16(uint32(frame[instr.src1]), instr.Offset())))

		case fopI64Load8S:
			frame[instr.dest] = uint64(int64(int8(f.module.mem0.Byte(uint32(frame[instr.src1]), instr.Offset()))))
		case fopI64Load8U:
			frame[instr.dest] = uint64(int64(f.module.mem0.Byte(uint32(frame[instr.src1]), instr.Offset())))
		case fopI64Load16S:
			frame[instr.dest] = uint64(int64(int16(f.module.mem0.Uint16(uint32(frame[instr.src1]), instr.Offset()))))
		case fopI64Load16U:
			frame[instr.dest] = uint64(int64(f.module.mem0.Uint16(uint32(frame[instr.src1]), instr.Offset())))
		case fopI64Load32S:
			frame[instr.dest] = uint64(int64(int32(f.module.mem0.Uint32(uint32(frame[instr.src1]), instr.Offset()))))
		case fopI64Load32U:
			frame[instr.dest] = uint64(int64(f.module.mem0.Uint32(uint32(frame[instr.src1]), instr.Offset())))

		case fopI32Store, fopF32Store:
			f.module.mem0.PutUint32(uint32(frame[instr.src1]), uint32(frame[instr.dest]), instr.Offset())
		case fopI64Store, fopF64Store:
			f.module.mem0.PutUint64(uint64(frame[instr.src1]), uint32(frame[instr.dest]), instr.Offset())

		case fopI32Store8:
			f.module.mem0.PutByte(byte(frame[instr.src1]), uint32(frame[instr.dest]), instr.Offset())
		case fopI32Store16:
			f.module.mem0.PutUint16(uint16(frame[instr.src1]), uint32(frame[instr.dest]), instr.Offset())

		case fopI64Store8:
			f.module.mem0.PutByte(byte(frame[instr.src1]), uint32(frame[instr.dest]), instr.Offset())
		case fopI64Store16:
			f.module.mem0.PutUint16(uint16(frame[instr.src1]), uint32(frame[instr.dest]), instr.Offset())
		case fopI64Store32:
			f.module.mem0.PutUint32(uint32(frame[instr.src1]), uint32(frame[instr.dest]), instr.Offset())

		case fopMemorySize:
			frame[instr.dest] = uint64(int32(f.module.mem0.Size()))
		case fopMemoryGrow:
			result, err := f.module.mem0.Grow(uint32(frame[instr.src1]))
			if err != nil {
				i := -1
				result = uint32(i)
			}
			frame[instr.dest] = uint64(int32(result))

		case fopI32Const, fopF32Const, fopI64Const, fopF64Const:
			frame[instr.dest] = instr.src2

		case fopI32Eqz:
			frame.setBool(int32(frame[instr.src1]) == 0, instr.dest)
		case fopI32Eq:
			v2, v1 := int32(frame[instr.Src2()]), int32(frame[instr.src1])
			frame.setBool(v1 == v2, instr.dest)
		case fopI32Ne:
			v2, v1 := int32(frame[instr.Src2()]), int32(frame[instr.src1])
			frame.setBool(v1 != v2, instr.dest)
		case fopI32LtS:
			v2, v1 := int32(frame[instr.Src2()]), int32(frame[instr.src1])
			frame.setBool(v1 < v2, instr.dest)
		case fopI32LtU:
			v2, v1 := uint32(frame[instr.Src2()]), uint32(frame[instr.src1])
			frame.setBool(v1 < v2, instr.dest)
		case fopI32GtS:
			v2, v1 := int32(frame[instr.Src2()]), int32(frame[instr.src1])
			frame.setBool(v1 > v2, instr.dest)
		case fopI32GtU:
			v2, v1 := uint32(frame[instr.Src2()]), uint32(frame[instr.src1])
			frame.setBool(v1 > v2, instr.dest)
		case fopI32LeS:
			v2, v1 := int32(frame[instr.Src2()]), int32(frame[instr.src1])
			frame.setBool(v1 <= v2, instr.dest)
		case fopI32LeU:
			v2, v1 := uint32(frame[instr.Src2()]), uint32(frame[instr.src1])
			frame.setBool(v1 <= v2, instr.dest)
		case fopI32GeS:
			v2, v1 := int32(frame[instr.Src2()]), int32(frame[instr.src1])
			frame.setBool(v1 >= v2, instr.dest)
		case fopI32GeU:
			v2, v1 := uint32(frame[instr.Src2()]), uint32(frame[instr.src1])
			frame.setBool(v1 >= v2, instr.dest)

		case fopI64Eqz:
			frame.setBool(int64(frame[instr.src1]) == 0, instr.dest)
		case fopI64Eq:
			v2, v1 := int64(frame[instr.Src2()]), int64(frame[instr.src1])
			frame.setBool(v1 == v2, instr.dest)
		case fopI64Ne:
			v2, v1 := int64(frame[instr.Src2()]), int64(frame[instr.src1])
			frame.setBool(v1 != v2, instr.dest)
		case fopI64LtS:
			v2, v1 := int64(frame[instr.Src2()]), int64(frame[instr.src1])
			frame.setBool(v1 < v2, instr.dest)
		case fopI64LtU:
			v2, v1 := frame[instr.Src2()], frame[instr.src1]
			frame.setBool(v1 < v2, instr.dest)
		case fopI64GtS:
			v2, v1 := int64(frame[instr.Src2()]), int64(frame[instr.src1])
			frame.setBool(v1 > v2, instr.dest)
		case fopI64GtU:
			v2, v1 := frame[instr.Src2()], frame[instr.src1]
			frame.setBool(v1 > v2, instr.dest)
		case fopI64LeS:
			v2, v1 := int64(frame[instr.Src2()]), int64(frame[instr.src1])
			frame.setBool(v1 <= v2, instr.dest)
		case fopI64LeU:
			v2, v1 := frame[instr.Src2()], frame[instr.src1]
			frame.setBool(v1 <= v2, instr.dest)
		case fopI64GeS:
			v2, v1 := int64(frame[instr.Src2()]), int64(frame[instr.src1])
			frame.setBool(v1 >= v2, instr.dest)
		case fopI64GeU:
			v2, v1 := frame[instr.Src2()], frame[instr.src1]
			frame.setBool(v1 >= v2, instr.dest)

		case fopF32Eq:
			v2, v1 := math.Float32frombits(uint32(frame[instr.Src2()])), math.Float32frombits(uint32(frame[instr.src1]))
			frame.setBool(v1 == v2, instr.dest)
		case fopF32Ne:
			v2, v1 := math.Float32frombits(uint32(frame[instr.Src2()])), math.Float32frombits(uint32(frame[instr.src1]))
			frame.setBool(v1 != v2, instr.dest)
		case fopF32Lt:
			v2, v1 := math.Float32frombits(uint32(frame[instr.Src2()])), math.Float32frombits(uint32(frame[instr.src1]))
			frame.setBool(v1 < v2, instr.dest)
		case fopF32Gt:
			v2, v1 := math.Float32frombits(uint32(frame[instr.Src2()])), math.Float32frombits(uint32(frame[instr.src1]))
			frame.setBool(v1 > v2, instr.dest)
		case fopF32Le:
			v2, v1 := math.Float32frombits(uint32(frame[instr.Src2()])), math.Float32frombits(uint32(frame[instr.src1]))
			frame.setBool(v1 <= v2, instr.dest)
		case fopF32Ge:
			v2, v1 := math.Float32frombits(uint32(frame[instr.Src2()])), math.Float32frombits(uint32(frame[instr.src1]))
			frame.setBool(v1 >= v2, instr.dest)

		case fopF64Eq:
			v2, v1 := math.Float64frombits(frame[instr.Src2()]), math.Float64frombits(frame[instr.src1])
			frame.setBool(v1 == v2, instr.dest)
		case fopF64Ne:
			v2, v1 := math.Float64frombits(frame[instr.Src2()]), math.Float64frombits(frame[instr.src1])
			frame.setBool(v1 != v2, instr.dest)
		case fopF64Lt:
			v2, v1 := math.Float64frombits(frame[instr.Src2()]), math.Float64frombits(frame[instr.src1])
			frame.setBool(v1 < v2, instr.dest)
		case fopF64Gt:
			v2, v1 := math.Float64frombits(frame[instr.Src2()]), math.Float64frombits(frame[instr.src1])
			frame.setBool(v1 > v2, instr.dest)
		case fopF64Le:
			v2, v1 := math.Float64frombits(frame[instr.Src2()]), math.Float64frombits(frame[instr.src1])
			frame.setBool(v1 <= v2, instr.dest)
		case fopF64Ge:
			v2, v1 := math.Float64frombits(frame[instr.Src2()]), math.Float64frombits(frame[instr.src1])
			frame.setBool(v1 >= v2, instr.dest)

		case fopI32Clz:
			frame[instr.dest] = uint64(bits.LeadingZeros32(uint32(frame[instr.src1])))
		case fopI32Ctz:
			frame[instr.dest] = uint64(bits.TrailingZeros32(uint32(frame[instr.src1])))
		case fopI32Popcnt:
			frame[instr.dest] = uint64(bits.OnesCount32(uint32(frame[instr.src1])))
		case fopI32Add:
			v2, v1 := int32(frame[instr.Src2()]), int32(frame[instr.src1])
			frame[instr.dest] = uint64(v1 + v2)
		case fopI32Sub:
			v2, v1 := int32(frame[instr.Src2()]), int32(frame[instr.src1])
			frame[instr.dest] = uint64(v1 - v2)
		case fopI32Mul:
			v2, v1 := int32(frame[instr.Src2()]), int32(frame[instr.src1])
			frame[instr.dest] = uint64(v1 * v2)
		case fopI32DivS:
			v2, v1 := int32(frame[instr.Src2()]), int32(frame[instr.src1])
			frame[instr.dest] = uint64(exec.I32DivS(v1, v2))
		case fopI32DivU:
			v2, v1 := uint32(frame[instr.Src2()]), uint32(frame[instr.src1])
			frame[instr.dest] = uint64(v1 / v2)
		case fopI32RemS:
			v2, v1 := int32(frame[instr.Src2()]), int32(frame[instr.src1])
			frame[instr.dest] = uint64(v1 % v2)
		case fopI32RemU:
			v2, v1 := uint32(frame[instr.Src2()]), uint32(frame[instr.src1])
			frame[instr.dest] = uint64(v1 % v2)
		case fopI32And:
			v2, v1 := int32(frame[instr.Src2()]), int32(frame[instr.src1])
			frame[instr.dest] = uint64(v1 & v2)
		case fopI32Or:
			v2, v1 := int32(frame[instr.Src2()]), int32(frame[instr.src1])
			frame[instr.dest] = uint64(v1 | v2)
		case fopI32Xor:
			v2, v1 := int32(frame[instr.Src2()]), int32(frame[instr.src1])
			frame[instr.dest] = uint64(v1 ^ v2)
		case fopI32Shl:
			v2, v1 := int32(frame[instr.Src2()]), int32(frame[instr.src1])
			frame[instr.dest] = uint64(v1 << (v2 & 31))
		case fopI32ShrS:
			v2, v1 := int32(frame[instr.Src2()]), int32(frame[instr.src1])
			frame[instr.dest] = uint64(v1 >> (v2 & 31))
		case fopI32ShrU:
			v2, v1 := uint32(frame[instr.Src2()]), uint32(frame[instr.src1])
			frame[instr.dest] = uint64(v1 >> (v2 & 31))
		case fopI32Rotl:
			v2, v1 := int(frame[instr.Src2()]), uint32(frame[instr.src1])
			frame[instr.dest] = uint64(bits.RotateLeft32(v1, v2))
		case fopI32Rotr:
			v2, v1 := int(frame[instr.Src2()]), uint32(frame[instr.src1])
			frame[instr.dest] = uint64(bits.RotateLeft32(v1, -v2))

		case fopI64Clz:
			frame[instr.dest] = uint64(bits.LeadingZeros64(uint64(frame[instr.src1])))
		case fopI64Ctz:
			frame[instr.dest] = uint64(bits.TrailingZeros64(uint64(frame[instr.src1])))
		case fopI64Popcnt:
			frame[instr.dest] = uint64(bits.OnesCount64(uint64(frame[instr.src1])))
		case fopI64Add:
			v2, v1 := int64(frame[instr.Src2()]), int64(frame[instr.src1])
			frame[instr.dest] = uint64(v1 + v2)
		case fopI64Sub:
			v2, v1 := int64(frame[instr.Src2()]), int64(frame[instr.src1])
			frame[instr.dest] = uint64(v1 - v2)
		case fopI64Mul:
			v2, v1 := int64(frame[instr.Src2()]), int64(frame[instr.src1])
			frame[instr.dest] = uint64(v1 * v2)
		case fopI64DivS:
			v2, v1 := int64(frame[instr.Src2()]), int64(frame[instr.src1])
			frame[instr.dest] = uint64(exec.I64DivS(v1, v2))
		case fopI64DivU:
			v2, v1 := frame[instr.Src2()], frame[instr.src1]
			frame[instr.dest] = uint64(v1 / v2)
		case fopI64RemS:
			v2, v1 := int64(frame[instr.Src2()]), int64(frame[instr.src1])
			frame[instr.dest] = uint64(v1 % v2)
		case fopI64RemU:
			v2, v1 := frame[instr.Src2()], frame[instr.src1]
			frame[instr.dest] = uint64(v1 % v2)
		case fopI64And:
			v2, v1 := int64(frame[instr.Src2()]), int64(frame[instr.src1])
			frame[instr.dest] = uint64(v1 & v2)
		case fopI64Or:
			v2, v1 := int64(frame[instr.Src2()]), int64(frame[instr.src1])
			frame[instr.dest] = uint64(v1 | v2)
		case fopI64Xor:
			v2, v1 := int64(frame[instr.Src2()]), int64(frame[instr.src1])
			frame[instr.dest] = uint64(v1 ^ v2)
		case fopI64Shl:
			v2, v1 := int64(frame[instr.Src2()]), int64(frame[instr.src1])
			frame[instr.dest] = uint64(v1 << (v2 & 63))
		case fopI64ShrS:
			v2, v1 := int64(frame[instr.Src2()]), int64(frame[instr.src1])
			frame[instr.dest] = uint64(v1 >> (v2 & 63))
		case fopI64ShrU:
			v2, v1 := frame[instr.Src2()], frame[instr.src1]
			frame[instr.dest] = uint64(v1 >> (v2 & 63))
		case fopI64Rotl:
			v2, v1 := int(frame[instr.Src2()]), uint64(frame[instr.src1])
			frame[instr.dest] = uint64(bits.RotateLeft64(v1, v2))
		case fopI64Rotr:
			v2, v1 := int(frame[instr.Src2()]), uint64(frame[instr.src1])
			frame[instr.dest] = uint64(bits.RotateLeft64(v1, -v2))

		case fopF32Abs:
			frame[instr.dest] = uint64(math.Float32bits(float32(math.Abs(float64(math.Float32frombits(uint32(frame[instr.src1])))))))
		case fopF32Neg:
			frame[instr.dest] = uint64(math.Float32bits(-math.Float32frombits(uint32(frame[instr.src1]))))
		case fopF32Ceil:
			frame[instr.dest] = uint64(math.Float32bits(float32(math.Ceil(float64(math.Float32frombits(uint32(frame[instr.src1])))))))
		case fopF32Floor:
			frame[instr.dest] = uint64(math.Float32bits(float32(math.Floor(float64(math.Float32frombits(uint32(frame[instr.src1])))))))
		case fopF32Trunc:
			frame[instr.dest] = uint64(math.Float32bits(float32(math.Trunc(float64(math.Float32frombits(uint32(frame[instr.src1])))))))
		case fopF32Nearest:
			frame[instr.dest] = uint64(math.Float32bits(float32(math.RoundToEven(float64(math.Float32frombits(uint32(frame[instr.src1])))))))
		case fopF32Sqrt:
			frame[instr.dest] = uint64(math.Float32bits(float32(math.Sqrt(float64(math.Float32frombits(uint32(frame[instr.src1])))))))
		case fopF32Add:
			v2, v1 := math.Float32frombits(uint32(frame[instr.Src2()])), math.Float32frombits(uint32(frame[instr.src1]))
			frame[instr.dest] = uint64(math.Float32bits(v1 + v2))
		case fopF32Sub:
			v2, v1 := math.Float32frombits(uint32(frame[instr.Src2()])), math.Float32frombits(uint32(frame[instr.src1]))
			frame[instr.dest] = uint64(math.Float32bits(v1 - v2))
		case fopF32Mul:
			v2, v1 := math.Float32frombits(uint32(frame[instr.Src2()])), math.Float32frombits(uint32(frame[instr.src1]))
			frame[instr.dest] = uint64(math.Float32bits(v1 * v2))
		case fopF32Div:
			v2, v1 := math.Float32frombits(uint32(frame[instr.Src2()])), math.Float32frombits(uint32(frame[instr.src1]))
			frame[instr.dest] = uint64(math.Float32bits(v1 / v2))
		case fopF32Min:
			v2, v1 := math.Float32frombits(uint32(frame[instr.Src2()])), math.Float32frombits(uint32(frame[instr.src1]))
			frame[instr.dest] = uint64(math.Float32bits(float32(exec.Fmin(float64(v1), float64(v2)))))
		case fopF32Max:
			v2, v1 := math.Float32frombits(uint32(frame[instr.Src2()])), math.Float32frombits(uint32(frame[instr.src1]))
			frame[instr.dest] = uint64(math.Float32bits(float32(exec.Fmax(float64(v1), float64(v2)))))
		case fopF32Copysign:
			v2, v1 := math.Float32frombits(uint32(frame[instr.Src2()])), math.Float32frombits(uint32(frame[instr.src1]))
			frame[instr.dest] = uint64(math.Float32bits(float32(math.Copysign(float64(v1), float64(v2)))))

		case fopF64Abs:
			frame[instr.dest] = uint64(math.Float64bits(math.Abs(math.Float64frombits(frame[instr.src1]))))
		case fopF64Neg:
			frame[instr.dest] = uint64(math.Float64bits(-math.Float64frombits(frame[instr.src1])))
		case fopF64Ceil:
			frame[instr.dest] = uint64(math.Float64bits(math.Ceil(math.Float64frombits(frame[instr.src1]))))
		case fopF64Floor:
			frame[instr.dest] = uint64(math.Float64bits(math.Floor(math.Float64frombits(frame[instr.src1]))))
		case fopF64Trunc:
			frame[instr.dest] = uint64(math.Float64bits(math.Trunc(math.Float64frombits(frame[instr.src1]))))
		case fopF64Nearest:
			frame[instr.dest] = uint64(math.Float64bits(math.RoundToEven(math.Float64frombits(frame[instr.src1]))))
		case fopF64Sqrt:
			frame[instr.dest] = uint64(math.Float64bits(math.Sqrt(math.Float64frombits(frame[instr.src1]))))
		case fopF64Add:
			v2, v1 := math.Float64frombits(frame[instr.Src2()]), math.Float64frombits(frame[instr.src1])
			frame[instr.dest] = uint64(math.Float64bits(v1 + v2))
		case fopF64Sub:
			v2, v1 := math.Float64frombits(frame[instr.Src2()]), math.Float64frombits(frame[instr.src1])
			frame[instr.dest] = uint64(math.Float64bits(v1 - v2))
		case fopF64Mul:
			v2, v1 := math.Float64frombits(frame[instr.Src2()]), math.Float64frombits(frame[instr.src1])
			frame[instr.dest] = uint64(math.Float64bits(v1 * v2))
		case fopF64Div:
			v2, v1 := math.Float64frombits(frame[instr.Src2()]), math.Float64frombits(frame[instr.src1])
			frame[instr.dest] = uint64(math.Float64bits(v1 / v2))
		case fopF64Min:
			v2, v1 := math.Float64frombits(frame[instr.Src2()]), math.Float64frombits(frame[instr.src1])
			frame[instr.dest] = uint64(math.Float64bits(exec.Fmin(v1, v2)))
		case fopF64Max:
			v2, v1 := math.Float64frombits(frame[instr.Src2()]), math.Float64frombits(frame[instr.src1])
			frame[instr.dest] = uint64(math.Float64bits(exec.Fmax(v1, v2)))
		case fopF64Copysign:
			v2, v1 := math.Float64frombits(frame[instr.Src2()]), math.Float64frombits(frame[instr.src1])
			frame[instr.dest] = uint64(math.Float64bits(math.Copysign(v1, v2)))

		case fopI32WrapI64:
			frame[instr.dest] = uint64(int32(int64(frame[instr.src1])))
		case fopI32TruncF32S:
			frame[instr.dest] = uint64(exec.I32TruncS(float64(math.Float32frombits(uint32(frame[instr.src1])))))
		case fopI32TruncF32U:
			frame[instr.dest] = uint64(exec.I32TruncU(float64(math.Float32frombits(uint32(frame[instr.src1])))))
		case fopI32TruncF64S:
			frame[instr.dest] = uint64(exec.I32TruncS(math.Float64frombits(frame[instr.src1])))
		case fopI32TruncF64U:
			frame[instr.dest] = uint64(exec.I32TruncU(math.Float64frombits(frame[instr.src1])))

		case fopI64ExtendI32S:
			frame[instr.dest] = uint64(int64(int32(frame[instr.src1])))
		case fopI64ExtendI32U:
			frame[instr.dest] = uint64(int64(uint32(frame[instr.src1])))
		case fopI64TruncF32S:
			frame[instr.dest] = uint64(exec.I64TruncS(float64(math.Float32frombits(uint32(frame[instr.src1])))))
		case fopI64TruncF32U:
			frame[instr.dest] = uint64(exec.I64TruncU(float64(math.Float32frombits(uint32(frame[instr.src1])))))
		case fopI64TruncF64S:
			frame[instr.dest] = uint64(exec.I64TruncS(math.Float64frombits(frame[instr.src1])))
		case fopI64TruncF64U:
			frame[instr.dest] = uint64(exec.I64TruncU(math.Float64frombits(frame[instr.src1])))

		case fopF32ConvertI32S:
			frame[instr.dest] = uint64(math.Float32bits(float32(int32(frame[instr.src1]))))
		case fopF32ConvertI32U:
			frame[instr.dest] = uint64(math.Float32bits(float32(uint32(frame[instr.src1]))))
		case fopF32ConvertI64S:
			frame[instr.dest] = uint64(math.Float32bits(float32(int64(frame[instr.src1]))))
		case fopF32ConvertI64U:
			frame[instr.dest] = uint64(math.Float32bits(float32(uint64(frame[instr.src1]))))
		case fopF32DemoteF64:
			frame[instr.dest] = uint64(math.Float32bits(float32(math.Float64frombits(frame[instr.src1]))))

		case fopF64ConvertI32S:
			frame[instr.dest] = uint64(math.Float64bits(float64(int32(frame[instr.src1]))))
		case fopF64ConvertI32U:
			frame[instr.dest] = uint64(math.Float64bits(float64(uint32(frame[instr.src1]))))
		case fopF64ConvertI64S:
			frame[instr.dest] = uint64(math.Float64bits(float64(int64(frame[instr.src1]))))
		case fopF64ConvertI64U:
			frame[instr.dest] = uint64(math.Float64bits(float64(uint64(frame[instr.src1]))))
		case fopF64PromoteF32:
			frame[instr.dest] = uint64(math.Float64bits(float64(math.Float32frombits(uint32(frame[instr.src1])))))

		case fopI32ReinterpretF32, fopI64ReinterpretF64, fopF32ReinterpretI32, fopF64ReinterpretI64:
			frame[instr.dest] = frame[instr.src1]

		case fopI32Extend8S:
			frame[instr.dest] = uint64(int32(int8(int32(frame[instr.src1]))))
		case fopI32Extend16S:
			frame[instr.dest] = uint64(int32(int16(int32(frame[instr.src1]))))
		case fopI64Extend8S:
			frame[instr.dest] = uint64(int64(int8(int64(frame[instr.src1]))))
		case fopI64Extend16S:
			frame[instr.dest] = uint64(int64(int16(int64(frame[instr.src1]))))
		case fopI64Extend32S:
			frame[instr.dest] = uint64(int64(int32(int64(frame[instr.src1]))))

		case fopI32TruncSatF32S:
			frame[instr.dest] = uint64(exec.I32TruncSatS(float64(math.Float32frombits(uint32(frame[instr.src1])))))
		case fopI32TruncSatF32U:
			frame[instr.dest] = uint64(exec.I32TruncSatU(float64(math.Float32frombits(uint32(frame[instr.src1])))))
		case fopI32TruncSatF64S:
			frame[instr.dest] = uint64(exec.I32TruncSatS(math.Float64frombits(frame[instr.src1])))
		case fopI32TruncSatF64U:
			frame[instr.dest] = uint64(exec.I32TruncSatU(math.Float64frombits(frame[instr.src1])))
		case fopI64TruncSatF32S:
			frame[instr.dest] = uint64(exec.I64TruncSatS(float64(math.Float32frombits(uint32(frame[instr.src1])))))
		case fopI64TruncSatF32U:
			frame[instr.dest] = uint64(exec.I64TruncSatU(float64(math.Float32frombits(uint32(frame[instr.src1])))))
		case fopI64TruncSatF64S:
			frame[instr.dest] = uint64(exec.I64TruncSatS(math.Float64frombits(frame[instr.src1])))
		case fopI64TruncSatF64U:
			frame[instr.dest] = uint64(exec.I64TruncSatU(math.Float64frombits(frame[instr.src1])))

		case fopBrIfI32Eqz:
			if int32(frame[instr.src1]) == 0 {
				ip = labels[instr.Labelidx()].Continuation()
				continue
			}
		case fopBrIfI32Eq:
			v2, v1 := int32(frame[instr.Src2()]), int32(frame[instr.src1])
			if v1 == v2 {
				ip = labels[instr.Labelidx()].Continuation()
				continue
			}
		case fopBrIfI32Ne:
			v2, v1 := int32(frame[instr.Src2()]), int32(frame[instr.src1])
			if v1 != v2 {
				ip = labels[instr.Labelidx()].Continuation()
				continue
			}
		case fopBrIfI32LtS:
			v2, v1 := int32(frame[instr.Src2()]), int32(frame[instr.src1])
			if v1 < v2 {
				ip = labels[instr.Labelidx()].Continuation()
				continue
			}
		case fopBrIfI32LtU:
			v2, v1 := uint32(frame[instr.Src2()]), uint32(frame[instr.src1])
			if v1 < v2 {
				ip = labels[instr.Labelidx()].Continuation()
				continue
			}
		case fopBrIfI32GtS:
			v2, v1 := int32(frame[instr.Src2()]), int32(frame[instr.src1])
			if v1 > v2 {
				ip = labels[instr.Labelidx()].Continuation()
				continue
			}
		case fopBrIfI32GtU:
			v2, v1 := uint32(frame[instr.Src2()]), uint32(frame[instr.src1])
			if v1 > v2 {
				ip = labels[instr.Labelidx()].Continuation()
				continue
			}
		case fopBrIfI32LeS:
			v2, v1 := int32(frame[instr.Src2()]), int32(frame[instr.src1])
			if v1 <= v2 {
				ip = labels[instr.Labelidx()].Continuation()
				continue
			}
		case fopBrIfI32LeU:
			v2, v1 := uint32(frame[instr.Src2()]), uint32(frame[instr.src1])
			if v1 <= v2 {
				ip = labels[instr.Labelidx()].Continuation()
				continue
			}
		case fopBrIfI32GeS:
			v2, v1 := int32(frame[instr.Src2()]), int32(frame[instr.src1])
			if v1 >= v2 {
				ip = labels[instr.Labelidx()].Continuation()
				continue
			}
		case fopBrIfI32GeU:
			v2, v1 := uint32(frame[instr.Src2()]), uint32(frame[instr.src1])
			if v1 >= v2 {
				ip = labels[instr.Labelidx()].Continuation()
				continue
			}

		case fopBrIfI64Eqz:
			if int64(frame[instr.src1]) == 0 {
				ip = labels[instr.Labelidx()].Continuation()
				continue
			}
		case fopBrIfI64Eq:
			v2, v1 := int64(frame[instr.Src2()]), int64(frame[instr.src1])
			if v1 == v2 {
				ip = labels[instr.Labelidx()].Continuation()
				continue
			}
		case fopBrIfI64Ne:
			v2, v1 := int64(frame[instr.Src2()]), int64(frame[instr.src1])
			if v1 != v2 {
				ip = labels[instr.Labelidx()].Continuation()
				continue
			}
		case fopBrIfI64LtS:
			v2, v1 := int64(frame[instr.Src2()]), int64(frame[instr.src1])
			if v1 < v2 {
				ip = labels[instr.Labelidx()].Continuation()
				continue
			}
		case fopBrIfI64LtU:
			v2, v1 := frame[instr.Src2()], frame[instr.src1]
			if v1 < v2 {
				ip = labels[instr.Labelidx()].Continuation()
				continue
			}
		case fopBrIfI64GtS:
			v2, v1 := int64(frame[instr.Src2()]), int64(frame[instr.src1])
			if v1 > v2 {
				ip = labels[instr.Labelidx()].Continuation()
				continue
			}
		case fopBrIfI64GtU:
			v2, v1 := frame[instr.Src2()], frame[instr.src1]
			if v1 > v2 {
				ip = labels[instr.Labelidx()].Continuation()
				continue
			}
		case fopBrIfI64LeS:
			v2, v1 := int64(frame[instr.Src2()]), int64(frame[instr.src1])
			if v1 <= v2 {
				ip = labels[instr.Labelidx()].Continuation()
				continue
			}
		case fopBrIfI64LeU:
			v2, v1 := frame[instr.Src2()], frame[instr.src1]
			if v1 <= v2 {
				ip = labels[instr.Labelidx()].Continuation()
				continue
			}
		case fopBrIfI64GeS:
			v2, v1 := int64(frame[instr.Src2()]), int64(frame[instr.src1])
			if v1 >= v2 {
				ip = labels[instr.Labelidx()].Continuation()
				continue
			}
		case fopBrIfI64GeU:
			v2, v1 := frame[instr.Src2()], frame[instr.src1]
			if v1 >= v2 {
				ip = labels[instr.Labelidx()].Continuation()
				continue
			}

		case fopBrIfF32Eq:
			v2, v1 := math.Float32frombits(uint32(frame[instr.Src2()])), math.Float32frombits(uint32(frame[instr.src1]))
			if v1 == v2 {
				ip = labels[instr.Labelidx()].Continuation()
				continue
			}
		case fopBrIfF32Ne:
			v2, v1 := math.Float32frombits(uint32(frame[instr.Src2()])), math.Float32frombits(uint32(frame[instr.src1]))
			if v1 != v2 {
				ip = labels[instr.Labelidx()].Continuation()
				continue
			}
		case fopBrIfF32Lt:
			v2, v1 := math.Float32frombits(uint32(frame[instr.Src2()])), math.Float32frombits(uint32(frame[instr.src1]))
			if v1 < v2 {
				ip = labels[instr.Labelidx()].Continuation()
				continue
			}
		case fopBrIfF32Gt:
			v2, v1 := math.Float32frombits(uint32(frame[instr.Src2()])), math.Float32frombits(uint32(frame[instr.src1]))
			if v1 > v2 {
				ip = labels[instr.Labelidx()].Continuation()
				continue
			}
		case fopBrIfF32Le:
			v2, v1 := math.Float32frombits(uint32(frame[instr.Src2()])), math.Float32frombits(uint32(frame[instr.src1]))
			if v1 <= v2 {
				ip = labels[instr.Labelidx()].Continuation()
				continue
			}
		case fopBrIfF32Ge:
			v2, v1 := math.Float32frombits(uint32(frame[instr.Src2()])), math.Float32frombits(uint32(frame[instr.src1]))
			if v1 >= v2 {
				ip = labels[instr.Labelidx()].Continuation()
				continue
			}

		case fopBrIfF64Eq:
			v2, v1 := math.Float64frombits(frame[instr.Src2()]), math.Float64frombits(frame[instr.src1])
			if v1 == v2 {
				ip = labels[instr.Labelidx()].Continuation()
				continue
			}
		case fopBrIfF64Ne:
			v2, v1 := math.Float64frombits(frame[instr.Src2()]), math.Float64frombits(frame[instr.src1])
			if v1 != v2 {
				ip = labels[instr.Labelidx()].Continuation()
				continue
			}
		case fopBrIfF64Lt:
			v2, v1 := math.Float64frombits(frame[instr.Src2()]), math.Float64frombits(frame[instr.src1])
			if v1 < v2 {
				ip = labels[instr.Labelidx()].Continuation()
				continue
			}
		case fopBrIfF64Gt:
			v2, v1 := math.Float64frombits(frame[instr.Src2()]), math.Float64frombits(frame[instr.src1])
			if v1 > v2 {
				ip = labels[instr.Labelidx()].Continuation()
				continue
			}
		case fopBrIfF64Le:
			v2, v1 := math.Float64frombits(frame[instr.Src2()]), math.Float64frombits(frame[instr.src1])
			if v1 <= v2 {
				ip = labels[instr.Labelidx()].Continuation()
				continue
			}
		case fopBrIfF64Ge:
			v2, v1 := math.Float64frombits(frame[instr.Src2()]), math.Float64frombits(frame[instr.src1])
			if v1 >= v2 {
				ip = labels[instr.Labelidx()].Continuation()
				continue
			}

		case fopLocalSetI:
			frame[instr.dest] = instr.src2
		case fopGlobalSetI:
			global, _ := f.module.getGlobal(instr.dest)
			global.Set(instr.src2)

		case fopI32LoadI, fopF32LoadI:
			frame[instr.dest] = uint64(f.module.mem0.Uint32At(uint32(frame[instr.src1])))
		case fopI64LoadI, fopF64LoadI:
			frame[instr.dest] = f.module.mem0.Uint64At(uint32(frame[instr.src1]))

		case fopI32Load8SI:
			frame[instr.dest] = uint64(int32(int8(f.module.mem0.ByteAt(uint32(frame[instr.src1])))))
		case fopI32Load8UI:
			frame[instr.dest] = uint64(int32(f.module.mem0.ByteAt(uint32(frame[instr.src1]))))
		case fopI32Load16SI:
			frame[instr.dest] = uint64(int32(int16(f.module.mem0.Uint16At(uint32(frame[instr.src1])))))
		case fopI32Load16UI:
			frame[instr.dest] = uint64(int32(f.module.mem0.Uint16At(uint32(frame[instr.src1]))))

		case fopI64Load8SI:
			frame[instr.dest] = uint64(int64(int8(f.module.mem0.ByteAt(uint32(frame[instr.src1])))))
		case fopI64Load8UI:
			frame[instr.dest] = uint64(int64(f.module.mem0.ByteAt(uint32(frame[instr.src1]))))
		case fopI64Load16SI:
			frame[instr.dest] = uint64(int64(int16(f.module.mem0.Uint16At(uint32(frame[instr.src1])))))
		case fopI64Load16UI:
			frame[instr.dest] = uint64(int64(f.module.mem0.Uint16At(uint32(frame[instr.src1]))))
		case fopI64Load32SI:
			frame[instr.dest] = uint64(int64(int32(f.module.mem0.Uint32At(uint32(frame[instr.src1])))))
		case fopI64Load32UI:
			frame[instr.dest] = uint64(int64(f.module.mem0.Uint32At(uint32(frame[instr.src1]))))

		case fopI32StoreI, fopF32StoreI:
			f.module.mem0.PutUint32At(uint32(frame[instr.src1]), uint32(frame[instr.dest]))
		case fopI64StoreI, fopF64StoreI:
			f.module.mem0.PutUint64At(uint64(frame[instr.src1]), uint32(frame[instr.dest]))

		case fopI32Store8I:
			f.module.mem0.PutByteAt(byte(frame[instr.src1]), uint32(frame[instr.dest]))
		case fopI32Store16I:
			f.module.mem0.PutUint16At(uint16(frame[instr.src1]), uint32(frame[instr.dest]))

		case fopI64Store8I:
			f.module.mem0.PutByteAt(byte(frame[instr.src1]), uint32(frame[instr.dest]))
		case fopI64Store16I:
			f.module.mem0.PutUint16At(uint16(frame[instr.src1]), uint32(frame[instr.dest]))
		case fopI64Store32I:
			f.module.mem0.PutUint32At(uint32(frame[instr.src1]), uint32(frame[instr.dest]))

		case fopI32EqI:
			v2, v1 := int32(instr.src2), int32(frame[instr.src1])
			frame.setBool(v1 == v2, instr.dest)
		case fopI32NeI:
			v2, v1 := int32(instr.src2), int32(frame[instr.src1])
			frame.setBool(v1 != v2, instr.dest)
		case fopI32LtSI:
			v2, v1 := int32(instr.src2), int32(frame[instr.src1])
			frame.setBool(v1 < v2, instr.dest)
		case fopI32LtUI:
			v2, v1 := uint32(instr.src2), uint32(frame[instr.src1])
			frame.setBool(v1 < v2, instr.dest)
		case fopI32GtSI:
			v2, v1 := int32(instr.src2), int32(frame[instr.src1])
			frame.setBool(v1 > v2, instr.dest)
		case fopI32GtUI:
			v2, v1 := uint32(instr.src2), uint32(frame[instr.src1])
			frame.setBool(v1 > v2, instr.dest)
		case fopI32LeSI:
			v2, v1 := int32(instr.src2), int32(frame[instr.src1])
			frame.setBool(v1 <= v2, instr.dest)
		case fopI32LeUI:
			v2, v1 := uint32(instr.src2), uint32(frame[instr.src1])
			frame.setBool(v1 <= v2, instr.dest)
		case fopI32GeSI:
			v2, v1 := int32(instr.src2), int32(frame[instr.src1])
			frame.setBool(v1 >= v2, instr.dest)
		case fopI32GeUI:
			v2, v1 := uint32(instr.src2), uint32(frame[instr.src1])
			frame.setBool(v1 >= v2, instr.dest)

		case fopI64EqI:
			v2, v1 := int64(instr.src2), int64(frame[instr.src1])
			frame.setBool(v1 == v2, instr.dest)
		case fopI64NeI:
			v2, v1 := int64(instr.src2), int64(frame[instr.src1])
			frame.setBool(v1 != v2, instr.dest)
		case fopI64LtSI:
			v2, v1 := int64(instr.src2), int64(frame[instr.src1])
			frame.setBool(v1 < v2, instr.dest)
		case fopI64LtUI:
			v2, v1 := instr.src2, frame[instr.src1]
			frame.setBool(v1 < v2, instr.dest)
		case fopI64GtSI:
			v2, v1 := int64(instr.src2), int64(frame[instr.src1])
			frame.setBool(v1 > v2, instr.dest)
		case fopI64GtUI:
			v2, v1 := instr.src2, frame[instr.src1]
			frame.setBool(v1 > v2, instr.dest)
		case fopI64LeSI:
			v2, v1 := int64(instr.src2), int64(frame[instr.src1])
			frame.setBool(v1 <= v2, instr.dest)
		case fopI64LeUI:
			v2, v1 := instr.src2, frame[instr.src1]
			frame.setBool(v1 <= v2, instr.dest)
		case fopI64GeSI:
			v2, v1 := int64(instr.src2), int64(frame[instr.src1])
			frame.setBool(v1 >= v2, instr.dest)
		case fopI64GeUI:
			v2, v1 := instr.src2, frame[instr.src1]
			frame.setBool(v1 >= v2, instr.dest)

		case fopF32EqI:
			v2, v1 := math.Float32frombits(uint32(instr.src2)), math.Float32frombits(uint32(frame[instr.src1]))
			frame.setBool(v1 == v2, instr.dest)
		case fopF32NeI:
			v2, v1 := math.Float32frombits(uint32(instr.src2)), math.Float32frombits(uint32(frame[instr.src1]))
			frame.setBool(v1 != v2, instr.dest)
		case fopF32LtI:
			v2, v1 := math.Float32frombits(uint32(instr.src2)), math.Float32frombits(uint32(frame[instr.src1]))
			frame.setBool(v1 < v2, instr.dest)
		case fopF32GtI:
			v2, v1 := math.Float32frombits(uint32(instr.src2)), math.Float32frombits(uint32(frame[instr.src1]))
			frame.setBool(v1 > v2, instr.dest)
		case fopF32LeI:
			v2, v1 := math.Float32frombits(uint32(instr.src2)), math.Float32frombits(uint32(frame[instr.src1]))
			frame.setBool(v1 <= v2, instr.dest)
		case fopF32GeI:
			v2, v1 := math.Float32frombits(uint32(instr.src2)), math.Float32frombits(uint32(frame[instr.src1]))
			frame.setBool(v1 >= v2, instr.dest)

		case fopF64EqI:
			v2, v1 := math.Float64frombits(instr.src2), math.Float64frombits(frame[instr.src1])
			frame.setBool(v1 == v2, instr.dest)
		case fopF64NeI:
			v2, v1 := math.Float64frombits(instr.src2), math.Float64frombits(frame[instr.src1])
			frame.setBool(v1 != v2, instr.dest)
		case fopF64LtI:
			v2, v1 := math.Float64frombits(instr.src2), math.Float64frombits(frame[instr.src1])
			frame.setBool(v1 < v2, instr.dest)
		case fopF64GtI:
			v2, v1 := math.Float64frombits(instr.src2), math.Float64frombits(frame[instr.src1])
			frame.setBool(v1 > v2, instr.dest)
		case fopF64LeI:
			v2, v1 := math.Float64frombits(instr.src2), math.Float64frombits(frame[instr.src1])
			frame.setBool(v1 <= v2, instr.dest)
		case fopF64GeI:
			v2, v1 := math.Float64frombits(instr.src2), math.Float64frombits(frame[instr.src1])
			frame.setBool(v1 >= v2, instr.dest)

		case fopI32AddI:
			v2, v1 := int32(instr.src2), int32(frame[instr.src1])
			frame[instr.dest] = uint64(v1 + v2)
		case fopI32SubI:
			v2, v1 := int32(instr.src2), int32(frame[instr.src1])
			frame[instr.dest] = uint64(v1 - v2)
		case fopI32MulI:
			v2, v1 := int32(instr.src2), int32(frame[instr.src1])
			frame[instr.dest] = uint64(v1 * v2)
		case fopI32DivSI:
			v2, v1 := int32(instr.src2), int32(frame[instr.src1])
			frame[instr.dest] = uint64(exec.I32DivS(v1, v2))
		case fopI32DivUI:
			v2, v1 := uint32(instr.src2), uint32(frame[instr.src1])
			frame[instr.dest] = uint64(v1 / v2)
		case fopI32RemSI:
			v2, v1 := int32(instr.src2), int32(frame[instr.src1])
			frame[instr.dest] = uint64(v1 % v2)
		case fopI32RemUI:
			v2, v1 := uint32(instr.src2), uint32(frame[instr.src1])
			frame[instr.dest] = uint64(v1 % v2)
		case fopI32AndI:
			v2, v1 := int32(instr.src2), int32(frame[instr.src1])
			frame[instr.dest] = uint64(v1 & v2)
		case fopI32OrI:
			v2, v1 := int32(instr.src2), int32(frame[instr.src1])
			frame[instr.dest] = uint64(v1 | v2)
		case fopI32XorI:
			v2, v1 := int32(instr.src2), int32(frame[instr.src1])
			frame[instr.dest] = uint64(v1 ^ v2)
		case fopI32ShlI:
			v2, v1 := int32(instr.src2), int32(frame[instr.src1])
			frame[instr.dest] = uint64(v1 << (v2 & 31))
		case fopI32ShrSI:
			v2, v1 := int32(instr.src2), int32(frame[instr.src1])
			frame[instr.dest] = uint64(v1 >> (v2 & 31))
		case fopI32ShrUI:
			v2, v1 := uint32(instr.src2), uint32(frame[instr.src1])
			frame[instr.dest] = uint64(v1 >> (v2 & 31))
		case fopI32RotlI:
			v2, v1 := int(instr.src2), uint32(frame[instr.src1])
			frame[instr.dest] = uint64(bits.RotateLeft32(v1, v2))
		case fopI32RotrI:
			v2, v1 := int(instr.src2), uint32(frame[instr.src1])
			frame[instr.dest] = uint64(bits.RotateLeft32(v1, -v2))

		case fopI64AddI:
			v2, v1 := int64(instr.src2), int64(frame[instr.src1])
			frame[instr.dest] = uint64(v1 + v2)
		case fopI64SubI:
			v2, v1 := int64(instr.src2), int64(frame[instr.src1])
			frame[instr.dest] = uint64(v1 - v2)
		case fopI64MulI:
			v2, v1 := int64(instr.src2), int64(frame[instr.src1])
			frame[instr.dest] = uint64(v1 * v2)
		case fopI64DivSI:
			v2, v1 := int64(instr.src2), int64(frame[instr.src1])
			frame[instr.dest] = uint64(exec.I64DivS(v1, v2))
		case fopI64DivUI:
			v2, v1 := instr.src2, frame[instr.src1]
			frame[instr.dest] = uint64(v1 / v2)
		case fopI64RemSI:
			v2, v1 := int64(instr.src2), int64(frame[instr.src1])
			frame[instr.dest] = uint64(v1 % v2)
		case fopI64RemUI:
			v2, v1 := instr.src2, frame[instr.src1]
			frame[instr.dest] = uint64(v1 % v2)
		case fopI64AndI:
			v2, v1 := int64(instr.src2), int64(frame[instr.src1])
			frame[instr.dest] = uint64(v1 & v2)
		case fopI64OrI:
			v2, v1 := int64(instr.src2), int64(frame[instr.src1])
			frame[instr.dest] = uint64(v1 | v2)
		case fopI64XorI:
			v2, v1 := int64(instr.src2), int64(frame[instr.src1])
			frame[instr.dest] = uint64(v1 ^ v2)
		case fopI64ShlI:
			v2, v1 := int64(instr.src2), int64(frame[instr.src1])
			frame[instr.dest] = uint64(v1 << (v2 & 63))
		case fopI64ShrSI:
			v2, v1 := int64(instr.src2), int64(frame[instr.src1])
			frame[instr.dest] = uint64(v1 >> (v2 & 63))
		case fopI64ShrUI:
			v2, v1 := instr.src2, frame[instr.src1]
			frame[instr.dest] = uint64(v1 >> (v2 & 63))
		case fopI64RotlI:
			v2, v1 := int(instr.src2), uint64(frame[instr.src1])
			frame[instr.dest] = uint64(bits.RotateLeft64(v1, v2))
		case fopI64RotrI:
			v2, v1 := int(instr.src2), uint64(frame[instr.src1])
			frame[instr.dest] = uint64(bits.RotateLeft64(v1, -v2))

		case fopF32AddI:
			v2, v1 := math.Float32frombits(uint32(instr.src2)), math.Float32frombits(uint32(frame[instr.src1]))
			frame[instr.dest] = uint64(math.Float32bits(v1 + v2))
		case fopF32SubI:
			v2, v1 := math.Float32frombits(uint32(instr.src2)), math.Float32frombits(uint32(frame[instr.src1]))
			frame[instr.dest] = uint64(math.Float32bits(v1 - v2))
		case fopF32MulI:
			v2, v1 := math.Float32frombits(uint32(instr.src2)), math.Float32frombits(uint32(frame[instr.src1]))
			frame[instr.dest] = uint64(math.Float32bits(v1 * v2))
		case fopF32DivI:
			v2, v1 := math.Float32frombits(uint32(instr.src2)), math.Float32frombits(uint32(frame[instr.src1]))
			frame[instr.dest] = uint64(math.Float32bits(v1 / v2))
		case fopF32MinI:
			v2, v1 := math.Float32frombits(uint32(instr.src2)), math.Float32frombits(uint32(frame[instr.src1]))
			frame[instr.dest] = uint64(math.Float32bits(float32(exec.Fmin(float64(v1), float64(v2)))))
		case fopF32MaxI:
			v2, v1 := math.Float32frombits(uint32(instr.src2)), math.Float32frombits(uint32(frame[instr.src1]))
			frame[instr.dest] = uint64(math.Float32bits(float32(exec.Fmax(float64(v1), float64(v2)))))
		case fopF32CopysignI:
			v2, v1 := math.Float32frombits(uint32(instr.src2)), math.Float32frombits(uint32(frame[instr.src1]))
			frame[instr.dest] = uint64(math.Float32bits(float32(math.Copysign(float64(v1), float64(v2)))))

		case fopF64AddI:
			v2, v1 := math.Float64frombits(instr.src2), math.Float64frombits(frame[instr.src1])
			frame[instr.dest] = uint64(math.Float64bits(v1 + v2))
		case fopF64SubI:
			v2, v1 := math.Float64frombits(instr.src2), math.Float64frombits(frame[instr.src1])
			frame[instr.dest] = uint64(math.Float64bits(v1 - v2))
		case fopF64MulI:
			v2, v1 := math.Float64frombits(instr.src2), math.Float64frombits(frame[instr.src1])
			frame[instr.dest] = uint64(math.Float64bits(v1 * v2))
		case fopF64DivI:
			v2, v1 := math.Float64frombits(instr.src2), math.Float64frombits(frame[instr.src1])
			frame[instr.dest] = uint64(math.Float64bits(v1 / v2))
		case fopF64MinI:
			v2, v1 := math.Float64frombits(instr.src2), math.Float64frombits(frame[instr.src1])
			frame[instr.dest] = uint64(math.Float64bits(exec.Fmin(v1, v2)))
		case fopF64MaxI:
			v2, v1 := math.Float64frombits(instr.src2), math.Float64frombits(frame[instr.src1])
			frame[instr.dest] = uint64(math.Float64bits(exec.Fmax(v1, v2)))
		case fopF64CopysignI:
			v2, v1 := math.Float64frombits(instr.src2), math.Float64frombits(frame[instr.src1])
			frame[instr.dest] = uint64(math.Float64bits(math.Copysign(v1, v2)))

		case fopBrIfI32EqzI:
			if int32(frame[instr.src1]) == 0 {
				ip = labels[instr.Labelidx()].Continuation()
				continue
			}
		case fopBrIfI32EqI:
			v2, v1 := int32(instr.src2), int32(frame[instr.src1])
			if v1 == v2 {
				ip = labels[instr.Labelidx()].Continuation()
				continue
			}
		case fopBrIfI32NeI:
			v2, v1 := int32(instr.src2), int32(frame[instr.src1])
			if v1 != v2 {
				ip = labels[instr.Labelidx()].Continuation()
				continue
			}
		case fopBrIfI32LtSI:
			v2, v1 := int32(instr.src2), int32(frame[instr.src1])
			if v1 < v2 {
				ip = labels[instr.Labelidx()].Continuation()
				continue
			}
		case fopBrIfI32LtUI:
			v2, v1 := uint32(instr.src2), uint32(frame[instr.src1])
			if v1 < v2 {
				ip = labels[instr.Labelidx()].Continuation()
				continue
			}
		case fopBrIfI32GtSI:
			v2, v1 := int32(instr.src2), int32(frame[instr.src1])
			if v1 > v2 {
				ip = labels[instr.Labelidx()].Continuation()
				continue
			}
		case fopBrIfI32GtUI:
			v2, v1 := uint32(instr.src2), uint32(frame[instr.src1])
			if v1 > v2 {
				ip = labels[instr.Labelidx()].Continuation()
				continue
			}
		case fopBrIfI32LeSI:
			v2, v1 := int32(instr.src2), int32(frame[instr.src1])
			if v1 <= v2 {
				ip = labels[instr.Labelidx()].Continuation()
				continue
			}
		case fopBrIfI32LeUI:
			v2, v1 := uint32(instr.src2), uint32(frame[instr.src1])
			if v1 <= v2 {
				ip = labels[instr.Labelidx()].Continuation()
				continue
			}
		case fopBrIfI32GeSI:
			v2, v1 := int32(instr.src2), int32(frame[instr.src1])
			if v1 >= v2 {
				ip = labels[instr.Labelidx()].Continuation()
				continue
			}
		case fopBrIfI32GeUI:
			v2, v1 := uint32(instr.src2), uint32(frame[instr.src1])
			if v1 >= v2 {
				ip = labels[instr.Labelidx()].Continuation()
				continue
			}

		case fopBrIfI64EqzI:
			if int64(frame[instr.src1]) == 0 {
				ip = labels[instr.Labelidx()].Continuation()
				continue
			}
		case fopBrIfI64EqI:
			v2, v1 := int64(instr.src2), int64(frame[instr.src1])
			if v1 == v2 {
				ip = labels[instr.Labelidx()].Continuation()
				continue
			}
		case fopBrIfI64NeI:
			v2, v1 := int64(instr.src2), int64(frame[instr.src1])
			if v1 != v2 {
				ip = labels[instr.Labelidx()].Continuation()
				continue
			}
		case fopBrIfI64LtSI:
			v2, v1 := int64(instr.src2), int64(frame[instr.src1])
			if v1 < v2 {
				ip = labels[instr.Labelidx()].Continuation()
				continue
			}
		case fopBrIfI64LtUI:
			v2, v1 := instr.src2, frame[instr.src1]
			if v1 < v2 {
				ip = labels[instr.Labelidx()].Continuation()
				continue
			}
		case fopBrIfI64GtSI:
			v2, v1 := int64(instr.src2), int64(frame[instr.src1])
			if v1 > v2 {
				ip = labels[instr.Labelidx()].Continuation()
				continue
			}
		case fopBrIfI64GtUI:
			v2, v1 := instr.src2, frame[instr.src1]
			if v1 > v2 {
				ip = labels[instr.Labelidx()].Continuation()
				continue
			}
		case fopBrIfI64LeSI:
			v2, v1 := int64(instr.src2), int64(frame[instr.src1])
			if v1 <= v2 {
				ip = labels[instr.Labelidx()].Continuation()
				continue
			}
		case fopBrIfI64LeUI:
			v2, v1 := instr.src2, frame[instr.src1]
			if v1 <= v2 {
				ip = labels[instr.Labelidx()].Continuation()
				continue
			}
		case fopBrIfI64GeSI:
			v2, v1 := int64(instr.src2), int64(frame[instr.src1])
			if v1 >= v2 {
				ip = labels[instr.Labelidx()].Continuation()
				continue
			}
		case fopBrIfI64GeUI:
			v2, v1 := instr.src2, frame[instr.src1]
			if v1 >= v2 {
				ip = labels[instr.Labelidx()].Continuation()
				continue
			}

		case fopBrIfF32EqI:
			v2, v1 := math.Float32frombits(uint32(instr.src2)), math.Float32frombits(uint32(frame[instr.src1]))
			if v1 == v2 {
				ip = labels[instr.Labelidx()].Continuation()
				continue
			}
		case fopBrIfF32NeI:
			v2, v1 := math.Float32frombits(uint32(instr.src2)), math.Float32frombits(uint32(frame[instr.src1]))
			if v1 != v2 {
				ip = labels[instr.Labelidx()].Continuation()
				continue
			}
		case fopBrIfF32LtI:
			v2, v1 := math.Float32frombits(uint32(instr.src2)), math.Float32frombits(uint32(frame[instr.src1]))
			if v1 < v2 {
				ip = labels[instr.Labelidx()].Continuation()
				continue
			}
		case fopBrIfF32GtI:
			v2, v1 := math.Float32frombits(uint32(instr.src2)), math.Float32frombits(uint32(frame[instr.src1]))
			if v1 > v2 {
				ip = labels[instr.Labelidx()].Continuation()
				continue
			}
		case fopBrIfF32LeI:
			v2, v1 := math.Float32frombits(uint32(instr.src2)), math.Float32frombits(uint32(frame[instr.src1]))
			if v1 <= v2 {
				ip = labels[instr.Labelidx()].Continuation()
				continue
			}
		case fopBrIfF32GeI:
			v2, v1 := math.Float32frombits(uint32(instr.src2)), math.Float32frombits(uint32(frame[instr.src1]))
			if v1 >= v2 {
				ip = labels[instr.Labelidx()].Continuation()
				continue
			}

		case fopBrIfF64EqI:
			v2, v1 := math.Float64frombits(instr.src2), math.Float64frombits(frame[instr.src1])
			if v1 == v2 {
				ip = labels[instr.Labelidx()].Continuation()
				continue
			}
		case fopBrIfF64NeI:
			v2, v1 := math.Float64frombits(instr.src2), math.Float64frombits(frame[instr.src1])
			if v1 != v2 {
				ip = labels[instr.Labelidx()].Continuation()
				continue
			}
		case fopBrIfF64LtI:
			v2, v1 := math.Float64frombits(instr.src2), math.Float64frombits(frame[instr.src1])
			if v1 < v2 {
				ip = labels[instr.Labelidx()].Continuation()
				continue
			}
		case fopBrIfF64GtI:
			v2, v1 := math.Float64frombits(instr.src2), math.Float64frombits(frame[instr.src1])
			if v1 > v2 {
				ip = labels[instr.Labelidx()].Continuation()
				continue
			}
		case fopBrIfF64LeI:
			v2, v1 := math.Float64frombits(instr.src2), math.Float64frombits(frame[instr.src1])
			if v1 <= v2 {
				ip = labels[instr.Labelidx()].Continuation()
				continue
			}
		case fopBrIfF64GeI:
			v2, v1 := math.Float64frombits(instr.src2), math.Float64frombits(frame[instr.src1])
			if v1 >= v2 {
				ip = labels[instr.Labelidx()].Continuation()
				continue
			}
		}

		ip++
	}
}
