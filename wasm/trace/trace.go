package trace

import (
	"io"

	"github.com/pgavlin/warp/wasm"
	"github.com/pgavlin/warp/wasm/code"
	"github.com/pgavlin/warp/wasm/leb128"
)

// EntryKind describes the type of a trace entry.
type EntryKind byte

const (
	// EntryEnter is an enter trace entry.
	EntryEnter = 0x01
	// EntryLeave is an leave trace entry.
	EntryLeave = 0x02
	// EntryInstruction is an instruction trace entry.
	EntryInstruction = 0x03
	// EntryEnd is an end trace entry.
	EntryEnd = 0x04
)

// A Entry represents a single entry in an execution trace.
type Entry interface {
	// Kind returns the kind of the trace entry.
	Kind() EntryKind
	// Encode encodes the trace entry to the given writer.
	Encode(w io.Writer) error

	decode(r io.Reader) error
}

// A Decoder decodes trace entries from an io.Reader.
type Decoder struct {
	r     io.Reader
	entry Entry
	err   error
}

// NewDecoder creates a new decoder that reads from the given io.Reader.
func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{r: r}
}

// Entry returns the trace entry decoded by the last call to Next, if any.
func (t *Decoder) Entry() Entry {
	return t.entry
}

// Error returns the error encoutered during decoding, if any.
func (t *Decoder) Error() error {
	return t.err
}

// Next decodes the next entry in the trace. Next returns false if an error occurs or if the end of the trace has been reached and true otherwise.
func (t *Decoder) Next() bool {
	var buf [1]byte
	if _, t.err = io.ReadFull(t.r, buf[:]); t.err != nil {
		if t.entry != nil && t.entry.Kind() == EntryEnd {
			t.err = nil
		}
		return false
	}

	switch buf[0] {
	case EntryEnter:
		var entry EnterEntry
		if t.err = entry.decode(t.r); t.err != nil {
			return false
		}
		t.entry = &entry
	case EntryLeave:
		t.entry = &LeaveEntry{}
	case EntryInstruction:
		var entry InstructionEntry
		if t.err = entry.decode(t.r); t.err != nil {
			return false
		}
		t.entry = &entry
	case EntryEnd:
		t.entry = &EndEntry{}
	default:
		return false
	}

	return true
}

// Decode decodes an execution trace from the given reader.
func Decode(r io.Reader) ([]Entry, error) {
	decoder := NewDecoder(r)

	var trace []Entry
	for decoder.Next() {
		trace = append(trace, decoder.Entry())
	}
	if err := decoder.Error(); err != nil {
		return nil, err
	}
	return trace, nil
}

type EnterEntry struct {
	ModuleName        string           `json:"moduleName"`
	FunctionIndex     uint32           `json:"functionIndex"`
	FunctionSignature wasm.FunctionSig `json:"functionSignature"`
}

func (t *EnterEntry) Kind() EntryKind {
	return EntryEnter
}

// Encode encodes an enter trace entry to the given writer.
//
// An enter trace entry is encoded as follows:
//
//     0x01 | Module Name vec(byte) | FunctionIndex u32 | FunctionSignature
//
// The signature is encoded in its WASM format. The function index is LEB128-encoded.
func (t *EnterEntry) Encode(w io.Writer) error {
	if _, err := w.Write([]byte{EntryEnter}); err != nil {
		return err
	}

	if _, err := leb128.WriteVarUint32(w, uint32(len(t.ModuleName))); err != nil {
		return err
	}
	if _, err := w.Write([]byte(t.ModuleName)); err != nil {
		return err
	}

	if _, err := leb128.WriteVarUint32(w, t.FunctionIndex); err != nil {
		return err
	}

	return t.FunctionSignature.MarshalWASM(w)
}

// decode decodes an enter trace entry from the given reader. The kind byte must already have been read.
func (t *EnterEntry) decode(r io.Reader) error {
	n, err := leb128.ReadVarUint32(r)
	if err != nil {
		return err
	}
	moduleName := make([]byte, int(n))
	if _, err = io.ReadFull(r, moduleName); err != nil {
		return err
	}

	functionIndex, err := leb128.ReadVarUint32(r)
	if err != nil {
		return err
	}

	if t.FunctionSignature.UnmarshalWASM(r); err != nil {
		return err
	}

	t.ModuleName = string(moduleName)
	t.FunctionIndex = functionIndex
	return nil
}

type LeaveEntry struct{}

func (t *LeaveEntry) Kind() EntryKind {
	return EntryLeave
}

