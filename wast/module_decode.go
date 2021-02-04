package wast

import (
	"bytes"
	"errors"
	"fmt"
	"math/bits"
	"strings"

	"github.com/pgavlin/warp/wasm"
	"github.com/pgavlin/warp/wasm/code"
)

func (m *Module) Decode() (*wasm.Module, error) {
	decoder := moduleDecoder{m: m}
	return decoder.decodeModule()
}

type indexes struct {
	functionTypes map[string]int

	types     []*FuncType
	functions []int
	tables    int
	memories  int
	globals   int
}

func valueTypeKey(t wasm.ValueType) rune {
	switch t {
	case wasm.ValueTypeI32:
		return 'i'
	case wasm.ValueTypeI64:
		return 'I'
	case wasm.ValueTypeF32:
		return 'f'
	case wasm.ValueTypeF64:
		return 'F'
	default:
		panic("unreachable")
	}
}

func functionTypeKey(params []*Param, results []wasm.ValueType) string {
	var b strings.Builder
	b.WriteRune('p')
	for _, p := range params {
		b.WriteRune(valueTypeKey(p.Type))
	}
	b.WriteRune('r')
	for _, t := range results {
		b.WriteRune(valueTypeKey(t))
	}
	return b.String()
}

func (i *indexes) functionType(params []*Param, results []wasm.ValueType) int {
	k := functionTypeKey(params, results)
	if typeidx, ok := i.functionTypes[k]; ok {
		return typeidx
	}
	return i.defType(&Typedef{Params: params, Results: results})
}

func (i *indexes) defType(type_ *Typedef) int {
	i.types = append(i.types, &FuncType{Params: type_.Params, Results: type_.Results})
	typeidx := len(i.types) - 1

	k := functionTypeKey(type_.Params, type_.Results)
	if _, ok := i.functionTypes[k]; !ok {
		i.functionTypes[k] = typeidx
	}
	return typeidx
}

func (i *indexes) defFunction(typeidx int) int {
	i.functions = append(i.functions, typeidx)
	return len(i.functions) - 1
}

func (i *indexes) defTable() int {
	i.tables++
	return i.tables - 1
}

func (i *indexes) defMemory() int {
	i.memories++
	return i.memories - 1
}

func (i *indexes) defGlobal() int {
	i.globals++
	return i.globals - 1
}

type names struct {
	types     map[string]int
	functions map[string]int
	tables    map[string]int
	memories  map[string]int
	globals   map[string]int
}

type context struct {
	*names

	indexes *indexes
	parent  *context

	locals map[string]int
	labels map[string]int
}

func (c *context) push() *context {
	var idx *indexes
	var nm *names
	if c == nil {
		idx = &indexes{functionTypes: map[string]int{}}
		nm = &names{
			types:     map[string]int{},
			functions: map[string]int{},
			tables:    map[string]int{},
			memories:  map[string]int{},
			globals:   map[string]int{},
		}
	} else {
		idx, nm = c.indexes, c.names
	}

	return &context{
		parent:  c,
		indexes: idx,
		names:   nm,
		locals:  map[string]int{},
		labels:  map[string]int{},
	}
}

func (c *context) pop() *context {
	return c.parent
}

func (c *context) functionType(type_ *FuncType) int {
	if type_.Var == nil {
		return c.indexes.functionType(type_.Params, type_.Results)
	}
	return c.useType(*type_.Var)
}

func (c *context) defType(name string, type_ *Typedef) {
	index := c.indexes.defType(type_)
	if name != "" {
		c.types[name] = index
	}
}

func (c *context) defFunction(name string, type_ *FuncType) {
	index := c.indexes.defFunction(c.functionType(type_))
	if name != "" {
		c.functions[name] = index
	}
}

func (c *context) defTable(name string) {
	index := c.indexes.defTable()
	if name != "" {
		c.tables[name] = index
	}
}

func (c *context) defMemory(name string) {
	index := c.indexes.defMemory()
	if name != "" {
		c.memories[name] = index
	}
}

func (c *context) defGlobal(name string) {
	index := c.indexes.defGlobal()
	if name != "" {
		c.globals[name] = index
	}
}

func (c *context) defLocal(name string, index int) {
	if name != "" {
		c.locals[name] = index
	}
}

func (c *context) defLabel(name string, depth int) {
	if name != "" {
		c.labels[name] = depth
	}
}

func (c *context) getType(v Var) *FuncType {
	if v.Name == "" {
		return c.indexes.types[int(v.Index)]
	}

	index, ok := c.types[v.Name]
	if !ok {
		index, ok = c.functions[v.Name]
		if !ok {
			panic("unknown type")
		}
	}
	return c.indexes.types[index]
}

func (c *context) useType(v Var) int {
	if v.Name == "" {
		return int(v.Index)
	}
	if index, ok := c.types[v.Name]; ok {
		return index
	}
	if c.parent != nil {
		return c.parent.useType(v)
	}
	panic("unknown type")
}

func (c *context) useFunction(v Var) int {
	if v.Name == "" {
		return int(v.Index)
	}
	if index, ok := c.functions[v.Name]; ok {
		return index
	}
	if c.parent != nil {
		return c.parent.useFunction(v)
	}
	panic("unknown function")
}

func (c *context) useTable(v Var) int {
	if v.Name == "" {
		return int(v.Index)
	}
	if index, ok := c.tables[v.Name]; ok {
		return index
	}
	if c.parent != nil {
		return c.parent.useTable(v)
	}
	panic("unknown table")
}

