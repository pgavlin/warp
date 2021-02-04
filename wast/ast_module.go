package wast

import "github.com/pgavlin/warp/wasm"

type Module struct {
	Pos Pos

	Name     string
	Types    []*Typedef
	Funcs    []*Func
	Imports  []*Import
	Exports  []*Export
	Tables   []*Table
	Memories []*Memory
	Globals  []*Global
	Elems    []*Elem
	Data     []*Data
	Start    *Var
}

func (m *Module) ModuleName() string {
	return m.Name
}
func (m *Module) CommandPos() Pos {
	return m.Pos
}
func (*Module) isCommand() {}

type Typedef struct {
	Name    string
	Params  []*Param
	Results []wasm.ValueType
}

type Func struct {
	Name    string
	Exports []string
	Import  *InlineImport
	Type    *FuncType
	Locals  []*Local
	Instrs  []Instr
}

type InlineImport struct {
	Module string
	Name   string
}

type Import struct {
	Module   string
	Name     string
	External External
}

type Export struct {
	Name string
	Kind wasm.External
	Var  Var
}

type Table struct {
	Name    string
	Exports []string
	Import  *InlineImport
	Range   *Range
	Values  []Var
}

type Memory struct {
	Name    string
	Exports []string
	Import  *InlineImport
	Range   *Range
	Data    []string
}

type Global struct {
	Name    string
	Exports []string
	Import  *InlineImport
	Type    GlobalType
	Init    []Instr
}

type Elem struct {
	Var    *Var
	Offset []Instr
	Values []Var
}

type Data struct {
	Var    *Var
	Offset []Instr
	Values []string
}

type Local struct {
	Name string
	Type wasm.ValueType
}

type Var struct {
	Name  string
	Index uint32
}

type Range struct {
	Min uint32
	Max *uint32
}

type FuncType struct {
	Var     *Var
	Params  []*Param
	Results []wasm.ValueType
}

type Param struct {
	Name string
	Type wasm.ValueType
}

type GlobalType struct {
	Mutable bool
	Type    wasm.ValueType
}

type External interface {
	isExternal()
}

type ExternalFunc struct {
	Name string
	Type *FuncType
}

func (*ExternalFunc) isExternal() {}

type ExternalGlobal struct {
	Name string
	Type GlobalType
}

func (*ExternalGlobal) isExternal() {}

type ExternalTable struct {
	Name  string
	Range Range
}

func (*ExternalTable) isExternal() {}

type ExternalMemory struct {
	Name  string
	Range Range
}

func (*ExternalMemory) isExternal() {}

type Instr interface {
	isInstr()
}

type Block struct {
	Name   string
	Type   *FuncType
	Instrs []Instr
}

func (*Block) isInstr() {}

type Loop struct {
	Name   string
	Type   *FuncType
	Instrs []Instr
}

func (*Loop) isInstr() {}

type If struct {
	Name      string
	Type      *FuncType
	Condition []Instr
	Then      []Instr
	Else      []Instr
}

func (*If) isInstr() {}

type Op struct {
	Code TokenKind
}

func (*Op) isInstr() {}

type VarOp struct {
	Code TokenKind
	Vars []Var
}

func (*VarOp) isInstr() {}

type CallIndirect struct {
	Type FuncType
}

func (*CallIndirect) isInstr() {}

type MemOp struct {
	Code   TokenKind
	Offset *int64
	Align  *int64
}

func (*MemOp) isInstr() {}

type ConstOp struct {
	Code  TokenKind
	Value interface{}
}

func (*ConstOp) isInstr() {}
