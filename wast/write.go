// Copyright 2018 The go-interpreter Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wast

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"math"
	"strconv"

	"github.com/pgavlin/warp/wasm"
	"github.com/pgavlin/warp/wasm/code"
)

const tab = `  `

// WriteTo writes a WASM module in a text representation.
func WriteTo(w io.Writer, m *wasm.Module) error {
	wr, err := newWriter(w, m)
	if err != nil {
		return err
	}
	return wr.writeModule()
}

type writer struct {
	bw *bufio.Writer
	m  *wasm.Module

	fnames map[uint32]string
	lnames map[uint32]map[uint32]string

	funcOff int

	importedFunctions []uint32
	importedGlobals   []wasm.GlobalVar

	locals []wasm.ValueType
}

func newWriter(w io.Writer, m *wasm.Module) (*writer, error) {
	wr := &writer{bw: bufio.NewWriter(w), m: m}

	if names, err := m.Names(); err == nil {
		wr.fnames, wr.lnames = map[uint32]string{}, map[uint32]map[uint32]string{}
		for _, subsection := range names.Entries {
			switch subsection := subsection.(type) {
			case *wasm.FunctionNamesSubsection:
				for _, name := range subsection.Names {
					wr.fnames[name.Index] = name.Name
				}
			case *wasm.LocalNamesSubsection:
				for _, func_ := range subsection.Funcs {
					m := map[uint32]string{}
					for _, name := range func_.Names {
						m[name.Index] = name.Name
					}
					wr.lnames[func_.Index] = m
				}
			}
		}
	}

	return wr, nil
}

func (w *writer) writeModule() (err error) {
	defer func() {
		if x := recover(); x != nil {
			if e, ok := x.(error); ok {
				err = e
				return
			}
			panic(x)
		}
	}()
	defer func() {
		err = w.bw.Flush()
	}()

	w.WriteString("(module")

	w.writeTypes()
	w.writeImports()
	w.writeFunctions()
	w.writeGlobals()
	w.writeTables()
	w.writeMemory()
	w.writeExports()
	w.writeElements()
	w.writeData()

	w.WriteString(")\n")
	return nil
}

func (w *writer) writeTypes() {
	if w.m.Types == nil {
		return
	}
	w.WriteString("\n")
	for i, t := range w.m.Types.Entries {
		if i != 0 {
			w.WriteString("\n")
		}
		w.Print(tab+"(type (;%d;) ", i)
		w.writeFuncSignature(t)
		w.WriteString(")")
	}
}

func (w *writer) writeFuncSignature(t wasm.FunctionSig) error {
	w.WriteString("(func")
	defer w.WriteString(")")
	return w.writeFuncType(t)
}

func (w *writer) writeFuncType(t wasm.FunctionSig) error {
	if len(t.ParamTypes) != 0 {
		w.WriteString(" (param")
		for _, p := range t.ParamTypes {
			w.WriteString(" ")
			w.WriteString(p.String())
		}
		w.WriteString(")")
	}
	if len(t.ReturnTypes) != 0 {
		w.WriteString(" (result")
		for _, p := range t.ReturnTypes {
			w.WriteString(" ")
			w.WriteString(p.String())
		}
		w.WriteString(")")
	}
	return nil
}

func (w *writer) writeImports() {
	w.funcOff = 0
	if w.m.Import == nil {
		return
	}
	w.WriteString("\n")
	for i, e := range w.m.Import.Entries {
		if i != 0 {
			w.WriteString("\n")
		}
		w.WriteString(tab + "(import ")
		w.Print("%q %q ", e.ModuleName, e.FieldName)
		switch im := e.Type.(type) {
		case wasm.FuncImport:
			w.Print("(func (;%d;) (type %d))", w.funcOff, im.Type)
			if w.fnames == nil {
				w.fnames = map[uint32]string{}
			}
			w.fnames[uint32(w.funcOff)] = e.ModuleName + "." + e.FieldName

			w.funcOff++
			w.importedFunctions = append(w.importedFunctions, im.Type)
		case wasm.TableImport:
			// TODO
		case wasm.MemoryImport:
			// TODO
		case wasm.GlobalVarImport:
			// TODO
			w.importedGlobals = append(w.importedGlobals, im.Type)
		}
		w.WriteString(")")
	}
}

