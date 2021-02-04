package interpreter

import (
	"github.com/pgavlin/warp/exec"
	"github.com/pgavlin/warp/wasm"
	"github.com/pgavlin/warp/wasm/code"
)

// A module holds an instance of a WASM module.
type module struct {
	name string // The name of the module.

	types     []wasm.FunctionSig // The types used by this module.
	functions []function         // The function table for this module.
	mem0      *exec.Memory       // The first memory for this module.
	table0    *exec.Table        // The first for this module.
	globals   []exec.Global      // The globals defined by this module.

	importedFunctions []exec.Function // The functions imported by this module.
	importedGlobals   []*exec.Global  // The globals imported by this module.

	exports map[string]interface{} // The module's exports.
}

func (m *module) blockType(instr *code.Instruction) (ins []wasm.ValueType, outs []wasm.ValueType) {
	blockType := instr.Immediate & code.BlockTypeMask
	switch blockType {
	case code.BlockTypeEmpty:
		return nil, nil
	case code.BlockTypeI32, code.BlockTypeI64, code.BlockTypeF32, code.BlockTypeF64:
		return nil, []wasm.ValueType{wasm.ValueType(blockType)}
	default:
		t := &m.types[int(blockType)]
		return t.ParamTypes, t.ReturnTypes
	}
}

func (m *module) blockArity(instr *code.Instruction, isLoop bool) int {
	ins, outs := m.blockType(instr)
	if isLoop {
		return len(ins)
	}
	return len(outs)
}

func (m *module) getFunction(index uint32) (exec.Function, bool) {
	if index < uint32(len(m.importedFunctions)) {
		return m.importedFunctions[int(index)], true
	}
	index -= uint32(len(m.importedFunctions))
	if index >= uint32(len(m.functions)) {
		return nil, false
	}
	return &m.functions[int(index)], true
}

func (m *module) getGlobal(index uint32) (*exec.Global, bool) {
	if index < uint32(len(m.importedGlobals)) {
		return m.importedGlobals[int(index)], true
	}
	index -= uint32(len(m.importedGlobals))
	if index >= uint32(len(m.globals)) {
		return nil, false
	}
	return &m.globals[int(index)], true
}

func (m *module) Name() string {
	return m.name
}

func (m *module) newExportError(name string, importKind wasm.External, export interface{}) error {
	if export == nil {
		return &exec.ExportNotFoundError{ModuleName: m.name, FieldName: name}
	}

	var exportKind wasm.External
	switch export.(type) {
	case *function:
		exportKind = wasm.ExternalFunction
	case *exec.Table:
		exportKind = wasm.ExternalTable
	case *exec.Memory:
		exportKind = wasm.ExternalMemory
	case *exec.Global:
		exportKind = wasm.ExternalGlobal
	default:
		panic("unreachable")
	}
	return exec.NewKindMismatchError(m.name, name, importKind, exportKind)
}

func (m *module) GetFunction(name string) (exec.Function, error) {
	export := m.exports[name]
	if function, ok := export.(exec.Function); ok {
		return function, nil
	}
	return nil, m.newExportError(name, wasm.ExternalFunction, export)
}

func (m *module) GetTable(name string) (*exec.Table, error) {
	export := m.exports[name]
	if table, ok := export.(*exec.Table); ok {
		return table, nil
	}
	return nil, m.newExportError(name, wasm.ExternalFunction, export)
}

func (m *module) GetMemory(name string) (*exec.Memory, error) {
	export := m.exports[name]
	if memory, ok := export.(*exec.Memory); ok {
		return memory, nil
	}
	return nil, m.newExportError(name, wasm.ExternalFunction, export)
}

func (m *module) GetGlobal(name string) (*exec.Global, error) {
	export := m.exports[name]
	if global, ok := export.(*exec.Global); ok {
		return global, nil
	}
	return nil, m.newExportError(name, wasm.ExternalFunction, export)
}
