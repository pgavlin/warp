//go:generate cargo run --manifest-path gen-wasi/Cargo.toml generate-api

package wasi

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/pgavlin/warp/exec"
)

type TrapExit int

type wasiSnapshotPreview1Impl struct {
	wasi *wasiSnapshotPreview1

	start time.Time

	env  []string
	args []string

	fs      FS
	files   fileTable
	preopen []Preopen
}

func newImpl(wasi *wasiSnapshotPreview1, opts *Options) (*wasiSnapshotPreview1Impl, error) {
	env, args := os.Environ(), []string(nil)
	fs, preopen := NewFS(), []Preopen(nil)
	stdin, stdout, stderr := NewFile(os.Stdin, 0), NewFile(os.Stdout, wasiFdflagsAppend), NewFile(os.Stderr, wasiFdflagsAppend)
	if opts != nil {
		env = make([]string, 0, len(opts.Env))
		for k, v := range opts.Env {
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}
		sort.Strings(env)

		args = opts.Args

		if opts.FS != nil {
			fs = opts.FS
		}
		preopen = opts.Preopen

		if opts.Stdin != nil {
			stdin = NewReader(opts.Stdin)
		}
		if opts.Stdout != nil {
			stdout = NewWriter(opts.Stdout)
		}
		if opts.Stderr != nil {
			stderr = NewWriter(opts.Stderr)
		}
	}

	impl := &wasiSnapshotPreview1Impl{
		wasi:    wasi,
		start:   time.Now(),
		env:     env,
		args:    args,
		fs:      fs,
		preopen: preopen,
	}
	impl.files.mustAllocateFile(stdin, 0, FileRights, 0)
	impl.files.mustAllocateFile(stdout, 1, FileRights, 0)
	impl.files.mustAllocateFile(stderr, 2, FileRights, 0)

	for i, p := range preopen {
		_, f, errno := impl.files.allocate(wasiRights(p.Rights), wasiRights(p.Inherit))
		switch errno {
		case wasiErrnoSuccess:
			// OK
		case wasiErrnoNfile:
			return nil, fmt.Errorf("too many open files")
		default:
			panic(fmt.Errorf("unexpected error allocting preopen fd: %v", errno))
		}

		dir, err := fs.OpenDirectory(p.FSPath)
		if err != nil {
			return nil, err
		}

		f.preopen, f.open, f.f = i+1, true, dir
	}

	return impl, nil
}

// Read command-line argument data.
// The size of the array should match that returned by `args_sizes_get`
func (m *wasiSnapshotPreview1Impl) argsGet(pargv pointer, pargvBuf pointer) (err wasiErrno) {
	for _, s := range m.args {
		buf := m.bytes(pargvBuf)
		copy(buf, s)
		buf[len(s)] = 0

		m.putUint32(uint32(pargvBuf), pargv)
		pargvBuf, pargv = pargvBuf+pointer(len(s))+1, pargv+4
	}
	return wasiErrnoSuccess
}

// Return command-line argument data sizes.
func (m *wasiSnapshotPreview1Impl) argsSizesGet() (r0 wasiSize, r1 wasiSize, err wasiErrno) {
	size := 0
	for _, s := range m.args {
		size += len(s) + 1
	}
	return wasiSize(len(m.args)), wasiSize(size), wasiErrnoSuccess
}

// Read environment variable data.
// The sizes of the buffers should match that returned by `environ_sizes_get`.
func (m *wasiSnapshotPreview1Impl) environGet(penviron pointer, penvironBuf pointer) (err wasiErrno) {
	for _, s := range m.env {
		buf := m.bytes(penvironBuf)
		copy(buf, s)
		buf[len(s)] = 0

		m.putUint32(uint32(penvironBuf), penviron)
		penvironBuf, penviron = penvironBuf+pointer(len(s))+1, penviron+4
	}
	return wasiErrnoSuccess
}

// Return environment variable data sizes.
func (m *wasiSnapshotPreview1Impl) environSizesGet() (r0 wasiSize, r1 wasiSize, err wasiErrno) {
	size := 0
	for _, s := range m.env {
		size += len(s) + 1
	}
	return wasiSize(len(m.env)), wasiSize(size), wasiErrnoSuccess
}

// Return the resolution of a clock.
// Implementations are required to provide a non-zero value for supported clocks. For unsupported clocks,
// return `errno::inval`.
// Note: This is similar to `clock_getres` in POSIX.
func (m *wasiSnapshotPreview1Impl) clockResGet(pid wasiClockid) (rv wasiTimestamp, err wasiErrno) {
	switch pid {
	case wasiClockidRealtime:
		// Guess at milliseconds.
		return wasiTimestamp(1 * time.Millisecond), wasiErrnoSuccess
	case wasiClockidMonotonic:
		return wasiTimestamp(1 * time.Nanosecond), wasiErrnoSuccess
	default:
		return 0, wasiErrnoInval
	}
}

// Return the time value of a clock.
// Note: This is similar to `clock_gettime` in POSIX.
func (m *wasiSnapshotPreview1Impl) clockTimeGet(pid wasiClockid, pprecision wasiTimestamp) (rv wasiTimestamp, err wasiErrno) {
	switch pid {
	case wasiClockidRealtime:
		return wasiTimestamp(time.Now().UnixNano()), wasiErrnoSuccess
	case wasiClockidMonotonic:
		return wasiTimestamp(time.Now().Sub(m.start)), wasiErrnoSuccess
	default:
		return 0, wasiErrnoInval
	}
}

