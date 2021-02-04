package wast

import "strings"

func ParseScript(scanner *Scanner) (script *Script, err error) {
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
	return p.parseScript(), nil
}

func (p *parser) parseScript() *Script {
	var commands []Command
	for p.tok.Kind != EOF {
		commands = append(commands, p.parseCommand())
	}
	p.expect(EOF)
	return &Script{Commands: commands}
}

func (p *parser) parseCommand() Command {
	if p.tok.Kind == '(' {
		switch p.peek() {
		case TYPE, FUNC, IMPORT, EXPORT, TABLE, MEMORY, GLOBAL, ELEM, DATA, START:
			return p.parseModuleBody("")
		}
	}

	pos := p.tok.Pos
	p.expect('(')

	switch p.tok.Kind {
	case MODULE:
		return p.parseModule(pos, true)
	case REGISTER:
		return p.parseRegister(pos)
	case INVOKE:
		return p.parseInvoke(pos)
	case GET:
		return p.parseGet(pos)
	case ASSERT_RETURN:
		return p.parseAssertReturn(pos)
	case ASSERT_TRAP:
		return p.parseAssertTrap(pos)
	case ASSERT_EXHAUSTION:
		return p.parseAssertExhaustion(pos)
	case ASSERT_MALFORMED, ASSERT_INVALID, ASSERT_UNLINKABLE:
		return p.parseModuleAssertion(pos)
	case SCRIPT:
		return p.parseScriptCommand(pos)
	case INPUT:
		return p.parseInput(pos)
	case OUTPUT:
		return p.parseOutput(pos)
	default:
		panic(p.errorf("expected action, assertion, or meta command"))
	}
}

func (p *parser) parseModuleLiteral(pos Pos, name string) *ModuleLiteral {
	defer p.closeSExpr()

	isBinary := false
	switch p.tok.Kind {
	case BINARY:
		isBinary = true
	case QUOTE:
		// OK
	default:
		panic(p.errorf("expected BINARY or QUOTE"))
	}
	p.scan()

	var data strings.Builder
	for p.tok.Kind != ')' {
		data.WriteString(p.expect(STRING).(string))
	}

	return &ModuleLiteral{
		Pos:      pos,
		Name:     name,
		IsBinary: isBinary,
		Data:     data.String(),
	}
}

func (p *parser) parseRegister(pos Pos) *Register {
	p.expect(REGISTER)
	defer p.closeSExpr()

	export := p.expect(STRING).(string)
	name, _ := p.maybe(VAR).(string)
	return &Register{
		Pos:    pos,
		Export: export,
		Name:   name,
	}
}

func (p *parser) parseAction(pos Pos) Action {
	p.expect('(')

	switch p.tok.Kind {
	case INVOKE:
		return p.parseInvoke(pos)
	case GET:
		return p.parseGet(pos)
	default:
		panic(p.errorf("exected INVOKE or GET"))
	}
}

func (p *parser) parseInvoke(pos Pos) *Invoke {
	defer p.closeSExpr()
	p.scan()

	name, _ := p.maybe(VAR).(string)

	export := p.expect(STRING).(string)

	var args []interface{}
	for p.tok.Kind != ')' {
		p.expect('(')
		switch p.tok.Kind {
		case F32_CONST, F64_CONST, I32_CONST, I64_CONST:
			args = append(args, p.parseOp().(*ConstOp).Value)
		default:
			panic(p.errorf("expected F32_CONST, F64_CONST, I32_CONST, or I64_CONST"))
		}
		p.closeSExpr()
	}

	return &Invoke{
		Pos:    pos,
		Name:   name,
		Export: export,
		Args:   args,
	}
}

func (p *parser) parseGet(pos Pos) *Get {
	defer p.closeSExpr()
	p.scan()

	name, _ := p.maybe(VAR).(string)
	return &Get{
		Pos:    pos,
		Name:   name,
		Export: p.expect(STRING).(string),
	}
}

func (p *parser) parseResult() interface{} {
	p.expect('(')
	defer p.closeSExpr()

	switch p.tok.Kind {
	case F32_CONST, F64_CONST:
		if n := p.peek(); n == NAN_ARITHMETIC || n == NAN_CANONICAL {
			p.scan()
			p.scan()
			return n
		}
		return p.parseOp().(*ConstOp).Value
	case I32_CONST, I64_CONST:
		return p.parseOp().(*ConstOp).Value
	default:
		panic(p.errorf("expected F32_CONST, F64_CONST, I32_CONST, or I64_CONST"))
	}
}

func (p *parser) parseAssertReturn(pos Pos) *AssertReturn {
	p.expect(ASSERT_RETURN)
	defer p.closeSExpr()

	action := p.parseAction(p.tok.Pos)

	var results []interface{}
	for p.tok.Kind != ')' {
		results = append(results, p.parseResult())
	}

	return &AssertReturn{
		Pos:     pos,
		Action:  action,
		Results: results,
	}
}

func (p *parser) parseAssertTrap(pos Pos) *AssertTrap {
	p.expect(ASSERT_TRAP)
	defer p.closeSExpr()

	return &AssertTrap{
		Pos:     pos,
		Command: p.parseCommand(),
		Failure: p.expect(STRING).(string),
	}
}

func (p *parser) parseAssertExhaustion(pos Pos) *AssertExhaustion {
	p.expect(ASSERT_EXHAUSTION)
	defer p.closeSExpr()

	return &AssertExhaustion{
		Pos:     pos,
		Action:  p.parseAction(p.tok.Pos),
		Failure: p.expect(STRING).(string),
	}
}

func (p *parser) parseModuleAssertion(pos Pos) *ModuleAssertion {
	defer p.closeSExpr()

	switch p.tok.Kind {
	case ASSERT_MALFORMED, ASSERT_INVALID, ASSERT_UNLINKABLE:
		// OK
	default:
		panic(p.errorf("expected ASSERT_MALFORMED, ASSERT_INVALID, or ASSERT_UNLINKABLE"))
	}

	kind := p.tok.Kind
	p.scan()

	modulePos := p.tok.Pos
	p.expect('(')
	module := p.parseModule(modulePos, true)

	return &ModuleAssertion{
		Pos:     pos,
		Kind:    kind,
		Module:  module,
		Failure: p.expect(STRING).(string),
	}
}

func (p *parser) parseScriptCommand(pos Pos) *ScriptCommand {
	p.expect(SCRIPT)
	defer p.closeSExpr()

	name, _ := p.maybe(VAR).(string)
	return &ScriptCommand{
		Pos:    pos,
		Name:   name,
		Script: p.parseScript(),
	}
}

func (p *parser) parseInput(pos Pos) *Input {
	p.expect(INPUT)
	defer p.closeSExpr()

	name, _ := p.maybe(VAR).(string)
	return &Input{
		Pos:  pos,
		Name: name,
		Path: p.expect(STRING).(string),
	}
}

func (p *parser) parseOutput(pos Pos) *Output {
	p.expect(OUTPUT)
	defer p.closeSExpr()

	name, _ := p.maybe(VAR).(string)
	return &Output{
		Pos:  pos,
		Name: name,
		Path: p.expect(STRING).(string),
	}
}
