package code

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/pgavlin/warp/wasm"
	"github.com/pgavlin/warp/wasm/leb128"
)

var ErrInvalidInstruction = errors.New("wasm: invalid instruction")

func decodeBlockType(body []byte) (uint64, []byte, error) {
	// Block encoding
	if len(body) == 0 {
		return 0, nil, io.ErrUnexpectedEOF
	}

	switch body[0] {
	case 0x40, 0x7f, 0x7e, 0x7d, 0x7c:
		return uint64(body[0]) | 0x8000000000000000, body[1:], nil
	default:
		index, read, err := leb128.GetVarint64(body)
		if err != nil {
			return 0, nil, err
		}
		return uint64(index) & 0x7fffffffffffffff, body[read:], nil
	}
}

type Metrics struct {
	MaxNesting    int  // The maximum block nesting for the function.
	MaxStackDepth int  // The maximum stack depth for the function.
	LabelCount    int  // The number of labels in the function.
	HasLoops      bool // True if this function has loops
}

type block struct {
	*Instruction

	in, out     []wasm.ValueType
	stackHeight int
	unreachable bool
}

type decoder struct {
	Scope

	ibuf    []Instruction
	metrics Metrics

	blocks []block
	stack  []wasm.ValueType
}

type Body struct {
	Instructions []Instruction
	Metrics      Metrics
}

func Decode(body []byte, scope Scope, out []wasm.ValueType) (Body, error) {
	decoder := decoder{Scope: scope}
	return decoder.decode(body, out)
}

func (d *decoder) GetStackType(num int) wasm.ValueType {
	if num > len(d.stack) {
		return wasm.ValueTypeT
	}
	return d.stack[len(d.stack)-1-num]
}

func (d *decoder) popOpd() (wasm.ValueType, error) {
	b := &d.blocks[len(d.blocks)-1]
	if b.unreachable && len(d.stack) == b.stackHeight {
		return wasm.ValueTypeT, nil
	}
	if len(d.stack) == b.stackHeight {
		return 0, wasm.ValidationError("stack underflow")
	}
	t := d.stack[len(d.stack)-1]
	d.stack = d.stack[:len(d.stack)-1]
	return t, nil
}

func (d *decoder) popOpds(types ...wasm.ValueType) error {
	for i := len(types) - 1; i >= 0; i-- {
		expected := types[i]
		actual, err := d.popOpd()
		if err != nil {
			return err
		}
		if actual != wasm.ValueTypeT && expected != wasm.ValueTypeT && actual != expected {
			return wasm.ValidationError("stack type mismatch")
		}
	}
	return nil
}

func (d *decoder) pushOpds(types ...wasm.ValueType) {
	d.stack = append(d.stack, types...)

	if len(d.stack) > d.metrics.MaxStackDepth {
		d.metrics.MaxStackDepth = len(d.stack)
	}
}

func (d *decoder) pushBlock(instr *Instruction, in, out []wasm.ValueType) {
	d.blocks = append(d.blocks, block{
		Instruction: instr,
		in:          in,
		out:         out,
		stackHeight: len(d.stack),
	})
	d.pushOpds(in...)

	if len(d.blocks) > d.metrics.MaxNesting {
		d.metrics.MaxNesting = len(d.blocks)
	}
	d.metrics.LabelCount++
}

func (d *decoder) popBlock() (*block, error) {
	if len(d.blocks) == 0 {
		return nil, wasm.ValidationError("label stack underflow")
	}
	b := &d.blocks[len(d.blocks)-1]
	if err := d.popOpds(b.out...); err != nil {
		return nil, err
	}
	if b.Instruction != nil && len(d.stack) != b.stackHeight {
		return nil, wasm.ValidationError("unbalanced stack")
	}
	d.blocks = d.blocks[:len(d.blocks)-1]
	return b, nil
}

func (d *decoder) labelTypes(n int) ([]wasm.ValueType, error) {
	if len(d.blocks)-1 < n {
		return nil, wasm.ValidationError("invalid label")
	}

	b := &d.blocks[len(d.blocks)-1-n]
	if b.Instruction != nil && b.Opcode == OpLoop {
		return b.in, nil
	}
	return b.out, nil
}