// Provide file advisory information on a file descriptor.
// Note: This is similar to `posix_fadvise` in POSIX.
func (m *wasiSnapshotPreview1Impl) fdAdvise(pfd wasiFd, poffset wasiFilesize, plen wasiFilesize, padvice wasiAdvice) (err wasiErrno) {
	f, err := m.files.getFile(pfd, wasiRightsFdAdvise)
	if err != wasiErrnoSuccess {
		return err
	}

	if err := f.Advise(poffset, plen, int(padvice)); err != nil {
		return fileErrno(err)
	}
	return wasiErrnoSuccess
}

// Force the allocation of space in a file.
// Note: This is similar to `posix_fallocate` in POSIX.
func (m *wasiSnapshotPreview1Impl) fdAllocate(pfd wasiFd, poffset wasiFilesize, plen wasiFilesize) (err wasiErrno) {
	err = wasiErrnoNotsup
	return
}

// Close a file descriptor.
// Note: This is similar to `close` in POSIX.
func (m *wasiSnapshotPreview1Impl) fdClose(pfd wasiFd) (err wasiErrno) {
	f, err := m.files.acquireFile(pfd, 0)
	if err != wasiErrnoSuccess {
		return err
	}
	defer m.files.releaseFile(pfd, f)

	if err := f.f.Close(); err != nil {
		return fileErrno(err)
	}

	f.open = false
	return wasiErrnoSuccess
}

// Synchronize the data of a file to disk.
// Note: This is similar to `fdatasync` in POSIX.
func (m *wasiSnapshotPreview1Impl) fdDatasync(pfd wasiFd) (err wasiErrno) {
	f, err := m.files.getFile(pfd, wasiRightsFdDatasync)
	if err != wasiErrnoSuccess {
		return err
	}

	if err := f.Datasync(); err != nil {
		return fileErrno(err)
	}
	return wasiErrnoSuccess
}

// Get the attributes of a file descriptor.
// Note: This returns similar flags to `fsync(fd, F_GETFL)` in POSIX, as well as additional fields.
func (m *wasiSnapshotPreview1Impl) fdFdstatGet(pfd wasiFd) (rv wasiFdstat, err wasiErrno) {
	f, err := m.files.acquireFile(pfd, wasiRightsFdDatasync)
	if err != wasiErrnoSuccess {
		return wasiFdstat{}, err
	}
	defer m.files.releaseFile(pfd, f)

	info, ferr := f.f.Stat()
	if ferr != nil {
		return wasiFdstat{}, fileErrno(ferr)
	}

	return wasiFdstat{
		fsFiletype:         filetype(info.Mode),
		fsFlags:            f.fdflags,
		fsRightsBase:       f.rights,
		fsRightsInheriting: f.inherit,
	}, wasiErrnoSuccess
}

// Adjust the flags associated with a file descriptor.
// Note: This is similar to `fcntl(fd, F_SETFL, flags)` in POSIX.
func (m *wasiSnapshotPreview1Impl) fdFdstatSetFlags(pfd wasiFd, pflags wasiFdflags) (err wasiErrno) {
	f, err := m.files.getFile(pfd, wasiRightsFdDatasync)
	if err != wasiErrnoSuccess {
		return err
	}

	ferr := f.SetFlags(int(pflags))
	if ferr != nil {
		return fileErrno(ferr)
	}
	return wasiErrnoSuccess
}

// Adjust the rights associated with a file descriptor.
// This can only be used to remove rights, and returns `errno::notcapable` if called in a way that would attempt to add rights
func (m *wasiSnapshotPreview1Impl) fdFdstatSetRights(pfd wasiFd, pfsRightsBase wasiRights, pfsRightsInheriting wasiRights) (err wasiErrno) {
	f, err := m.files.acquireFile(pfd, wasiRightsFdDatasync)
	if err != wasiErrnoSuccess {
		return err
	}
	defer m.files.releaseFile(pfd, f)

	if f.rights&pfsRightsBase != pfsRightsBase || f.inherit&pfsRightsInheriting != pfsRightsInheriting {
		return wasiErrnoNotcapable
	}

	f.rights, f.inherit = pfsRightsBase, pfsRightsInheriting
	return wasiErrnoSuccess
}

// Return the attributes of an open file.
func (m *wasiSnapshotPreview1Impl) fdFilestatGet(pfd wasiFd) (rv wasiFilestat, err wasiErrno) {
	f, err := m.files.getFile(pfd, wasiRightsFdRead)
	if err != wasiErrnoSuccess {
		return wasiFilestat{}, err
	}

	info, ferr := f.Stat()
	if ferr != nil {
		return wasiFilestat{}, fileErrno(ferr)
	}

	return wasiFilestat{
		dev:      info.Dev,
		ino:      info.Inode,
		filetype: filetype(info.Mode),
		nlink:    info.LinkCount,
		size:     info.Size,
		atim:     uint64(info.AccessTime.UnixNano()),
		mtim:     uint64(info.ModTime.UnixNano()),
		ctim:     uint64(info.ChangeTime.UnixNano()),
	}, wasiErrnoSuccess
}

