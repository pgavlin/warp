package main

import (
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"

	"github.com/spf13/cobra"

	"github.com/pgavlin/warp/cmd/warp/compile"
	"github.com/pgavlin/warp/cmd/warp/dump"
	"github.com/pgavlin/warp/cmd/warp/run"
	"github.com/pgavlin/warp/wasi"
)

var version = "<unknown>"

func configureCLI() *cobra.Command {
	var cpuProfile string
	var memProfile string

	rootCommand := &cobra.Command{
		Use:           "warp",
		Short:         "warp WebAssembly suite",
		Long:          "warp - a tool suite for WebAssembly",
		Version:       version,
		SilenceErrors: true,
		SilenceUsage:  true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if cpuProfile != "" {
				f, err := os.Create(cpuProfile)
				if err != nil {
					return err
				}
				pprof.StartCPUProfile(f)
			}
			return nil
		},
		PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
			if cpuProfile != "" {
				pprof.StopCPUProfile()
			}

			if memProfile != "" {
				f, err := os.Create(memProfile)
				if err != nil {
					return err
				}
				runtime.GC()
				pprof.WriteHeapProfile(f)
			}

			return nil
		},
	}

	rootCommand.AddCommand(compile.Command())
	rootCommand.AddCommand(dump.Command())
	rootCommand.AddCommand(run.Command())

	rootCommand.PersistentFlags().StringVar(&cpuProfile, "cpu", "", "emit Go CPU profile data to this path")
	rootCommand.PersistentFlags().StringVar(&memProfile, "mem", "", "emit Go memory profile data to this path")

	rootCommand.PersistentFlags().MarkHidden("cpu")
	rootCommand.PersistentFlags().MarkHidden("mem")

	return rootCommand
}

func main() {
	rootCommand := configureCLI()

	if err := rootCommand.Execute(); err != nil {
		if exit, ok := err.(*wasi.ExitError); ok {
			os.Exit(exit.Code())
		}

		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
