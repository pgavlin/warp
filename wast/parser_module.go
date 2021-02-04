package wast

import (
	"github.com/pgavlin/warp/wasm"
)

func ParseModule(scanner *Scanner) (module *Module, err error) {
	defer func() {
		if v := recover(); v != nil {
			e, ok := v.(error)
			if !ok {
				panic(v)
			}
			err = e
		}
	}()

	p := parser{s: scanner}
	p.start()
	pos := p.tok.Pos
	p.expect('(')
	m := p.parseModule(pos, false).(*Module)
	p.expect(EOF)
	return m, nil
}

func (p *parser) parseModule(pos Pos, allowCommand bool) ModuleCommand {
	p.expect(MODULE)

	name, _ := p.maybe(VAR).(string)

	if allowCommand && (p.tok.Kind == BINARY || p.tok.Kind == QUOTE) {
		return p.parseModuleLiteral(pos, name)
	}

	m := p.parseModuleBody(name)
	m.Pos = pos

	p.expect(')')
	return m
}

func (p *parser) parseModuleBody(name string) *Module {
	m := Module{Name: name}

	for p.tok.Kind == '(' {
		switch p.peek() {
		case TYPE:
			m.Types = append(m.Types, p.parseTypedef())
		case FUNC:
			m.Funcs = append(m.Funcs, p.parseFunc())
		case IMPORT:
			m.Imports = append(m.Imports, p.parseImport())
		case EXPORT:
			m.Exports = append(m.Exports, p.parseExport())
		case TABLE:
			m.Tables = append(m.Tables, p.parseTable())
		case MEMORY:
			m.Memories = append(m.Memories, p.parseMemory())
		case GLOBAL:
			m.Globals = append(m.Globals, p.parseGlobal())
		case ELEM:
			m.Elems = append(m.Elems, p.parseElem())
		case DATA:
			m.Data = append(m.Data, p.parseData())
		case START:
			if m.Start != nil {
				panic(p.errorf("multiple start sections"))
			}
			m.Start = p.parseStart()
		default:
			panic(p.errorf("expected TYPE, FUNC, IMPORT, EXPORT, TABLE, MEMORY, GLOBAL, ELEM, DATA, or START (got %v)", p.tok.Kind))
		}
	}

	return &m
}

func (p *parser) parseTypedef() *Typedef {
	p.expectSExpr(TYPE)
	defer p.closeSExpr()

	name, _ := p.maybe(VAR).(string)

	p.expectSExpr(FUNC)
	defer p.closeSExpr()

	return &Typedef{
		Name:    name,
		Params:  p.parseParams(),
		Results: p.parseResults(),
	}
}

func (p *parser) parseFunc() *Func {
	p.expectSExpr(FUNC)
	defer p.closeSExpr()

	name, _ := p.maybe(VAR).(string)

	exports := p.parseInlineExports(wasm.ExternalFunction)
	import_ := p.parseInlineImport()
	typ := p.parseFuncType()

	locals, instrs := []*Local(nil), []Instr(nil)
	if import_ == nil {
		locals, instrs = p.parseLocals(), p.parseInstrs(')')
	}

	return &Func{
		Name:    name,
		Exports: exports,
		Import:  import_,
		Type:    typ,
		Locals:  locals,
		Instrs:  instrs,
	}
}

func (p *parser) parseImport() *Import {
	p.expectSExpr(IMPORT)
	defer p.closeSExpr()

	module, name := p.expect(STRING).(string), p.expect(STRING).(string)

	var external External
	switch p.peek() {
	case FUNC:
		external = p.parseExternalFunc()
	case GLOBAL:
		external = p.parseExternalGlobal()
	case TABLE:
		external = p.parseExternalTable()
	case MEMORY:
		external = p.parseExternalMemory()
	}

	return &Import{
		Module:   module,
		Name:     name,
		External: external,
	}
}

func (p *parser) parseExport() *Export {
	p.expectSExpr(EXPORT)
	defer p.closeSExpr()

	name := p.expect(STRING).(string)

	p.expect('(')
	defer p.closeSExpr()

	var external wasm.External
	switch p.tok.Kind {
	case FUNC:
		external = wasm.ExternalFunction
	case GLOBAL:
		external = wasm.ExternalGlobal
	case TABLE:
		external = wasm.ExternalTable
	case MEMORY:
		external = wasm.ExternalMemory
	}
	p.scan()

	return &Export{
		Name: name,
		Kind: external,
		Var:  *p.parseVar(),
	}
}

