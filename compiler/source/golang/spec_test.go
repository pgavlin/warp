package golang

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"unicode"

	"github.com/pgavlin/warp/wast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var specTest = flag.String("spec", "", "spec test to run")

func TestMain(m *testing.M) {
	flag.Parse()
	os.Exit(m.Run())
}

func TestSpec(t *testing.T) {
	err := os.Mkdir("test", 0700)
	if err != nil && !os.IsExist(err) {
		t.Fatal(err)
	}

	dir, err := ioutil.TempDir("test", "spec")
	require.NoError(t, err)

	t.Logf("Directory: %v", dir)

	any := false
	if *specTest != "" {
		any = prepareScript(t, dir, *specTest)
	} else {
		specDir := filepath.Join("..", "..", "..", "internal", "testdata", "spec")

		entries, err := ioutil.ReadDir(specDir)
		require.NoError(t, err)

		for _, entry := range entries {
			if entry.IsDir() || filepath.Ext(entry.Name()) != ".wast" {
				continue
			}

			if prepareScript(t, dir, filepath.Join(specDir, entry.Name())) {
				any = true
			}
		}
	}
	if !any {
		return
	}

	cmd := exec.Command("go", "test", "./...")
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if !assert.NoError(t, err) {
		t.Logf("%v", string(output))
	}
}

func prepareScript(t *testing.T, root, path string) bool {
	script, err := parseScript(path)
	require.NoError(t, err)

	dir, any, err := compileScript(root, filepath.Base(path), script)
	require.NoError(t, err)

	if !any {
		err = os.RemoveAll(dir)
		require.NoError(t, err)
		return false
	}
	return true
}

func parseScript(path string) (*wast.Script, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return wast.ParseScript(wast.NewScanner(f))
}

func skip(r rune) bool {
	return !unicode.IsLetter(r) && !unicode.IsDigit(r)
}

func testName(name string) string {
	ext := filepath.Ext(name)
	name = name[:len(name)-len(ext)]

	var s strings.Builder
	for i, r := range name {
		switch {
		case i == 0 || i > 0 && skip(rune(name[i-1])):
			s.WriteRune(unicode.ToUpper(r))
		case skip(r):
			// Skip
		default:
			s.WriteRune(r)
		}
	}
	return s.String()
}

func compileScript(root, name string, script *wast.Script) (string, bool, error) {
	dir := filepath.Join(root, name)
	if err := os.Mkdir(dir, 0700); err != nil && !os.IsNotExist(err) {
		return "", false, err
	}

	// Emit the script prologue
	test, err := os.Create(filepath.Join(dir, name+"_test.go"))
	if err != nil {
		return "", false, err
	}
	defer test.Close()

	err = printf(test, `package test

import (
	"math"
	"testing"

	wt "github.com/pgavlin/warp/testing"
	"github.com/pgavlin/warp/wast"
	"github.com/stretchr/testify/require"
)

var _ = math.MaxInt32
var _ = wast.EOF

func Test%s(t *testing.T) {
	env, err := wt.NewEnvironment(nil)
	require.NoError(t, err)

`, testName(name))
	if err != nil {
		return "", false, err
	}

	ignores := map[wast.Pos]bool{}
	for _, pos := range ignore[name] {
		ignores[pos] = true
	}

	any := false
	for _, command := range script.Commands {
		pos := command.CommandPos()
		if ignores[pos] {
			continue
		}

		switch command := command.(type) {
		case wast.Action:
			if err := printf(test, "\t%s.Run(env)\n", compileAction(command)); err != nil {
				return "", false, err
			}
			any = true
		case wast.ModuleCommand:
			_, err := compileModuleCommand(test, dir, command, true)
			if err != nil {
				return "", false, err
			}
			any = true
		case *wast.Register:
			if err := printf(test, "\tenv.Register(%q, %q)\n", command.Export, command.Name); err != nil {
				return "", false, err
			}
			any = true
		case *wast.AssertReturn:
			if err := printf(test, "\tenv.AssertReturn(t, wt.Pos(%d, %d), %s, false, %v)\n", pos.Line, pos.Column, compileAction(command.Action), args(command.Results)); err != nil {
				return "", false, err
			}
			any = true
		case *wast.AssertTrap:
			action := ""
			switch command := command.Command.(type) {
			case wast.Action:
				action = compileAction(command)
			case wast.ModuleCommand:
				a, err := compileModuleAction(test, dir, command)
				if err != nil {
					return "", false, err
				}
				action = a
			default:
				continue
			}
			if err := printf(test, "\tenv.AssertTrap(t, wt.Pos(%d, %d), %s, %q)\n", pos.Line, pos.Column, action, command.Failure); err != nil {
				return "", false, err
			}
			any = true
		case *wast.AssertExhaustion:
			if err := printf(test, "\tenv.AssertExhaustion(t, wt.Pos(%d, %d), %s, %q, false)\n", pos.Line, pos.Column, compileAction(command.Action), command.Failure); err != nil {
				return "", false, err
			}
			any = true
		case *wast.ModuleAssertion:
			if command.Kind == wast.ASSERT_UNLINKABLE {
				name, err := compileModuleCommand(test, dir, command.Module, false)
				if err != nil {
					return "", false, err
				}

				if err := printf(test, "\tenv.AssertUnlinkable(t, wt.Pos(%d, %d), %s, %q, false)\n", pos.Line, pos.Column, name, command.Failure); err != nil {
					return "", false, err
				}
				any = true
			}
		case *wast.ScriptCommand, *wast.Input, *wast.Output:
			return "", false, errors.New("meta commands are not supported")
		default:
			panic("unreachable")
		}
	}

	if err := printf(test, "}\n"); err != nil {
		return "", false, err
	}

	return dir, any, nil
}

