package compile

import (
	"bufio"
	"errors"
	"io"
	"os"
	"path/filepath"

	"github.com/pgavlin/warp/compiler/source/golang"
	"github.com/pgavlin/warp/load"
	"github.com/pgavlin/warp/wasm"
	"github.com/spf13/cobra"
)

func Command() *cobra.Command {
	var packageName string
	var isCommand bool
	var outputPath string
	var format bool
	var useRawPointers bool
	var noInternalThreads bool

	command := &cobra.Command{
		Use:   "compile",
		Short: "Compile a WebAssembly module to Go source",
		Long:  "Compile a WebAssembly module to Go source",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return errors.New("expected exactly one argument")
			}

			if isCommand != (packageName == "") {
				return errors.New("exactly one of --pkg and --cmd must be specified")
			}

			mod, err := load.LoadFile(args[0])
			if err != nil {
				return err
			}

			baseName := filepath.Base(args[0])
			baseName = baseName[:len(baseName)-len(filepath.Ext(baseName))]

			var dest io.Writer
			switch outputPath {
			case "":
				f, err := os.Create(baseName + ".go")
				if err != nil {
					return err
				}
				defer f.Close()

				dest = f
			case "-":
				dest = os.Stdout
			default:
				f, err := os.Create(outputPath)
				if err != nil {
					return err
				}
				defer f.Close()

				dest = f
			}

			w := bufio.NewWriter(dest)
			defer w.Flush()
			dest = w

			if format {
				dest = golang.Format(dest)
			}

			modName := ""
			if names, err := mod.Names(); err == nil {
				for _, entry := range names.Entries {
					if m, ok := entry.(*wasm.ModuleNameSubsection); ok {
						modName = m.Name
					}
				}
			}
			if modName == "" {
				modName = baseName
			}

			options := golang.Options{
				UseRawPointers:    useRawPointers,
				NoInternalThreads: noInternalThreads,
			}
			if !isCommand {
				return golang.CompileModule(dest, packageName, modName, mod, &options)
			}

			return golang.CompileCommand(dest, modName, mod, &options)
		},
	}

	command.PersistentFlags().StringVar(&packageName, "pkg", "", "the name of the generated package")
	command.PersistentFlags().BoolVarP(&isCommand, "cmd", "c", true, "true to automatically detect WASI commands")
	command.PersistentFlags().StringVarP(&outputPath, "out", "o", "", "the path for the output file. Defaults to the name of the input file + '.go'")
	command.PersistentFlags().BoolVarP(&format, "format", "f", false, "true to gofmt the generated source code")
	command.PersistentFlags().BoolVar(&useRawPointers, "raw-pointers", false, "true to compile loads and stores to raw pointer accesses")
	command.PersistentFlags().BoolVar(&noInternalThreads, "no-internal-threads", false, "true to elide stack depth tracking in generated code")

	return command
}