// Adjust the size of an open file. If this increases the file's size, the extra bytes are filled with zeros.
// Note: This is similar to `ftruncate` in POSIX.
func (m *wasiSnapshotPreview1Impl) fdFilestatSetSize(pfd wasiFd, psize wasiFilesize) (err wasiErrno) {
	f, err := m.files.getFile(pfd, wasiRightsFdRead)
	if err != wasiErrnoSuccess {
		return err
	}

	if ferr := f.SetSize(psize); ferr != nil {
		return fileErrno(ferr)
	}
	return wasiErrnoSuccess
}

// Adjust the timestamps of an open file or directory.
// Note: This is similar to `futimens` in POSIX.
func (m *wasiSnapshotPreview1Impl) fdFilestatSetTimes(pfd wasiFd, patim wasiTimestamp, pmtim wasiTimestamp, pfstFlags wasiFstflags) (err wasiErrno) {
	f, err := m.files.getFile(pfd, wasiRightsFdRead)
	if err != wasiErrnoSuccess {
		return err
	}

	var accessTime *time.Time
	switch {
	case pfstFlags&wasiFstflagsAtim != 0:
		t := time.Unix(0, int64(patim))
		accessTime = &t
	case pfstFlags&wasiFstflagsAtimNow != 0:
		t := time.Now()
		accessTime = &t
	}

	var modTime *time.Time
	switch {
	case pfstFlags&wasiFstflagsMtim != 0:
		t := time.Unix(0, int64(patim))
		modTime = &t
	case pfstFlags&wasiFstflagsMtimNow != 0:
		t := time.Now()
		modTime = &t
	}

	if ferr := f.SetTimes(accessTime, modTime); ferr != nil {
		return fileErrno(ferr)
	}
	return wasiErrnoSuccess
}

// Read from a file descriptor, without using and updating the file descriptor's offset.
// Note: This is similar to `preadv` in POSIX.
func (m *wasiSnapshotPreview1Impl) fdPread(pfd wasiFd, piovs list, poffset wasiFilesize) (rv wasiSize, err wasiErrno) {
	f, err := m.files.getFile(pfd, wasiRightsFdRead)
	if err != wasiErrnoSuccess {
		return 0, err
	}

	n, ferr := f.Pread(m.buffers(wasiIovecArray(piovs)), int64(poffset))
	if ferr != nil {
		return n, fileErrno(ferr)
	}
	return n, wasiErrnoSuccess
}

// Return a description of the given preopened file descriptor.
func (m *wasiSnapshotPreview1Impl) fdPrestatGet(pfd wasiFd) (rv wasiPrestat, err wasiErrno) {
	index, err := m.files.getPreopen(pfd)
	if err != wasiErrnoSuccess {
		return wasiPrestat{}, err
	}
	preopen := m.preopen[index]

	return wasiPrestat{
		tag: wasiPreopentypeDir,
		dir: wasiPrestatDir{
			prNameLen: wasiSize(len(preopen.Path)),
		},
	}, wasiErrnoSuccess
}

// Return a description of the given preopened file descriptor.
func (m *wasiSnapshotPreview1Impl) fdPrestatDirName(pfd wasiFd, ppath pointer, ppathLen wasiSize) (err wasiErrno) {
	index, err := m.files.getPreopen(pfd)
	if err != wasiErrnoSuccess {
		return err
	}
	preopen := m.preopen[index]

	copy(m.slice(ppath, ppathLen), preopen.Path)
	return wasiErrnoSuccess
}

// Write to a file descriptor, without using and updating the file descriptor's offset.
// Note: This is similar to `pwritev` in POSIX.
func (m *wasiSnapshotPreview1Impl) fdPwrite(pfd wasiFd, piovs list, poffset wasiFilesize) (rv wasiSize, err wasiErrno) {
	f, err := m.files.getFile(pfd, wasiRightsFdWrite)
	if err != wasiErrnoSuccess {
		return 0, err
	}

	n, ferr := f.Pwrite(m.buffers(wasiIovecArray(piovs)), int64(poffset))
	if ferr != nil {
		return n, fileErrno(ferr)
	}
	return n, wasiErrnoSuccess
}

// Read from a file descriptor.
// Note: This is similar to `readv` in POSIX.
func (m *wasiSnapshotPreview1Impl) fdRead(pfd wasiFd, piovs list) (rv wasiSize, err wasiErrno) {
	f, err := m.files.getFile(pfd, wasiRightsFdRead)
	if err != wasiErrnoSuccess {
		return 0, err
	}

	n, ferr := f.Readv(m.buffers(wasiIovecArray(piovs)))
	if ferr != nil {
		return n, fileErrno(ferr)
	}
	return n, wasiErrnoSuccess
}