func (d *decoder) unreachable() {
	b := &d.blocks[len(d.blocks)-1]
	d.stack = d.stack[:b.stackHeight]
	b.unreachable = true
}

func (d *decoder) doStack(i *Instruction) error {
	const (
		I32 = wasm.ValueTypeI32
		I64 = wasm.ValueTypeI64
		F32 = wasm.ValueTypeF32
		F64 = wasm.ValueTypeF64
	)

	switch i.Opcode {
	case OpLocalSet:
		t, ok := d.GetLocalType(i.Localidx())
		if !ok {
			return wasm.ValidationError("unknown local")
		}
		return d.popOpds(t)

	case OpGlobalSet:
		t, ok := d.GetGlobalType(i.Globalidx())
		if !ok {
			return wasm.ValidationError("unknown global")
		}
		if !t.Mutable {
			return wasm.ValidationError("global is immutable")
		}
		return d.popOpds(t.Type)

	case OpI32Store, OpI32Store8, OpI32Store16:
		if !d.HasMemory(0) {
			return wasm.ValidationError("unknown memory")
		}
		return d.popOpds(I32, I32)

	case OpI64Store, OpI64Store8, OpI64Store16, OpI64Store32:
		if !d.HasMemory(0) {
			return wasm.ValidationError("unknown memory")
		}
		return d.popOpds(I32, I64)

	case OpF32Store:
		if !d.HasMemory(0) {
			return wasm.ValidationError("unknown memory")
		}
		return d.popOpds(I32, F32)

	case OpF64Store:
		if !d.HasMemory(0) {
			return wasm.ValidationError("unknown memory")
		}
		return d.popOpds(I32, F64)

	case OpLocalGet:
		t, ok := d.GetLocalType(i.Localidx())
		if !ok {
			return wasm.ValidationError("unknown local")
		}
		d.pushOpds(t)

	case OpGlobalGet:
		t, ok := d.GetGlobalType(i.Globalidx())
		if !ok {
			return wasm.ValidationError("unknown global")
		}
		d.pushOpds(t.Type)

	case OpI32Load, OpI32Load8S, OpI32Load8U, OpI32Load16S, OpI32Load16U:
		if !d.HasMemory(0) {
			return wasm.ValidationError("unknown memory")
		}
		if err := d.popOpds(I32); err != nil {
			return err
		}
		d.pushOpds(I32)

	case OpI64Load, OpI64Load8S, OpI64Load8U, OpI64Load16S, OpI64Load16U, OpI64Load32S, OpI64Load32U:
		if !d.HasMemory(0) {
			return wasm.ValidationError("unknown memory")
		}
		if err := d.popOpds(I32); err != nil {
			return err
		}
		d.pushOpds(I64)

	case OpF32Load:
		if !d.HasMemory(0) {
			return wasm.ValidationError("unknown memory")
		}
		if err := d.popOpds(I32); err != nil {
			return err
		}
		d.pushOpds(F32)

	case OpF64Load:
		if !d.HasMemory(0) {
			return wasm.ValidationError("unknown memory")
		}
		if err := d.popOpds(I32); err != nil {
			return err
		}
		d.pushOpds(F64)

	case OpMemorySize:
		if !d.HasMemory(0) {
			return wasm.ValidationError("unknown memory")
		}
		d.pushOpds(I32)

	case OpI32Const:
		d.pushOpds(I32)

	case OpI64Const:
		d.pushOpds(I64)

	case OpF32Const:
		d.pushOpds(F32)

	case OpF64Const:
		d.pushOpds(F64)

	case OpLocalTee:
		t, ok := d.GetLocalType(i.Localidx())
		if !ok {
			return wasm.ValidationError("unknown local")
		}
		if err := d.popOpds(t); err != nil {
			return err
		}
		d.pushOpds(t)

	case OpMemoryGrow:
		if !d.HasMemory(0) {
			return wasm.ValidationError("unknown memory")
		}
		if err := d.popOpds(I32); err != nil {
			return err
		}
		d.pushOpds(I32)

	case OpI32Eqz, OpI32Clz, OpI32Ctz, OpI32Popcnt:
		if err := d.popOpds(I32); err != nil {
			return err
		}
		d.pushOpds(I32)

	case OpI64Eqz:
		if err := d.popOpds(I64); err != nil {
			return err
		}
		d.pushOpds(I32)

	case OpI64Clz, OpI64Ctz, OpI64Popcnt:
		if err := d.popOpds(I64); err != nil {
			return err
		}
		d.pushOpds(I64)

	case OpF32Abs, OpF32Neg, OpF32Ceil, OpF32Floor, OpF32Trunc, OpF32Nearest, OpF32Sqrt:
		if err := d.popOpds(F32); err != nil {
			return err
		}
		d.pushOpds(F32)

	case OpF64Abs, OpF64Neg, OpF64Ceil, OpF64Floor, OpF64Trunc, OpF64Nearest, OpF64Sqrt:
		if err := d.popOpds(F64); err != nil {
			return err
		}
		d.pushOpds(F64)

	case OpI32WrapI64:
		if err := d.popOpds(I64); err != nil {
			return err
		}
		d.pushOpds(I32)

	case OpI32TruncF32S, OpI32TruncF32U:
		if err := d.popOpds(F32); err != nil {
			return err
		}
		d.pushOpds(I32)

	case OpI32TruncF64S, OpI32TruncF64U:
		if err := d.popOpds(F64); err != nil {
			return err
		}
		d.pushOpds(I32)

	case OpI64ExtendI32S, OpI64ExtendI32U:
		if err := d.popOpds(I32); err != nil {
			return err
		}
		d.pushOpds(I64)

	case OpI64TruncF32S, OpI64TruncF32U:
		if err := d.popOpds(F32); err != nil {
			return err
		}
		d.pushOpds(I64)

	case OpI64TruncF64S, OpI64TruncF64U:
		if err := d.popOpds(F64); err != nil {
			return err
		}
		d.pushOpds(I64)

	case OpF32ConvertI32S, OpF32ConvertI32U:
		if err := d.popOpds(I32); err != nil {
			return err
		}
		d.pushOpds(F32)

	case OpF32ConvertI64S, OpF32ConvertI64U:
		if err := d.popOpds(I64); err != nil {
			return err
		}
		d.pushOpds(F32)

	case OpF32DemoteF64:
		if err := d.popOpds(F64); err != nil {
			return err
		}
		d.pushOpds(F32)

	case OpF64ConvertI32S, OpF64ConvertI32U:
		if err := d.popOpds(I32); err != nil {
			return err
		}
		d.pushOpds(F64)

	case OpF64ConvertI64S, OpF64ConvertI64U:
		if err := d.popOpds(I64); err != nil {
			return err
		}
		d.pushOpds(F64)

	case OpF64PromoteF32:
		if err := d.popOpds(F32); err != nil {
			return err
		}
		d.pushOpds(F64)

	case OpI32ReinterpretF32:
		if err := d.popOpds(F32); err != nil {
			return err
		}
		d.pushOpds(I32)

	case OpI64ReinterpretF64:
		if err := d.popOpds(F64); err != nil {
			return err
		}
		d.pushOpds(I64)

	case OpF32ReinterpretI32:
		if err := d.popOpds(I32); err != nil {
			return err
		}
		d.pushOpds(F32)

	case OpF64ReinterpretI64:
		if err := d.popOpds(I64); err != nil {
			return err
		}
		d.pushOpds(F64)

	case OpI32Extend8S, OpI32Extend16S:
		if err := d.popOpds(I32); err != nil {
			return err
		}
		d.pushOpds(I32)

	case OpI64Extend8S, OpI64Extend16S, OpI64Extend32S:
		if err := d.popOpds(I64); err != nil {
			return err
		}
		d.pushOpds(I64)

	case OpI32Eq, OpI32Ne, OpI32LtS, OpI32LtU, OpI32GtS, OpI32GtU, OpI32LeS, OpI32LeU, OpI32GeS, OpI32GeU:
		if err := d.popOpds(I32, I32); err != nil {
			return err
		}
		d.pushOpds(I32)

	case OpI64Eq, OpI64Ne, OpI64LtS, OpI64LtU, OpI64GtS, OpI64GtU, OpI64LeS, OpI64LeU, OpI64GeS, OpI64GeU:
		if err := d.popOpds(I64, I64); err != nil {
			return err
		}
		d.pushOpds(I32)

	case OpF32Eq, OpF32Ne, OpF32Lt, OpF32Gt, OpF32Le, OpF32Ge:
		if err := d.popOpds(F32, F32); err != nil {
			return err
		}
		d.pushOpds(I32)

	case OpF64Eq, OpF64Ne, OpF64Lt, OpF64Gt, OpF64Le, OpF64Ge:
		if err := d.popOpds(F64, F64); err != nil {
			return err
		}
		d.pushOpds(I32)

	case OpI32Add, OpI32Sub, OpI32Mul, OpI32DivS, OpI32DivU, OpI32RemS, OpI32RemU, OpI32And, OpI32Or, OpI32Xor, OpI32Shl, OpI32ShrS, OpI32ShrU, OpI32Rotl, OpI32Rotr:
		if err := d.popOpds(I32, I32); err != nil {
			return err
		}
		d.pushOpds(I32)

	case OpI64Add, OpI64Sub, OpI64Mul, OpI64DivS, OpI64DivU, OpI64RemS, OpI64RemU, OpI64And, OpI64Or, OpI64Xor, OpI64Shl, OpI64ShrS, OpI64ShrU, OpI64Rotl, OpI64Rotr:
		if err := d.popOpds(I64, I64); err != nil {
			return err
		}
		d.pushOpds(I64)

	case OpF32Add, OpF32Sub, OpF32Mul, OpF32Div, OpF32Min, OpF32Max, OpF32Copysign:
		if err := d.popOpds(F32, F32); err != nil {
			return err
		}
		d.pushOpds(F32)

	case OpF64Add, OpF64Sub, OpF64Mul, OpF64Div, OpF64Min, OpF64Max, OpF64Copysign:
		if err := d.popOpds(F64, F64); err != nil {
			return err
		}
		d.pushOpds(F64)

	case OpCall:
		sig, ok := d.GetFunctionSignature(i.Funcidx())
		if !ok {
			return wasm.ValidationError("unknown function")
		}
		if err := d.popOpds(sig.ParamTypes...); err != nil {
			return err
		}
		d.pushOpds(sig.ReturnTypes...)

	case OpCallIndirect:
		if !d.HasTable(0) {
			return wasm.ValidationError("unknown table")
		}
		sig, ok := d.GetType(i.Typeidx())
		if !ok {
			return wasm.ValidationError("unknown type")
		}
		if err := d.popOpds(I32); err != nil {
			return err
		}
		if err := d.popOpds(sig.ParamTypes...); err != nil {
			return err
		}
		d.pushOpds(sig.ReturnTypes...)

	case OpPrefix:
		switch i.Immediate {
		case OpI32TruncSatF32S, OpI32TruncSatF32U:
			if err := d.popOpds(F32); err != nil {
				return err
			}
			d.pushOpds(I32)
		case OpI32TruncSatF64S, OpI32TruncSatF64U:
			if err := d.popOpds(F64); err != nil {
				return err
			}
			d.pushOpds(I32)
		case OpI64TruncSatF32S, OpI64TruncSatF32U:
			if err := d.popOpds(F32); err != nil {
				return err
			}
			d.pushOpds(I64)
		case OpI64TruncSatF64S, OpI64TruncSatF64U:
			if err := d.popOpds(F64); err != nil {
				return err
			}
			d.pushOpds(I64)
		}
	}

	return nil
}

