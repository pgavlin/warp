package dump

import (
	"bufio"
	"errors"
	"os"

	"github.com/pgavlin/warp/load"
	"github.com/pgavlin/warp/wasm"
	"github.com/pgavlin/warp/wasm/trace"
	"github.com/pgavlin/warp/wast"
	"github.com/spf13/cobra"
)

type names struct {
	moduleName    string
	functionNames map[uint32]string
	localNames    map[uint32]map[uint32]string
}

func (n *names) FunctionName(moduleName string, index uint32) (string, bool) {
	name, ok := n.functionNames[index]
	return name, ok
}

func (n *names) LocalName(moduleName string, functionIndex, localIndex uint32) (string, bool) {
	name, ok := n.localNames[functionIndex][localIndex]
	return name, ok
}

func Command() *cobra.Command {
	var traceFile string
	var stats bool

	command := &cobra.Command{
		Use:   "dump [path to module]",
		Short: "Dump WebAssembly modules and traces",
		Long:  "Dump WebAssembly modules as WebAssembly text",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return errors.New("expected exactly one argument")
			}
			mod, err := load.LoadFile(args[0])
			if err != nil {
				return err
			}

			n := names{
				functionNames: map[uint32]string{},
				localNames:    map[uint32]map[uint32]string{},
			}
			if names, err := mod.Names(); err == nil {
				for _, subsection := range names.Entries {
					switch subsection := subsection.(type) {
					case *wasm.ModuleNameSubsection:
						n.moduleName = subsection.Name
					case *wasm.FunctionNamesSubsection:
						for _, name := range subsection.Names {
							n.functionNames[name.Index] = name.Name
						}
					case *wasm.LocalNamesSubsection:
						for _, func_ := range subsection.Funcs {
							m := map[uint32]string{}
							for _, name := range func_.Names {
								m[name.Index] = name.Name
							}
							n.localNames[func_.Index] = m
						}
					}
				}
			}
			if mod.Import != nil {
				funcIdx := uint32(0)
				for _, import_ := range mod.Import.Entries {
					if _, ok := import_.Type.(wasm.FuncImport); ok {
						n.functionNames[uint32(funcIdx)] = import_.FieldName
						funcIdx++
					}
				}
			}

			switch {
			case traceFile != "":
				f, err := os.Open(traceFile)
				if err != nil {
					return err
				}
				defer f.Close()

				w := bufio.NewWriter(os.Stdout)
				defer w.Flush()

				return trace.PrintTrace(w, bufio.NewReader(f), &n)
			case stats:
				return dumpStats(os.Stdout, mod, &n)
			default:
				return wast.WriteTo(os.Stdout, mod)
			}
		},
	}

	command.PersistentFlags().StringVarP(&traceFile, "trace", "t", "", "dump an execution trace from the specified file")
	command.PersistentFlags().BoolVarP(&stats, "stats", "s", false, "dump module statistics in CSV format")

	return command
}