func (c *context) useMemory(v Var) int {
	if v.Name == "" {
		return int(v.Index)
	}
	if index, ok := c.memories[v.Name]; ok {
		return index
	}
	if c.parent != nil {
		return c.parent.useMemory(v)
	}
	panic("unknown memory")
}

func (c *context) useGlobal(v Var) int {
	if v.Name == "" {
		return int(v.Index)
	}
	if index, ok := c.globals[v.Name]; ok {
		return index
	}
	if c.parent != nil {
		return c.parent.useGlobal(v)
	}
	panic("unknown global")
}

func (c *context) useLocal(v Var) int {
	if v.Name == "" {
		return int(v.Index)
	}
	if index, ok := c.locals[v.Name]; ok {
		return index
	}
	if c.parent != nil {
		return c.parent.useLocal(v)
	}
	panic("unknown local")
}

func (c *context) useLabel(v Var) int {
	if v.Name == "" {
		return int(v.Index)
	}
	if index, ok := c.labels[v.Name]; ok {
		return index
	}
	if c.parent != nil {
		return c.parent.useLabel(v)
	}
	panic("unknown label")
}

type moduleDecoder struct {
	m *Module

	context *context
	depth   int

	imports               int
	functionImports       int
	inlineFunctionImports int
	functionBodies        int
	tableImports          int
	inlineTableImports    int
	definedTables         int
	memoryImports         int
	inlineMemoryImports   int
	definedMemories       int
	globalImports         int
	inlineGlobalImports   int
	definedGlobals        int
}

func (b *moduleDecoder) pushModuleNames() {
	b.context = b.context.push()

	for _, item := range b.m.Types {
		b.context.defType(item.Name, item)
	}

	for _, item := range b.m.Imports {
		switch external := item.External.(type) {
		case *ExternalFunc:
			b.context.defFunction(external.Name, external.Type)
			b.functionImports++
		case *ExternalTable:
			b.context.defTable(external.Name)
			b.tableImports++
		case *ExternalMemory:
			b.context.defMemory(external.Name)
			b.memoryImports++
		case *ExternalGlobal:
			b.context.defGlobal(external.Name)
			b.globalImports++
		}
		b.imports++
	}
	for _, item := range b.m.Funcs {
		if item.Import != nil {
			b.context.defFunction(item.Name, item.Type)
			b.inlineFunctionImports++
			b.imports++
		}
	}
	for _, item := range b.m.Tables {
		if item.Import != nil {
			b.context.defTable(item.Name)
			b.inlineTableImports++
			b.imports++
		}
	}
	for _, item := range b.m.Memories {
		if item.Import != nil {
			b.context.defMemory(item.Name)
			b.inlineMemoryImports++
			b.imports++
		}
	}
	for _, item := range b.m.Globals {
		if item.Import != nil {
			b.context.defGlobal(item.Name)
			b.inlineGlobalImports++
			b.imports++
		}
	}

	for _, item := range b.m.Funcs {
		if item.Import == nil {
			b.context.defFunction(item.Name, item.Type)
			b.functionBodies++
		}
	}
	for _, item := range b.m.Tables {
		if item.Import == nil {
			b.context.defTable(item.Name)
			b.definedTables++
		}
	}
	for _, item := range b.m.Memories {
		if item.Import == nil {
			b.context.defMemory(item.Name)
			b.definedMemories++
		}
	}
	for _, item := range b.m.Globals {
		if item.Import == nil {
			b.context.defGlobal(item.Name)
			b.definedGlobals++
		}
	}
}

func (b *moduleDecoder) pushTypeNames(typ *Typedef) {
	b.context = b.context.push()
	b.defParamNames(typ.Params)
}

func (b *moduleDecoder) pushFuncNames(fn *Func) {
	b.context = b.context.push()

	arity := 0
	if fn.Type.Var != nil {
		typ := b.context.getType(*fn.Type.Var)
		b.defParamNames(typ.Params)
		arity = len(typ.Params)
	} else {
		arity = len(fn.Type.Params)
	}

	b.defParamNames(fn.Type.Params)

	for i, l := range fn.Locals {
		b.context.defLocal(l.Name, arity+i)
	}
}

func (b *moduleDecoder) pushBlock(name string, type_ *FuncType) {
	b.context = b.context.push()
	b.context.labels[name] = b.depth
	if type_ != nil {
		b.defParamNames(type_.Params)
	}
	b.depth++
}

func (b *moduleDecoder) pop() {
	b.context = b.context.pop()
}

func (b *moduleDecoder) popBlock() {
	b.pop()
	b.depth--
}

func (b *moduleDecoder) useLabel(v Var) int {
	if v.Name == "" {
		return int(v.Index)
	}
	return b.depth - b.context.useLabel(v) - 1
}

func (b *moduleDecoder) defParamNames(params []*Param) {
	for i, p := range params {
		b.context.defLocal(p.Name, i)
	}
}