// Read directory entries from a directory.
// When successful, the contents of the output buffer consist of a sequence of
// directory entries. Each directory entry consists of a `dirent` object,
// followed by `dirent::d_namlen` bytes holding the name of the directory
// entry.
// This function fills the output buffer as much as possible, potentially
// truncating the last directory entry. This allows the caller to grow its
// read buffer size in case it's too small to fit a single large directory
// entry, or skip the oversized directory entry.
func (m *wasiSnapshotPreview1Impl) fdReaddir(pfd wasiFd, pbuf pointer, pbufLen wasiSize, pcookie wasiDircookie) (rv wasiSize, err wasiErrno) {
	f, err := m.files.acquireFile(pfd, wasiRightsFdReaddir)
	if err != wasiErrnoSuccess {
		return 0, err
	}

	entries, err := func() ([]os.DirEntry, wasiErrno) {
		defer m.files.releaseFile(pfd, f)

		dir, ok := f.f.(Directory)
		if !ok {
			return nil, wasiErrnoNotdir
		}

		if f.entries == nil {
			entries, ferr := dir.ReadDir(0)
			if ferr != nil {
				return nil, fileErrno(ferr)
			}
			f.entries = entries
		}
		entries := f.entries
		switch {
		case pcookie > uint64(len(entries)):
			return nil, wasiErrnoInval
		case pcookie == uint64(len(entries)):
			f.entries = nil
			return nil, wasiErrnoSuccess
		default:
			return entries[int(pcookie):], wasiErrnoSuccess
		}
	}()
	if err != wasiErrnoSuccess {
		return 0, err
	}

	dest := m.slice(pbuf, pbufLen)

	var dirent wasiDirent
	direntSize, _ := dirent.layout()

	written := wasiSize(0)
	buf := exec.NewMemory(1, 1)
	for i, entry := range entries[int(pcookie):] {
		name := entry.Name()
		bytes := buf.Bytes()
		entrySize := direntSize + uint32(len(name))
		if entrySize > uint32(len(bytes)) {
			return written, wasiErrnoNobufs
		}

		info, ferr := entry.Info()
		if ferr != nil {
			return written, fileErrno(ferr)
		}
		stat := fileStat(info)

		dirent.dIno = stat.Inode
		dirent.dNamlen = wasiDirnamlen(len(name))
		dirent.dNext = pcookie + uint64(i)
		dirent.dType = filetype(stat.Mode)

		dirent.store(&buf, 0, 0)
		copy(bytes[int(direntSize):], name)

		n := copy(dest, bytes[:int(entrySize)])
		if n == 0 {
			break
		}
		written += wasiSize(n)
		dest = dest[n:]
	}
	return written, wasiErrnoSuccess
}

// Atomically replace a file descriptor by renumbering another file descriptor.
// Due to the strong focus on thread safety, this environment does not provide
// a mechanism to duplicate or renumber a file descriptor to an arbitrary
// number, like `dup2()`. This would be prone to race conditions, as an actual
// file descriptor with the same number could be allocated by a different
// thread at the same time.
// This function provides a way to atomically renumber file descriptors, which
// would disappear if `dup2()` were to be removed entirely.
func (m *wasiSnapshotPreview1Impl) fdRenumber(pfd wasiFd, pto wasiFd) (err wasiErrno) {
	from, err := m.files.acquireFile(pfd, 0)
	if err != wasiErrnoSuccess {
		return err
	}
	defer m.files.releaseFile(pfd, from)

	to, err := m.files.acquireFile(pto, 0)
	if err != wasiErrnoSuccess {
		return err
	}
	defer m.files.releaseFile(pto, to)

	if err := to.f.Close(); err != nil {
		return fileErrno(err)
	}

	to.open = true
	to.preopen = from.preopen
	to.fdflags = from.fdflags
	to.rights = from.rights
	to.inherit = from.inherit
	to.f = from.f
	to.entries = from.entries

	return wasiErrnoSuccess
}

// Move the offset of a file descriptor.
// Note: This is similar to `lseek` in POSIX.
func (m *wasiSnapshotPreview1Impl) fdSeek(pfd wasiFd, poffset wasiFiledelta, pwhence wasiWhence) (rv wasiFilesize, err wasiErrno) {
	f, err := m.files.getFile(pfd, wasiRightsFdSeek)
	if err != wasiErrnoSuccess {
		return 0, err
	}

	pos, ferr := f.Seek(int64(poffset), int(pwhence))
	if ferr != nil {
		return 0, fileErrno(ferr)
	}
	return wasiFilesize(pos), wasiErrnoSuccess
}

// Synchronize the data and metadata of a file to disk.
// Note: This is similar to `fsync` in POSIX.
func (m *wasiSnapshotPreview1Impl) fdSync(pfd wasiFd) (err wasiErrno) {
	f, err := m.files.getFile(pfd, wasiRightsFdSync)
	if err != wasiErrnoSuccess {
		return err
	}

	if err := f.Sync(); err != nil {
		return fileErrno(err)
	}
	return wasiErrnoSuccess
}

// Return the current offset of a file descriptor.
// Note: This is similar to `lseek(fd, 0, SEEK_CUR)` in POSIX.
func (m *wasiSnapshotPreview1Impl) fdTell(pfd wasiFd) (rv wasiFilesize, err wasiErrno) {
	f, err := m.files.getFile(pfd, wasiRightsFdTell)
	if err != wasiErrnoSuccess {
		return 0, err
	}

	pos, ferr := f.Seek(0, 1)
	if ferr != nil {
		return 0, fileErrno(ferr)
	}
	return wasiFilesize(pos), wasiErrnoSuccess
}