func (p *parser) parseTable() *Table {
	p.expectSExpr(TABLE)
	defer p.closeSExpr()

	name, _ := p.maybe(VAR).(string)

	exports := p.parseInlineExports(wasm.ExternalFunction)

	if p.tok.Kind == FUNCREF {
		p.scan()

		p.expectSExpr(ELEM)
		defer p.closeSExpr()

		var values []Var
		for p.tok.Kind != ')' {
			values = append(values, *p.parseVar())
		}

		return &Table{
			Name:    name,
			Exports: exports,
			Values:  values,
		}
	}

	import_ := p.parseInlineImport()
	rng := p.parseRange()
	p.expect(FUNCREF)

	return &Table{
		Name:    name,
		Exports: exports,
		Import:  import_,
		Range:   rng,
	}
}

func (p *parser) parseMemory() *Memory {
	p.expectSExpr(MEMORY)
	defer p.closeSExpr()

	name, _ := p.maybe(VAR).(string)

	exports := p.parseInlineExports(wasm.ExternalFunction)

	if p.scanSExpr(DATA) {
		defer p.closeSExpr()

		var data []string
		for p.tok.Kind != ')' {
			data = append(data, p.expect(STRING).(string))
		}

		return &Memory{
			Name:    name,
			Exports: exports,
			Data:    data,
		}
	}

	return &Memory{
		Name:    name,
		Exports: exports,
		Import:  p.parseInlineImport(),
		Range:   p.parseRange(),
	}
}

func (p *parser) parseGlobal() *Global {
	p.expectSExpr(GLOBAL)
	defer p.closeSExpr()

	name, _ := p.maybe(VAR).(string)

	exports := p.parseInlineExports(wasm.ExternalFunction)
	import_ := p.parseInlineImport()
	typ := p.parseGlobalType()

	var init []Instr
	if import_ == nil {
		init = p.parseInstrs(')')
	}

	return &Global{
		Name:    name,
		Exports: exports,
		Import:  import_,
		Type:    typ,
		Init:    init,
	}
}

func (p *parser) parseElem() *Elem {
	p.expectSExpr(ELEM)
	defer p.closeSExpr()

	var_ := p.parseVar()

	var offset []Instr
	if p.scanSExpr(OFFSET) {
		offset = p.parseInstrs(')')
		p.closeSExpr()
	} else {
		offset = p.parseExpr()
	}

	var vars []Var
	for p.tok.Kind != ')' {
		vars = append(vars, *p.parseVar())
	}

	return &Elem{
		Var:    var_,
		Offset: offset,
		Values: vars,
	}
}

func (p *parser) parseData() *Data {
	p.expectSExpr(DATA)
	defer p.closeSExpr()

	var_ := p.parseVar()

	var offset []Instr
	if p.scanSExpr(OFFSET) {
		offset = p.parseInstrs(')')
		p.closeSExpr()
	} else {
		offset = p.parseExpr()
	}

	var values []string
	for p.tok.Kind != ')' {
		values = append(values, p.expect(STRING).(string))
	}

	return &Data{
		Var:    var_,
		Offset: offset,
		Values: values,
	}
}

func (p *parser) parseStart() *Var {
	p.expectSExpr(START)
	defer p.closeSExpr()

	return p.parseVar()
}

func (p *parser) parseBlock() *Block {
	p.expect(BLOCK)

	name, _ := p.maybe(VAR).(string)
	typ := p.parseFuncType()
	instrs := p.parseInstrs(END)

	p.expect(END)
	p.maybe(VAR)

	return &Block{
		Name:   name,
		Type:   typ,
		Instrs: instrs,
	}
}

func (p *parser) parseBlockExpr() *Block {
	p.expectSExpr(BLOCK)
	defer p.closeSExpr()

	name, _ := p.maybe(VAR).(string)
	return &Block{
		Name:   name,
		Type:   p.parseFuncType(),
		Instrs: p.parseInstrs(')'),
	}
}

func (p *parser) parseInlineExports(kind wasm.External) []string {
	var exports []string
	for p.scanSExpr(EXPORT) {
		exports = append(exports, p.expect(STRING).(string))
		p.closeSExpr()
	}
	return exports
}

