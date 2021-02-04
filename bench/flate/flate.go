// Package flate provides a WASM module that implements a simple compress -> decompress
// pipeline. The module is a WASI command that compresses data from stdin, then
// decompresses the compressed data and writes the result to stdout.

package flate

import (
	"bytes"
	_ "embed"

	"github.com/pgavlin/warp/wasm"
)

//go:generate go run ../build.go -lang=rust-wasi
//go:embed target/wasm32-wasi/release/flate.wasm
var flate []byte

// Module is the module definition for the `flate` command.
var Module = wasm.MustDecode(bytes.NewReader(flate))