// Write to a file descriptor.
// Note: This is similar to `writev` in POSIX.
func (m *wasiSnapshotPreview1Impl) fdWrite(pfd wasiFd, piovs list) (rv wasiSize, err wasiErrno) {
	f, err := m.files.getFile(pfd, wasiRightsFdWrite)
	if err != wasiErrnoSuccess {
		return 0, err
	}

	n, ferr := f.Writev(m.buffers(wasiIovecArray(piovs)))
	if ferr != nil {
		return n, fileErrno(ferr)
	}
	return n, wasiErrnoSuccess
}

// Create a directory.
// Note: This is similar to `mkdirat` in POSIX.
func (m *wasiSnapshotPreview1Impl) pathCreateDirectory(pfd wasiFd, ppath list) (err wasiErrno) {
	path, err := m.loadPath(ppath)
	if err != wasiErrnoSuccess {
		return err
	}

	dir, err := m.files.getDirectory(pfd, wasiRightsPathCreateDirectory)
	if err != wasiErrnoSuccess {
		return err
	}

	if ferr := dir.Mkdir(path); ferr != nil {
		return fileErrno(ferr)
	}
	return wasiErrnoSuccess
}

// Return the attributes of a file or directory.
// Note: This is similar to `stat` in POSIX.
func (m *wasiSnapshotPreview1Impl) pathFilestatGet(pfd wasiFd, pflags wasiLookupflags, ppath list) (rv wasiFilestat, err wasiErrno) {
	path, err := m.loadPath(ppath)
	if err != wasiErrnoSuccess {
		return wasiFilestat{}, err
	}

	dir, err := m.files.getDirectory(pfd, wasiRightsPathFilestatGet)
	if err != wasiErrnoSuccess {
		return wasiFilestat{}, err
	}

	followSymlinks := pflags&wasiLookupflagsSymlinkFollow != 0
	info, ferr := dir.FileStat(path, followSymlinks)
	if ferr != nil {
		return wasiFilestat{}, fileErrno(ferr)
	}

	return wasiFilestat{
		dev:      info.Dev,
		ino:      info.Inode,
		filetype: filetype(info.Mode),
		nlink:    info.LinkCount,
		size:     info.Size,
		atim:     uint64(info.AccessTime.UnixNano()),
		mtim:     uint64(info.ModTime.UnixNano()),
		ctim:     uint64(info.ChangeTime.UnixNano()),
	}, wasiErrnoSuccess
}

// Adjust the timestamps of a file or directory.
// Note: This is similar to `utimensat` in POSIX.
func (m *wasiSnapshotPreview1Impl) pathFilestatSetTimes(pfd wasiFd, pflags wasiLookupflags, ppath list, patim wasiTimestamp, pmtim wasiTimestamp, pfstFlags wasiFstflags) (err wasiErrno) {
	path, err := m.loadPath(ppath)
	if err != wasiErrnoSuccess {
		return err
	}

	dir, err := m.files.getDirectory(pfd, wasiRightsPathFilestatSetTimes)
	if err != wasiErrnoSuccess {
		return err
	}

	var accessTime *time.Time
	switch {
	case pfstFlags&wasiFstflagsAtim != 0:
		t := time.Unix(0, int64(patim))
		accessTime = &t
	case pfstFlags&wasiFstflagsAtimNow != 0:
		t := time.Now()
		accessTime = &t
	}

	var modTime *time.Time
	switch {
	case pfstFlags&wasiFstflagsMtim != 0:
		t := time.Unix(0, int64(patim))
		modTime = &t
	case pfstFlags&wasiFstflagsMtimNow != 0:
		t := time.Now()
		modTime = &t
	}

	followSymlinks := pflags&wasiLookupflagsSymlinkFollow != 0
	if ferr := dir.SetFileTimes(path, accessTime, modTime, followSymlinks); ferr != nil {
		return fileErrno(ferr)
	}
	return wasiErrnoSuccess
}

// Create a hard link.
// Note: This is similar to `linkat` in POSIX.
func (m *wasiSnapshotPreview1Impl) pathLink(poldFd wasiFd, poldFlags wasiLookupflags, poldPath list, pnewFd wasiFd, pnewPath list) (err wasiErrno) {
	oldPath, err := m.loadPath(poldPath)
	if err != wasiErrnoSuccess {
		return err
	}
	newPath, err := m.loadPath(pnewPath)
	if err != wasiErrnoSuccess {
		return err
	}

	oldDir, err := m.files.getDirectory(poldFd, wasiRightsPathLinkSource)
	if err != wasiErrnoSuccess {
		return err
	}

	newDir, err := m.files.getDirectory(pnewFd, wasiRightsPathLinkTarget)
	if err != wasiErrnoSuccess {
		return err
	}

	if ferr := m.fs.Link(oldDir, oldPath, newDir, newPath); ferr != nil {
		return fileErrno(ferr)
	}

	return wasiErrnoSuccess
}

