// Copyright 2017 The go-interpreter Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wasm

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"reflect"

	"github.com/pgavlin/warp/wasm/internal/readpos"
)

var ErrInvalidMagic = errors.New("magic header not detected")

const (
	Magic   uint32 = 0x6d736100
	Version uint32 = 0x1
)

// Function represents an entry in the function index space of a module.
type Function struct {
	Sig  *FunctionSig
	Body *FunctionBody
	Host reflect.Value
	Name string
}

// IsHost indicates whether this function is a host function as defined in:
//  https://webassembly.github.io/spec/core/exec/modules.html#host-functions
func (fct *Function) IsHost() bool {
	return fct.Host != reflect.Value{}
}

// Module represents a parsed WebAssembly module:
// http://webassembly.org/docs/modules/
type Module struct {
	Version  uint32
	Sections []Section

	Types    *SectionTypes
	Import   *SectionImports
	Function *SectionFunctions
	Table    *SectionTables
	Memory   *SectionMemories
	Global   *SectionGlobals
	Export   *SectionExports
	Start    *SectionStartFunction
	Elements *SectionElements
	Code     *SectionCode
	Data     *SectionData
	Customs  []*SectionCustom
}

// TableEntry represents a table index and tracks its initialized state.
type TableEntry struct {
	Index       uint32
	Initialized bool
}

// Names returns the names section. If no names section exists, this function returns a MissingSectionError.
func (m *Module) Names() (*NameSection, error) {
	s := m.Custom(CustomSectionName)
	if s == nil {
		return nil, MissingSectionError(0)
	}

	var names NameSection
	if err := names.UnmarshalWASM(bytes.NewReader(s.Data)); err != nil {
		return nil, err
	}

	return &names, nil
}

// Custom returns a custom section with a specific name, if it exists.
func (m *Module) Custom(name string) *SectionCustom {
	for _, s := range m.Customs {
		if s.Name == name {
			return s
		}
	}
	return nil
}

// NewModule creates a new empty module
func NewModule() *Module {
	return &Module{
		Types:    &SectionTypes{},
		Import:   &SectionImports{},
		Table:    &SectionTables{},
		Memory:   &SectionMemories{},
		Global:   &SectionGlobals{},
		Export:   &SectionExports{},
		Start:    &SectionStartFunction{},
		Elements: &SectionElements{},
		Data:     &SectionData{},
	}
}

// ResolveFunc is a function that takes a module name and
// returns a valid resolved module.
type ResolveFunc func(name string) (*Module, error)

// DecodeModule decodes a WASM module.
func DecodeModule(r io.Reader) (*Module, error) {
	reader := &readpos.ReadPos{
		R:      r,
		CurPos: 0,
	}
	m := &Module{}
	magic, err := readU32(reader)
	if err != nil {
		return nil, err
	}
	if magic != Magic {
		return nil, ErrInvalidMagic
	}
	if m.Version, err = readU32(reader); err != nil {
		return nil, err
	}
	if m.Version != Version {
		return nil, errors.New("unknown binary version")
	}

	err = newSectionsReader(m).readSections(reader)
	if err != nil {
		return nil, err
	}

	return m, nil
}

// MustDecode decodes a WASM module and panics on failure.
func MustDecode(r io.Reader) *Module {
	m, err := DecodeModule(r)
	if err != nil {
		panic(fmt.Errorf("decoding module: %w", err))
	}
	return m
}
