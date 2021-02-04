package wasi

import (
	"fmt"
	"math/bits"
	"os"
	"sync"
	"sync/atomic"
)

const (
	maxFiles = 4096
)

type file struct {
	m    sync.Mutex
	open bool

	preopen int
	fdflags wasiFdflags

	rights  wasiRights
	inherit wasiRights

	f File

	entries []os.DirEntry
}

type fileTable struct {
	files  [maxFiles]file
	bitmap [maxFiles / 64]uint64
}

func (t *fileTable) allocate(rights, inherit wasiRights) (wasiFd, *file, wasiErrno) {
	for i, b := range t.bitmap {
		bi := bits.TrailingZeros64(^b)
		if bi != 64 {
			if atomic.CompareAndSwapUint64(&t.bitmap[i], b, b|1<<bi) {
				fd := 64*i + bi
				f := &t.files[fd]
				f.rights, f.inherit = rights, inherit
				return wasiFd(fd), f, wasiErrnoSuccess
			}
		}
	}
	return wasiFd(0), nil, wasiErrnoNfile
}

func (t *fileTable) mustAllocateFile(f File, fd wasiFd, rights, inherit wasiRights) {
	i, bi := fd/64, fd%64
	b := t.bitmap[i]
	if !atomic.CompareAndSwapUint64(&t.bitmap[i], b, b|1<<bi) {
		panic(fmt.Errorf("failed to allocate file descriptor %d", fd))
	}

	wf := &t.files[fd]
	wf.open, wf.rights, wf.inherit, wf.f = true, rights, inherit, f
}

func (t *fileTable) acquireFile(fd wasiFd, rights wasiRights) (*file, wasiErrno) {
	if fd >= handle(len(t.files)) {
		return nil, wasiErrnoBadf
	}

	f := &t.files[fd]
	f.m.Lock()
	if !f.open {
		f.m.Unlock()
		return nil, wasiErrnoBadf
	}
	if f.rights&rights != rights {
		f.m.Unlock()
		return nil, wasiErrnoNotcapable
	}
	return f, wasiErrnoSuccess
}

func (t *fileTable) releaseFile(fd wasiFd, f *file) wasiErrno {
	if f.open {
		f.m.Unlock()
		return wasiErrnoSuccess
	}

	f.preopen, f.rights, f.inherit, f.f = 0, 0, 0, nil
	f.m.Unlock()

	i, bi := fd/64, fd%64
	for {
		b := t.bitmap[i]
		if atomic.CompareAndSwapUint64(&t.bitmap[i], b, b&^(1<<bi)) {
			return wasiErrnoSuccess
		}
	}
}

func (t *fileTable) getFile(fd wasiFd, rights wasiRights) (File, wasiErrno) {
	f, errno := t.acquireFile(fd, rights)
	if errno != wasiErrnoSuccess {
		return nil, errno
	}
	defer f.m.Unlock()

	return f.f, wasiErrnoSuccess
}

func (t *fileTable) getDirectory(fd wasiFd, rights wasiRights) (Directory, wasiErrno) {
	f, errno := t.acquireFile(fd, rights)
	if errno != wasiErrnoSuccess {
		return nil, errno
	}
	defer f.m.Unlock()

	d, ok := f.f.(Directory)
	if !ok {
		return nil, wasiErrnoNotdir
	}

	return d, wasiErrnoSuccess
}

func (t *fileTable) getPreopen(fd wasiFd) (int, wasiErrno) {
	f, errno := t.acquireFile(fd, 0)
	if errno != wasiErrnoSuccess {
		return 0, errno
	}
	defer f.m.Unlock()

	if f.preopen == 0 {
		return 0, wasiErrnoInval
	}
	return f.preopen - 1, wasiErrnoSuccess
}
