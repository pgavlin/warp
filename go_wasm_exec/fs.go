package go_wasm_exec

import (
	"errors"
	"io"
	"os"

	"github.com/pgavlin/warp/wasi"
)

type fsObject struct {
	Object

	fs    wasi.FS
	files map[int]wasi.File
}

func (fs *fsObject) read(fd int, b []byte, offset int64) (uint32, error) {
	f, ok := fs.files[fd]
	if !ok {
		return 0, errors.New("bad file descriptor")
	}

	if offset == -1 {
		return f.Readv([][]byte{b})
	}
	return f.Pread([][]byte{b}, offset)
}

func (fs *fsObject) write(fd int, b []byte, offset int64) (uint32, error) {
	f, ok := fs.files[fd]
	if !ok {
		return 0, errors.New("bad file descriptor")
	}

	if offset == -1 {
		return f.Writev([][]byte{b})
	}
	return f.Pwrite([][]byte{b}, offset)
}

func NewFS(stdin, stdout, stderr wasi.File, fs wasi.FS) Value {
	o := &fsObject{
		fs: fs,
		files: map[int]wasi.File{
			0: stdin,
			1: stdout,
			2: stderr,
		},
	}
	o.Object = NewObject(ObjectClass, map[string]Value{
		"constants": ValueOf(map[string]Value{
			"O_WRONLY": ValueOf(os.O_WRONLY),
			"O_RDWR":   ValueOf(os.O_RDWR),
			"O_CREAT":  ValueOf(os.O_CREATE),
			"O_TRUNC":  ValueOf(os.O_TRUNC),
			"O_APPEND": ValueOf(os.O_APPEND),
			"O_EXCL":   ValueOf(os.O_EXCL),
		}),
		"read": ValueOf(func(args []Value) (Value, error) {
			fd, start, end := args[0].Int(), args[2].Int(), args[3].Int()
			b, _ := args[1].Uint8Array()

			cb, _ := args[5].Function()

			offset := int64(-1)
			if args[4].Type() == TypeNumber && args[4].Int() != -1 {
				offset = int64(args[4].Int())
			}
			n, err := o.read(fd, b[start:end], offset)
			if err == io.EOF {
				err = nil
			}
			return cb.Invoke([]Value{Undefined(), ValueOf(err), ValueOf(n), args[1]})
		}),
		"write": ValueOf(func(args []Value) (Value, error) {
			fd, start, end := args[0].Int(), args[2].Int(), args[3].Int()
			b, _ := args[1].Uint8Array()

			cb, _ := args[5].Function()

			offset := int64(-1)
			if args[4].Type() == TypeNumber && args[4].Int() != -1 {
				offset = int64(args[4].Int())
			}
			n, err := o.write(fd, b[start:end], offset)
			return cb.Invoke([]Value{Undefined(), ValueOf(err), ValueOf(n), args[1]})
		}),
	})
	return ValueOf(o)
}
