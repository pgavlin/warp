package load

import (
	"bufio"
	"encoding/binary"
	"io"
	"os"

	"github.com/pgavlin/warp/wasm"
	"github.com/pgavlin/warp/wast"
)

func LoadModule(r io.Reader) (*wasm.Module, error) {
	br := bufio.NewReader(r)

	buf, err := br.Peek(4)
	if err != nil {
		return nil, err
	}
	magic := binary.LittleEndian.Uint32(buf)

	if magic == wasm.Magic {
		return wasm.DecodeModule(br)
	}

	syntax, err := wast.ParseModule(wast.NewScanner(br))
	if err != nil {
		return nil, err
	}
	return syntax.Decode()
}

func LoadFile(path string) (*wasm.Module, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return LoadModule(f)
}
