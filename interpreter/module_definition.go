package interpreter

import (
	"fmt"
	"io"
	"reflect"

	"github.com/pgavlin/warp/exec"
	"github.com/pgavlin/warp/wasm"
)

// ErrInvalidMemoryIndex indicates that the memory index associated with a data section is
// not valid.
var ErrInvalidMemoryIndex = fmt.Errorf("invalid memory index")

const (
	mixedCode = 0
	icodeOnly = 1
	fcodeOnly = 2
)

type moduleDefinition struct {
	mod      *wasm.Module
	codeKind int
}

// NewModuleDefinition creates a new ModuleDefinition from the given WASM module. The
// module's functions will be executed by the intepreter.
func NewModuleDefinition(module *wasm.Module) exec.ModuleDefinition {
	return newModuleDefinition(module, mixedCode)
}

func newModuleDefinition(module *wasm.Module, codeKind int) exec.ModuleDefinition {
	return &moduleDefinition{mod: module, codeKind: codeKind}
}

// LoadModuleDefinition decodes a WASM module from the given Reader and uses it to create
// a ModuleDefinition.
func LoadModuleDefinition(r io.Reader) (exec.ModuleDefinition, error) {
	mod, err := wasm.DecodeModule(r)
	if err != nil {
		return nil, err
	}
	return NewModuleDefinition(mod), nil
}

func (def *moduleDefinition) Allocate(name string) (exec.AllocatedModule, error) {
	module := allocatedModule{
		module: &module{name: name, codeKind: def.codeKind},
	}

	// Allocate import entries.
	if def.mod.Import != nil {
		module.imports = def.mod.Import.Entries

		funcImports, globalImports := 0, 0
		for _, import_ := range def.mod.Import.Entries {
			switch import_.Type.(type) {
			case wasm.FuncImport:
				funcImports++
			case wasm.GlobalVarImport:
				globalImports++
			}
		}
		module.importedFunctions = make([]exec.Function, funcImports)
		module.importedGlobals = make([]*exec.Global, globalImports)
	}

	if def.mod.Types != nil {
		module.types = def.mod.Types.Entries
	}

	// Allocate globals, functions, memories, and tables.
	if def.mod.Global != nil {
		module.globals = def.mod.Global.Globals
		module.module.globals = def.allocateGlobals()
	}

	functions, err := def.allocateFunctions(module.module)
	if err != nil {
		return nil, err
	}
	module.functions = functions

	if def.mod.Memory != nil && len(def.mod.Memory.Entries) != 0 {
		mem0Def := def.mod.Memory.Entries[0]
		min := mem0Def.Limits.Initial
		max := mem0Def.Limits.Maximum
		if mem0Def.Limits.Flags == 0 {
			max = 65536
		}
		m := exec.NewMemory(min, max)
		module.mem0 = &m
	}

	if def.mod.Table != nil && len(def.mod.Table.Entries) != 0 {
		table0Def := def.mod.Table.Entries[0]
		min := table0Def.Limits.Initial
		max := table0Def.Limits.Maximum
		if table0Def.Limits.Flags == 0 {
			max = ^uint32(0)
		}
		t := exec.NewTable(min, max)
		module.table0 = &t
	}

	// Define exports.
	if def.mod.Export != nil {
		module.exports = def.mod.Export.Entries

		exports := map[string]interface{}{}
		for _, export := range def.mod.Export.Entries {
			switch export.Kind {
			case wasm.ExternalFunction:
				exports[export.FieldStr], _ = module.getFunction(export.Index)
			case wasm.ExternalMemory:
				if export.Index != 0 {
					return nil, ErrInvalidMemoryIndex
				}
				exports[export.FieldStr] = module.mem0
			case wasm.ExternalTable:
				if export.Index != 0 {
					return nil, exec.InvalidTableIndexError(export.Index)
				}
				exports[export.FieldStr] = module.table0
			case wasm.ExternalGlobal:
				exports[export.FieldStr], _ = module.getGlobal(export.Index)
			}
		}
		module.module.exports = exports
	}

	// Record initialization info.
	if def.mod.Elements != nil {
		module.elements = def.mod.Elements.Entries
	}
	if def.mod.Data != nil {
		module.data = def.mod.Data.Entries
	}
	module.start = def.mod.Start

	return &module, nil
}