func compileModuleCommand(test io.Writer, dir string, command wast.ModuleCommand, instantiate bool) (string, error) {
	m, err := command.Decode()
	if err != nil {
		return "", err
	}

	pos := command.CommandPos()
	name := fmt.Sprintf("Mod_L%vC%v", pos.Line, pos.Column)
	path := filepath.Join(dir, name+".go")

	f, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	if err = CompileModule(f, "test", name, m); err != nil {
		return "", fmt.Errorf("%v: %w", path, err)
	}

	if instantiate {
		if err = printf(test, "\tenv.InstantiateModule(t, wt.Pos(%d, %d), %q, %s)\n", pos.Line, pos.Column, command.ModuleName(), name); err != nil {
			return "", err
		}
	}

	return name, nil
}

func compileModuleAction(test io.Writer, dir string, command wast.ModuleCommand) (string, error) {
	name, err := compileModuleCommand(test, dir, command, false)
	if err != nil {
		return "", err
	}
	pos := command.CommandPos()
	return fmt.Sprintf("wt.InstantiateModule(wt.Pos(%d, %d), %q, %s)", pos.Line, pos.Column, command.ModuleName(), name), nil
}

func compileAction(action wast.Action) string {
	pos := action.CommandPos()
	switch action := action.(type) {
	case *wast.Invoke:
		return fmt.Sprintf("wt.Invoke(wt.Pos(%d, %d), %q, %q, %v)", pos.Line, pos.Column, action.Name, action.Export, args(action.Args))
	case *wast.Get:
		return fmt.Sprintf("wt.Get(wt.Pos(%d, %d), %q, %q)", pos.Line, pos.Column, action.Name, action.Export)
	default:
		panic("unreachable")
	}
}

type args []interface{}

func (a args) String() string {
	var b strings.Builder
	for i, a := range a {
		switch a := a.(type) {
		case int32:
			fmt.Fprintf(&b, "%vint32(%d)", comma(i), a)
		case int64:
			fmt.Fprintf(&b, "%vint64(%d)", comma(i), a)
		case float32:
			fmt.Fprintf(&b, "%vfloat32(%s)", comma(i), f32Const(a))
		case float64:
			fmt.Fprintf(&b, "%vfloat64(%s)", comma(i), f64Const(a))
		case wast.TokenKind:
			fmt.Fprintf(&b, "%vwast.TokenKind(wast.%v)", comma(i), a)
		}
	}
	return b.String()
}

