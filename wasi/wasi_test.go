package wasi

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pgavlin/warp/exec"
	"github.com/pgavlin/warp/interpreter"
	"github.com/pgavlin/warp/wasm"
	"github.com/pgavlin/warp/wast"
)

func parseModule(path string) (exec.ModuleDefinition, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	syntax, err := wast.ParseModule(wast.NewScanner(f))
	if err != nil {
		return nil, err
	}

	module, err := syntax.Decode()
	if err != nil {
		return nil, err
	}

	return interpreter.NewModuleDefinition(module), nil
}

func loadModule(path string) (exec.ModuleDefinition, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	module, err := wasm.DecodeModule(f)
	if err != nil {
		return nil, err
	}

	return interpreter.NewModuleDefinition(module), nil
}

func TestHelloWorld(t *testing.T) {
	def, err := parseModule("./testdata/hello_world.wast")
	require.NoError(t, err)

	var buf bytes.Buffer
	err = Run("hello_world", def, &RunOptions{
		Options: &Options{
			Stdout: &buf,
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "hello world\n", buf.String())
}

func TestHelloWorldFile(t *testing.T) {
	def, err := parseModule("./testdata/hello_world_file.wast")
	require.NoError(t, err)

	dir, err := ioutil.TempDir("", "hwfile")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	err = Run("hello_world_file", def, &RunOptions{
		Options: &Options{
			Preopen: []Preopen{
				{FSPath: dir, Path: dir, Rights: AllRights, Inherit: AllRights},
			},
		},
	})
	require.NoError(t, err)

	actual, err := os.ReadFile(filepath.Join(dir, "hello.txt"))
	require.NoError(t, err)

	assert.Equal(t, "hello world\n", string(actual))
}

func TestHello(t *testing.T) {
	def, err := loadModule("./testdata/hello.wasm")
	require.NoError(t, err)

	var buf bytes.Buffer
	err = Run("hello", def, &RunOptions{
		Options: &Options{
			Stdout: &buf,
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "Hello, world!\n", buf.String())
}

func TestDemo(t *testing.T) {
	def, err := loadModule("./testdata/demo.wasm")
	require.NoError(t, err)

	dir, err := ioutil.TempDir("", "demo")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	const str = "Hello, world!\n"

	err = os.WriteFile(filepath.Join(dir, "hello.txt"), []byte(str), 0600)
	require.NoError(t, err)

	var stdout, stderr bytes.Buffer
	err = Run("demo", def, &RunOptions{
		Options: &Options{
			Args:   []string{"hello.txt", "world.txt"},
			Stdout: &stdout,
			Stderr: &stderr,
			Preopen: []Preopen{
				{Path: ".", FSPath: dir, Rights: AllRights, Inherit: AllRights},
			},
		},
	})
	require.NoError(t, err)

	assert.Equal(t, "", stdout.String())
	assert.Equal(t, "", stderr.String())

	actual, err := os.ReadFile(filepath.Join(dir, "world.txt"))
	require.NoError(t, err)

	assert.Equal(t, str, string(actual))
}
