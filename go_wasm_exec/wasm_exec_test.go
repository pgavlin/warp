package go_wasm_exec

import (
	"bytes"
	"os"
	"testing"

	"github.com/pgavlin/warp/exec"
	"github.com/pgavlin/warp/interpreter"
	"github.com/pgavlin/warp/wasm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

func TestHello(t *testing.T) {
	def, err := loadModule("./testdata/hello.wasm")
	require.NoError(t, err)

	var stdout bytes.Buffer
	err = Run("hello", def, &Options{
		Stdout: &stdout,
	})
	require.NoError(t, err)
	assert.Equal(t, []byte("Hello, WebAssembly!\n"), stdout.Bytes())
}