func (w *writer) writeFunctions() {
	if w.m.Function == nil {
		return
	}
	w.WriteString("\n")
	for i, t := range w.m.Function.Types {
		if i != 0 {
			w.WriteString("\n")
		}
		ind := w.funcOff + i
		w.WriteString(tab + "(func")
		if name, ok := w.fnames[uint32(ind)]; ok {
			w.WriteString(" $" + name)
		}
		fmt.Fprintf(w.bw, " (;%d;) (type %d)", ind, int(t))
		var sig wasm.FunctionSig
		if int(t) < len(w.m.Types.Entries) {
			sig = w.m.Types.Entries[t]
			w.writeFuncType(sig)
		}
		if w.m.Code != nil && i < len(w.m.Code.Bodies) {
			b := w.m.Code.Bodies[i]
			w.locals = append(w.locals[:0], sig.ParamTypes...)
			for _, l := range b.Locals {
				for i := uint32(0); i < l.Count; i++ {
					w.locals = append(w.locals, l.Type)
				}
			}

			if len(b.Locals) > 0 {
				w.WriteString("\n" + tab + tab + "(local")

				names := w.lnames[uint32(ind)]

				idx := uint32(0)
				for _, l := range b.Locals {
					for i := 0; i < int(l.Count); i++ {
						if name, ok := names[idx]; ok {
							w.WriteString(" $" + name)
						}
						w.WriteString(" ")
						w.WriteString(l.Type.String())

						idx++
					}
				}

				w.WriteString(")")
			}
			w.writeCode(b.Code, false, sig.ReturnTypes)
		}
		w.WriteString(")")
	}
}

func (w *writer) writeGlobals() {
	if w.m.Global == nil {
		return
	}
	for i, e := range w.m.Global.Globals {
		w.WriteString("\n")
		w.WriteString(tab + "(global ")
		w.Print("(;%d;)", i)
		if e.Type.Mutable {
			w.WriteString(" (mut")
		}
		w.Print(" %v", e.Type.Type)
		if e.Type.Mutable {
			w.WriteString(")")
		}
		w.WriteString(" (")
		w.writeCode(e.Init, true, []wasm.ValueType{e.Type.Type})
		w.WriteString("))")
	}
}

func (w *writer) writeTables() {
	if w.m.Table == nil {
		return
	}
	w.WriteString("\n")
	for i, t := range w.m.Table.Entries {
		w.WriteString(tab + "(table ")
		w.Print("(;%d;)", i)
		w.Print(" %d %d ", t.Limits.Initial, t.Limits.Maximum)
		switch t.ElementType {
		case wasm.ElemTypeAnyFunc:
			w.WriteString("anyfunc")
		}
		w.WriteString(")")
	}
}

func (w *writer) writeMemory() {
	if w.m.Memory == nil {
		return
	}
	w.WriteString("\n")
	for i, e := range w.m.Memory.Entries {
		w.WriteString(tab + "(memory ")
		w.Print("(;%d;)", i)
		w.Print(" %d", e.Limits.Initial)
		if e.Limits.Flags&0x1 != 0 {
			w.Print(" %d", e.Limits.Maximum)
		}
		w.WriteString(")")
	}
}

func (w *writer) writeExports() {
	if w.m.Export == nil {
		return
	}
	w.WriteString("\n")
	for i, e := range w.m.Export.Entries {
		if i != 0 {
			w.WriteString("\n")
		}
		w.Print(tab+"(export %q (", e.FieldStr)
		switch e.Kind {
		case wasm.ExternalFunction:
			w.WriteString("func")
		case wasm.ExternalMemory:
			w.WriteString("memory")
		case wasm.ExternalTable:
			w.WriteString("table")
		case wasm.ExternalGlobal:
			w.WriteString("global")
		}
		w.Print(" %d))", e.Index)
	}
}

func (w *writer) writeElements() {
	if w.m.Elements == nil {
		return
	}
	for _, d := range w.m.Elements.Entries {
		w.WriteString("\n")
		w.WriteString(tab + "(elem")
		if d.Index != 0 {
			w.Print(" %d", d.Index)
		}
		w.WriteString(" (")
		w.writeCode(d.Offset, true, []wasm.ValueType{wasm.ValueTypeI32})
		w.WriteString(")")
		for _, v := range d.Elems {
			w.Print(" %d", v)
		}
		w.WriteString(")")
	}
}