func (def *moduleDefinition) allocateGlobals() []exec.Global {
	globals := make([]exec.Global, len(def.mod.Global.Globals))
	for i, globalEntry := range def.mod.Global.Globals {
		switch globalEntry.Type.Type {
		case wasm.ValueTypeI32:
			globals[i] = exec.NewGlobalI32(!globalEntry.Type.Mutable, 0)
		case wasm.ValueTypeI64:
			globals[i] = exec.NewGlobalI64(!globalEntry.Type.Mutable, 0)
		case wasm.ValueTypeF32:
			globals[i] = exec.NewGlobalF32(!globalEntry.Type.Mutable, 0)
		case wasm.ValueTypeF64:
			globals[i] = exec.NewGlobalF64(!globalEntry.Type.Mutable, 0)
		default:
			panic("unreachable")
		}
	}
	return globals
}

func (def *moduleDefinition) allocateFunctions(module *module) ([]function, error) {
	if def.mod.Code == nil {
		return nil, nil
	}

	functions := make([]function, len(def.mod.Code.Bodies))
	for i, body := range def.mod.Code.Bodies {
		f := &functions[i]

		f.module = module
		f.index = uint32(len(module.importedFunctions) + i)
		f.bytecode = body.Code
		f.localEntries = body.Locals

		typeIndex := def.mod.Function.Types[i]
		f.signature = def.mod.Types.Entries[typeIndex]
	}
	return functions, nil
}

type allocatedModule struct {
	*module

	imports  []wasm.ImportEntry         // The module's imports.
	globals  []wasm.GlobalEntry         // The module's globals.
	exports  []wasm.ExportEntry         // The module's exports.
	elements []wasm.ElementSegment      // The module's element segments.
	data     []wasm.DataSegment         // The module's data segments.
	start    *wasm.SectionStartFunction // The module's start function, if any.
}

func (m *allocatedModule) Instantiate(imports exec.ImportResolver) (exec.Module, error) {
	// Resolve imports.
	funcidx, globalidx := 0, 0
	for _, import_ := range m.imports {
		switch type_ := import_.Type.(type) {
		case wasm.FuncImport:
			if type_.Type >= uint32(len(m.types)) {
				return nil, exec.ErrInvalidTypeIndex
			}
			sig := m.types[int(type_.Type)]
			f, err := imports.ResolveFunction(import_.ModuleName, import_.FieldName, sig)
			if err != nil {
				return nil, err
			}
			m.importedFunctions[funcidx] = f
			funcidx++
		case wasm.MemoryImport:
			if m.mem0 != nil {
				return nil, &exec.InvalidImportError{}
			}
			mem, err := imports.ResolveMemory(import_.ModuleName, import_.FieldName, type_.Type)
			if err != nil {
				return nil, err
			}
			m.mem0 = mem
		case wasm.TableImport:
			if m.table0 != nil {
				return nil, &exec.InvalidImportError{}
			}
			table, err := imports.ResolveTable(import_.ModuleName, import_.FieldName, type_.Type)
			if err != nil {
				return nil, err
			}
			m.table0 = table
		case wasm.GlobalVarImport:
			g, err := imports.ResolveGlobal(import_.ModuleName, import_.FieldName, type_.Type)
			if err != nil {
				return nil, err
			}
			m.importedGlobals[globalidx] = g
			globalidx++
		default:
			panic("unreachable")
		}
	}

	// Initialize globals.
	if err := m.initializeGlobals(); err != nil {
		return nil, err
	}

	// Define exports.
	for _, export := range m.exports {
		switch export.Kind {
		case wasm.ExternalFunction:
			m.module.exports[export.FieldStr], _ = m.getFunction(export.Index)
		case wasm.ExternalMemory:
			if export.Index != 0 {
				return nil, ErrInvalidMemoryIndex
			}
			m.module.exports[export.FieldStr] = m.mem0
		case wasm.ExternalTable:
			if export.Index != 0 {
				return nil, exec.InvalidTableIndexError(export.Index)
			}
			m.module.exports[export.FieldStr] = m.table0
		case wasm.ExternalGlobal:
			m.module.exports[export.FieldStr], _ = m.getGlobal(export.Index)
		}
	}

	// Check element and data segments.
	elementOffsets, err := m.checkElementSegments()
	if err != nil {
		return nil, err
	}
	dataOffsets, err := m.checkDataSegments()
	if err != nil {
		return nil, err
	}

	// Evaluate element and segments.
	m.evaluateElementSegments(elementOffsets)
	m.evaluateDataSegments(dataOffsets)

	// Evaluate the module's start function, if any.
	if m.start != nil {
		thread := exec.NewThread(0)
		func_, _ := m.getFunction(m.start.Index)
		func_.UncheckedCall(&thread, nil, nil)
	}

	return m.module, nil
}

