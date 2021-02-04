package load

import (
	"io/fs"

	"github.com/pgavlin/warp/exec"
	"github.com/pgavlin/warp/interpreter"
	"github.com/pgavlin/warp/wasm"
)

type ModuleDefinitionFunc func(m *wasm.Module) (exec.ModuleDefinition, error)

func Intepret(m *wasm.Module) (exec.ModuleDefinition, error) {
	return interpreter.NewModuleDefinition(m), nil
}

type FSResolver struct {
	fs             fs.FS
	definitionFunc ModuleDefinitionFunc
}

func NewFSResolver(fs fs.FS, definitionFunc ModuleDefinitionFunc) *FSResolver {
	return &FSResolver{fs: fs, definitionFunc: definitionFunc}
}

func (r *FSResolver) loadModule(name string) (*wasm.Module, error) {
	extensions := []string{".wasm", ".wast", ".wat", ""}
	for _, ext := range extensions {
		if f, err := r.fs.Open(name + ext); err == nil {
			defer f.Close()
			return LoadModule(f)
		}
	}
	return nil, exec.ErrModuleNotFound
}

func (r *FSResolver) ResolveModule(name string) (exec.ModuleDefinition, error) {
	m, err := r.loadModule(name)
	if err != nil {
		return nil, err
	}
	return r.definitionFunc(m)
}