func (w *writer) writeData() {
	if w.m.Data == nil {
		return
	}
	for _, d := range w.m.Data.Entries {
		w.WriteString("\n")
		w.WriteString(tab + "(data")
		if d.Index != 0 {
			w.Print(" %d", d.Index)
		}
		w.WriteString(" (")
		w.writeCode(d.Offset, true, []wasm.ValueType{wasm.ValueTypeI32})
		w.Print(") %s)", quoteData(d.Data))
	}
}

func (w *writer) WriteString(s string) {
	if _, err := w.bw.WriteString(s); err != nil {
		panic(err)
	}
}

func (w *writer) Print(format string, args ...interface{}) {
	if _, err := fmt.Fprintf(w.bw, format, args...); err != nil {
		panic(err)
	}
}

func quoteData(p []byte) string {
	buf := new(bytes.Buffer)
	buf.WriteRune('"')
	for _, b := range p {
		if strconv.IsGraphic(rune(b)) && b < 0xa0 && b != '"' && b != '\\' {
			buf.WriteByte(b)
		} else {
			s := strconv.FormatInt(int64(b), 16)
			if len(s) == 1 {
				s = "0" + s
			}
			buf.WriteString(`\` + s)
		}
	}
	buf.WriteRune('"')
	return buf.String()
}

func (w *writer) writeCode(bytecode []byte, isInit bool, out []wasm.ValueType) {
	body, err := code.Decode(bytecode, w, out)
	if err != nil {
		panic(err)
	}
	instrs := body.Instructions

	tabs := 2
	block := 0
	writeBlock := func(d int) {
		w.Print(" %d (;@%d;)", d, block-d)
	}
	for i, ins := range instrs {
		if i == len(instrs)-1 && ins.Opcode == code.OpEnd {
			break
		}

		if !isInit {
			w.WriteString("\n")
		}
		switch ins.Opcode {
		case code.OpEnd, code.OpElse:
			tabs--
			block--
		}
		if isInit {
			if i > 0 {
				w.WriteString(" ")
			}
		} else {
			for i := 0; i < tabs; i++ {
				w.WriteString(tab)
			}
		}
		w.WriteString(ins.OpString())
		switch ins.Opcode {
		case code.OpElse:
			tabs++
			block++
		case code.OpBlock, code.OpLoop, code.OpIf:
			tabs++
			block++
			if ins.Immediate != code.BlockTypeEmpty {
				w.WriteString(" (result ")
				switch ins.Immediate {
				case code.BlockTypeI32:
					w.WriteString("i32")
				case code.BlockTypeI64:
					w.WriteString("i64")
				case code.BlockTypeF32:
					w.WriteString("f32")
				case code.BlockTypeF64:
					w.WriteString("f64")
				default:
					w.WriteString(strconv.FormatUint(uint64(ins.Typeidx()), 10))
				}
				w.WriteString(")")
			}
			w.Print("  ;; label = @%d", block)
		case code.OpI32Const, code.OpI64Const:
			w.WriteString(" " + strconv.FormatInt(ins.I64(), 10))
		case code.OpF32Const:
			w.WriteString(" " + formatFloat32(ins.F32()))
		case code.OpF64Const:
			w.WriteString(" " + formatFloat64(ins.F64()))
		case code.OpBrIf, code.OpBr:
			writeBlock(ins.Labelidx())
		case code.OpBrTable:
			for _, l := range ins.Labels {
				writeBlock(l)
			}
			writeBlock(ins.Default())
		case code.OpCall:
			i1 := ins.Funcidx()
			if name, ok := w.fnames[i1]; ok {
				w.WriteString(" $")
				w.WriteString(name)
			} else {
				w.Print(" %v", i1)
			}
		case code.OpCallIndirect:
			w.Print(" (type %d)", ins.Typeidx())
		case code.OpLocalGet, code.OpLocalSet, code.OpLocalTee, code.OpGlobalGet, code.OpGlobalSet:
			w.Print(" %v", ins.Immediate)
		case code.OpI32Store, code.OpI64Store,
			code.OpI32Store8, code.OpI64Store8,
			code.OpI32Store16, code.OpI64Store16,
			code.OpI64Store32,
			code.OpF32Store, code.OpF64Store,
			code.OpI32Load, code.OpI64Load,
			code.OpI32Load8U, code.OpI32Load8S,
			code.OpI32Load16U, code.OpI32Load16S,
			code.OpI64Load8U, code.OpI64Load8S,
			code.OpI64Load16U, code.OpI64Load16S,
			code.OpI64Load32U, code.OpI64Load32S,
			code.OpF32Load, code.OpF64Load:

			i1, i2 := ins.Memarg()
			dst := 0 // in log 2 (i8)
			switch ins.Opcode {
			case code.OpI64Load, code.OpI64Store,
				code.OpF64Load, code.OpF64Store:
				dst = 3
			case code.OpI32Load, code.OpI64Load32S, code.OpI64Load32U,
				code.OpI32Store, code.OpI64Store32,
				code.OpF32Load, code.OpF32Store:
				dst = 2
			case code.OpI32Load16U, code.OpI32Load16S, code.OpI64Load16U, code.OpI64Load16S,
				code.OpI32Store16, code.OpI64Store16:
				dst = 1
			case code.OpI32Load8U, code.OpI32Load8S, code.OpI64Load8U, code.OpI64Load8S,
				code.OpI32Store8, code.OpI64Store8:
				dst = 0
			}
			if i1 != 0 {
				w.Print(" offset=%d", i1)
			}
			if int(i2) != dst {
				w.Print(" align=%d", 1<<i2)
			}
		}
	}
}

func (w *writer) GetLocalType(localidx uint32) (wasm.ValueType, bool) {
	if localidx >= uint32(len(w.locals)) {
		return 0, false
	}
	return w.locals[int(localidx)], true
}

func (w *writer) GetGlobalType(globalidx uint32) (wasm.GlobalVar, bool) {
	if globalidx < uint32(len(w.importedGlobals)) {
		return w.importedGlobals[int(globalidx)], true
	}
	globalidx -= uint32(len(w.importedGlobals))
	if w.m.Global == nil || globalidx >= uint32(len(w.m.Global.Globals)) {
		return wasm.GlobalVar{}, false
	}
	return w.m.Global.Globals[int(globalidx)].Type, true
}

func (w *writer) GetFunctionSignature(funcidx uint32) (wasm.FunctionSig, bool) {
	if funcidx < uint32(len(w.importedFunctions)) {
		return w.GetType(w.importedFunctions[int(funcidx)])
	}
	funcidx -= uint32(len(w.importedFunctions))
	if w.m.Function == nil || funcidx >= uint32(len(w.m.Function.Types)) {
		return wasm.FunctionSig{}, false
	}
	return w.GetType(w.m.Function.Types[int(funcidx)])
}

func (w *writer) GetType(typeidx uint32) (wasm.FunctionSig, bool) {
	if w.m.Types == nil || typeidx >= uint32(len(w.m.Types.Entries)) {
		return wasm.FunctionSig{}, false
	}
	return w.m.Types.Entries[int(typeidx)], true
}

func (w *writer) HasMemory(index uint32) bool {
	return true
}

func (w *writer) HasTable(index uint32) bool {
	return true
}

func formatFloat32(v float32) string {
	s := ""
	if v == float32(int32(v)) {
		s = strconv.FormatInt(int64(v), 10)
	} else {
		s = strconv.FormatFloat(float64(v), 'g', -1, 32)
	}
	return fmt.Sprintf("%#0x (;=%s;)", math.Float32bits(v), s)
}

func formatFloat64(v float64) string {
	// TODO: https://github.com/WebAssembly/wabt/blob/master/src/literal.cc (FloatWriter<T>::WriteHex)
	s := ""
	if v == float64(int64(v)) {
		s = strconv.FormatInt(int64(v), 10)
	} else {
		s = strconv.FormatFloat(float64(v), 'g', -1, 64)
	}
	return fmt.Sprintf("%#0x (;=%v;)", math.Float64bits(v), s)
}
