// Copyright 2017 The go-interpreter Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wasm

import (
	"io"

	"github.com/pgavlin/warp/wasm/leb128"
)

// Import is an interface implemented by types that can be imported by a WebAssembly module.
type Import interface {
	Kind() External
	Marshaler
	isImport()
}

// ImportEntry describes an import statement in a Wasm module.
type ImportEntry struct {
	ModuleName string // module name string
	FieldName  string // field name string

	// If Kind is Function, Type is a FuncImport containing the type index of the function signature
	// If Kind is Table, Type is a TableImport containing the type of the imported table
	// If Kind is Memory, Type is a MemoryImport containing the type of the imported memory
	// If the Kind is Global, Type is a GlobalVarImport
	Type Import
}

type FuncImport struct {
	Type uint32
}

func (FuncImport) isImport() {}
func (FuncImport) Kind() External {
	return ExternalFunction
}
func (f FuncImport) MarshalWASM(w io.Writer) error {
	_, err := leb128.WriteVarUint32(w, uint32(f.Type))
	return err
}

type TableImport struct {
	Type Table
}

func (TableImport) isImport() {}
func (TableImport) Kind() External {
	return ExternalTable
}
func (t TableImport) MarshalWASM(w io.Writer) error {
	return t.Type.MarshalWASM(w)
}

type MemoryImport struct {
	Type Memory
}

func (MemoryImport) isImport() {}
func (MemoryImport) Kind() External {
	return ExternalMemory
}
func (t MemoryImport) MarshalWASM(w io.Writer) error {
	return t.Type.MarshalWASM(w)
}

type GlobalVarImport struct {
	Type GlobalVar
}

func (GlobalVarImport) isImport() {}
func (GlobalVarImport) Kind() External {
	return ExternalGlobal
}
func (t GlobalVarImport) MarshalWASM(w io.Writer) error {
	return t.Type.MarshalWASM(w)
}
