// Copyright 2017 The go-interpreter Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wasm

import (
	"bytes"
	"debug/dwarf"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"

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

// DWARF returns the DWARF debugging info for the module, if any.
func (m *Module) DWARF() (*dwarf.Data, error) {
	dwarfSuffix := func(s *SectionCustom) string {
		switch {
		case strings.HasPrefix(s.Name, ".debug_"):
			return s.Name[7:]
		default:
			return ""
		}

	}

	// There are many DWARF sections, but these are the ones
	// the debug/dwarf package started with.
	var dat = map[string][]byte{"abbrev": nil, "info": nil, "str": nil, "line": nil, "ranges": nil}
	for _, s := range m.Customs {
		suffix := dwarfSuffix(s)
		if suffix == "" {
			continue
		}
		if _, ok := dat[suffix]; !ok {
			continue
		}
		dat[suffix] = s.Data
	}

	d, err := dwarf.New(dat["abbrev"], nil, nil, dat["info"], dat["line"], nil, dat["ranges"], dat["str"])
	if err != nil {
		return nil, err
	}

	// Look for DWARF4 .debug_types sections and DWARF5 sections.
	for i, s := range m.Customs {
		suffix := dwarfSuffix(s)
		if suffix == "" {
			continue
		}
		if _, ok := dat[suffix]; ok {
			// Already handled.
			continue
		}

		if suffix == "types" {
			if err := d.AddTypes(fmt.Sprintf("types-%d", i), s.Data); err != nil {
				return nil, err
			}
		} else {
			if err := d.AddSection(".debug_"+suffix, s.Data); err != nil {
				return nil, err
			}
		}
	}

	return d, nil
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
