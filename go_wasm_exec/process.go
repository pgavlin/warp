package go_wasm_exec

import (
	"os"
)

var process = objectValue(map[string]Value{
	"getuid":  functionValue(getuid),
	"getgid":  functionValue(getgid),
	"geteuid": functionValue(geteuid),
	"getegid": functionValue(getegid),
	"pid":     ValueOf(os.Getpid()),
	"ppid":    ValueOf(os.Getppid()),
})

func getuid(_ []Value) (Value, error) {
	uid := os.Getuid()
	return ValueOf(uid), nil
}

func getgid(_ []Value) (Value, error) {
	gid := os.Getgid()
	return ValueOf(gid), nil
}

func geteuid(_ []Value) (Value, error) {
	euid := os.Geteuid()
	return ValueOf(euid), nil
}

func getegid(_ []Value) (Value, error) {
	egid := os.Getegid()
	return ValueOf(egid), nil
}