func (b *moduleDecoder) decodeModule() (module *wasm.Module, err error) {
	defer func() {
		if x := recover(); x != nil {
			e := x.(error)
			if e == nil {
				panic(x)
			}
			err = e
		}
	}()

	b.pushModuleNames()

	import_, err := b.decodeImports()
	if err != nil {
		return nil, err
	}
	function, code, err := b.decodeFuncs()
	if err != nil {
		return nil, err
	}
	table, err := b.decodeTables()
	if err != nil {
		return nil, err
	}
	memory, err := b.decodeMemories()
	if err != nil {
		return nil, err
	}
	global, err := b.decodeGlobals()
	if err != nil {
		return nil, err
	}
	export, err := b.decodeExports()
	if err != nil {
		return nil, err
	}
	start, err := b.decodeStart()
	if err != nil {
		return nil, err
	}
	elements, err := b.decodeElems()
	if err != nil {
		return nil, err
	}
	data, err := b.decodeData()
	if err != nil {
		return nil, err
	}
	types, err := b.decodeTypes()
	if err != nil {
		return nil, err
	}

	return &wasm.Module{
		Types:    types,
		Import:   import_,
		Function: function,
		Table:    table,
		Memory:   memory,
		Global:   global,
		Export:   export,
		Start:    start,
		Elements: elements,
		Code:     code,
		Data:     data,
	}, nil
}

func (b *moduleDecoder) decodeTypes() (*wasm.SectionTypes, error) {
	section := wasm.SectionTypes{
		Entries: make([]wasm.FunctionSig, len(b.context.indexes.types)),
	}
	for i, type_ := range b.context.indexes.types {
		section.Entries[i] = b.decodeFunctionSig(type_.Params, type_.Results)
	}
	return &section, nil
}

func (b *moduleDecoder) decodeImports() (*wasm.SectionImports, error) {
	section := wasm.SectionImports{
		Entries: make([]wasm.ImportEntry, len(b.m.Imports), b.imports),
	}
	for i, import_ := range b.m.Imports {
		var type_ wasm.Import
		switch external := import_.External.(type) {
		case *ExternalFunc:
			type_ = wasm.FuncImport{Type: uint32(b.context.functionType(external.Type))}
		case *ExternalTable:
			type_ = wasm.TableImport{Type: b.decodeTableRange(external.Range)}
		case *ExternalMemory:
			type_ = wasm.MemoryImport{Type: b.decodeMemoryRange(external.Range)}
		case *ExternalGlobal:
			type_ = wasm.GlobalVarImport{Type: b.decodeGlobalType(external.Type)}
		}
		section.Entries[i] = wasm.ImportEntry{
			ModuleName: import_.Module,
			FieldName:  import_.Name,
			Type:       type_,
		}
	}

	for _, item := range b.m.Funcs {
		if item.Import != nil {
			section.Entries = append(section.Entries, wasm.ImportEntry{
				ModuleName: item.Import.Module,
				FieldName:  item.Import.Name,
				Type:       wasm.FuncImport{Type: uint32(b.context.functionType(item.Type))},
			})
		}
	}
	for _, item := range b.m.Tables {
		if item.Import != nil {
			section.Entries = append(section.Entries, wasm.ImportEntry{
				ModuleName: item.Import.Module,
				FieldName:  item.Import.Name,
				Type:       wasm.TableImport{Type: b.decodeTableType(item)},
			})
		}
	}
	for _, item := range b.m.Memories {
		if item.Import != nil {
			section.Entries = append(section.Entries, wasm.ImportEntry{
				ModuleName: item.Import.Module,
				FieldName:  item.Import.Name,
				Type:       wasm.MemoryImport{Type: b.decodeMemoryType(item)},
			})
		}
	}
	for _, item := range b.m.Globals {
		if item.Import != nil {
			section.Entries = append(section.Entries, wasm.ImportEntry{
				ModuleName: item.Import.Module,
				FieldName:  item.Import.Name,
				Type:       wasm.GlobalVarImport{Type: b.decodeGlobalType(item.Type)},
			})
		}
	}

	return &section, nil
}

func (b *moduleDecoder) decodeFuncs() (*wasm.SectionFunctions, *wasm.SectionCode, error) {
	functions := wasm.SectionFunctions{
		Types: make([]uint32, 0, b.functionBodies),
	}
	code := wasm.SectionCode{
		Bodies: make([]wasm.FunctionBody, 0, b.functionBodies),
	}
	for _, f := range b.m.Funcs {
		if f.Import == nil {
			functions.Types = append(functions.Types, uint32(b.context.functionType(f.Type)))

			body, err := b.decodeFunctionBody(f)
			if err != nil {
				return nil, nil, err
			}
			code.Bodies = append(code.Bodies, body)
		}
	}
	return &functions, &code, nil
}

func (b *moduleDecoder) decodeTables() (*wasm.SectionTables, error) {
	tables := wasm.SectionTables{
		Entries: make([]wasm.Table, 0, len(b.m.Tables)),
	}
	for _, t := range b.m.Tables {
		if t.Import == nil {
			tables.Entries = append(tables.Entries, b.decodeTableType(t))
		}
	}
	return &tables, nil
}

func (b *moduleDecoder) decodeMemories() (*wasm.SectionMemories, error) {
	memories := wasm.SectionMemories{
		Entries: make([]wasm.Memory, 0, len(b.m.Memories)),
	}
	for _, m := range b.m.Memories {
		if m.Import == nil {
			memories.Entries = append(memories.Entries, b.decodeMemoryType(m))
		}
	}
	return &memories, nil
}

func (b *moduleDecoder) decodeGlobals() (*wasm.SectionGlobals, error) {
	section := wasm.SectionGlobals{
		Globals: make([]wasm.GlobalEntry, 0, b.definedGlobals),
	}
	for _, global := range b.m.Globals {
		if global.Import == nil {
			init, err := b.decodeBytecode(global.Init, empty)
			if err != nil {
				return nil, err
			}

			section.Globals = append(section.Globals, wasm.GlobalEntry{
				Type: b.decodeGlobalType(global.Type),
				Init: init,
			})
		}
	}
	return &section, nil
}

