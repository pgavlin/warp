package testing

import (
	"fmt"
	"math"
	"os"
	"testing"

	"github.com/pgavlin/warp/exec"
	"github.com/pgavlin/warp/wasm"
	"github.com/pgavlin/warp/wasm/validate"
	"github.com/pgavlin/warp/wast"
)

type Loader func(m *wasm.Module) (exec.ModuleDefinition, error)

type Environment struct {
	modules map[string]exec.Module

	loader   Loader
	resolver exec.MapResolver
	store    *exec.Store

	ignore map[string]bool
}

func RunScript(t *testing.T, loader Loader, path string, strict bool, ignore []string) {
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("opening script: %v", err)
	}
	defer f.Close()

	s, err := wast.ParseScript(wast.NewScanner(f))
	if err != nil {
		t.Fatalf("parsing script: %v", err)
	}

	env, err := NewEnvironment(loader)
	if err != nil {
		t.Fatalf("creating environment: %v", err)
	}

	env.RunScript(t, s, strict, ignore)
}

func NewEnvironment(loader Loader) (*Environment, error) {
	resolver := exec.MapResolver{
		"spectest": SpecTest,
	}
	store := exec.NewStore(resolver)
	spectest, err := store.InstantiateModule("spectest")
	if err != nil {
		return nil, err
	}
	return &Environment{
		modules:  map[string]exec.Module{"spectest": spectest},
		loader:   loader,
		resolver: resolver,
		store:    store,
	}, nil
}

func (e *Environment) instantiateModule(name string, definition exec.ModuleDefinition) error {
	m, err := e.store.InstantiateModuleDefinition(name, definition)
	if err != nil {
		return err
	}

	e.modules[""], e.modules[name] = m, m
	return nil
}

func (e *Environment) InstantiateModule(t *testing.T, pos *wast.Pos, name string, definition exec.ModuleDefinition) {
	if err := e.instantiateModule(name, definition); err != nil {
		e.errorf(t, posOrDefault(pos), "unexpected error: %v", err)
	}
}

func (e *Environment) Register(export, module string) {
	e.store.RegisterModule(export, e.modules[module])
}

func (e *Environment) AssertReturn(t *testing.T, pos *wast.Pos, action Action, strict bool, expected ...interface{}) {
	p := posOrDefault(pos)
	results, err := e.runAction(action)
	if err != nil {
		e.errorf(t, action.Pos(), "%v", err)
		return
	}
	if len(results) != len(expected) {
		e.errorf(t, p, "assert_return: expected %v results, got %v", len(expected), len(results))
	} else {
		ok := true
		for i, v := range expected {
			if !isEqual(v, results[i], strict) {
				ok = false
				break
			}
		}
		if !ok {
			e.errorf(t, p, "assert_return: expected %v, got %v", expected, results)
		}
	}
}

func (e *Environment) AssertTrap(t *testing.T, pos *wast.Pos, action Action, failure string) {
	p := posOrDefault(pos)

	_, err := e.runAction(action)
	if err == nil {
		e.errorf(t, p, "assert_trap: action did not trap")
	} else if err.Error() != failure {
		e.errorf(t, p, "assert_trap: expected %v, got %v", failure, err.Error())
	}
}

func (e *Environment) AssertExhaustion(t *testing.T, pos *wast.Pos, action Action, failure string, strict bool) {
	p := posOrDefault(pos)

	_, err := e.runAction(action)
	if err == nil {
		e.errorf(t, p, "assert_exhaustion: action did not trap")
	} else if strict && err.Error() != failure {
		e.errorf(t, p, "assert_exhaustion: expected %v, got %v", failure, err.Error())
	}
}

func (e *Environment) AssertUnlinkable(t *testing.T, pos *wast.Pos, definition exec.ModuleDefinition, failure string, strict bool) {
	p := posOrDefault(pos)
	err := e.instantiateModule("", definition)
	if err == nil {
		e.errorf(t, p, "assert_unlinkable: module linked successfully")
	} else if strict && err.Error() != failure {
		e.errorf(t, p, "assert_unlinkable: expected %v, got %v", failure, err.Error())
	}
}

func (e *Environment) RunScript(t *testing.T, script *wast.Script, strict bool, ignore []string) {
	e.ignore = map[string]bool{}
	for _, ignore := range ignore {
		e.ignore[ignore] = true
	}

	for _, command := range script.Commands {
		e.RunCommand(t, command, strict)
	}
}

func (e *Environment) fatalf(t *testing.T, pos wast.Pos, msg string, args ...interface{}) {
	t.Fatalf("%v,%v: %s", pos.Line, pos.Column, fmt.Sprintf(msg, args...))
}

