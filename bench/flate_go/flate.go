// Package flate_go provides a WASM module that implements a simple compress -> decompress
// pipeline. The module is a Go command that compresses data from stdin, then
// decompresses the compressed data and writes the result to stdout.

package flate_go

import (
	"bytes"
	_ "embed"

	"github.com/pgavlin/warp/wasm"
)

//go:generate go run ../build.go -lang=go -- -o ./flate/flate.wasm ./flate/main.go
//go:embed flate/flate.wasm
var flate []byte

// Module is the module definition for the `flate` command.
var Module = wasm.MustDecode(bytes.NewReader(flate))