func (p *parser) parseExpr() []Instr {
	switch p.peek() {
	case BLOCK:
		return []Instr{p.parseBlockExpr()}
	case LOOP:
		return []Instr{p.parseLoopExpr()}
	case IF:
		return []Instr{p.parseIfExpr()}
	}

	p.expect('(')
	defer p.closeSExpr()

	final := p.parseOp()
	var instrs []Instr
	for p.tok.Kind != ')' {
		instrs = append(instrs, p.parseExpr()...)
	}

	return append(instrs, final)
}

func (p *parser) parseExternalFunc() *ExternalFunc {
	p.expectSExpr(FUNC)
	defer p.closeSExpr()

	name, _ := p.maybe(VAR).(string)
	return &ExternalFunc{
		Name: name,
		Type: p.parseTypeUse(),
	}
}

func (p *parser) parseExternalGlobal() *ExternalGlobal {
	p.expectSExpr(GLOBAL)
	defer p.closeSExpr()

	name, _ := p.maybe(VAR).(string)
	return &ExternalGlobal{
		Name: name,
		Type: p.parseGlobalType(),
	}
}

func (p *parser) parseExternalMemory() *ExternalMemory {
	p.expectSExpr(MEMORY)
	defer p.closeSExpr()

	name, _ := p.maybe(VAR).(string)
	return &ExternalMemory{
		Name:  name,
		Range: *p.parseRange(),
	}
}

func (p *parser) parseExternalTable() *ExternalTable {
	p.expectSExpr(TABLE)
	defer p.closeSExpr()

	name, _ := p.maybe(VAR).(string)
	typ := p.parseRange()

	p.expect(FUNCREF)

	return &ExternalTable{
		Name:  name,
		Range: *typ,
	}
}

func (p *parser) parseFuncType() *FuncType {
	var var_ *Var
	if p.scanSExpr(TYPE) {
		var_ = p.parseVar()
		p.closeSExpr()
	}

	return &FuncType{
		Var:     var_,
		Params:  p.parseParams(),
		Results: p.parseResults(),
	}
}

func (p *parser) parseGlobalType() GlobalType {
	if p.scanSExpr(MUT) {
		defer p.closeSExpr()

		return GlobalType{Mutable: true, Type: p.parseValType()}
	}
	return GlobalType{Type: p.parseValType()}
}

func (p *parser) parseIf() *If {
	p.expect(IF)

	name, _ := p.maybe(VAR).(string)

	typ := p.parseFuncType()
	then := p.parseInstrs(END, ELSE)

	var else_ []Instr
	if p.tok.Kind == ELSE {
		p.scan()
		p.maybe(VAR)

		else_ = p.parseInstrs(END)
	}

	p.expect(END)
	p.maybe(VAR)

	return &If{
		Name: name,
		Type: typ,
		Then: then,
		Else: else_,
	}
}

func (p *parser) parseIfExpr() *If {
	p.expectSExpr(IF)
	defer p.closeSExpr()

	name, _ := p.maybe(VAR).(string)

	typ := p.parseFuncType()

	var condition []Instr
	for !p.peekSExpr(THEN) {
		condition = append(condition, p.parseExpr()...)
	}

	p.expectSExpr(THEN)
	then := p.parseInstrs(')')
	p.closeSExpr()

	var else_ []Instr
	if p.scanSExpr(ELSE) {
		else_ = p.parseInstrs(')')
		p.closeSExpr()
	}

	return &If{
		Name:      name,
		Type:      typ,
		Condition: condition,
		Then:      then,
		Else:      else_,
	}
}

func (p *parser) parseInlineImport() *InlineImport {
	if !p.scanSExpr(IMPORT) {
		return nil
	}
	defer p.closeSExpr()

	return &InlineImport{
		Module: p.expect(STRING).(string),
		Name:   p.expect(STRING).(string),
	}
}

func (p *parser) parseInstrs(term ...TokenKind) []Instr {
	var instrs []Instr
	for !any(p.tok.Kind, term) {
		switch p.tok.Kind {
		case BLOCK:
			instrs = append(instrs, p.parseBlock())
		case LOOP:
			instrs = append(instrs, p.parseLoop())
		case IF:
			instrs = append(instrs, p.parseIf())
		case '(':
			instrs = append(instrs, p.parseExpr()...)
		default:
			instrs = append(instrs, p.parseOp())
		}
	}
	return instrs
}

func (p *parser) parseLocals() []*Local {
	var locals []*Local
	for p.scanSExpr(LOCAL) {
		if p.tok.Kind == VAR {
			locals = append(locals, &Local{
				Name: p.expect(VAR).(string),
				Type: p.parseValType(),
			})
		} else {
			for p.tok.Kind != ')' {
				locals = append(locals, &Local{Type: p.parseValType()})
			}
		}
		p.closeSExpr()
	}
	return locals
}

