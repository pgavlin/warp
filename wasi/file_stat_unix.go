// +build aix dragonfly linux openbsd solaris

package wasi

import (
	"os"
	"syscall"
	"time"
)

func fileStat(info os.FileInfo) FileStat {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return fileStatUnknown(info)
	}

	return FileStat{
		Dev:        uint64(stat.Dev),
		Inode:      uint64(stat.Ino),
		Mode:       info.Mode(),
		LinkCount:  uint64(stat.Nlink),
		Size:       uint64(info.Size()),
		AccessTime: time.Unix(stat.Atim.Unix()),
		ModTime:    time.Unix(stat.Mtim.Unix()),
		ChangeTime: time.Unix(stat.Ctim.Unix()),
	}
}