// Encode encodes a leave trace entry to the given writer.
//
// A leave trace entry is encoded as follows:
//
//     0x02
//
func (t *LeaveEntry) Encode(w io.Writer) error {
	_, err := w.Write([]byte{EntryLeave})
	return err
}

// decode decodes a leave trace entry from the given reader. The kind byte must already have been read.
func (t *LeaveEntry) decode(r io.Reader) error {
	return nil
}

type InstructionEntry struct {
	IP          int              `json:"ip"`
	Instruction code.Instruction `json:"instruction"`
	ArgTypes    []wasm.ValueType `json:"argTypes"`
	ResultTypes []wasm.ValueType `json:"resultTypes"`
	Args        []uint64         `json:"args"`
	Results     []uint64         `json:"results"`
}

func (t *InstructionEntry) Kind() EntryKind {
	return EntryInstruction
}

// Encode encodes an instruction trace entry to the given writer.
//
// An instruction trace entry is encoded as follows:
//
//     0x03 | IP (u32) | Instruction | Args vec(byte, u64) | Results vec(byte, u64)
//
// The instruction is encoded in its WASM format. The instruction pointer, args, and results are all LEB128-encoded.
func (t *InstructionEntry) Encode(w io.Writer) error {
	if _, err := w.Write([]byte{EntryInstruction}); err != nil {
		return err
	}

	if _, err := leb128.WriteVarUint32(w, uint32(t.IP)); err != nil {
		return err
	}

	if err := t.Instruction.Encode(w); err != nil {
		return err
	}

	if _, err := leb128.WriteVarUint32(w, uint32(len(t.Args))); err != nil {
		return err
	}
	for i, arg := range t.Args {
		type_ := wasm.ValueTypeT
		if i < len(t.ArgTypes) {
			type_ = t.ArgTypes[i]
		}
		if _, err := w.Write([]byte{byte(type_)}); err != nil {
			return err
		}
		if _, err := leb128.WriteVarUint64(w, arg); err != nil {
			return err
		}
	}

	if _, err := leb128.WriteVarUint32(w, uint32(len(t.Results))); err != nil {
		return err
	}
	for i, result := range t.Results {
		type_ := wasm.ValueTypeT
		if i < len(t.ResultTypes) {
			type_ = t.ResultTypes[i]
		}
		if _, err := w.Write([]byte{byte(type_)}); err != nil {
			return err
		}
		if _, err := leb128.WriteVarUint64(w, result); err != nil {
			return err
		}
	}

	return nil
}

// decode decodes an instruction trace entry from the given reader. The kind byte must already have been read.
func (t *InstructionEntry) decode(r io.Reader) error {
	var type_ [1]byte

	ip, err := leb128.ReadVarUint32(r)
	if err != nil {
		return err
	}

	var instr code.Instruction
	if err := instr.Decode(r); err != nil {
		return err
	}

	n, err := leb128.ReadVarUint32(r)
	if err != nil {
		return err
	}
	argTypes, args := make([]wasm.ValueType, int(n)), make([]uint64, int(n))
	for i := range args {
		if _, err := io.ReadFull(r, type_[:]); err != nil {
			return err
		}
		argTypes[i] = wasm.ValueType(type_[0])

		arg, err := leb128.ReadVarUint64(r)
		if err != nil {
			return err
		}
		args[i] = arg
	}

	n, err = leb128.ReadVarUint32(r)
	if err != nil {
		return err
	}
	resultTypes, results := make([]wasm.ValueType, int(n)), make([]uint64, int(n))
	for i := range results {
		if _, err := io.ReadFull(r, type_[:]); err != nil {
			return err
		}
		resultTypes[i] = wasm.ValueType(type_[0])

		result, err := leb128.ReadVarUint64(r)
		if err != nil {
			return err
		}
		results[i] = result
	}

	t.IP = int(ip)
	t.Instruction = instr
	t.ArgTypes = argTypes
	t.ResultTypes = resultTypes
	t.Args = args
	t.Results = results
	return nil
}

type EndEntry struct{}

func (t *EndEntry) Kind() EntryKind {
	return EntryEnd
}

// Encode encodes an end trace entry to the given writer.
//
// An end trace entry is encoded as follows:
//
//     0x04
//
func (t *EndEntry) Encode(w io.Writer) error {
	_, err := w.Write([]byte{EntryEnd})
	return err
}

// decode decodes an end trace entry from the given reader. The kind byte must already have been read.
func (t *EndEntry) decode(r io.Reader) error {
	return nil
}