func (p *parser) parseLoop() *Loop {
	p.expect(LOOP)

	name, _ := p.maybe(VAR).(string)

	typ := p.parseFuncType()
	instrs := p.parseInstrs(END)

	p.expect(END)
	p.maybe(VAR)

	return &Loop{
		Name:   name,
		Type:   typ,
		Instrs: instrs,
	}
}

func (p *parser) parseLoopExpr() *Loop {
	p.expectSExpr(LOOP)
	defer p.closeSExpr()

	name, _ := p.maybe(VAR).(string)
	return &Loop{
		Name:   name,
		Type:   p.parseFuncType(),
		Instrs: p.parseInstrs(')'),
	}
}

func (p *parser) parseParams() []*Param {
	var params []*Param
	for p.scanSExpr(PARAM) {
		if p.tok.Kind == VAR {
			params = append(params, &Param{
				Name: p.expect(VAR).(string),
				Type: p.parseValType(),
			})
		} else {
			for p.tok.Kind != ')' {
				params = append(params, &Param{Type: p.parseValType()})
			}
		}
		p.closeSExpr()
	}
	return params
}

func (p *parser) parseRange() *Range {
	min := uint32(p.expectI(INT))

	var max *uint32
	if p.tok.Kind == INT {
		m := uint32(p.expectI(INT))
		max = &m
	}

	return &Range{
		Min: min,
		Max: max,
	}
}

func (p *parser) parseResults() []wasm.ValueType {
	var results []wasm.ValueType
	for p.scanSExpr(RESULT) {
		for p.tok.Kind != ')' {
			results = append(results, p.parseValType())
		}
		p.closeSExpr()
	}
	return results
}

func (p *parser) parseTypeUse() *FuncType {
	return p.parseFuncType()
}

func (p *parser) parseValType() wasm.ValueType {
	switch p.tok.Kind {
	case I32:
		p.scan()
		return wasm.ValueTypeI32
	case I64:
		p.scan()
		return wasm.ValueTypeI64
	case F32:
		p.scan()
		return wasm.ValueTypeF32
	case F64:
		p.scan()
		return wasm.ValueTypeF64
	default:
		panic(p.errorf("expected I32, I64, F32, or F64"))
	}
}

func (p *parser) parseVar() *Var {
	switch p.tok.Kind {
	case INT, NAT:
		return &Var{Index: uint32(p.expectI(p.tok.Kind))}
	case VAR:
		return &Var{Name: p.expect(VAR).(string)}
	default:
		return nil
	}
}

