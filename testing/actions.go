package testing

import (
	"errors"

	"github.com/pgavlin/warp/exec"
	"github.com/pgavlin/warp/wasm/validate"
	"github.com/pgavlin/warp/wast"
)

type Action interface {
	Pos() wast.Pos
	Run(e *Environment) ([]interface{}, error)
}

func posOrDefault(p *wast.Pos) wast.Pos {
	if p == nil {
		return wast.Pos{}
	}
	return *p
}

func Pos(line, column int) *wast.Pos {
	return &wast.Pos{Line: line, Column: column}
}

type decodeAndInstantiate struct {
	wast.ModuleCommand
}

func (i *decodeAndInstantiate) Pos() wast.Pos {
	return i.CommandPos()
}

func (i *decodeAndInstantiate) decode(e *Environment) (exec.ModuleDefinition, error) {
	m, err := i.Decode()
	if err != nil {
		return nil, err
	}

	if err := validate.ValidateModule(m, true); err != nil {
		return nil, err
	}

	if e.loader == nil {
		return nil, errors.New("module: no loader")
	}
	return e.loader(m)
}

func (i *decodeAndInstantiate) Run(e *Environment) ([]interface{}, error) {
	def, err := i.decode(e)
	if err != nil {
		return nil, err
	}
	return nil, e.instantiateModule(i.ModuleName(), def)
}

type instantiate struct {
	pos  wast.Pos
	name string
	def  exec.ModuleDefinition
}

func (i *instantiate) Pos() wast.Pos {
	return i.pos
}

func (i *instantiate) Run(e *Environment) ([]interface{}, error) {
	return nil, e.instantiateModule(i.name, i.def)
}

func InstantiateModule(pos *wast.Pos, name string, def exec.ModuleDefinition) Action {
	return &instantiate{pos: posOrDefault(pos), name: name, def: def}
}

type invoke struct {
	wast.Invoke
}

func (i *invoke) Pos() wast.Pos {
	return i.Invoke.Pos
}

func (i *invoke) Run(e *Environment) ([]interface{}, error) {
	m, ok := e.modules[i.Name]
	if !ok {
		return nil, errors.New("unknown module")
	}
	f, err := m.GetFunction(i.Export)
	if err != nil {
		return nil, err
	}
	thread := exec.NewThread(300)
	return f.Call(&thread, i.Args...), nil
}

func Invoke(pos *wast.Pos, module, export string, args ...interface{}) Action {
	return &invoke{Invoke: wast.Invoke{Pos: posOrDefault(pos), Name: module, Export: export, Args: args}}
}

type get struct {
	wast.Get
}

func (ga *get) Pos() wast.Pos {
	return ga.Get.Pos
}

func (ga *get) Run(e *Environment) ([]interface{}, error) {
	m, ok := e.modules[ga.Name]
	if !ok {
		return nil, errors.New("unknown module")
	}
	g, err := m.GetGlobal(ga.Export)
	if err != nil {
		return nil, err
	}
	return []interface{}{g.GetValue()}, nil
}

func Get(pos *wast.Pos, module, export string) Action {
	return &get{Get: wast.Get{Pos: posOrDefault(pos), Name: module, Export: export}}
}