// Open a file or directory.
// The returned file descriptor is not guaranteed to be the lowest-numbered
// file descriptor not currently open; it is randomized to prevent
// applications from depending on making assumptions about indexes, since this
// is error-prone in multi-threaded contexts. The returned file descriptor is
// guaranteed to be less than 2**31.
// Note: This is similar to `openat` in POSIX.
func (m *wasiSnapshotPreview1Impl) pathOpen(pfd wasiFd, pdirflags wasiLookupflags, ppath list, poflags wasiOflags, pfsRightsBase wasiRights, pfsRightsInheriting wasiRights, pfdflags wasiFdflags) (rv wasiFd, err wasiErrno) {
	path, err := m.loadPath(ppath)
	if err != wasiErrnoSuccess {
		return 0, err
	}

	requiredRights := uint64(wasiRightsPathOpen)
	if poflags&wasiOflagsCreat != 0 {
		requiredRights |= wasiRightsPathCreateFile
	}

	f, err := m.files.acquireFile(pfd, requiredRights)
	if err != wasiErrnoSuccess {
		return 0, err
	}
	defer m.files.releaseFile(pfd, f)

	if f.rights&pfsRightsBase != pfsRightsBase {
		return 0, wasiErrnoNotcapable
	}

	dir, ok := f.f.(Directory)
	if !ok {
		return 0, wasiErrnoNotdir
	}

	fd, wasiFile, err := m.files.allocate(pfsRightsBase, pfsRightsInheriting)
	if err != wasiErrnoSuccess {
		return 0, err
	}

	fsFile, ferr := dir.Open(path, int(poflags), int(pfdflags))
	if ferr != nil {
		return 0, fileErrno(ferr)
	}

	wasiFile.open, wasiFile.fdflags, wasiFile.f = true, pfdflags, fsFile
	return fd, wasiErrnoSuccess
}

// Read the contents of a symbolic link.
// Note: This is similar to `readlinkat` in POSIX.
func (m *wasiSnapshotPreview1Impl) pathReadlink(pfd wasiFd, ppath list, pbuf pointer, pbufLen wasiSize) (rv wasiSize, err wasiErrno) {
	path, err := m.loadPath(ppath)
	if err != wasiErrnoSuccess {
		return 0, err
	}

	dir, err := m.files.getDirectory(pfd, wasiRightsPathReadlink)
	if err != wasiErrnoSuccess {
		return 0, err
	}

	dest, ferr := dir.ReadLink(path)
	if ferr != nil {
		return 0, fileErrno(ferr)
	}
	return wasiSize(copy(m.slice(pbuf, pbufLen), dest)), wasiErrnoSuccess
}

// Remove a directory.
// Return `errno::notempty` if the directory is not empty.
// Note: This is similar to `unlinkat(fd, path, AT_REMOVEDIR)` in POSIX.
func (m *wasiSnapshotPreview1Impl) pathRemoveDirectory(pfd wasiFd, ppath list) (err wasiErrno) {
	path, err := m.loadPath(ppath)
	if err != wasiErrnoSuccess {
		return err
	}

	dir, err := m.files.getDirectory(pfd, wasiRightsPathRemoveDirectory)
	if err != wasiErrnoSuccess {
		return err
	}

	if ferr := dir.Rmdir(path); ferr != nil {
		return fileErrno(ferr)
	}
	return wasiErrnoSuccess
}

// Rename a file or directory.
// Note: This is similar to `renameat` in POSIX.
func (m *wasiSnapshotPreview1Impl) pathRename(pfd wasiFd, poldPath list, pnewFd wasiFd, pnewPath list) (err wasiErrno) {
	oldPath, err := m.loadPath(poldPath)
	if err != wasiErrnoSuccess {
		return err
	}
	newPath, err := m.loadPath(pnewPath)
	if err != wasiErrnoSuccess {
		return err
	}

	oldDir, err := m.files.getDirectory(pfd, wasiRightsPathRenameSource)
	if err != wasiErrnoSuccess {
		return err
	}

	newDir, err := m.files.getDirectory(pnewFd, wasiRightsPathRenameTarget)
	if err != wasiErrnoSuccess {
		return err
	}

	if ferr := m.fs.Rename(oldDir, oldPath, newDir, newPath); ferr != nil {
		return fileErrno(ferr)
	}

	return wasiErrnoSuccess
}

// Create a symbolic link.
// Note: This is similar to `symlinkat` in POSIX.
func (m *wasiSnapshotPreview1Impl) pathSymlink(poldPath list, pfd wasiFd, pnewPath list) (err wasiErrno) {
	oldPath, err := m.loadPath(poldPath)
	if err != wasiErrnoSuccess {
		return err
	}
	newPath, err := m.loadPath(pnewPath)
	if err != wasiErrnoSuccess {
		return err
	}

	oldDir, err := m.files.getDirectory(pfd, wasiRightsPathSymlink)
	if err != wasiErrnoSuccess {
		return err
	}

	newDir, err := m.files.getDirectory(pfd, wasiRightsPathSymlink)
	if err != wasiErrnoSuccess {
		return err
	}

	if ferr := m.fs.Symlink(oldDir, oldPath, newDir, newPath); ferr != nil {
		return fileErrno(ferr)
	}

	return wasiErrnoSuccess
}