func (p *parser) parseOp() Instr {
	switch p.tok.Kind {
	case BR_TABLE:
		code := p.tok.Kind
		p.scan()

		var vars []Var
		for p.tok.Kind == VAR || p.tok.Kind == NAT || p.tok.Kind == INT {
			vars = append(vars, *p.parseVar())
		}
		return &VarOp{Code: code, Vars: vars}

	case CALL_INDIRECT:
		p.scan()

		typ := p.parseFuncType()
		return &CallIndirect{Type: *typ}

	case BR, BR_IF, CALL, LOCAL_GET, LOCAL_SET, LOCAL_TEE, GLOBAL_GET, GLOBAL_SET:
		code := p.tok.Kind
		p.scan()

		return &VarOp{Code: code, Vars: []Var{*p.parseVar()}}

	case F32_LOAD, F64_LOAD, I32_LOAD, I64_LOAD, I32_LOAD16_S, I32_LOAD16_U, I32_LOAD8_S, I32_LOAD8_U, I64_LOAD16_S, I64_LOAD16_U, I64_LOAD32_S, I64_LOAD32_U, I64_LOAD8_S, I64_LOAD8_U, F32_STORE, F64_STORE, I32_STORE, I64_STORE, I32_STORE16, I32_STORE8, I64_STORE16, I64_STORE32, I64_STORE8:
		code := p.tok.Kind
		p.scan()

		var offset *int64
		if p.tok.Kind == OFFSET {
			o := p.expectI(OFFSET, '=', INT)
			offset = &o
		}

		var align *int64
		if p.tok.Kind == ALIGN {
			a := p.expectI(ALIGN, '=', INT)
			align = &a
		}

		return &MemOp{Code: code, Offset: offset, Align: align}

	case F32_CONST:
		p.scan()

		v, ok := p.F32()
		if !ok {
			panic(p.errorf("expected INT or FLOAT"))
		}
		p.scan()
		return &ConstOp{Code: F32_CONST, Value: v}

	case F64_CONST:
		p.scan()

		v, ok := p.F64()
		if !ok {
			panic(p.errorf("expected INT or FLOAT"))
		}
		p.scan()
		return &ConstOp{Code: F64_CONST, Value: v}

	case I32_CONST:
		p.scan()

		v, ok := p.I32()
		if !ok {
			panic(p.errorf("expected INT"))
		}
		p.scan()
		return &ConstOp{Code: I32_CONST, Value: v}

	case I64_CONST:
		p.scan()

		v, ok := p.I64()
		if !ok {
			panic(p.errorf("expected INT"))
		}
		p.scan()
		return &ConstOp{Code: I64_CONST, Value: v}

	case UNREACHABLE, NOP, RETURN, DROP, SELECT, MEMORY_GROW, MEMORY_SIZE,
		F32_ABS, F32_ADD, F32_CEIL, F32_CONVERT_I32_S, F32_CONVERT_I32_U, F32_CONVERT_I64_S, F32_CONVERT_I64_U, F32_COPYSIGN, F32_DEMOTE_F64, F32_DIV, F32_EQ, F32_FLOOR, F32_GE, F32_GT, F32_LE, F32_LT, F32_MAX, F32_MIN, F32_MUL, F32_NE, F32_NEAREST, F32_NEG, F32_REINTERPRET_I32, F32_SQRT, F32_SUB, F32_TRUNC,
		F64_ABS, F64_ADD, F64_CEIL, F64_CONVERT_I32_S, F64_CONVERT_I32_U, F64_CONVERT_I64_S, F64_CONVERT_I64_U, F64_COPYSIGN, F64_DIV, F64_EQ, F64_FLOOR, F64_GE, F64_GT, F64_LE, F64_LT, F64_MAX, F64_MIN, F64_MUL, F64_NE, F64_NEAREST, F64_NEG, F64_PROMOTE_F32, F64_REINTERPRET_I64, F64_SQRT, F64_SUB, F64_TRUNC,
		I32_ADD, I32_AND, I32_CLZ, I32_CTZ, I32_DIV_S, I32_DIV_U, I32_EQ, I32_EQZ, I32_EXTEND16_S, I32_EXTEND8_S, I32_GE_S, I32_GE_U, I32_GT_S, I32_GT_U, I32_LE_S, I32_LE_U, I32_LT_S, I32_LT_U, I32_MUL, I32_NE, I32_OR, I32_POPCNT, I32_REINTERPRET_F32, I32_REM_S, I32_REM_U, I32_ROTL, I32_ROTR, I32_SHL, I32_SHR_S, I32_SHR_U, I32_SUB, I32_TRUNC_F32_S, I32_TRUNC_F32_U, I32_TRUNC_F64_S, I32_TRUNC_F64_U, I32_TRUNC_SAT_F32_S, I32_TRUNC_SAT_F32_U, I32_TRUNC_SAT_F64_S, I32_TRUNC_SAT_F64_U, I32_WRAP_I64, I32_XOR,
		I64_ADD, I64_AND, I64_CLZ, I64_CTZ, I64_DIV_S, I64_DIV_U, I64_EQ, I64_EQZ, I64_EXTEND16_S, I64_EXTEND32_S, I64_EXTEND8_S, I64_EXTEND_I32_S, I64_EXTEND_I32_U, I64_GE_S, I64_GE_U, I64_GT_S, I64_GT_U, I64_LE_S, I64_LE_U, I64_LT_S, I64_LT_U, I64_MUL, I64_NE, I64_OR, I64_POPCNT, I64_REINTERPRET_F64, I64_REM_S, I64_REM_U, I64_ROTL, I64_ROTR, I64_SHL, I64_SHR_S, I64_SHR_U, I64_SUB, I64_TRUNC_F32_S, I64_TRUNC_F32_U, I64_TRUNC_F64_S, I64_TRUNC_F64_U, I64_TRUNC_SAT_F32_S, I64_TRUNC_SAT_F32_U, I64_TRUNC_SAT_F64_S, I64_TRUNC_SAT_F64_U, I64_XOR:

		code := p.tok.Kind
		p.scan()
		return &Op{Code: code}

	default:
		panic(p.errorf("unknown operator"))
	}
}
