package trace

import (
	"bytes"
	"fmt"
	"io"
	"math"

	"github.com/pgavlin/warp/wasm"
	"github.com/pgavlin/warp/wasm/code"
)

func printValues(w io.Writer, values []uint64, types []wasm.ValueType) error {
	var b bytes.Buffer

	fmt.Fprint(&b, " [")
	if len(types) != len(values) {
		for i, v := range values {
			if i != 0 {
				fmt.Fprint(&b, ", ")
			}
			fmt.Fprintf(&b, "0x%x", v)
		}
	} else {
		for i, v := range values {
			if i != 0 {
				fmt.Fprint(&b, ", ")
			}
			switch types[i] {
			case wasm.ValueTypeI32:
				fmt.Fprintf(&b, "%d (0x%08x)", int32(v), uint32(v))
			case wasm.ValueTypeI64:
				fmt.Fprintf(&b, "%d (0x%016x)", int64(v), uint64(v))
			case wasm.ValueTypeF32:
				fmt.Fprintf(&b, "%g (0x%08x)", math.Float32frombits(uint32(v)), uint32(v))
			case wasm.ValueTypeF64:
				fmt.Fprintf(&b, "%g (0x%016x)", math.Float64frombits(v), v)
			default:
				fmt.Fprintf(&b, "0x%x", v)
			}
		}
	}
	fmt.Fprint(&b, "]")

	_, err := w.Write(b.Bytes())
	return err
}

type Names interface {
	FunctionName(moduleName string, index uint32) (string, bool)
	LocalName(moduleName string, functionIndex, localIndex uint32) (string, bool)
}

type frame struct {
	moduleName    string
	functionIndex uint32
}

type Printer struct {
	names  Names
	frames []frame
}

func NewPrinter(names Names) *Printer {
	return &Printer{names: names}
}

func (p *Printer) where() (string, string) {
	if len(p.frames) == 0 {
		return "", ""
	}
	f := p.frames[len(p.frames)-1]
	if name, ok := p.names.FunctionName(f.moduleName, f.functionIndex); ok {
		return f.moduleName, "$" + name
	}
	return f.moduleName, fmt.Sprintf("%v", f.functionIndex)
}

// Print prints a textual representation of the given trace entry to the given io.Writer.
func (p *Printer) Print(w io.Writer, entry Entry) error {
	switch entry := entry.(type) {
	case *EnterEntry:
		p.frames = append(p.frames, frame{moduleName: entry.ModuleName, functionIndex: entry.FunctionIndex})

		moduleName, functionName := p.where()
		if _, err := fmt.Fprintf(w, "enter(%q, %v, %v)\n", moduleName, functionName, entry.FunctionSignature); err != nil {
			return err
		}
	case *LeaveEntry:
		moduleName, functionName := p.where()
		if _, err := fmt.Fprintf(w, "leave(%q, %v)\n", moduleName, functionName); err != nil {
			return err
		}

		if len(p.frames) > 0 {
			p.frames = p.frames[:len(p.frames)-1]
			moduleName, functionName = p.where()
			if _, err := fmt.Fprintf(w, "resume(%q, %v)\n", moduleName, functionName); err != nil {
				return err
			}
		}
	case *InstructionEntry:
		instruction := ""
		if len(p.frames) > 0 {
			frame := p.frames[len(p.frames)-1]

			switch entry.Instruction.Opcode {
			case code.OpCall:
				if name, ok := p.names.FunctionName(frame.moduleName, entry.Instruction.Funcidx()); ok {
					instruction = fmt.Sprintf("call $%s", name)
				}
			case code.OpLocalGet, code.OpLocalSet, code.OpLocalTee:
				if name, ok := p.names.LocalName(frame.moduleName, frame.functionIndex, entry.Instruction.Localidx()); ok {
					instruction = fmt.Sprintf("%s $%s", entry.Instruction.OpString(), name)
				}
			case code.OpI32Load, code.OpI64Load, code.OpF32Load, code.OpF64Load, code.OpI32Load8S, code.OpI32Load8U, code.OpI32Load16S, code.OpI32Load16U, code.OpI64Load8S, code.OpI64Load8U, code.OpI64Load16S, code.OpI64Load16U, code.OpI64Load32S, code.OpI64Load32U, code.OpI32Store, code.OpI64Store, code.OpF32Store, code.OpF64Store, code.OpI32Store8, code.OpI32Store16, code.OpI64Store8, code.OpI64Store16, code.OpI64Store32:
				if len(entry.Args) > 0 {
					base := uint32(entry.Args[0])
					offset, _ := entry.Instruction.Memarg()
					instruction = fmt.Sprintf("%v (0x%08x)", &entry.Instruction, base+offset)
				}
			}
		}
		if instruction == "" {
			instruction = fmt.Sprintf("%v", &entry.Instruction)
		}

		if _, err := fmt.Fprintf(w, "%04x: %v;", entry.IP, instruction); err != nil {
			return err
		}

		if err := printValues(w, entry.Args, entry.ArgTypes); err != nil {
			return err
		}
		if _, err := fmt.Fprint(w, " ->"); err != nil {
			return err
		}
		if err := printValues(w, entry.Results, entry.ResultTypes); err != nil {
			return err
		}
		if _, err := fmt.Fprint(w, "\n"); err != nil {
			return err
		}
	}
	return nil
}

func PrintTrace(w io.Writer, r io.Reader, names Names) error {
	decoder, printer := NewDecoder(r), NewPrinter(names)
	for decoder.Next() {
		if err := printer.Print(w, decoder.Entry()); err != nil {
			return err
		}
	}
	return decoder.Error()
}