func (d *decoder) decodeInstruction(body []byte) (*Instruction, []byte, error) {
	if len(body) == 0 {
		return nil, nil, io.ErrUnexpectedEOF
	}

	ip := len(d.ibuf)
	opcode := body[0]
	body = body[1:]

	var immediate uint64
	var labels []int
	var err error
	switch opcode {
	case OpBlock:
		immediate, body, err = decodeBlockType(body)
		if err != nil {
			return nil, nil, err
		}
		labels = []int{0}
	case OpLoop:
		d.metrics.HasLoops = true

		immediate, body, err = decodeBlockType(body)
		if err != nil {
			return nil, nil, err
		}
		labels = []int{ip}
	case OpIf:
		immediate, body, err = decodeBlockType(body)
		if err != nil {
			return nil, nil, err
		}
		labels = []int{0, 0}
	case OpElse:
		labels = []int{0}
	case OpBr, OpBrIf, OpCall, OpLocalGet, OpLocalSet, OpLocalTee, OpGlobalGet, OpGlobalSet:
		// Index encoding
		index, read, err := leb128.GetVarUint32(body)
		if err != nil {
			return nil, nil, err
		}
		immediate, body = uint64(index), body[read:]
	case OpBrTable:
		numLabels, read, err := leb128.GetVarUint32(body)
		if err != nil {
			return nil, nil, err
		}
		body = body[read:]

		labels = make([]int, int(numLabels))
		for i := 0; i < len(labels); i++ {
			label, read, err := leb128.GetVarUint32(body)
			if err != nil {
				return nil, nil, err
			}
			labels[i], body = int(label), body[read:]
		}

		defaultLabel, read, err := leb128.GetVarUint32(body)
		if err != nil {
			return nil, nil, err
		}
		immediate, body = uint64(defaultLabel), body[read:]
	case OpCallIndirect:
		index, read, err := leb128.GetVarUint32(body)
		if err != nil {
			return nil, nil, err
		}
		immediate, body = uint64(index), body[read:]

		if len(body) == 0 {
			return nil, nil, io.ErrUnexpectedEOF
		}
		if body[0] != 0x00 {
			return nil, nil, ErrInvalidInstruction
		}
		body = body[1:]
	case OpI32Load, OpI64Load, OpF32Load, OpF64Load, OpI32Load8S, OpI32Load8U, OpI32Load16S, OpI32Load16U, OpI64Load8S, OpI64Load8U, OpI64Load16S, OpI64Load16U, OpI64Load32S, OpI64Load32U, OpI32Store, OpI64Store, OpF32Store, OpF64Store, OpI32Store8, OpI32Store16, OpI64Store8, OpI64Store16, OpI64Store32:
		// Memory encoding
		align, read, err := leb128.GetVarUint32(body)
		if err != nil {
			return nil, nil, err
		}
		body = body[read:]

		offset, read, err := leb128.GetVarUint32(body)
		if err != nil {
			return nil, nil, err
		}
		body = body[read:]

		immediate = memarg(offset, align)
	case OpMemorySize, OpMemoryGrow:
		if len(body) == 0 {
			return nil, nil, io.ErrUnexpectedEOF
		}
		if body[0] != 0x00 {
			return nil, nil, ErrInvalidInstruction
		}
		body = body[1:]
	case OpI32Const:
		value, read, err := leb128.GetVarint32(body)
		if err != nil {
			return nil, nil, err
		}
		immediate, body = uint64(value), body[read:]
	case OpI64Const:
		value, read, err := leb128.GetVarint64(body)
		if err != nil {
			return nil, nil, err
		}
		immediate, body = uint64(value), body[read:]
	case OpF32Const:
		if len(body) < 4 {
			return nil, nil, io.ErrUnexpectedEOF
		}
		immediate, body = uint64(binary.LittleEndian.Uint32(body)), body[4:]
	case OpF64Const:
		// f64.const
		if len(body) < 8 {
			return nil, nil, io.ErrUnexpectedEOF
		}
		immediate, body = binary.LittleEndian.Uint64(body), body[8:]
	case OpPrefix:
		// Saturating truncation encoding
		satOp, read, err := leb128.GetVarUint32(body)
		if err != nil {
			return nil, nil, err
		}
		immediate, body = uint64(satOp), body[read:]
	default:
		// Single-byte encoding; already done
	}

	instr := Instruction{
		Opcode:    opcode,
		Immediate: immediate,
		Labels:    labels,
	}
	d.ibuf = append(d.ibuf, instr)
	return &d.ibuf[len(d.ibuf)-1], body, nil
}

