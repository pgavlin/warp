package run

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pgavlin/warp/go_wasm_exec"
	"github.com/pgavlin/warp/load"
	"github.com/pgavlin/warp/wasi"

	"github.com/spf13/cobra"
)

// [to=]from(,flags)
type preopens struct {
	values  []wasi.Preopen
	strings []string
}

var preopenRE = regexp.MustCompile(`^([^=]+=)?([^,]+)(,[^,]+)*$`)

func (p *preopens) parseOne(s string) (wasi.Preopen, error) {
	match := preopenRE.FindStringSubmatch(s)
	if len(match) == 0 {
		return wasi.Preopen{}, fmt.Errorf("malformed preopen '%v': preopens must be of the form (to=)from(,flags)", s)
	}

	to, from, flags := match[1], match[2], strings.Split(match[3], ",")

	if to == "" {
		to = from
	}
	preopen := wasi.Preopen{
		FSPath:  from,
		Path:    to,
		Rights:  wasi.AllRights,
		Inherit: wasi.AllRights,
	}

	if match[3] != "" {
		for _, f := range flags {
			r := &preopen.Rights
			if strings.HasPrefix(f, "inherit:") {
				r, f = &preopen.Inherit, f[len("inherit:"):]
			}

			switch f {
			case "all":
				*r |= wasi.AllRights
			case "dir":
				*r |= wasi.DirectoryRights
			case "file":
				*r |= wasi.FileRights
			case "fd_datasync":
				*r |= wasi.RightsFdDatasync
			case "fd_read":
				*r |= wasi.RightsFdRead
			case "fd_seek":
				*r |= wasi.RightsFdSeek
			case "fd_fdstat_set_flags":
				*r |= wasi.RightsFdFdstatSetFlags
			case "fd_sync":
				*r |= wasi.RightsFdSync
			case "fd_tell":
				*r |= wasi.RightsFdTell
			case "fd_write":
				*r |= wasi.RightsFdWrite
			case "fd_advise":
				*r |= wasi.RightsFdAdvise
			case "fd_allocate":
				*r |= wasi.RightsFdAllocate
			case "path_create_directory":
				*r |= wasi.RightsPathCreateDirectory
			case "path_create_file":
				*r |= wasi.RightsPathCreateFile
			case "path_link_source":
				*r |= wasi.RightsPathLinkSource
			case "path_link_target":
				*r |= wasi.RightsPathLinkTarget
			case "path_open":
				*r |= wasi.RightsPathOpen
			case "fd_readdir":
				*r |= wasi.RightsFdReaddir
			case "path_readlink":
				*r |= wasi.RightsPathReadlink
			case "path_rename_source":
				*r |= wasi.RightsPathRenameSource
			case "path_rename_target":
				*r |= wasi.RightsPathRenameTarget
			case "path_filestat_get":
				*r |= wasi.RightsPathFilestatGet
			case "path_filestat_set_size":
				*r |= wasi.RightsPathFilestatSetSize
			case "path_filestat_set_times":
				*r |= wasi.RightsPathFilestatSetTimes
			case "fd_filestat_get":
				*r |= wasi.RightsFdFilestatGet
			case "fd_filestat_set_size":
				*r |= wasi.RightsFdFilestatSetSize
			case "fd_filestat_set_times":
				*r |= wasi.RightsFdFilestatSetTimes
			case "path_symlink":
				*r |= wasi.RightsPathSymlink
			case "path_remove_directory":
				*r |= wasi.RightsPathRemoveDirectory
			case "path_unlink_file":
				*r |= wasi.RightsPathUnlinkFile
			case "poll_fd_readwrite":
				*r |= wasi.RightsPollFdReadwrite
			case "sock_shutdown":
				*r |= wasi.RightsSockShutdown

			case "=all":
				*r = wasi.AllRights
			case "=dir":
				*r = wasi.DirectoryRights
			case "=file":
				*r = wasi.FileRights
			case "=ro":
				*r = wasi.ReadOnlyRights

			case "-all":
				*r &^= wasi.AllRights
			case "-dir":
				*r &^= wasi.DirectoryRights
			case "-file":
				*r &^= wasi.FileRights
			case "-fd_datasync":
				*r &^= wasi.RightsFdDatasync
			case "-fd_read":
				*r &^= wasi.RightsFdRead
			case "-fd_seek":
				*r &^= wasi.RightsFdSeek
			case "-fd_fdstat_set_flags":
				*r &^= wasi.RightsFdFdstatSetFlags
			case "-fd_sync":
				*r &^= wasi.RightsFdSync
			case "-fd_tell":
				*r &^= wasi.RightsFdTell
			case "-fd_write":
				*r &^= wasi.RightsFdWrite
			case "-fd_advise":
				*r &^= wasi.RightsFdAdvise
			case "-fd_allocate":
				*r &^= wasi.RightsFdAllocate
			case "-path_create_directory":
				*r &^= wasi.RightsPathCreateDirectory
			case "-path_create_file":
				*r &^= wasi.RightsPathCreateFile
			case "-path_link_source":
				*r &^= wasi.RightsPathLinkSource
			case "-path_link_target":
				*r &^= wasi.RightsPathLinkTarget
			case "-path_open":
				*r &^= wasi.RightsPathOpen
			case "-fd_readdir":
				*r &^= wasi.RightsFdReaddir
			case "-path_readlink":
				*r &^= wasi.RightsPathReadlink
			case "-path_rename_source":
				*r &^= wasi.RightsPathRenameSource
			case "-path_rename_target":
				*r &^= wasi.RightsPathRenameTarget
			case "-path_filestat_get":
				*r &^= wasi.RightsPathFilestatGet
			case "-path_filestat_set_size":
				*r &^= wasi.RightsPathFilestatSetSize
			case "-path_filestat_set_times":
				*r &^= wasi.RightsPathFilestatSetTimes
			case "-fd_filestat_get":
				*r &^= wasi.RightsFdFilestatGet
			case "-fd_filestat_set_size":
				*r &^= wasi.RightsFdFilestatSetSize
			case "-fd_filestat_set_times":
				*r &^= wasi.RightsFdFilestatSetTimes
			case "-path_symlink":
				*r &^= wasi.RightsPathSymlink
			case "-path_remove_directory":
				*r &^= wasi.RightsPathRemoveDirectory
			case "-path_unlink_file":
				*r &^= wasi.RightsPathUnlinkFile
			case "-poll_fd_readwrite":
				*r &^= wasi.RightsPollFdReadwrite
			case "-sock_shutdown":
				*r &^= wasi.RightsSockShutdown

			default:
				return wasi.Preopen{}, fmt.Errorf("unknown preopen flag '%v'", f)
			}
		}
	}

	return preopen, nil
}

