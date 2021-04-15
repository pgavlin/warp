package wasi

import (
	"fmt"
	"io"
	"os"
	"runtime/pprof"
	"strings"

	"github.com/pgavlin/warp/exec"
	"github.com/pgavlin/warp/wasm"
)

type Preopen struct {
	FSPath  string
	Path    string
	Rights  Rights
	Inherit Rights
}

type Options struct {
	Env  map[string]string
	Args []string

	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer

	FS      FS
	Preopen []Preopen
}

// SnapshotPreview1 is the definition for the module canonically named "wasi_snapshot_preview1".
var SnapshotPreview1 exec.ModuleDefinition = wasiSnapshotPreview1Definition(0)

type ExitError struct {
	code int
}

func (e *ExitError) Code() int {
	return e.code
}

func (e *ExitError) Error() string {
	return fmt.Sprintf("exit status %d", e.code)
}

type moduleEventHandler struct {
	options *Options
}

func NewModuleEventHandler(options *Options) exec.ModuleEventHandler {
	return &moduleEventHandler{options: options}
}

func (h *moduleEventHandler) ModuleAllocated(m exec.AllocatedModule) (err error) {
	wasi, ok := m.(*allocatedWasiSnapshotPreview1)
	if !ok {
		return nil
	}

	wasi.impl, err = newImpl(wasi.wasiSnapshotPreview1, h.options)
	return err
}

func (h *moduleEventHandler) ModuleInstantiated(m exec.Module) error {
	if initialize, err := m.GetFunction("_initialize"); err == nil && initialize.GetSignature().Equals(wasm.FunctionSig{}) {
		t := exec.NewThread(0)
		initialize.UncheckedCall(&t, nil, nil)
	}
	return nil
}

type Resolver struct {
	inner exec.ModuleResolver
}

func NewResolver(inner exec.ModuleResolver) Resolver {
	if inner == nil {
		inner = exec.MapResolver{}
	}
	return Resolver{inner: inner}
}

func (r Resolver) ResolveModule(name string) (exec.ModuleDefinition, error) {
	switch name {
	case "wasi_snapshot_preview1":
		return SnapshotPreview1, nil
	default:
		return r.inner.ResolveModule(name)
	}
}

type RunOptions struct {
	*Options

	Debug    bool
	Trace    io.Writer
	Resolver exec.ModuleResolver
}

func Run(name string, def exec.ModuleDefinition, runOptions *RunOptions) error {
	options, resolver := (*Options)(nil), exec.ModuleResolver(nil)
	if runOptions != nil {
		options = runOptions.Options
		if runOptions.Resolver != nil {
			resolver = runOptions.Resolver
		}
	}
	if options == nil {
		options = &Options{}
	}
	options.Args = append([]string{name}, options.Args...)

	store := exec.NewStore(NewResolver(resolver), NewModuleEventHandler(options))

	mod, err := store.InstantiateModuleDefinition("", def)
	if err != nil {
		return err
	}

	start, err := mod.GetFunction("_start")
	if err != nil {
		return err
	}

	if !start.GetSignature().Equals(wasm.FunctionSig{}) {
		return fmt.Errorf("_start must not accept or return parameters")
	}

	code := run(start, runOptions.Debug, runOptions.Trace)
	if code != 0 {
		return &ExitError{code: code}
	}
	return nil
}

func Main(def exec.ModuleDefinition) {
	if err := MainErr(def); err != nil {
		if exit, ok := err.(*ExitError); ok {
			os.Exit(exit.code)
		}
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(-1)
	}
}

func MainErr(def exec.ModuleDefinition) error {
	if profile := os.Getenv("WASI_CPU_PROFILE"); profile != "" {
		f, err := os.Create(profile)
		if err != nil {
			return fmt.Errorf("failed to open CPU profile: %w", err)
		}
		defer f.Close()

		if err = pprof.StartCPUProfile(f); err != nil {
			return fmt.Errorf("failed to start CPU profile: %w", err)
		}
		defer pprof.StopCPUProfile()
	}

	env := map[string]string{}
	for _, v := range os.Environ() {
		kvp := strings.SplitN(v, "=", 2)
		env[kvp[0]] = kvp[1]
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	err = Run(os.Args[0], def, &RunOptions{
		Options: &Options{
			Env:  env,
			Args: os.Args[1:],
			Preopen: []Preopen{
				{
					Path:    ".",
					FSPath:  cwd,
					Rights:  AllRights,
					Inherit: AllRights,
				},
				{
					Path:    "/",
					FSPath:  "/",
					Rights:  AllRights,
					Inherit: AllRights,
				},
			},
		},
	})
	if err != nil {
		if _, ok := err.(*ExitError); ok {
			return err
		}
		return fmt.Errorf("error running program: %w", err)
	}
	return nil
}

func run(start exec.Function, debug bool, trace io.Writer) (code int) {
	var thread exec.Thread
	if debug || trace != nil {
		thread = exec.NewDebugThread(trace, 0)
	} else {
		thread = exec.NewThread(0)
	}

	defer func() {
		if x := recover(); x != nil {
			if ec, ok := x.(TrapExit); ok {
				code = int(ec)
				return
			}
			panic(x)
		}
	}()

	start.UncheckedCall(&thread, nil, nil)
	thread.Close()
	return 0
}