func (d *decoder) decode(body []byte, out []wasm.ValueType) (Body, error) {
	d.ibuf = make([]Instruction, 0, len(body))

	var instr *Instruction
	d.pushBlock(instr, nil, out)

	var err error
	for {
		ip := len(d.ibuf)
		if instr, body, err = d.decodeInstruction(body); err != nil {
			return Body{}, err
		}

		switch instr.Opcode {
		default:
			if err := d.doStack(instr); err != nil {
				return Body{}, err
			}

		case OpDrop:
			if _, err := d.popOpd(); err != nil {
				return Body{}, err
			}

		case OpSelect:
			if err := d.popOpds(wasm.ValueTypeI32); err != nil {
				return Body{}, err
			}
			t, err := d.popOpd()
			if err != nil {
				return Body{}, err
			}
			if err := d.popOpds(t); err != nil {
				return Body{}, err
			}
			d.pushOpds(t)

		case OpUnreachable:
			d.unreachable()

		case OpIf:
			d.popOpds(wasm.ValueTypeI32)
			fallthrough

		case OpBlock, OpLoop:
			in, out, ok := instr.BlockType(d)
			if !ok {
				return Body{}, wasm.ValidationError("unknown type")
			}
			if err := d.popOpds(in...); err != nil {
				return Body{}, err
			}
			d.pushBlock(instr, in, out)

			stackHeight := d.blocks[len(d.blocks)-1].stackHeight
			instr.Immediate |= (uint64(stackHeight) << 32) & StackHeightMask

		case OpElse:
			b, err := d.popBlock()
			if err != nil {
				return Body{}, err
			}

			if b.Opcode != OpIf || b.Labels[1] != 0 {
				return Body{}, wasm.ValidationError("invalid nesting")
			}
			b.Labels[1] = ip

			d.pushBlock(b.Instruction, b.in, b.out)

		case OpEnd:
			b, err := d.popBlock()
			if err != nil {
				return Body{}, err
			}

			switch {
			case b.Instruction != nil:
				if b.Opcode != OpLoop {
					b.Labels[0] = ip + 1
				}
				if b.Opcode == OpIf && b.Labels[1] != 0 {
					d.ibuf[b.Labels[1]].Labels[0] = ip + 1
				}
				d.pushOpds(b.out...)
			case len(body) != 0:
				return Body{}, wasm.ValidationError("unexpected end instruction")
			default:
				if len(d.stack) != 0 {
					return Body{}, wasm.ValidationError("type mismatch")
				}

				// Condense the instruction list.
				if cap(d.ibuf)-len(d.ibuf) > len(d.ibuf)/10 {
					result := make([]Instruction, len(d.ibuf))
					copy(result, d.ibuf)
					d.ibuf = result
				}
				return Body{
					Instructions: d.ibuf,
					Metrics:      d.metrics,
				}, nil
			}

		case OpBr:
			pop, err := d.labelTypes(instr.Labelidx())
			if err != nil {
				return Body{}, err
			}
			if err := d.popOpds(pop...); err != nil {
				return Body{}, err
			}
			d.unreachable()

		case OpBrIf:
			pop, err := d.labelTypes(instr.Labelidx())
			if err != nil {
				return Body{}, err
			}
			if err := d.popOpds(wasm.ValueTypeI32); err != nil {
				return Body{}, err
			}
			if err := d.popOpds(pop...); err != nil {
				return Body{}, err
			}
			d.pushOpds(pop...)

		case OpBrTable:
			pop, err := d.labelTypes(instr.Default())
			if err != nil {
				return Body{}, err
			}
			for _, l := range instr.Labels {
				typs, err := d.labelTypes(l)
				if err != nil {
					return Body{}, err
				}
				if len(typs) != len(pop) {
					return Body{}, wasm.ValidationError("br_table type mismatch")
				}
				for i, t := range typs {
					if pop[i] != t {
						return Body{}, wasm.ValidationError("br_table type mismatch")
					}
				}
			}
			if err := d.popOpds(wasm.ValueTypeI32); err != nil {
				return Body{}, err
			}
			if err := d.popOpds(pop...); err != nil {
				return Body{}, err
			}
			d.unreachable()

		case OpReturn:
			if err := d.popOpds(d.blocks[0].out...); err != nil {
				return Body{}, err
			}
			d.unreachable()
		}
	}
}

