package exec

import "fmt"

var ErrModuleNotFound = fmt.Errorf("module not found")

// A ModuleResolver resolves module names to module definitions.
type ModuleResolver interface {
	// ResolveModule resolves the given module name to a module definition.
	ResolveModule(name string) (ModuleDefinition, error)
}

// A MapResolver is a ModuleResolver that maps module names to definitions using the contents of a map.
type MapResolver map[string]ModuleDefinition

// ResolveModule resolves the given module name to a module definition.
func (r MapResolver) ResolveModule(name string) (ModuleDefinition, error) {
	def, ok := r[name]
	if !ok {
		return nil, ErrModuleNotFound
	}
	return def, nil
}
