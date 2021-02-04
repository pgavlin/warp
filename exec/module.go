package exec

import (
	"errors"
	"fmt"

	"github.com/pgavlin/warp/wasm"
)

// ErrDataSegmentDoesNotFit should be returned by Instantiate if a data segment attempts to write outside of
// its target memory's bounds.
var ErrDataSegmentDoesNotFit = errors.New("data segment does not fit")

// ErrElementSegmentDoesNotFit should be returned by Instantiate if a element segment attempts to write outside
// of its target table's bounds.
var ErrElementSegmentDoesNotFit = errors.New("element segment does not fit")

type InvalidTableIndexError uint32

func (e InvalidTableIndexError) Error() string {
	return fmt.Sprintf("wasm: Invalid table to table index space: %d", uint32(e))
}

// An ExportNotFoundError is returned by InstantiateModule if an export could not be found.
type ExportNotFoundError struct {
	ModuleName string
	FieldName  string
}

type KindMismatchError struct {
	ModuleName string
	FieldName  string
	Import     wasm.External
	Export     wasm.External
}

func (e *KindMismatchError) Error() string {
	return fmt.Sprintf("wasm: mismatching import and export external kind values for %s.%s (%v, %v)", e.FieldName, e.ModuleName, e.Import, e.Export)
}

func (e *ExportNotFoundError) Error() string {
	return fmt.Sprintf("wasm: couldn't find export with name %s in module %s", e.FieldName, e.ModuleName)
}

// An ImportResolver resolves import entries to function, memory, table, and global instances.
type ImportResolver interface {
	ResolveFunction(moduleName, functionName string, type_ wasm.FunctionSig) (Function, error)
	ResolveMemory(moduleName, memoryName string, type_ wasm.Memory) (*Memory, error)
	ResolveTable(moduleName, tableName string, type_ wasm.Table) (*Table, error)
	ResolveGlobal(moduleName, globalName string, type_ wasm.GlobalVar) (*Global, error)
}

// A ModuleEventHandler responds to module allocations and instantiations.
type ModuleEventHandler interface {
	ModuleAllocated(m AllocatedModule) error
	ModuleInstantiated(m Module) error
}

// ModuleDefinition represents a WASM module definition.
type ModuleDefinition interface {
	// Allocate creates an allocated, uninitialized module with the given name from this module definition.
	Allocate(name string) (AllocatedModule, error)
}

// NewKindMismatchError creates a new error that reports a mismatch between an import and export kind. This function
// should be used to create the errors returned by Module.Get{Function,Table,Memory,Global} if the requested name
// refers to an export of a different kind.
func NewKindMismatchError(exportingModuleName, exportName string, importKind, exportKind wasm.External) error {
	return &KindMismatchError{
		FieldName:  exportName,
		ModuleName: exportingModuleName,
		Import:     importKind,
		Export:     exportKind,
	}
}

// An AllocatedModule is an allocated but uninitialized WASM module.
type AllocatedModule interface {
	Module

	// Instantiate initializes the allocated module with imports supplied by the given resolver.
	Instantiate(imports ImportResolver) (Module, error)
}

// A Module is an instantiated WASM module.
type Module interface {
	// Name returns the name of this module.
	Name() string
	// GetFunction returns the exported function with the given name. If the function does not exist or the name
	// refers to an export of a different kind, this function returns an error.
	GetFunction(name string) (Function, error)
	// GetTable returns the exported table with the given name. If the table does not exist or the name
	// refers to an export of a different kind, this function returns an error.
	GetTable(name string) (*Table, error)
	// GetMemory returns the exported memory with the given name. If the memory does not exist or the name
	// refers to an export of a different kind, this function returns an error.
	GetMemory(name string) (*Memory, error)
	// GetGlobal returns the exported global with the given name. If the global does not exist or the name
	// refers to an export of a different kind, this function returns an error.
	GetGlobal(name string) (*Global, error)
}