func (b *moduleDecoder) decodeExports() (*wasm.SectionExports, error) {
	section := wasm.SectionExports{
		Entries: make([]wasm.ExportEntry, 0, len(b.m.Exports)+b.functionBodies+b.definedTables+b.definedMemories+b.definedGlobals),
	}
	for _, export := range b.m.Exports {
		var index int
		switch export.Kind {
		case wasm.ExternalFunction:
			index = b.context.useFunction(export.Var)
		case wasm.ExternalTable:
			index = b.context.useTable(export.Var)
		case wasm.ExternalMemory:
			index = b.context.useMemory(export.Var)
		case wasm.ExternalGlobal:
			index = b.context.useGlobal(export.Var)
		}
		section.Entries = append(section.Entries, wasm.ExportEntry{
			FieldStr: export.Name,
			Kind:     export.Kind,
			Index:    uint32(index),
		})
	}

	importidx := b.functionImports
	idx := b.functionImports + b.inlineFunctionImports

	for _, fn := range b.m.Funcs {
		var index int
		if fn.Import != nil {
			index, importidx = importidx, importidx+1
		} else {
			index, idx = idx, idx+1
		}

		for _, export := range fn.Exports {
			section.Entries = append(section.Entries, wasm.ExportEntry{
				FieldStr: export,
				Kind:     wasm.ExternalFunction,
				Index:    uint32(index),
			})
		}
	}

	importidx = b.tableImports
	idx = b.tableImports + b.inlineTableImports

	for _, table := range b.m.Tables {
		var index int
		if table.Import != nil {
			index, importidx = importidx, importidx+1
		} else {
			index, idx = idx, idx+1
		}

		for _, export := range table.Exports {
			section.Entries = append(section.Entries, wasm.ExportEntry{
				FieldStr: export,
				Kind:     wasm.ExternalTable,
				Index:    uint32(index),
			})
		}
	}

	importidx = b.memoryImports
	idx = b.memoryImports + b.inlineMemoryImports

	for _, memory := range b.m.Memories {
		var index int
		if memory.Import != nil {
			index, importidx = importidx, importidx+1
		} else {
			index, idx = idx, idx+1
		}

		for _, export := range memory.Exports {
			section.Entries = append(section.Entries, wasm.ExportEntry{
				FieldStr: export,
				Kind:     wasm.ExternalMemory,
				Index:    uint32(index),
			})
		}
	}

	importidx = b.globalImports
	idx = b.globalImports + b.inlineGlobalImports

	for _, global := range b.m.Globals {
		var index int
		if global.Import != nil {
			index, importidx = importidx, importidx+1
		} else {
			index, idx = idx, idx+1
		}

		for _, export := range global.Exports {
			section.Entries = append(section.Entries, wasm.ExportEntry{
				FieldStr: export,
				Kind:     wasm.ExternalGlobal,
				Index:    uint32(index),
			})
		}
	}

	return &section, nil
}

func (b *moduleDecoder) decodeStart() (*wasm.SectionStartFunction, error) {
	if b.m.Start == nil {
		return nil, nil
	}
	return &wasm.SectionStartFunction{Index: uint32(b.context.useFunction(*b.m.Start))}, nil
}

func (b *moduleDecoder) decodeElems() (*wasm.SectionElements, error) {
	section := wasm.SectionElements{
		Entries: make([]wasm.ElementSegment, len(b.m.Elems)),
	}
	for i, elem := range b.m.Elems {
		tableidx := 0
		if elem.Var != nil {
			tableidx = b.context.useTable(*elem.Var)
		}

		offset, err := b.decodeBytecode(elem.Offset, empty)
		if err != nil {
			return nil, err
		}

		elems := make([]uint32, len(elem.Values))
		for i, v := range elem.Values {
			elems[i] = uint32(b.context.useFunction(v))
		}

		section.Entries[i] = wasm.ElementSegment{
			Index:  uint32(tableidx),
			Offset: offset,
			Elems:  elems,
		}
	}

	importidx := b.tableImports
	idx := b.tableImports + b.inlineTableImports

	for _, table := range b.m.Tables {
		var index int
		if table.Import != nil {
			index, importidx = importidx, importidx+1
		} else {
			index, idx = idx, idx+1
		}

		if len(table.Values) != 0 {
			elems := make([]uint32, len(table.Values))
			for i, v := range table.Values {
				elems[i] = uint32(b.context.useFunction(v))
			}
			section.Entries = append(section.Entries, wasm.ElementSegment{
				Index:  uint32(index),
				Offset: zeroI32,
				Elems:  elems,
			})
		}
	}

	return &section, nil
}

func (b *moduleDecoder) decodeData() (*wasm.SectionData, error) {
	section := wasm.SectionData{
		Entries: make([]wasm.DataSegment, len(b.m.Data)),
	}
	for i, data := range b.m.Data {
		tableidx := 0
		if data.Var != nil {
			tableidx = b.context.useMemory(*data.Var)
		}

		offset, err := b.decodeBytecode(data.Offset, empty)
		if err != nil {
			return nil, err
		}

		var bytes []byte
		for _, v := range data.Values {
			bytes = append(bytes, []byte(v)...)
		}

		section.Entries[i] = wasm.DataSegment{
			Index:  uint32(tableidx),
			Offset: offset,
			Data:   bytes,
		}
	}

	importidx := b.memoryImports
	idx := b.memoryImports + b.inlineMemoryImports

	for _, memory := range b.m.Memories {
		var index int
		if memory.Import != nil {
			index, importidx = importidx, importidx+1
		} else {
			index, idx = idx, idx+1
		}

		if len(memory.Data) != 0 {
			var bytes []byte
			for _, v := range memory.Data {
				bytes = append(bytes, []byte(v)...)
			}
			section.Entries = append(section.Entries, wasm.DataSegment{
				Index:  uint32(index),
				Offset: zeroI32,
				Data:   bytes,
			})
		}
	}

	return &section, nil
}