func (m *allocatedModule) initializeGlobals() error {
	for i, globalEntry := range m.globals {
		value, err := exec.EvalConstantExpression(m.importedGlobals, globalEntry.Init)
		if err != nil {
			return err
		}

		switch value := value.(type) {
		case int32:
			m.module.globals[i] = exec.NewGlobalI32(!globalEntry.Type.Mutable, value)
		case int64:
			m.module.globals[i] = exec.NewGlobalI64(!globalEntry.Type.Mutable, value)
		case float32:
			m.module.globals[i] = exec.NewGlobalF32(!globalEntry.Type.Mutable, value)
		case float64:
			m.module.globals[i] = exec.NewGlobalF64(!globalEntry.Type.Mutable, value)
		default:
			panic("unreachable")
		}
	}
	return nil
}

func (m *allocatedModule) checkElementSegments() ([]int, error) {
	offsets := make([]int, len(m.elements))
	for i, element := range m.elements {
		offsetV, err := exec.EvalConstantExpression(m.importedGlobals, element.Offset)
		if err != nil {
			return nil, err
		}
		offset, ok := offsetV.(int32)
		if !ok {
			return nil, exec.InvalidValueTypeInitExprError{Wanted: reflect.Int32, Got: reflect.ValueOf(offsetV).Kind()}
		}

		if element.Index != 0 || m.table0 == nil {
			return nil, exec.InvalidTableIndexError(element.Index)
		}

		entries := m.table0.Entries()
		if offset < 0 || offset > int32(len(entries)) || len(element.Elems) > len(entries[int(offset):]) {
			return nil, exec.ErrElementSegmentDoesNotFit
		}
		offsets[i] = int(offset)
	}
	return offsets, nil
}

func (m *allocatedModule) evaluateElementSegments(offsets []int) {
	for i, element := range m.elements {
		offset, entries := offsets[i], m.table0.Entries()
		for j, funcIndex := range element.Elems {
			entries[offset+j], _ = m.getFunction(funcIndex)
		}
	}
}

func (m *allocatedModule) checkDataSegments() ([]int, error) {
	offsets := make([]int, len(m.data))
	for i, data := range m.data {
		offsetV, err := exec.EvalConstantExpression(m.importedGlobals, data.Offset)
		if err != nil {
			return nil, err
		}
		offset, ok := offsetV.(int32)
		if !ok {
			return nil, exec.InvalidValueTypeInitExprError{Wanted: reflect.Int32, Got: reflect.ValueOf(offsetV).Kind()}
		}

		if data.Index != 0 || m.mem0 == nil {
			return nil, exec.InvalidTableIndexError(data.Index)
		}

		bytes := m.mem0.Bytes()
		if offset < 0 || offset > int32(len(bytes)) || len(bytes[int(offset):]) < len(data.Data) {
			return nil, exec.ErrDataSegmentDoesNotFit
		}
		offsets[i] = int(offset)
	}
	return offsets, nil
}

func (m *allocatedModule) evaluateDataSegments(offsets []int) {
	for i, data := range m.data {
		offset, bytes := offsets[i], m.mem0.Bytes()
		copy(bytes[offset:], data.Data)
	}
}
