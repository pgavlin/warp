package exec

import (
	"errors"
	"fmt"

	"github.com/pgavlin/warp/wasm"
)

// ErrInvalidTypeIndex is returned by InstantiateModule if the module's imports contain an invalid type index.
var ErrInvalidTypeIndex = fmt.Errorf("invalid type index")

// InvalidImportError is returned when the export of a resolved module doesn't
// match the signature of its import declaration.
type InvalidImportError struct {
	ModuleName string
	FieldName  string
	TypeIndex  uint32
}

func (e *InvalidImportError) Error() string {
	return fmt.Sprintf("wasm: invalid signature for import %#x with name '%s' in module %s", e.TypeIndex, e.FieldName, e.ModuleName)
}

var ErrTableType = errors.New("table type mismatch")
var ErrMemoryType = errors.New("memory type mismatch")
var ErrGlobalType = errors.New("global type mismatch")

// A Store is responsible for instantiating modules.
type Store struct {
	resolver ModuleResolver
	handlers []ModuleEventHandler
	modules  map[string]Module
}

// NewStore creates a new store that will use the given resolver to resolve modules.
func NewStore(resolver ModuleResolver, handlers ...ModuleEventHandler) *Store {
	return &Store{
		resolver: resolver,
		handlers: handlers,
		modules:  map[string]Module{},
	}
}

func (s *Store) allocateModule(def ModuleDefinition, name string) (AllocatedModule, error) {
	a, err := def.Allocate(name)
	if err != nil {
		return nil, err
	}
	for _, h := range s.handlers {
		if err = h.ModuleAllocated(a); err != nil {
			return nil, err
		}
	}
	return a, nil
}

func (s *Store) instantiateModule(a AllocatedModule, resolver *resolver) (Module, error) {
	m, err := a.Instantiate(resolver)
	if err != nil {
		return nil, err
	}
	for _, h := range s.handlers {
		if err = h.ModuleInstantiated(m); err != nil {
			return nil, err
		}
	}
	return m, nil
}

// InstantiateModule instantiates the given module. The name is resolved to a module definition using the store's ModuleResolver.
func (s *Store) InstantiateModule(name string) (Module, error) {
	if m, ok := s.modules[name]; ok {
		return m, nil
	}

	definition, err := s.resolver.ResolveModule(name)
	if err != nil {
		return nil, err
	}
	return s.InstantiateModuleDefinition(name, definition)
}

// RegisterModule registers an instantiated module with the store, replacing any existing module with the same name.
func (s *Store) RegisterModule(name string, module Module) {
	s.modules[name] = module
}

// InstantiateModuleDefinition instantiates the given module definition.
func (s *Store) InstantiateModuleDefinition(name string, def ModuleDefinition) (Module, error) {
	a, err := s.allocateModule(def, name)
	if err != nil {
		return nil, err
	}

	m, err := s.instantiateModule(a, newResolver(s, a))
	if err != nil {
		return nil, err
	}
	s.modules[name] = m
	return m, nil
}

type resolver struct {
	s *Store

	allocated map[string]AllocatedModule
}

func newResolver(s *Store, m AllocatedModule) *resolver {
	return &resolver{
		s:         s,
		allocated: map[string]AllocatedModule{m.Name(): m},
	}
}

func (r *resolver) instantiateModule(moduleName string) (Module, error) {
	if m, ok := r.s.modules[moduleName]; ok {
		return m, nil
	}
	if m, ok := r.allocated[moduleName]; ok {
		return Module(m), nil
	}

	def, err := r.s.resolver.ResolveModule(moduleName)
	if err != nil {
		return nil, err
	}

	a, err := r.s.allocateModule(def, moduleName)
	if err != nil {
		return nil, err
	}
	r.allocated[moduleName] = a

	m, err := r.s.instantiateModule(a, r)
	if err != nil {
		return nil, err
	}
	r.s.modules[moduleName] = m

	return m, nil
}

func (r *resolver) ResolveFunction(moduleName, functionName string, type_ wasm.FunctionSig) (Function, error) {
	m, err := r.instantiateModule(moduleName)
	if err != nil {
		return nil, err
	}
	f, err := m.GetFunction(functionName)
	if err != nil {
		return nil, err
	}
	if !f.GetSignature().Equals(type_) {
		return nil, &InvalidImportError{
			ModuleName: moduleName,
			FieldName:  functionName,
		}
	}
	return f, nil
}

func (r *resolver) ResolveMemory(moduleName, memoryName string, type_ wasm.Memory) (*Memory, error) {
	m, err := r.instantiateModule(moduleName)
	if err != nil {
		return nil, err
	}
	memory, err := m.GetMemory(memoryName)
	if err != nil {
		return nil, err
	}
	if !limitsMatch(memory.min, memory.max, type_.Limits) {
		return nil, ErrMemoryType
	}
	return memory, nil
}

func (r *resolver) ResolveTable(moduleName, tableName string, type_ wasm.Table) (*Table, error) {
	m, err := r.instantiateModule(moduleName)
	if err != nil {
		return nil, err
	}
	table, err := m.GetTable(tableName)
	if err != nil {
		return nil, err
	}
	if !limitsMatch(table.min, table.max, type_.Limits) {
		return nil, ErrMemoryType
	}
	return table, nil
}

func (r *resolver) ResolveGlobal(moduleName, globalName string, type_ wasm.GlobalVar) (*Global, error) {
	m, err := r.instantiateModule(moduleName)
	if err != nil {
		return nil, err
	}
	global, err := m.GetGlobal(globalName)
	if err != nil {
		return nil, err
	}
	if global.Type() != type_ {
		return nil, ErrGlobalType
	}
	return global, nil
}

func limitsMatch(min, max uint32, expected wasm.ResizableLimits) bool {
	return min >= expected.Initial && (expected.Flags == 0 || max <= expected.Maximum)
}