func (b *moduleDecoder) decodeFunctionSig(params []*Param, results []wasm.ValueType) wasm.FunctionSig {
	paramTypes := make([]wasm.ValueType, len(params))
	for i, p := range params {
		paramTypes[i] = p.Type
	}
	return wasm.FunctionSig{Form: 0x60, ParamTypes: paramTypes, ReturnTypes: results}
}

func (b *moduleDecoder) decodeTableType(table *Table) wasm.Table {
	var range_ Range
	if table.Range != nil {
		range_ = *table.Range
	} else {
		range_ = Range{Min: uint32(len(table.Values))}
	}
	return b.decodeTableRange(range_)
}

func (b *moduleDecoder) decodeTableRange(range_ Range) wasm.Table {
	return wasm.Table{
		ElementType: wasm.ElemTypeAnyFunc,
		Limits:      b.decodeResizableLimits(range_),
	}
}

func (b *moduleDecoder) decodeMemoryType(memory *Memory) wasm.Memory {
	var range_ Range
	if memory.Range != nil {
		range_ = *memory.Range
	} else {
		for _, d := range memory.Data {
			range_.Min += uint32(len(d))
		}
	}
	return b.decodeMemoryRange(range_)
}

func (b *moduleDecoder) decodeMemoryRange(range_ Range) wasm.Memory {
	return wasm.Memory{Limits: b.decodeResizableLimits(range_)}
}

func (b *moduleDecoder) decodeGlobalType(global GlobalType) wasm.GlobalVar {
	return wasm.GlobalVar{
		Type:    global.Type,
		Mutable: global.Mutable,
	}
}

func (b *moduleDecoder) decodeResizableLimits(range_ Range) wasm.ResizableLimits {
	max, flags := uint32(0), uint8(0)
	if range_.Max != nil {
		max, flags = *range_.Max, 1
	}
	return wasm.ResizableLimits{
		Flags:   flags,
		Initial: range_.Min,
		Maximum: max,
	}
}

func (b *moduleDecoder) decodeFunctionBody(f *Func) (wasm.FunctionBody, error) {
	locals := make([]wasm.LocalEntry, 0, len(f.Locals))

	run := 0
	for i, l := range f.Locals {
		if i == 0 || f.Locals[i-1].Type == l.Type {
			run++
		} else {
			locals = append(locals, wasm.LocalEntry{
				Count: uint32(run),
				Type:  f.Locals[i-1].Type,
			})
			run = 1
		}
	}
	if run > 0 {
		locals = append(locals, wasm.LocalEntry{
			Count: uint32(run),
			Type:  f.Locals[len(f.Locals)-1].Type,
		})
	}

	b.pushFuncNames(f)
	defer b.pop()

	bytecode, err := b.decodeBytecode(f.Instrs, empty)
	if err != nil {
		return wasm.FunctionBody{}, err
	}

	return wasm.FunctionBody{
		Locals: locals,
		Code:   bytecode,
	}, nil
}