// Unlink a file.
// Return `errno::isdir` if the path refers to a directory.
// Note: This is similar to `unlinkat(fd, path, 0)` in POSIX.
func (m *wasiSnapshotPreview1Impl) pathUnlinkFile(pfd wasiFd, ppath list) (err wasiErrno) {
	path, err := m.loadPath(ppath)
	if err != wasiErrnoSuccess {
		return err
	}

	dir, err := m.files.getDirectory(pfd, wasiRightsPathUnlinkFile)
	if err != wasiErrnoSuccess {
		return err
	}

	if ferr := dir.UnlinkFile(path); ferr != nil {
		return fileErrno(ferr)
	}
	return wasiErrnoSuccess
}

// Concurrently poll for the occurrence of a set of events.
func (m *wasiSnapshotPreview1Impl) pollOneoff(pin pointer, pout pointer, pnsubscriptions wasiSize) (rv wasiSize, err wasiErrno) {
	subscriptions := make([]Subscription, int(pnsubscriptions))
	for i := range subscriptions {
		var wasiSub wasiSubscription
		wasiSub.load(m.wasi.memory, uint32(pin), 0)
		size, align := wasiSub.layout()
		pin += pointer(alignTo(size, align))

		sub := &subscriptions[i]
		switch wasiSub.u.tag {
		case wasiEventtypeClock:
			sub.Kind = SubscriptionTimer

			switch wasiSub.u.clock.id {
			case wasiClockidMonotonic:
				timeout := time.Duration(wasiSub.u.clock.timeout) / time.Nanosecond
				if wasiSub.u.clock.flags&wasiSubclockflagsSubscriptionClockAbstime != 0 {
					timeout = time.Unix(0, int64(wasiSub.u.clock.timeout)).Sub(time.Now())
				}
				sub.Timeout = timeout
			case wasiClockidRealtime:
				deadline := time.Now().Add(time.Duration(wasiSub.u.clock.timeout) / time.Nanosecond)
				if wasiSub.u.clock.flags&wasiSubclockflagsSubscriptionClockAbstime != 0 {
					deadline = time.Unix(0, int64(wasiSub.u.clock.timeout))
				}
				sub.Deadline = deadline
			default:
				return 0, wasiErrnoNotsup
			}
		case wasiEventtypeFdRead:
			sub.Kind = SubscriptionRead

			f, err := m.files.getFile(wasiSub.u.fdRead.fileDescriptor, wasiRightsPollFdReadwrite)
			if err != wasiErrnoSuccess {
				return 0, err
			}

			sub.File = f
		case wasiEventtypeFdWrite:
			sub.Kind = SubscriptionWrite

			f, err := m.files.getFile(wasiSub.u.fdWrite.fileDescriptor, wasiRightsPollFdReadwrite)
			if err != wasiErrnoSuccess {
				return 0, err
			}

			sub.File = f
		default:
			return 0, wasiErrnoInval
		}

		sub.Userdata = wasiSub.userdata
	}

	events, ferr := m.fs.Poll(subscriptions)
	if ferr != nil {
		return 0, fileErrno(ferr)
	}

	for _, event := range events {
		var wasiEvent wasiEvent

		switch event.Kind {
		case SubscriptionTimer:
			wasiEvent.type_ = wasiEventtypeClock
		case SubscriptionRead:
			wasiEvent.type_ = wasiEventtypeFdRead
		case SubscriptionWrite:
			wasiEvent.type_ = wasiEventtypeFdWrite
		default:
			return 0, wasiErrnoInval
		}

		wasiEvent.fdReadwrite = wasiEventFdReadwrite{
			nbytes: wasiFilesize(event.Available),
			flags:  wasiEventrwflags(event.Flags),
		}
		wasiEvent.error = wasiErrno(event.Error)
		wasiEvent.userdata = event.Userdata

		wasiEvent.store(m.wasi.memory, uint32(pout), 0)
		size, align := wasiEvent.layout()
		pout += pointer(alignTo(size, align))
	}

	return wasiSize(len(events)), wasiErrnoSuccess
}

// Terminate the process normally. An exit code of 0 indicates successful
// termination of the program. The meanings of other values is dependent on
// the environment.
func (m *wasiSnapshotPreview1Impl) procExit(prval wasiExitcode) {
	panic(TrapExit(int(int32(prval))))
}

// Send a signal to the process of the calling thread.
// Note: This is similar to `raise` in POSIX.
func (m *wasiSnapshotPreview1Impl) procRaise(psig wasiSignal) (err wasiErrno) {
	err = wasiErrnoNotsup
	return
}

// Temporarily yield execution of the calling thread.
// Note: This is similar to `sched_yield` in POSIX.
func (m *wasiSnapshotPreview1Impl) schedYield() (err wasiErrno) {
	return wasiErrnoSuccess
}

// Write high-quality random data into a buffer.
// This function blocks when the implementation is unable to immediately
// provide sufficient high-quality random data.
// This function may execute slowly, so when large mounts of random data are
// required, it's advisable to use this function to seed a pseudo-random
// number generator, rather than to provide the random data directly.
func (m *wasiSnapshotPreview1Impl) randomGet(pbuf pointer, pbufLen wasiSize) (err wasiErrno) {
	_, rerr := rand.Read(m.slice(pbuf, pbufLen))
	if rerr != nil {
		return fileErrno(rerr)
	}
	return wasiErrnoSuccess
}