var ignore = map[string][]wast.Pos{
	"const.wast": {
		wast.Pos{Line: 445, Column: 2},
		wast.Pos{Line: 447, Column: 2},
		wast.Pos{Line: 461, Column: 2},
		wast.Pos{Line: 463, Column: 2},
		wast.Pos{Line: 493, Column: 2},
		wast.Pos{Line: 495, Column: 2},
		wast.Pos{Line: 551, Column: 2},
		wast.Pos{Line: 553, Column: 2},
		wast.Pos{Line: 555, Column: 2},
		wast.Pos{Line: 557, Column: 2},
		wast.Pos{Line: 569, Column: 2},
		wast.Pos{Line: 571, Column: 2},
		wast.Pos{Line: 585, Column: 2},
		wast.Pos{Line: 587, Column: 2},
		wast.Pos{Line: 617, Column: 2},
		wast.Pos{Line: 619, Column: 2},
		wast.Pos{Line: 735, Column: 2},
		wast.Pos{Line: 737, Column: 2},
		wast.Pos{Line: 745, Column: 2},
		wast.Pos{Line: 747, Column: 2},
		wast.Pos{Line: 761, Column: 2},
		wast.Pos{Line: 763, Column: 2},
		wast.Pos{Line: 789, Column: 2},
		wast.Pos{Line: 791, Column: 2},
		wast.Pos{Line: 798, Column: 2},
		wast.Pos{Line: 800, Column: 2},
		wast.Pos{Line: 814, Column: 2},
		wast.Pos{Line: 816, Column: 2},
		wast.Pos{Line: 842, Column: 2},
		wast.Pos{Line: 844, Column: 2},
		wast.Pos{Line: 851, Column: 2},
		wast.Pos{Line: 853, Column: 2},
		wast.Pos{Line: 855, Column: 2},
		wast.Pos{Line: 857, Column: 2},
		wast.Pos{Line: 869, Column: 2},
		wast.Pos{Line: 871, Column: 2},
		wast.Pos{Line: 885, Column: 2},
		wast.Pos{Line: 887, Column: 2},
		wast.Pos{Line: 917, Column: 2},
		wast.Pos{Line: 919, Column: 2},
		wast.Pos{Line: 926, Column: 2},
		wast.Pos{Line: 928, Column: 2},
		wast.Pos{Line: 942, Column: 2},
		wast.Pos{Line: 944, Column: 2},
		wast.Pos{Line: 974, Column: 2},
		wast.Pos{Line: 976, Column: 2},
		wast.Pos{Line: 1049, Column: 2},
		wast.Pos{Line: 1051, Column: 2},
		wast.Pos{Line: 1059, Column: 2},
		wast.Pos{Line: 1061, Column: 2},
	},
	"custom.wast": {
		wast.Pos{Line: 14, Column: 2},
		wast.Pos{Line: 84, Column: 2},
	},
	"float_exprs.wast": {
		// These are all quiet -> signaling NaN transitions that occur when the Go compiler
		// folds away multiplication or division by 1 or -1.
		wast.Pos{Line: 2351, Column: 2},
		wast.Pos{Line: 2352, Column: 2},
		wast.Pos{Line: 2353, Column: 2},
		wast.Pos{Line: 2354, Column: 2},
		wast.Pos{Line: 2357, Column: 2},
		wast.Pos{Line: 2358, Column: 2},
		wast.Pos{Line: 2359, Column: 2},
		wast.Pos{Line: 2360, Column: 2},
	},
	"linking.wast": {
		wast.Pos{Line: 136, Column: 2},
		wast.Pos{Line: 137, Column: 2},
		wast.Pos{Line: 139, Column: 2},
		wast.Pos{Line: 141, Column: 2},
		wast.Pos{Line: 142, Column: 2},
		wast.Pos{Line: 144, Column: 2},
		wast.Pos{Line: 146, Column: 2},
		wast.Pos{Line: 147, Column: 2},
		wast.Pos{Line: 148, Column: 2},
		wast.Pos{Line: 149, Column: 2},
		wast.Pos{Line: 152, Column: 2},
		wast.Pos{Line: 184, Column: 2},
		wast.Pos{Line: 185, Column: 2},
		wast.Pos{Line: 187, Column: 2},
		wast.Pos{Line: 188, Column: 2},
		wast.Pos{Line: 190, Column: 2},
		wast.Pos{Line: 225, Column: 2},
		wast.Pos{Line: 236, Column: 2},
		wast.Pos{Line: 248, Column: 2},
	},
}