func (e *Environment) errorf(t *testing.T, pos wast.Pos, msg string, args ...interface{}) {
	msg = fmt.Sprintf("%v,%v: %s", pos.Line, pos.Column, fmt.Sprintf(msg, args...))
	if e.ignore[msg] {
		t.Logf("ignored: %s", msg)
	} else {
		t.Error(msg)
	}
}

func (e *Environment) RunCommand(t *testing.T, command wast.Command, strict bool) {
	pos := command.CommandPos()

	switch command := command.(type) {
	case *wast.Register:
		e.store.RegisterModule(command.Export, e.modules[command.Name])
	case *wast.AssertReturn:
		e.AssertReturn(t, &pos, e.action(command.Action), strict, command.Results...)
	case *wast.AssertTrap:
		e.AssertTrap(t, &pos, e.action(command.Command), command.Failure)
	case *wast.AssertExhaustion:
		e.AssertExhaustion(t, &pos, e.action(command.Action), command.Failure, strict)
	case *wast.ModuleAssertion:
		switch command.Kind {
		case wast.ASSERT_MALFORMED:
			_, err := command.Module.Decode()
			if err == nil {
				e.errorf(t, pos, "assert_malformed: module was not malformed")
			} else if strict && err.Error() != command.Failure {
				e.errorf(t, pos, "assert_malformed: expected %v, got %v", command.Failure, err.Error())
			}
		case wast.ASSERT_INVALID:
			m, err := command.Module.Decode()
			if err != nil {
				if strict {
					e.errorf(t, pos, "assert_invalid: module was malformed")
				}
				return
			}

			err = validate.ValidateModule(m, true)
			if err == nil {
				e.errorf(t, pos, "assert_invalid: module was not invalid")
			} else if strict && err.Error() != command.Failure {
				e.errorf(t, pos, "assert_invalid: expected %v, got %v", command.Failure, err.Error())
			}
		case wast.ASSERT_UNLINKABLE:
			def, err := (&decodeAndInstantiate{ModuleCommand: command.Module}).decode(e)
			if err != nil {
				e.errorf(t, pos, "assert_unlinkable: module failed to decode (%v)", err)
			} else {
				e.AssertUnlinkable(t, &pos, def, command.Failure, strict)
			}
		}
	case *wast.ScriptCommand, *wast.Input, *wast.Output:
		e.fatalf(t, pos, "meta commands are not supported")
	default:
		if _, err := e.runAction(e.action(command)); err != nil {
			e.errorf(t, pos, "unexpected error: %v", err)
		}
	}
}

func (e *Environment) action(command wast.Command) Action {
	switch command := command.(type) {
	case wast.ModuleCommand:
		return &decodeAndInstantiate{ModuleCommand: command}
	case *wast.Invoke:
		return &invoke{Invoke: *command}
	case *wast.Get:
		return &get{Get: *command}
	default:
		panic("unreachable")
	}
}

func (e *Environment) runAction(action Action) (results []interface{}, err error) {
	defer func() {
		if x := recover(); x != nil {
			if e, ok := x.(error); ok {
				err = e
			}
		}
	}()
	return action.Run(e)
}

func isEqual(expected, actual interface{}, strict bool) bool {
	const nanMask32 = 0x7fffffff
	const canonicalNaN32 = 0x7fc00000
	const nanMask64 = 0x7fffffffffffffff
	const canonicalNaN64 = 0x7ff8000000000000

	switch expected := expected.(type) {
	case wast.TokenKind:
		switch expected {
		case wast.NAN_ARITHMETIC:
			switch actual := actual.(type) {
			case float32:
				return math.IsNaN(float64(actual)) && (!strict || math.Float32bits(actual)&nanMask32 != canonicalNaN32)
			case float64:
				return math.IsNaN(actual) && (!strict || math.Float64bits(actual)&nanMask64 != canonicalNaN64)
			default:
				return false
			}
		case wast.NAN_CANONICAL:
			switch actual := actual.(type) {
			case float32:
				return math.Float32bits(actual)&nanMask32 == canonicalNaN32
			case float64:
				return math.Float64bits(actual)&nanMask64 == canonicalNaN64
			default:
				return false
			}
		default:
			return false
		}
	case float32:
		f, ok := actual.(float32)
		if !ok {
			return false
		}
		if math.IsNaN(float64(f)) && math.IsNaN(float64(expected)) {
			return true
		}
		return f == expected
	case float64:
		f, ok := actual.(float64)
		if !ok {
			return false
		}
		if math.IsNaN(f) && math.IsNaN(expected) {
			return true
		}
		return f == expected
	default:
		return actual == expected
	}
}
