package code

import (
	"encoding/binary"
	"io"

	"github.com/pgavlin/warp/wasm/leb128"
)

func encodeBlockType(w io.Writer, instr Instruction) error {
	// Check for special block types.
	if instr.Immediate&0x8000000000000000 != 0 {
		_, err := w.Write([]byte{byte(instr.Immediate)})
		return err
	}

	_, err := leb128.WriteVarint64(w, int64(instr.Immediate))
	return err
}

func encodeInstruction(w io.Writer, instr Instruction) error {
	if _, err := w.Write([]byte{byte(instr.Opcode)}); err != nil {
		return err
	}

	switch instr.Opcode {
	case OpBlock, OpLoop, OpIf:
		// Block encoding
		if err := encodeBlockType(w, instr); err != nil {
			return err
		}
	case OpBr, OpBrIf, OpCall, OpLocalGet, OpLocalSet, OpLocalTee, OpGlobalGet, OpGlobalSet:
		// Index encoding
		if _, err := leb128.WriteVarUint32(w, uint32(instr.Immediate)); err != nil {
			return err
		}
	case OpBrTable:
		// br_table
		if _, err := leb128.WriteVarUint32(w, uint32(len(instr.Labels))); err != nil {
			return err
		}
		for _, l := range instr.Labels {
			if _, err := leb128.WriteVarUint32(w, uint32(l)); err != nil {
				return err
			}
		}

		if _, err := leb128.WriteVarUint32(w, uint32(instr.Immediate)); err != nil {
			return err
		}
	case OpCallIndirect:
		// call_indirect
		if _, err := leb128.WriteVarUint32(w, uint32(instr.Immediate)); err != nil {
			return err
		}
		if _, err := w.Write([]byte{0x00}); err != nil {
			return err
		}
	case OpI32Load, OpI64Load, OpF32Load, OpF64Load, OpI32Load8S, OpI32Load8U, OpI32Load16S, OpI32Load16U, OpI64Load8S, OpI64Load8U, OpI64Load16S, OpI64Load16U, OpI64Load32S, OpI64Load32U, OpI32Store, OpI64Store, OpF32Store, OpF64Store, OpI32Store8, OpI32Store16, OpI64Store8, OpI64Store16, OpI64Store32:
		// Memory encoding
		offset, align := instr.Memarg()
		if _, err := leb128.WriteVarUint32(w, align); err != nil {
			return err
		}
		if _, err := leb128.WriteVarUint32(w, offset); err != nil {
			return err
		}
	case OpMemorySize, OpMemoryGrow:
		if _, err := w.Write([]byte{0x00}); err != nil {
			return err
		}
	case OpI32Const:
		if _, err := leb128.WriteVarint64(w, int64(int32(instr.Immediate))); err != nil {
			return err
		}
	case OpI64Const:
		if _, err := leb128.WriteVarint64(w, int64(instr.Immediate)); err != nil {
			return err
		}
	case OpF32Const:
		// f32.const
		var buf [4]byte
		binary.LittleEndian.PutUint32(buf[:], uint32(instr.Immediate))
		if _, err := w.Write(buf[:]); err != nil {
			return err
		}
	case OpF64Const:
		// f64.const
		var buf [8]byte
		binary.LittleEndian.PutUint64(buf[:], instr.Immediate)
		if _, err := w.Write(buf[:]); err != nil {
			return err
		}
	case OpPrefix:
		// Saturating truncation encoding
		if _, err := leb128.WriteVarUint32(w, uint32(instr.Immediate)); err != nil {
			return err
		}
	default:
		// Single-byte encoding; already done
	}

	return nil
}

func Encode(w io.Writer, body []Instruction) error {
	for {
		if len(body) == 0 {
			return io.ErrUnexpectedEOF
		}

		if err := encodeInstruction(w, body[0]); err != nil {
			return err
		}
		if body[0].Opcode == OpEnd && len(body) == 1 {
			return nil
		}
		body = body[1:]
	}
}
