// +build !aix,!darwin,!dragonfly,!freebsd,!linux,!netbsd,!openbsd,!solaris

package wasi

import (
	"os"
)

func fileStat(info os.FileInfo) FileStat {
	return fileStatUnknown(info)
}
