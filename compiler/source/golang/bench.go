//go:build ignore
// +build ignore

package main

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"text/template"

	"github.com/pgavlin/warp/bench/flate"
	"github.com/pgavlin/warp/compiler/source/golang"
	"github.com/pgavlin/warp/wasm"
)

type benchmark struct {
	name     string
	modules  map[string]*wasm.Module
	template string
	args     interface{}
}

func useRawPointers() bool {
	switch runtime.GOOS + "/" + runtime.GOARCH {
	case "android/amd64", "android/arm64", "darwin/amd64", "darwin/arm64", "dragonfly/amd64", "freebsd/amd64", "freebsd/arm64", "illumos/amd64", "ios/amd64", "ios/arm64", "linux/amd64", "linux/arm64", "linux/mips64le", "linux/ppc64le", "linux/riscv64", "netbsd/amd64", "netbsd/arm64", "openbsd/amd64", "openbsd/arm64", "solaris/amd64":
		return true
	}
	return false
}

func (b *benchmark) compileModule(root, name string, module *wasm.Module) error {
	mt, err := os.Create(filepath.Join(root, name+"_module_test.go"))
	if err != nil {
		return err
	}
	defer mt.Close()

	return golang.CompileModule(mt, "bench", name, module, &golang.Options{
		UseRawPointers:    useRawPointers(),
		NoInternalThreads: true,
	})
}

func (b *benchmark) compile(root string) error {
	for name, mod := range b.modules {
		if err := b.compileModule(root, name, mod); err != nil {
			return err
		}
	}

	t, err := os.Create(filepath.Join(root, b.name+"_test.go"))
	if err != nil {
		return err
	}
	defer t.Close()

	tmpl, err := template.New(b.name + "_test.go").Parse(b.template)
	if err != nil {
		return err
	}

	return tmpl.Execute(t, b.args)
}

func main() {
	// create a directory to hold the compiled code
	dir, err := os.MkdirTemp("test", "bench")
	if err != nil {
		log.Fatal(err)
	}

	// compile tests
	benchmarks := []benchmark{
		{
			name: "Flate",
			modules: map[string]*wasm.Module{
				"flate": flate.Module,
			},
			template: `package bench

import (
	"bytes"
	"io"
	"testing"

	"github.com/pgavlin/warp/bench/data"
	"github.com/pgavlin/warp/wasi"
)

func BenchmarkFlate(b *testing.B) {
	for i := 0; i < b.N; i++ {
		err := wasi.Run("flate", Flate, &wasi.RunOptions{
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
`,
		},
	}

	for _, b := range benchmarks {
		if err := b.compile(dir); err != nil {
			log.Fatalf("compiling %v: %v", b.name, err)
		}
	}

	args := []string{"test", "-bench", "."}
	args = append(args, os.Args[1:]...)
	cmd := exec.Command("go", args...)
	cmd.Env = os.Environ()
	cmd.Dir = dir
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			os.Exit(exit.ExitCode())
		}
	}
}