func (b *moduleDecoder) decodeBytecode(instrs []Instr, or []byte) ([]byte, error) {
	if len(instrs) == 0 {
		return or, nil
	}

	// linearize the body and synthesize an end if necessary
	var body []code.Instruction
	if err := b.linearizeInstrs(&body, instrs); err != nil {
		return nil, err
	}
	if op, ok := instrs[len(instrs)-1].(*Op); !ok || op.Code != END {
		body = append(body, code.End())
	}

	var buf bytes.Buffer
	if err := code.Encode(&buf, body); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (b *moduleDecoder) linearizeInstrs(dest *[]code.Instruction, instrs []Instr) error {
	for _, i := range instrs {
		if err := b.linearize(dest, i); err != nil {
			return err
		}
	}
	return nil
}

func (b *moduleDecoder) linearize(dest *[]code.Instruction, instr Instr) error {
	switch instr := instr.(type) {
	case *Block:
		b.pushBlock(instr.Name, instr.Type)
		defer b.popBlock()

		*dest = append(*dest, code.Block(b.decodeBlockType(instr.Type)))
		if err := b.linearizeInstrs(dest, instr.Instrs); err != nil {
			return err
		}
		*dest = append(*dest, code.End())
		return nil
	case *Loop:
		b.pushBlock(instr.Name, instr.Type)
		defer b.popBlock()

		*dest = append(*dest, code.Loop(b.decodeBlockType(instr.Type)))
		if err := b.linearizeInstrs(dest, instr.Instrs); err != nil {
			return err
		}
		*dest = append(*dest, code.End())
		return nil
	case *If:
		b.pushBlock(instr.Name, instr.Type)
		defer b.popBlock()

		if err := b.linearizeInstrs(dest, instr.Condition); err != nil {
			return err
		}
		*dest = append(*dest, code.If(b.decodeBlockType(instr.Type)))
		if err := b.linearizeInstrs(dest, instr.Then); err != nil {
			return err
		}
		if len(instr.Else) != 0 {
			*dest = append(*dest, code.Else())
			if err := b.linearizeInstrs(dest, instr.Else); err != nil {
				return err
			}
		}
		*dest = append(*dest, code.End())
		return nil
	case *Op:
		*dest = append(*dest, b.decodeOp(instr))
		return nil
	case *VarOp:
		*dest = append(*dest, b.decodeVarOp(instr))
		return nil
	case *CallIndirect:
		*dest = append(*dest, code.CallIndirect(uint32(b.context.functionType(&instr.Type))))
		return nil
	case *MemOp:
		*dest = append(*dest, b.decodeMemOp(instr))
		return nil
	case *ConstOp:
		*dest = append(*dest, b.decodeConstOp(instr))
		return nil
	default:
		panic("unreachable")
	}
}

func (b *moduleDecoder) decodeBlockType(t *FuncType) uint64 {
	switch {
	case t == nil:
		return code.BlockTypeEmpty
	case t.Var != nil:
		return code.BlockType(uint32(b.context.functionType(t)))
	case len(t.Params) == 0 && len(t.Results) == 0:
		return code.BlockTypeEmpty
	case len(t.Params) == 0 && len(t.Results) == 1:
		switch t.Results[0] {
		case wasm.ValueTypeI32:
			return code.BlockTypeI32
		case wasm.ValueTypeI64:
			return code.BlockTypeI64
		case wasm.ValueTypeF32:
			return code.BlockTypeF32
		case wasm.ValueTypeF64:
			return code.BlockTypeF64
		default:
			panic("unreachable")
		}
	default:
		return code.BlockType(uint32(b.context.functionType(t)))
	}
}

func (b *moduleDecoder) decodeOp(op *Op) code.Instruction {
	switch op.Code {
	case UNREACHABLE:
		return code.Unreachable()
	case NOP:
		return code.Nop()
	case RETURN:
		return code.Return()
	case DROP:
		return code.Drop()
	case SELECT:
		return code.Select()
	case MEMORY_GROW:
		return code.MemoryGrow()
	case MEMORY_SIZE:
		return code.MemorySize()
	case F32_ABS:
		return code.F32Abs()
	case F32_ADD:
		return code.F32Add()
	case F32_CEIL:
		return code.F32Ceil()
	case F32_CONVERT_I32_S:
		return code.F32ConvertI32S()
	case F32_CONVERT_I32_U:
		return code.F32ConvertI32U()
	case F32_CONVERT_I64_S:
		return code.F32ConvertI64S()
	case F32_CONVERT_I64_U:
		return code.F32ConvertI64U()
	case F32_COPYSIGN:
		return code.F32Copysign()
	case F32_DEMOTE_F64:
		return code.F32DemoteF64()
	case F32_DIV:
		return code.F32Div()
	case F32_EQ:
		return code.F32Eq()
	case F32_FLOOR:
		return code.F32Floor()
	case F32_GE:
		return code.F32Ge()
	case F32_GT:
		return code.F32Gt()
	case F32_LE:
		return code.F32Le()
	case F32_LT:
		return code.F32Lt()
	case F32_MAX:
		return code.F32Max()
	case F32_MIN:
		return code.F32Min()
	case F32_MUL:
		return code.F32Mul()
	case F32_NE:
		return code.F32Ne()
	case F32_NEAREST:
		return code.F32Nearest()
	case F32_NEG:
		return code.F32Neg()
	case F32_REINTERPRET_I32:
		return code.F32ReinterpretI32()
	case F32_SQRT:
		return code.F32Sqrt()
	case F32_SUB:
		return code.F32Sub()
	case F32_TRUNC:
		return code.F32Trunc()
	case F64_ABS:
		return code.F64Abs()
	case F64_ADD:
		return code.F64Add()
	case F64_CEIL:
		return code.F64Ceil()
	case F64_CONVERT_I32_S:
		return code.F64ConvertI32S()
	case F64_CONVERT_I32_U:
		return code.F64ConvertI32U()
	case F64_CONVERT_I64_S:
		return code.F64ConvertI64S()
	case F64_CONVERT_I64_U:
		return code.F64ConvertI64U()
	case F64_COPYSIGN:
		return code.F64Copysign()
	case F64_DIV:
		return code.F64Div()
	case F64_EQ:
		return code.F64Eq()
	case F64_FLOOR:
		return code.F64Floor()
	case F64_GE:
		return code.F64Ge()
	case F64_GT:
		return code.F64Gt()
	case F64_LE:
		return code.F64Le()
	case F64_LT:
		return code.F64Lt()
	case F64_MAX:
		return code.F64Max()
	case F64_MIN:
		return code.F64Min()
	case F64_MUL:
		return code.F64Mul()
	case F64_NE:
		return code.F64Ne()
	case F64_NEAREST:
		return code.F64Nearest()
	case F64_NEG:
		return code.F64Neg()
	case F64_PROMOTE_F32:
		return code.F64PromoteF32()
	case F64_REINTERPRET_I64:
		return code.F64ReinterpretI64()
	case F64_SQRT:
		return code.F64Sqrt()
	case F64_SUB:
		return code.F64Sub()
	case F64_TRUNC:
		return code.F64Trunc()
	case I32_ADD:
		return code.I32Add()
	case I32_AND:
		return code.I32And()
	case I32_CLZ:
		return code.I32Clz()
	case I32_CTZ:
		return code.I32Ctz()
	case I32_DIV_S:
		return code.I32DivS()
	case I32_DIV_U:
		return code.I32DivU()
	case I32_EQ:
		return code.I32Eq()
	case I32_EQZ:
		return code.I32Eqz()
	case I32_EXTEND16_S:
		return code.I32Extend16S()
	case I32_EXTEND8_S:
		return code.I32Extend8S()
	case I32_GE_S:
		return code.I32GeS()
	case I32_GE_U:
		return code.I32GeU()
	case I32_GT_S:
		return code.I32GtS()
	case I32_GT_U:
		return code.I32GtU()
	case I32_LE_S:
		return code.I32LeS()
	case I32_LE_U:
		return code.I32LeU()
	case I32_LT_S:
		return code.I32LtS()
	case I32_LT_U:
		return code.I32LtU()
	case I32_MUL:
		return code.I32Mul()
	case I32_NE:
		return code.I32Ne()
	case I32_OR:
		return code.I32Or()
	case I32_POPCNT:
		return code.I32Popcnt()
	case I32_REINTERPRET_F32:
		return code.I32ReinterpretF32()
	case I32_REM_S:
		return code.I32RemS()
	case I32_REM_U:
		return code.I32RemU()
	case I32_ROTL:
		return code.I32Rotl()
	case I32_ROTR:
		return code.I32Rotr()
	case I32_SHL:
		return code.I32Shl()
	case I32_SHR_S:
		return code.I32ShrS()
	case I32_SHR_U:
		return code.I32ShrU()
	case I32_SUB:
		return code.I32Sub()
	case I32_TRUNC_F32_S:
		return code.I32TruncF32S()
	case I32_TRUNC_F32_U:
		return code.I32TruncF32U()
	case I32_TRUNC_F64_S:
		return code.I32TruncF64S()
	case I32_TRUNC_F64_U:
		return code.I32TruncF64U()
	case I32_TRUNC_SAT_F32_S:
		return code.I32TruncSatF32S()
	case I32_TRUNC_SAT_F32_U:
		return code.I32TruncSatF32U()
	case I32_TRUNC_SAT_F64_S:
		return code.I32TruncSatF64S()
	case I32_TRUNC_SAT_F64_U:
		return code.I32TruncSatF64U()
	case I32_WRAP_I64:
		return code.I32WrapI64()
	case I32_XOR:
		return code.I32Xor()
	case I64_ADD:
		return code.I64Add()
	case I64_AND:
		return code.I64And()
	case I64_CLZ:
		return code.I64Clz()
	case I64_CTZ:
		return code.I64Ctz()
	case I64_DIV_S:
		return code.I64DivS()
	case I64_DIV_U:
		return code.I64DivU()
	case I64_EQ:
		return code.I64Eq()
	case I64_EQZ:
		return code.I64Eqz()
	case I64_EXTEND16_S:
		return code.I64Extend16S()
	case I64_EXTEND32_S:
		return code.I64Extend32S()
	case I64_EXTEND8_S:
		return code.I64Extend8S()
	case I64_EXTEND_I32_S:
		return code.I64ExtendI32S()
	case I64_EXTEND_I32_U:
		return code.I64ExtendI32U()
	case I64_GE_S:
		return code.I64GeS()
	case I64_GE_U:
		return code.I64GeU()
	case I64_GT_S:
		return code.I64GtS()
	case I64_GT_U:
		return code.I64GtU()
	case I64_LE_S:
		return code.I64LeS()
	case I64_LE_U:
		return code.I64LeU()
	case I64_LT_S:
		return code.I64LtS()
	case I64_LT_U:
		return code.I64LtU()
	case I64_MUL:
		return code.I64Mul()
	case I64_NE:
		return code.I64Ne()
	case I64_OR:
		return code.I64Or()
	case I64_POPCNT:
		return code.I64Popcnt()
	case I64_REINTERPRET_F64:
		return code.I64ReinterpretF64()
	case I64_REM_S:
		return code.I64RemS()
	case I64_REM_U:
		return code.I64RemU()
	case I64_ROTL:
		return code.I64Rotl()
	case I64_ROTR:
		return code.I64Rotr()
	case I64_SHL:
		return code.I64Shl()
	case I64_SHR_S:
		return code.I64ShrS()
	case I64_SHR_U:
		return code.I64ShrU()
	case I64_SUB:
		return code.I64Sub()
	case I64_TRUNC_F32_S:
		return code.I64TruncF32S()
	case I64_TRUNC_F32_U:
		return code.I64TruncF32U()
	case I64_TRUNC_F64_S:
		return code.I64TruncF64S()
	case I64_TRUNC_F64_U:
		return code.I64TruncF64U()
	case I64_TRUNC_SAT_F32_S:
		return code.I64TruncSatF32S()
	case I64_TRUNC_SAT_F32_U:
		return code.I64TruncSatF32U()
	case I64_TRUNC_SAT_F64_S:
		return code.I64TruncSatF64S()
	case I64_TRUNC_SAT_F64_U:
		return code.I64TruncSatF64U()
	case I64_XOR:
		return code.I64Xor()
	default:
		panic(fmt.Errorf("invalid Op %v", op.Code))
	}
}

func (b *moduleDecoder) decodeVarOp(op *VarOp) code.Instruction {
	switch op.Code {
	case BR_TABLE:
		indices := make([]int, len(op.Vars))
		for i, v := range op.Vars {
			indices[i] = b.useLabel(v)
		}
		return code.BrTable(indices[0], indices[1:]...)
	case BR:
		return code.Br(b.useLabel(op.Vars[0]))
	case BR_IF:
		return code.BrIf(b.useLabel(op.Vars[0]))
	}

	switch op.Code {
	case CALL:
		return code.Call(uint32(b.context.useFunction(op.Vars[0])))
	case LOCAL_GET:
		return code.LocalGet(uint32(b.context.useLocal(op.Vars[0])))
	case LOCAL_SET:
		return code.LocalSet(uint32(b.context.useLocal(op.Vars[0])))
	case LOCAL_TEE:
		return code.LocalTee(uint32(b.context.useLocal(op.Vars[0])))
	case GLOBAL_GET:
		return code.GlobalGet(uint32(b.context.useGlobal(op.Vars[0])))
	case GLOBAL_SET:
		return code.GlobalSet(uint32(b.context.useGlobal(op.Vars[0])))
	default:
		panic(fmt.Errorf("invalid VarOp %v", op.Code))
	}
}

func (b *moduleDecoder) decodeMemOp(op *MemOp) code.Instruction {
	offset, align := uint32(0), uint32(0)
	if op.Offset != nil {
		offset = uint32(*op.Offset)
	}
	if op.Align != nil {
		align = uint32(*op.Align)
		if ones := bits.OnesCount32(align); ones != 1 {
			panic(errors.New("alignment"))
		}
		b.checkAlignment(op.Code, align)
	}

	switch op.Code {
	case F32_LOAD:
		return code.F32Load(offset, align)
	case F64_LOAD:
		return code.F64Load(offset, align)
	case I32_LOAD:
		return code.I32Load(offset, align)
	case I64_LOAD:
		return code.I64Load(offset, align)
	case I32_LOAD16_S:
		return code.I32Load16S(offset, align)
	case I32_LOAD16_U:
		return code.I32Load16U(offset, align)
	case I32_LOAD8_S:
		return code.I32Load8S(offset, align)
	case I32_LOAD8_U:
		return code.I32Load8U(offset, align)
	case I64_LOAD16_S:
		return code.I64Load16S(offset, align)
	case I64_LOAD16_U:
		return code.I64Load16U(offset, align)
	case I64_LOAD32_S:
		return code.I64Load32S(offset, align)
	case I64_LOAD32_U:
		return code.I64Load32U(offset, align)
	case I64_LOAD8_S:
		return code.I64Load8S(offset, align)
	case I64_LOAD8_U:
		return code.I64Load8U(offset, align)
	case F32_STORE:
		return code.F32Store(offset, align)
	case F64_STORE:
		return code.F64Store(offset, align)
	case I32_STORE:
		return code.I32Store(offset, align)
	case I64_STORE:
		return code.I64Store(offset, align)
	case I32_STORE16:
		return code.I32Store16(offset, align)
	case I32_STORE8:
		return code.I32Store8(offset, align)
	case I64_STORE16:
		return code.I64Store16(offset, align)
	case I64_STORE32:
		return code.I64Store32(offset, align)
	case I64_STORE8:
		return code.I64Store8(offset, align)
	default:
		panic(fmt.Errorf("invalid MemOp %v", op.Code))
	}
}

func (b *moduleDecoder) decodeConstOp(op *ConstOp) code.Instruction {
	switch op.Code {
	case F32_CONST:
		v, ok := op.Value.(float32)
		if !ok {
			panic(fmt.Errorf("invalid F32 constant %v", op.Value))
		}
		return code.F32Const(v)
	case F64_CONST:
		v, ok := op.Value.(float64)
		if !ok {
			panic(fmt.Errorf("invalid F64 constant %v", op.Value))
		}
		return code.F64Const(v)
	case I32_CONST:
		v, ok := op.Value.(int32)
		if !ok {
			panic(fmt.Errorf("invalid I32 constant %v", op.Value))
		}
		return code.I32Const(v)
	case I64_CONST:
		v, ok := op.Value.(int64)
		if !ok {
			panic(fmt.Errorf("invalid I64 constant %v", op.Value))
		}
		return code.I64Const(v)
	default:
		panic(fmt.Errorf("invalid ConstOp %v", op.Value))
	}
}

func (b *moduleDecoder) checkAlignment(code TokenKind, align uint32) {
	natural := uint32(0)
	switch code {
	case I32_LOAD8_S, I32_LOAD8_U, I64_LOAD8_S, I64_LOAD8_U, I32_STORE8, I64_STORE8:
		natural = 1
	case I32_LOAD16_S, I32_LOAD16_U, I64_LOAD16_S, I64_LOAD16_U, I32_STORE16, I64_STORE16:
		natural = 2
	case F32_LOAD, I32_LOAD, I64_LOAD32_S, I64_LOAD32_U, I32_STORE, F32_STORE, I64_STORE32:
		natural = 4
	case F64_LOAD, I64_LOAD, F64_STORE, I64_STORE:
		natural = 8
	}
	if align > natural {
		panic(fmt.Errorf("alignment must not be larger than natural"))
	}
}

func mustEncode(expr ...code.Instruction) []byte {
	var b bytes.Buffer
	err := code.Encode(&b, expr)
	if err != nil {
		panic(err)
	}
	return b.Bytes()
}

var empty = mustEncode(code.End())
var zeroI32 = mustEncode(code.I32Const(0), code.End())
var zeroI64 = mustEncode(code.I64Const(0), code.End())
var zeroF32 = mustEncode(code.F32Const(0), code.End())
var zeroF64 = mustEncode(code.I64Const(0), code.End())