func decodeSingleBlockType(r io.Reader) (uint64, error) {
	n, err := leb128.ReadVarint64(r)
	if err != nil {
		return 0, err
	}
	if n >= 0 {
		return uint64(n) & 0x7fffffffffffffff, nil
	}

	switch n & 0x7f {
	case 0x40, 0x7f, 0x7e, 0x7d, 0x7c:
		return uint64(n&0x7f) | 0x8000000000000000, nil
	default:
		return 0, fmt.Errorf("unexpected block type 0x%02x", byte(n&0x7f))
	}
}

func decodeSingleInstruction(r io.Reader) (Instruction, error) {
	var buf [8]byte
	if _, err := io.ReadFull(r, buf[:1]); err != nil {
		return Instruction{}, err
	}

	opcode := buf[0]
	var immediate uint64
	var labels []int
	switch opcode {
	case OpBlock, OpLoop, OpIf:
		blockType, err := decodeSingleBlockType(r)
		if err != nil {
			return Instruction{}, err
		}
		immediate = blockType
	case OpBr, OpBrIf, OpCall, OpLocalGet, OpLocalSet, OpLocalTee, OpGlobalGet, OpGlobalSet:
		// Index encoding
		index, err := leb128.ReadVarUint32(r)
		if err != nil {
			return Instruction{}, err
		}
		immediate = uint64(index)
	case OpBrTable:
		numLabels, err := leb128.ReadVarUint32(r)
		if err != nil {
			return Instruction{}, err
		}

		labels = make([]int, int(numLabels))
		for i := 0; i < len(labels); i++ {
			label, err := leb128.ReadVarUint32(r)
			if err != nil {
				return Instruction{}, err
			}
			labels[i] = int(label)
		}

		defaultLabel, err := leb128.ReadVarUint32(r)
		if err != nil {
			return Instruction{}, err
		}
		immediate = uint64(defaultLabel)
	case OpCallIndirect:
		index, err := leb128.ReadVarUint32(r)
		if err != nil {
			return Instruction{}, err
		}
		immediate = uint64(index)

		if _, err = io.ReadFull(r, buf[:1]); err != nil {
			return Instruction{}, err
		}
		if buf[0] != 0x00 {
			return Instruction{}, ErrInvalidInstruction
		}
	case OpI32Load, OpI64Load, OpF32Load, OpF64Load, OpI32Load8S, OpI32Load8U, OpI32Load16S, OpI32Load16U, OpI64Load8S, OpI64Load8U, OpI64Load16S, OpI64Load16U, OpI64Load32S, OpI64Load32U, OpI32Store, OpI64Store, OpF32Store, OpF64Store, OpI32Store8, OpI32Store16, OpI64Store8, OpI64Store16, OpI64Store32:
		// Memory encoding
		align, err := leb128.ReadVarUint32(r)
		if err != nil {
			return Instruction{}, err
		}

		offset, err := leb128.ReadVarUint32(r)
		if err != nil {
			return Instruction{}, err
		}

		immediate = memarg(offset, align)
	case OpMemorySize, OpMemoryGrow:
		// memory.size, memory.grow
		if _, err := io.ReadFull(r, buf[:1]); err != nil {
			return Instruction{}, err
		}
		if buf[0] != 0x00 {
			return Instruction{}, ErrInvalidInstruction
		}
	case OpI32Const:
		value, err := leb128.ReadVarint32(r)
		if err != nil {
			return Instruction{}, err
		}
		immediate = uint64(value)
	case OpI64Const:
		value, err := leb128.ReadVarint64(r)
		if err != nil {
			return Instruction{}, err
		}
		immediate = uint64(value)
	case OpF32Const:
		if _, err := io.ReadFull(r, buf[:4]); err != nil {
			return Instruction{}, err
		}
		immediate = uint64(binary.LittleEndian.Uint32(buf[:4]))
	case OpF64Const:
		if _, err := io.ReadFull(r, buf[:8]); err != nil {
			return Instruction{}, err
		}
		immediate = uint64(binary.LittleEndian.Uint64(buf[:8]))
	case OpPrefix:
		// Saturating truncation encoding
		satOp, err := leb128.ReadVarUint32(r)
		if err != nil {
			return Instruction{}, err
		}
		immediate = uint64(satOp)
	default:
		// Single-byte encoding; already done
	}

	return Instruction{
		Opcode:    opcode,
		Immediate: immediate,
		Labels:    labels,
	}, nil
}
