package interpreter

import (
	"bytes"
	"io"
	"testing"

	"github.com/pgavlin/warp/bench/data"
	"github.com/pgavlin/warp/bench/flate"
	"github.com/pgavlin/warp/bench/flate_go"
	"github.com/pgavlin/warp/go_wasm_exec"
	"github.com/pgavlin/warp/wasi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFlate(t *testing.T) {
	var stdout bytes.Buffer
	err := wasi.Run("flate", NewModuleDefinition(flate.Module), &wasi.RunOptions{
		Options: &wasi.Options{
			Stdin:  bytes.NewReader(data.Enwik8[:1<<20]),
			Stdout: &stdout,
		},
	})
	require.NoError(t, err)
	assert.Equal(t, data.Enwik8[:1<<20], stdout.Bytes())
}

func TestFlateGo(t *testing.T) {
	var stdout bytes.Buffer
	err := go_wasm_exec.Run("flate", NewModuleDefinition(flate_go.Module), &go_wasm_exec.Options{
		Stdin:  bytes.NewReader(data.Enwik8[:1<<20]),
		Stdout: &stdout,
	})
	require.NoError(t, err)
	assert.Equal(t, data.Enwik8[:1<<20], stdout.Bytes())
}

func BenchmarkFlate(b *testing.B) {
	for i := 0; i < b.N; i++ {
		err := wasi.Run("flate", NewModuleDefinition(flate.Module), &wasi.RunOptions{
			Options: &wasi.Options{
				Stdin:  bytes.NewReader(data.Enwik8[:1<<16]),
				Stdout: io.Discard,
			},
		})
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkFlateGo(b *testing.B) {
	for i := 0; i < b.N; i++ {
		err := go_wasm_exec.Run("flate", NewModuleDefinition(flate_go.Module), &go_wasm_exec.Options{
			Stdin:  bytes.NewReader(data.Enwik8[:1<<16]),
			Stdout: io.Discard,
		})
		if err != nil {
			b.Fatal(err)
		}
	}
}
