//go:build ignore
// +build ignore

package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"

	"github.com/pgavlin/warp/wasm"
	"github.com/pgavlin/warp/wast"
)

func buildGo(opts []string) error {
	opts = append([]string{"build"}, opts...)
	cmd := exec.Command("go", opts...)
	cmd.Env = append(append([]string{}, os.Environ()...), "GOOS=js", "GOARCH=wasm")
	return cmd.Run()
}

func buildRustWASI(opts []string) error {
	opts = append([]string{"build", "--target=wasm32-wasi", "--release"}, opts...)
	cmd := exec.Command("cargo", opts...)
	return cmd.Run()
}

func buildWAST(opts []string) error {
	wastFlags := flag.NewFlagSet("wast", flag.ContinueOnError)
	out := wastFlags.String("out", "", "output path")
	if err := wastFlags.Parse(opts); err != nil {
		return err
	}

	if *out == "" {
		return fmt.Errorf("-out must be specified")
	}

	in, err := os.Open(wastFlags.Arg(0))
	if err != nil {
		return err
	}

	syntax, err := wast.ParseModule(wast.NewScanner(in))
	if err != nil {
		return err
	}
	module, err := syntax.Decode()
	if err != nil {
		return err
	}

	o, err := os.Create(*out)
	if err != nil {
		return err
	}
	return wasm.EncodeModule(o, module)
}

func main() {
	lang := flag.String("lang", "", "the language to build")
	flag.Parse()

	var builder func([]string) error
	switch *lang {
	case "go":
		builder = buildGo
	case "rust-wasi":
		builder = buildRustWASI
	case "wast":
		builder = buildWAST
	default:
		fmt.Fprintf(os.Stderr, "unknown language '%s'; supported languages are 'rust-wasi' and 'wast'\n", *lang)
		os.Exit(-1)
	}

	if err := builder(flag.Args()); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(-1)
	}
}
