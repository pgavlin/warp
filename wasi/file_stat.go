package wasi

import (
	"os"
	"sync/atomic"
)

const unknownDevice = (1 << 64) - 1

var fileCookie uint64

func fileStatUnknown(info os.FileInfo) FileStat {
	modTime := info.ModTime()
	return FileStat{
		Dev:        unknownDevice,
		Inode:      atomic.AddUint64(&fileCookie, 1),
		Mode:       info.Mode(),
		LinkCount:  1,
		Size:       uint64(info.Size()),
		AccessTime: modTime,
		ModTime:    modTime,
		ChangeTime: modTime,
	}
}