func (p *preopens) String() string {
	return strings.Join(p.strings, ";")
}

func (p *preopens) Set(s string) error {
	preopen, err := p.parseOne(s)
	if err != nil {
		return err
	}
	p.values, p.strings = append(p.values, preopen), append(p.strings, s)
	return nil
}

func (p *preopens) Type() string {
	return "mount"
}

func Command() *cobra.Command {
	var preopen preopens
	var debug bool
	var trace string

	command := &cobra.Command{
		Use:   "run [path to module]",
		Short: "Run WebAssembly commands",
		Long:  "Run WebAssembly commands inside a WASI- or Go-compliant environment.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return errors.New("expected at least one argument")
			}

			mod, err := load.LoadFile(args[0])
			if err != nil {
				return err
			}

			isGo := false
			if mod.Import != nil {
				for _, entry := range mod.Import.Entries {
					if entry.ModuleName == "go" {
						isGo = true
						break
					}
				}
			}

			def, err := load.Intepret(mod)
			if err != nil {
				return err
			}

			env := map[string]string{}
			for _, v := range os.Environ() {
				kvp := strings.SplitN(v, "=", 2)
				env[kvp[0]] = kvp[1]
			}

			ext := filepath.Ext(args[0])
			name := args[0][:len(args[0])-len(ext)]

			var traceWriter io.Writer
			if trace != "" {
				traceFile, err := os.Create(trace)
				if err != nil {
					return err
				}
				defer traceFile.Close()

				w := bufio.NewWriter(traceFile)
				defer w.Flush()

				c := make(chan os.Signal)
				signal.Notify(c, os.Interrupt, os.Kill)
				go func() {
					for range c {
						w.Flush()
						os.Exit(-1)
					}
				}()

				traceWriter = w
			}

			if isGo {
				return go_wasm_exec.Run(name, def, &go_wasm_exec.Options{
					Env:  env,
					Args: args[1:],

					Debug:    debug,
					Trace:    traceWriter,
					Resolver: load.NewFSResolver(os.DirFS("."), load.Intepret),
				})
			}

			return wasi.Run(name, def, &wasi.RunOptions{
				Options: &wasi.Options{
					Env:     env,
					Args:    args[1:],
					Preopen: preopen.values,
				},
				Debug:    debug,
				Trace:    traceWriter,
				Resolver: load.NewFSResolver(os.DirFS("."), load.Intepret),
			})
		},
	}

	command.PersistentFlags().VarP(&preopen, "mount", "m", "list of directories to mount in the form (to=)from(,flags)")
	command.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "enable debugging support")
	command.PersistentFlags().StringVarP(&trace, "trace", "t", "", "write an execution trace to the specified file. Implies -d.")

	return command
}