// Receive a message from a socket.
// Note: This is similar to `recv` in POSIX, though it also supports reading
// the data into multiple buffers in the manner of `readv`.
func (m *wasiSnapshotPreview1Impl) sockRecv(pfd wasiFd, priData list, priFlags wasiRiflags) (r0 wasiSize, r1 wasiRoflags, err wasiErrno) {
	err = wasiErrnoNotsup
	return
}

// Send a message on a socket.
// Note: This is similar to `send` in POSIX, though it also supports writing
// the data from multiple buffers in the manner of `writev`.
func (m *wasiSnapshotPreview1Impl) sockSend(pfd wasiFd, psiData list, psiFlags wasiSiflags) (rv wasiSize, err wasiErrno) {
	err = wasiErrnoNotsup
	return
}

// Shut down socket send and receive channels.
// Note: This is similar to `shutdown` in POSIX.
func (m *wasiSnapshotPreview1Impl) sockShutdown(pfd wasiFd, phow wasiSdflags) (err wasiErrno) {
	err = wasiErrnoNotsup
	return
}

func (m *wasiSnapshotPreview1Impl) buffers(iovs wasiIovecArray) [][]byte {
	buffers := make([][]byte, int(iovs.length))
	for i := range buffers {
		vec := iovs.loadIndex(m.wasi.memory, i)
		buffers[i] = m.slice(vec.buf, vec.bufLen)
	}
	return buffers
}

func (m *wasiSnapshotPreview1Impl) loadString(l list) string {
	bytes := m.wasi.memory.Bytes()
	return string(bytes[int(l.pointer) : int(l.pointer)+int(l.length)])
}

func (m *wasiSnapshotPreview1Impl) loadPath(l list) (string, wasiErrno) {
	path := path.Clean(m.loadString(l))
	if strings.HasPrefix(path, "..") {
		return "", wasiErrnoAcces
	}
	return path, wasiErrnoSuccess
}

func (m *wasiSnapshotPreview1Impl) bytes(p pointer) []byte {
	return m.wasi.memory.Bytes()[int(p):]
}

func (m *wasiSnapshotPreview1Impl) slice(p pointer, l wasiSize) []byte {
	bytes := m.wasi.memory.Bytes()

	if wasiSize(p)+l > wasiSize(len(bytes)) {
		return bytes[int(p):]
	}
	return bytes[int(p) : int(p)+int(l)]
}

func (m *wasiSnapshotPreview1Impl) byte(p pointer) byte {
	return m.wasi.memory.Byte(uint32(p), 0)
}

func (m *wasiSnapshotPreview1Impl) putByte(v byte, p pointer) {
	m.wasi.memory.PutByte(v, uint32(p), 0)
}

func (m *wasiSnapshotPreview1Impl) uint16(p pointer) uint16 {
	return m.wasi.memory.Uint16(uint32(p), 0)
}

func (m *wasiSnapshotPreview1Impl) putUint16(v uint16, p pointer) {
	m.wasi.memory.PutUint16(v, uint32(p), 0)
}

func (m *wasiSnapshotPreview1Impl) uint32(p pointer) uint32 {
	return m.wasi.memory.Uint32(uint32(p), 0)
}

func (m *wasiSnapshotPreview1Impl) putUint32(v uint32, p pointer) {
	m.wasi.memory.PutUint32(v, uint32(p), 0)
}

func (m *wasiSnapshotPreview1Impl) uint64(p pointer) uint64 {
	return m.wasi.memory.Uint64(uint32(p), 0)
}

func (m *wasiSnapshotPreview1Impl) putUint64(v uint64, p pointer) {
	m.wasi.memory.PutUint64(v, uint32(p), 0)
}

func filetype(mode os.FileMode) wasiFiletype {
	mode = mode & os.ModeType
	switch {
	case mode == 0:
		return wasiFiletypeRegularFile
	case mode&os.ModeDevice != 0:
		if mode&os.ModeCharDevice == 0 {
			return wasiFiletypeBlockDevice
		}
		return wasiFiletypeCharacterDevice
	case mode&os.ModeDir != 0:
		return wasiFiletypeDirectory
	case mode&os.ModeSocket != 0:
		return wasiFiletypeSocketStream
	case mode&os.ModeSymlink != 0:
		return wasiFiletypeSymbolicLink
	default:
		return wasiFiletypeUnknown
	}
}

func fileErrno(err error) wasiErrno {
	switch {
	case errors.Is(err, io.EOF):
		return wasiErrnoSuccess
	case errors.Is(err, io.ErrClosedPipe):
		return wasiErrnoPipe
	case errors.Is(err, os.ErrInvalid):
		return wasiErrnoInval
	case errors.Is(err, os.ErrPermission):
		return wasiErrnoPerm
	case errors.Is(err, os.ErrExist):
		return wasiErrnoExist
	case errors.Is(err, os.ErrNotExist):
		return wasiErrnoNoent
	case errors.Is(err, os.ErrClosed):
		return wasiErrnoBadf
	default:
		return wasiErrnoIo
	}
}

func alignTo(size, align uint32) uint32 {
	return size + align - 1 - ((size + align - 1) % align)
}
