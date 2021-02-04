package golang

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pgavlin/warp/wasm"
	"github.com/pgavlin/warp/wasm/code"
	"github.com/pgavlin/warp/wast"
)

func testModule(t *testing.T, def *wasm.Module, entrypoint string, expected ...uint64) {
	testT := template.Must(template.New("module_test.go").Parse(`package test

import (
	"testing"

	"github.com/pgavlin/warp/exec"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/assert"
)

func TestCompiledModule(t *testing.T) {
	store := exec.NewStore(exec.MapResolver{
		"test": Test,
	})

	mod, err := store.InstantiateModule("test")
	require.NoError(t, err)

	main, err := mod.GetFunction("{{.Entrypoint}}")
	require.NoError(t, err)

	thread := exec.NewThread(0)

	expected := {{printf "%#v" .Expected}}
	returns := make([]uint64, len(expected))
	main.UncheckedCall(&thread, nil, returns)
	assert.Equal(t, expected, returns)

	thread.Close()
}
`))

	if expected == nil {
		expected = []uint64{}
	}

	var source bytes.Buffer
	err := CompileModule(&source, "test", "test", def)
	require.NoError(t, err)

	var test bytes.Buffer
	err = testT.Execute(&test, map[string]interface{}{
		"Entrypoint": entrypoint,
		"Expected":   expected,
	})
	require.NoError(t, err)

	err = os.Mkdir("test", 0700)
	if !os.IsExist(err) {
		require.NoError(t, err)
	}

	dir, err := ioutil.TempDir("test", "source_test")
	require.NoError(t, err)

	err = ioutil.WriteFile(filepath.Join(dir, "module.go"), source.Bytes(), 0600)
	require.NoError(t, err)

	err = ioutil.WriteFile(filepath.Join(dir, "module_test.go"), test.Bytes(), 0600)
	require.NoError(t, err)

	cmd := exec.Command("go", "test", ".")
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if !assert.NoError(t, err) {
		t.Logf("Output: %v", string(output))
		t.Logf("Directory: %v", dir)
	}
}

func TestEmptyFunction(t *testing.T) {
	testModule(t, EmptyFunction, "main")
}

func TestFibRecursive(t *testing.T) {
	testModule(t, FibRecursive, "app_main", 9227465)
}

func expr(instrs ...code.Instruction) []byte {
	var buf bytes.Buffer
	if err := code.Encode(&buf, instrs); err != nil {
		panic(fmt.Errorf("encoding expression: %w", err))
	}
	return buf.Bytes()
}

func i32Const(v int32) []byte {
	return expr(code.I32Const(v), code.End())
}

func mustParseModule(source string) *wasm.Module {
	syntax, err := wast.ParseModule(wast.NewScanner(strings.NewReader(source)))
	if err != nil {
		panic(err)
	}
	m, err := syntax.Decode()
	if err != nil {
		panic(err)
	}
	return m
}

var EmptyFunction = &wasm.Module{
	Version: 1,

	Types: &wasm.SectionTypes{
		Entries: []wasm.FunctionSig{
			{Form: 0x60, ParamTypes: []wasm.ValueType{}, ReturnTypes: []wasm.ValueType{}},
		},
	},
	Function: &wasm.SectionFunctions{
		Types: []uint32{0},
	},
	Export: &wasm.SectionExports{
		Entries: []wasm.ExportEntry{
			{FieldStr: "main", Kind: wasm.ExternalFunction, Index: 0},
		},
	},
	Code: &wasm.SectionCode{
		Bodies: []wasm.FunctionBody{{Code: []byte{code.OpReturn, code.OpEnd}}},
	},
}

var FibRecursive = &wasm.Module{
	Version: 1,

	Types: &wasm.SectionTypes{
		Entries: []wasm.FunctionSig{
			{Form: 0x60, ParamTypes: []wasm.ValueType{wasm.ValueTypeI32}, ReturnTypes: []wasm.ValueType{wasm.ValueTypeI32}},
			{Form: 0x60, ParamTypes: []wasm.ValueType{}, ReturnTypes: []wasm.ValueType{wasm.ValueTypeI32}},
		},
	},
	Function: &wasm.SectionFunctions{
		Types: []uint32{0, 1},
	},
	Table: &wasm.SectionTables{
		Entries: []wasm.Table{
			{ElementType: wasm.ElemTypeAnyFunc, Limits: wasm.ResizableLimits{Flags: 1, Initial: 1, Maximum: 1}},
		},
	},
	Memory: &wasm.SectionMemories{
		Entries: []wasm.Memory{
			{Limits: wasm.ResizableLimits{Initial: 16}},
		},
	},
	Global: &wasm.SectionGlobals{
		Globals: []wasm.GlobalEntry{
			{Type: wasm.GlobalVar{Type: wasm.ValueTypeI32, Mutable: true}, Init: i32Const(1048576)},
			{Type: wasm.GlobalVar{Type: wasm.ValueTypeI32}, Init: i32Const(1048576)},
			{Type: wasm.GlobalVar{Type: wasm.ValueTypeI32}, Init: i32Const(1048576)},
		},
	},
	Export: &wasm.SectionExports{
		Entries: []wasm.ExportEntry{
			{FieldStr: "memory", Kind: wasm.ExternalMemory, Index: 0},
			{FieldStr: "fib", Kind: wasm.ExternalFunction, Index: 0},
			{FieldStr: "app_main", Kind: wasm.ExternalFunction, Index: 1},
			{FieldStr: "__data_end", Kind: wasm.ExternalGlobal, Index: 1},
			{FieldStr: "__heap_base", Kind: wasm.ExternalGlobal, Index: 2},
		},
	},
	Code: &wasm.SectionCode{
		Bodies: []wasm.FunctionBody{
			{
				Locals: []wasm.LocalEntry{
					{Count: 1, Type: wasm.ValueTypeI32},
				},
				Code: expr(
					code.I32Const(1),
					code.LocalSet(1),
					code.Block(), // label = @1
					code.LocalGet(0),
					code.I32Const(-1),
					code.I32Add(),
					code.LocalTee(0),
					code.I32Const(2),
					code.I32LtU(),
					code.BrIf(0), // @1
					code.I32Const(0),
					code.LocalSet(1),
					code.Loop(), // label = @2
					code.LocalGet(0),
					code.Call(0), // fib
					code.LocalGet(1),
					code.I32Add(),
					code.LocalSet(1),
					code.LocalGet(0),
					code.I32Const(-2),
					code.I32Add(),
					code.LocalTee(0),
					code.I32Const(1),
					code.I32GtU(),
					code.BrIf(0), // @2
					code.End(),
					code.LocalGet(1),
					code.I32Const(1),
					code.I32Add(),
					code.LocalSet(1),
					code.End(),
					code.LocalGet(1),
					code.End(),
				),
			},
			{
				Code: expr(
					code.I32Const(35),
					code.Call(0), // fib
					code.End(),
				),
			},
		},
	},
}
