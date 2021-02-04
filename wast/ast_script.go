package wast

import (
	"strings"

	"github.com/pgavlin/warp/wasm"
)

type Command interface {
	CommandPos() Pos

	isCommand()
}

type Script struct {
	Commands []Command
}

type ModuleCommand interface {
	Command

	Decode() (*wasm.Module, error)
	ModuleName() string
}

type ModuleLiteral struct {
	Pos Pos

	Name     string
	IsBinary bool
	Data     string
}

func (m *ModuleLiteral) Decode() (*wasm.Module, error) {
	if m.IsBinary {
		return wasm.DecodeModule(strings.NewReader(m.Data))
	}

	tm, err := ParseModule(NewScanner(strings.NewReader(m.Data)))
	if err != nil {
		return nil, err
	}
	return tm.Decode()
}

func (m *ModuleLiteral) ModuleName() string {
	return m.Name
}

func (m *ModuleLiteral) CommandPos() Pos {
	return m.Pos
}

func (*ModuleLiteral) isCommand() {}

type Register struct {
	Pos Pos

	Export string
	Name   string
}

func (r *Register) CommandPos() Pos {
	return r.Pos
}

func (*Register) isCommand() {}

type Action interface {
	Command
	isAction()
}

type Invoke struct {
	Pos Pos

	Name   string
	Export string
	Args   []interface{}
}

func (i *Invoke) CommandPos() Pos {
	return i.Pos
}

func (*Invoke) isCommand() {}
func (*Invoke) isAction()  {}

type Get struct {
	Pos Pos

	Name   string
	Export string
}

func (g *Get) CommandPos() Pos {
	return g.Pos
}

func (*Get) isCommand() {}
func (*Get) isAction()  {}

type AssertReturn struct {
	Pos Pos

	Action  Action
	Results []interface{}
}

func (a *AssertReturn) CommandPos() Pos {
	return a.Pos
}

func (*AssertReturn) isCommand() {}

type AssertTrap struct {
	Pos Pos

	Command Command
	Failure string
}

func (a *AssertTrap) CommandPos() Pos {
	return a.Pos
}

func (*AssertTrap) isCommand() {}

type AssertExhaustion struct {
	Pos Pos

	Action  Action
	Failure string
}

func (a *AssertExhaustion) CommandPos() Pos {
	return a.Pos
}

func (*AssertExhaustion) isCommand() {}

type ModuleAssertion struct {
	Pos Pos

	Kind    TokenKind
	Module  ModuleCommand
	Failure string
}

func (m *ModuleAssertion) CommandPos() Pos {
	return m.Pos
}

func (*ModuleAssertion) isCommand() {}

type ScriptCommand struct {
	Pos Pos

	Name   string
	Script *Script
}

func (s *ScriptCommand) CommandPos() Pos {
	return s.Pos
}

func (*ScriptCommand) isCommand() {}

type Input struct {
	Pos Pos

	Name string
	Path string
}

func (i *Input) CommandPos() Pos {
	return i.Pos
}

func (*Input) isCommand() {}

type Output struct {
	Pos Pos

	Name string
	Path string
}

func (o *Output) CommandPos() Pos {
	return o.Pos
}

func (*Output) isCommand() {}
