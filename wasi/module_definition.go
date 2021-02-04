// THIS FILE IS AUTO-GENERATED from the following files:
//
//   wasi_snapshot_preview1.witx
//
// To regenerate this file execute:
//
//     cargo run --manifest-path gen-wasi/Cargo.toml generate-api
//
// Modifications to this file will cause CI to fail, the code generator tool
// must be modified to change this file.
//
// This file describes the [WASI] interface, consisting of functions, types,
// and defined values (macros).
//
// The interface described here is greatly inspired by [CloudABI]'s clean,
// thoughtfully-designed, capability-oriented, POSIX-style API.
//
// [CloudABI]: https://github.com/NuxiNL/cloudlibc
// [WASI]: https://github.com/WebAssembly/WASI/

package wasi

import (
	"errors"
	"reflect"

	"github.com/pgavlin/warp/exec"
	"github.com/pgavlin/warp/wasm"
)

type wasiSnapshotPreview1Definition int

func (def wasiSnapshotPreview1Definition) GetImports() []wasm.ImportEntry {
	return []wasm.ImportEntry{}
}

func (def wasiSnapshotPreview1Definition) Allocate(name string) (exec.AllocatedModule, error) {
	m := allocatedWasiSnapshotPreview1{
		wasiSnapshotPreview1: &wasiSnapshotPreview1{name: name},
	}
	return &m, nil
}

type wasiSnapshotPreview1 struct {
	name   string
	impl   *wasiSnapshotPreview1Impl
	memory *exec.Memory
}

type allocatedWasiSnapshotPreview1 struct {
	*wasiSnapshotPreview1
}

func (m *allocatedWasiSnapshotPreview1) Instantiate(imports exec.ImportResolver) (mod exec.Module, err error) {
	m.memory, err = imports.ResolveMemory("", "memory", wasm.Memory{})
	if err != nil {
		return nil, err
	}

	return m.wasiSnapshotPreview1, nil
}

func (m *wasiSnapshotPreview1) Name() string {
	return m.name
}

func (m *wasiSnapshotPreview1) GetTable(name string) (*exec.Table, error) {
	return nil, errors.New("unknown table")
}

func (m *wasiSnapshotPreview1) GetMemory(name string) (*exec.Memory, error) {
	return nil, errors.New("unknown memory")
}

func (m *wasiSnapshotPreview1) GetGlobal(name string) (*exec.Global, error) {
	return nil, errors.New("unknown global")
}

func (m *wasiSnapshotPreview1) GetFunction(name string) (exec.Function, error) {
	switch name {
	case "args_get":
		return exec.NewHostFunction(m, 0, reflect.ValueOf(m.wasiArgsGet)), nil
	case "args_sizes_get":
		return exec.NewHostFunction(m, 1, reflect.ValueOf(m.wasiArgsSizesGet)), nil
	case "environ_get":
		return exec.NewHostFunction(m, 2, reflect.ValueOf(m.wasiEnvironGet)), nil
	case "environ_sizes_get":
		return exec.NewHostFunction(m, 3, reflect.ValueOf(m.wasiEnvironSizesGet)), nil
	case "clock_res_get":
		return exec.NewHostFunction(m, 4, reflect.ValueOf(m.wasiClockResGet)), nil
	case "clock_time_get":
		return exec.NewHostFunction(m, 5, reflect.ValueOf(m.wasiClockTimeGet)), nil
	case "fd_advise":
		return exec.NewHostFunction(m, 6, reflect.ValueOf(m.wasiFdAdvise)), nil
	case "fd_allocate":
		return exec.NewHostFunction(m, 7, reflect.ValueOf(m.wasiFdAllocate)), nil
	case "fd_close":
		return exec.NewHostFunction(m, 8, reflect.ValueOf(m.wasiFdClose)), nil
	case "fd_datasync":
		return exec.NewHostFunction(m, 9, reflect.ValueOf(m.wasiFdDatasync)), nil
	case "fd_fdstat_get":
		return exec.NewHostFunction(m, 10, reflect.ValueOf(m.wasiFdFdstatGet)), nil
	case "fd_fdstat_set_flags":
		return exec.NewHostFunction(m, 11, reflect.ValueOf(m.wasiFdFdstatSetFlags)), nil
	case "fd_fdstat_set_rights":
		return exec.NewHostFunction(m, 12, reflect.ValueOf(m.wasiFdFdstatSetRights)), nil
	case "fd_filestat_get":
		return exec.NewHostFunction(m, 13, reflect.ValueOf(m.wasiFdFilestatGet)), nil
	case "fd_filestat_set_size":
		return exec.NewHostFunction(m, 14, reflect.ValueOf(m.wasiFdFilestatSetSize)), nil
	case "fd_filestat_set_times":
		return exec.NewHostFunction(m, 15, reflect.ValueOf(m.wasiFdFilestatSetTimes)), nil
	case "fd_pread":
		return exec.NewHostFunction(m, 16, reflect.ValueOf(m.wasiFdPread)), nil
	case "fd_prestat_get":
		return exec.NewHostFunction(m, 17, reflect.ValueOf(m.wasiFdPrestatGet)), nil
	case "fd_prestat_dir_name":
		return exec.NewHostFunction(m, 18, reflect.ValueOf(m.wasiFdPrestatDirName)), nil
	case "fd_pwrite":
		return exec.NewHostFunction(m, 19, reflect.ValueOf(m.wasiFdPwrite)), nil
	case "fd_read":
		return exec.NewHostFunction(m, 20, reflect.ValueOf(m.wasiFdRead)), nil
	case "fd_readdir":
		return exec.NewHostFunction(m, 21, reflect.ValueOf(m.wasiFdReaddir)), nil
	case "fd_renumber":
		return exec.NewHostFunction(m, 22, reflect.ValueOf(m.wasiFdRenumber)), nil
	case "fd_seek":
		return exec.NewHostFunction(m, 23, reflect.ValueOf(m.wasiFdSeek)), nil
	case "fd_sync":
		return exec.NewHostFunction(m, 24, reflect.ValueOf(m.wasiFdSync)), nil
	case "fd_tell":
		return exec.NewHostFunction(m, 25, reflect.ValueOf(m.wasiFdTell)), nil
	case "fd_write":
		return exec.NewHostFunction(m, 26, reflect.ValueOf(m.wasiFdWrite)), nil
	case "path_create_directory":
		return exec.NewHostFunction(m, 27, reflect.ValueOf(m.wasiPathCreateDirectory)), nil
	case "path_filestat_get":
		return exec.NewHostFunction(m, 28, reflect.ValueOf(m.wasiPathFilestatGet)), nil
	case "path_filestat_set_times":
		return exec.NewHostFunction(m, 29, reflect.ValueOf(m.wasiPathFilestatSetTimes)), nil
	case "path_link":
		return exec.NewHostFunction(m, 30, reflect.ValueOf(m.wasiPathLink)), nil
	case "path_open":
		return exec.NewHostFunction(m, 31, reflect.ValueOf(m.wasiPathOpen)), nil
	case "path_readlink":
		return exec.NewHostFunction(m, 32, reflect.ValueOf(m.wasiPathReadlink)), nil
	case "path_remove_directory":
		return exec.NewHostFunction(m, 33, reflect.ValueOf(m.wasiPathRemoveDirectory)), nil
	case "path_rename":
		return exec.NewHostFunction(m, 34, reflect.ValueOf(m.wasiPathRename)), nil
	case "path_symlink":
		return exec.NewHostFunction(m, 35, reflect.ValueOf(m.wasiPathSymlink)), nil
	case "path_unlink_file":
		return exec.NewHostFunction(m, 36, reflect.ValueOf(m.wasiPathUnlinkFile)), nil
	case "poll_oneoff":
		return exec.NewHostFunction(m, 37, reflect.ValueOf(m.wasiPollOneoff)), nil
	case "proc_exit":
		return exec.NewHostFunction(m, 38, reflect.ValueOf(m.wasiProcExit)), nil
	case "proc_raise":
		return exec.NewHostFunction(m, 39, reflect.ValueOf(m.wasiProcRaise)), nil
	case "sched_yield":
		return exec.NewHostFunction(m, 40, reflect.ValueOf(m.wasiSchedYield)), nil
	case "random_get":
		return exec.NewHostFunction(m, 41, reflect.ValueOf(m.wasiRandomGet)), nil
	case "sock_recv":
		return exec.NewHostFunction(m, 42, reflect.ValueOf(m.wasiSockRecv)), nil
	case "sock_send":
		return exec.NewHostFunction(m, 43, reflect.ValueOf(m.wasiSockSend)), nil
	case "sock_shutdown":
		return exec.NewHostFunction(m, 44, reflect.ValueOf(m.wasiSockShutdown)), nil
	default:
		return nil, errors.New("unknown function")
	}
}

func (m *wasiSnapshotPreview1) mem() *exec.Memory {
	return m.memory
}

// Read command-line argument data.
// The size of the array should match that returned by `args_sizes_get`
func (m *wasiSnapshotPreview1) wasiArgsGet(p0 int32, p1 int32) int32 {
	err := m.impl.argsGet(pointer(p0), pointer(p1))
	res := int32(wasiErrnoSuccess)
	if err != wasiErrnoSuccess {
		res = int32(err)
	}
	return res
}

// Return command-line argument data sizes.
func (m *wasiSnapshotPreview1) wasiArgsSizesGet(p0 int32, p1 int32) int32 {
	rv0, rv1, err := m.impl.argsSizesGet()
	res := int32(wasiErrnoSuccess)
	if err == wasiErrnoSuccess {
		m.mem().PutUint32(uint32(rv1), uint32(p1), 0)
		m.mem().PutUint32(uint32(rv0), uint32(p0), 0)
	} else {
		res = int32(err)
	}
	return res
}

// Read environment variable data.
// The sizes of the buffers should match that returned by `environ_sizes_get`.
func (m *wasiSnapshotPreview1) wasiEnvironGet(p0 int32, p1 int32) int32 {
	err := m.impl.environGet(pointer(p0), pointer(p1))
	res := int32(wasiErrnoSuccess)
	if err != wasiErrnoSuccess {
		res = int32(err)
	}
	return res
}

// Return environment variable data sizes.
func (m *wasiSnapshotPreview1) wasiEnvironSizesGet(p0 int32, p1 int32) int32 {
	rv0, rv1, err := m.impl.environSizesGet()
	res := int32(wasiErrnoSuccess)
	if err == wasiErrnoSuccess {
		m.mem().PutUint32(uint32(rv1), uint32(p1), 0)
		m.mem().PutUint32(uint32(rv0), uint32(p0), 0)
	} else {
		res = int32(err)
	}
	return res
}

// Return the resolution of a clock.
// Implementations are required to provide a non-zero value for supported clocks. For unsupported clocks,
// return `errno::inval`.
// Note: This is similar to `clock_getres` in POSIX.
func (m *wasiSnapshotPreview1) wasiClockResGet(p0 int32, p1 int32) int32 {
	rv, err := m.impl.clockResGet(uint32(p0))
	res := int32(wasiErrnoSuccess)
	if err == wasiErrnoSuccess {
		m.mem().PutUint64(uint64(rv), uint32(p1), 0)
	} else {
		res = int32(err)
	}
	return res
}

// Return the time value of a clock.
// Note: This is similar to `clock_gettime` in POSIX.
func (m *wasiSnapshotPreview1) wasiClockTimeGet(p0 int32, p1 int64, p2 int32) int32 {
	rv, err := m.impl.clockTimeGet(uint32(p0), uint64(p1))
	res := int32(wasiErrnoSuccess)
	if err == wasiErrnoSuccess {
		m.mem().PutUint64(uint64(rv), uint32(p2), 0)
	} else {
		res = int32(err)
	}
	return res
}

// Provide file advisory information on a file descriptor.
// Note: This is similar to `posix_fadvise` in POSIX.
func (m *wasiSnapshotPreview1) wasiFdAdvise(p0 int32, p1 int64, p2 int64, p3 int32) int32 {
	err := m.impl.fdAdvise(wasiFd(p0), uint64(p1), uint64(p2), uint8(p3))
	res := int32(wasiErrnoSuccess)
	if err != wasiErrnoSuccess {
		res = int32(err)
	}
	return res
}

// Force the allocation of space in a file.
// Note: This is similar to `posix_fallocate` in POSIX.
func (m *wasiSnapshotPreview1) wasiFdAllocate(p0 int32, p1 int64, p2 int64) int32 {
	err := m.impl.fdAllocate(wasiFd(p0), uint64(p1), uint64(p2))
	res := int32(wasiErrnoSuccess)
	if err != wasiErrnoSuccess {
		res = int32(err)
	}
	return res
}

// Close a file descriptor.
// Note: This is similar to `close` in POSIX.
func (m *wasiSnapshotPreview1) wasiFdClose(p0 int32) int32 {
	err := m.impl.fdClose(wasiFd(p0))
	res := int32(wasiErrnoSuccess)
	if err != wasiErrnoSuccess {
		res = int32(err)
	}
	return res
}

// Synchronize the data of a file to disk.
// Note: This is similar to `fdatasync` in POSIX.
func (m *wasiSnapshotPreview1) wasiFdDatasync(p0 int32) int32 {
	err := m.impl.fdDatasync(wasiFd(p0))
	res := int32(wasiErrnoSuccess)
	if err != wasiErrnoSuccess {
		res = int32(err)
	}
	return res
}

// Get the attributes of a file descriptor.
// Note: This returns similar flags to `fsync(fd, F_GETFL)` in POSIX, as well as additional fields.
func (m *wasiSnapshotPreview1) wasiFdFdstatGet(p0 int32, p1 int32) int32 {
	rv, err := m.impl.fdFdstatGet(wasiFd(p0))
	res := int32(wasiErrnoSuccess)
	if err == wasiErrnoSuccess {
		rv.store(m.mem(), uint32(p1), 0)
	} else {
		res = int32(err)
	}
	return res
}

// Adjust the flags associated with a file descriptor.
// Note: This is similar to `fcntl(fd, F_SETFL, flags)` in POSIX.
func (m *wasiSnapshotPreview1) wasiFdFdstatSetFlags(p0 int32, p1 int32) int32 {
	err := m.impl.fdFdstatSetFlags(wasiFd(p0), wasiFdflags(p1))
	res := int32(wasiErrnoSuccess)
	if err != wasiErrnoSuccess {
		res = int32(err)
	}
	return res
}

// Adjust the rights associated with a file descriptor.
// This can only be used to remove rights, and returns `errno::notcapable` if called in a way that would attempt to add rights
func (m *wasiSnapshotPreview1) wasiFdFdstatSetRights(p0 int32, p1 int64, p2 int64) int32 {
	err := m.impl.fdFdstatSetRights(wasiFd(p0), wasiRights(p1), wasiRights(p2))
	res := int32(wasiErrnoSuccess)
	if err != wasiErrnoSuccess {
		res = int32(err)
	}
	return res
}

// Return the attributes of an open file.
func (m *wasiSnapshotPreview1) wasiFdFilestatGet(p0 int32, p1 int32) int32 {
	rv, err := m.impl.fdFilestatGet(wasiFd(p0))
	res := int32(wasiErrnoSuccess)
	if err == wasiErrnoSuccess {
		rv.store(m.mem(), uint32(p1), 0)
	} else {
		res = int32(err)
	}
	return res
}

// Adjust the size of an open file. If this increases the file's size, the extra bytes are filled with zeros.
// Note: This is similar to `ftruncate` in POSIX.
func (m *wasiSnapshotPreview1) wasiFdFilestatSetSize(p0 int32, p1 int64) int32 {
	err := m.impl.fdFilestatSetSize(wasiFd(p0), uint64(p1))
	res := int32(wasiErrnoSuccess)
	if err != wasiErrnoSuccess {
		res = int32(err)
	}
	return res
}

// Adjust the timestamps of an open file or directory.
// Note: This is similar to `futimens` in POSIX.
func (m *wasiSnapshotPreview1) wasiFdFilestatSetTimes(p0 int32, p1 int64, p2 int64, p3 int32) int32 {
	err := m.impl.fdFilestatSetTimes(wasiFd(p0), uint64(p1), uint64(p2), wasiFstflags(p3))
	res := int32(wasiErrnoSuccess)
	if err != wasiErrnoSuccess {
		res = int32(err)
	}
	return res
}

// Read from a file descriptor, without using and updating the file descriptor's offset.
// Note: This is similar to `preadv` in POSIX.
func (m *wasiSnapshotPreview1) wasiFdPread(p0 int32, p1 int32, p2 int32, p3 int64, p4 int32) int32 {
	rv, err := m.impl.fdPread(wasiFd(p0), list{pointer: pointer(p1), length: int32(p2)}, uint64(p3))
	res := int32(wasiErrnoSuccess)
	if err == wasiErrnoSuccess {
		m.mem().PutUint32(uint32(rv), uint32(p4), 0)
	} else {
		res = int32(err)
	}
	return res
}

// Return a description of the given preopened file descriptor.
func (m *wasiSnapshotPreview1) wasiFdPrestatGet(p0 int32, p1 int32) int32 {
	rv, err := m.impl.fdPrestatGet(wasiFd(p0))
	res := int32(wasiErrnoSuccess)
	if err == wasiErrnoSuccess {
		rv.store(m.mem(), uint32(p1), 0)
	} else {
		res = int32(err)
	}
	return res
}

// Return a description of the given preopened file descriptor.
func (m *wasiSnapshotPreview1) wasiFdPrestatDirName(p0 int32, p1 int32, p2 int32) int32 {
	err := m.impl.fdPrestatDirName(wasiFd(p0), pointer(p1), uint32(p2))
	res := int32(wasiErrnoSuccess)
	if err != wasiErrnoSuccess {
		res = int32(err)
	}
	return res
}

// Write to a file descriptor, without using and updating the file descriptor's offset.
// Note: This is similar to `pwritev` in POSIX.
func (m *wasiSnapshotPreview1) wasiFdPwrite(p0 int32, p1 int32, p2 int32, p3 int64, p4 int32) int32 {
	rv, err := m.impl.fdPwrite(wasiFd(p0), list{pointer: pointer(p1), length: int32(p2)}, uint64(p3))
	res := int32(wasiErrnoSuccess)
	if err == wasiErrnoSuccess {
		m.mem().PutUint32(uint32(rv), uint32(p4), 0)
	} else {
		res = int32(err)
	}
	return res
}

// Read from a file descriptor.
// Note: This is similar to `readv` in POSIX.
func (m *wasiSnapshotPreview1) wasiFdRead(p0 int32, p1 int32, p2 int32, p3 int32) int32 {
	rv, err := m.impl.fdRead(wasiFd(p0), list{pointer: pointer(p1), length: int32(p2)})
	res := int32(wasiErrnoSuccess)
	if err == wasiErrnoSuccess {
		m.mem().PutUint32(uint32(rv), uint32(p3), 0)
	} else {
		res = int32(err)
	}
	return res
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
func (m *wasiSnapshotPreview1) wasiFdReaddir(p0 int32, p1 int32, p2 int32, p3 int64, p4 int32) int32 {
	rv, err := m.impl.fdReaddir(wasiFd(p0), pointer(p1), uint32(p2), uint64(p3))
	res := int32(wasiErrnoSuccess)
	if err == wasiErrnoSuccess {
		m.mem().PutUint32(uint32(rv), uint32(p4), 0)
	} else {
		res = int32(err)
	}
	return res
}

// Atomically replace a file descriptor by renumbering another file descriptor.
// Due to the strong focus on thread safety, this environment does not provide
// a mechanism to duplicate or renumber a file descriptor to an arbitrary
// number, like `dup2()`. This would be prone to race conditions, as an actual
// file descriptor with the same number could be allocated by a different
// thread at the same time.
// This function provides a way to atomically renumber file descriptors, which
// would disappear if `dup2()` were to be removed entirely.
func (m *wasiSnapshotPreview1) wasiFdRenumber(p0 int32, p1 int32) int32 {
	err := m.impl.fdRenumber(wasiFd(p0), wasiFd(p1))
	res := int32(wasiErrnoSuccess)
	if err != wasiErrnoSuccess {
		res = int32(err)
	}
	return res
}

// Move the offset of a file descriptor.
// Note: This is similar to `lseek` in POSIX.
func (m *wasiSnapshotPreview1) wasiFdSeek(p0 int32, p1 int64, p2 int32, p3 int32) int32 {
	rv, err := m.impl.fdSeek(wasiFd(p0), p1, uint8(p2))
	res := int32(wasiErrnoSuccess)
	if err == wasiErrnoSuccess {
		m.mem().PutUint64(uint64(rv), uint32(p3), 0)
	} else {
		res = int32(err)
	}
	return res
}

// Synchronize the data and metadata of a file to disk.
// Note: This is similar to `fsync` in POSIX.
func (m *wasiSnapshotPreview1) wasiFdSync(p0 int32) int32 {
	err := m.impl.fdSync(wasiFd(p0))
	res := int32(wasiErrnoSuccess)
	if err != wasiErrnoSuccess {
		res = int32(err)
	}
	return res
}

// Return the current offset of a file descriptor.
// Note: This is similar to `lseek(fd, 0, SEEK_CUR)` in POSIX.
func (m *wasiSnapshotPreview1) wasiFdTell(p0 int32, p1 int32) int32 {
	rv, err := m.impl.fdTell(wasiFd(p0))
	res := int32(wasiErrnoSuccess)
	if err == wasiErrnoSuccess {
		m.mem().PutUint64(uint64(rv), uint32(p1), 0)
	} else {
		res = int32(err)
	}
	return res
}

// Write to a file descriptor.
// Note: This is similar to `writev` in POSIX.
func (m *wasiSnapshotPreview1) wasiFdWrite(p0 int32, p1 int32, p2 int32, p3 int32) int32 {
	rv, err := m.impl.fdWrite(wasiFd(p0), list{pointer: pointer(p1), length: int32(p2)})
	res := int32(wasiErrnoSuccess)
	if err == wasiErrnoSuccess {
		m.mem().PutUint32(uint32(rv), uint32(p3), 0)
	} else {
		res = int32(err)
	}
	return res
}

// Create a directory.
// Note: This is similar to `mkdirat` in POSIX.
func (m *wasiSnapshotPreview1) wasiPathCreateDirectory(p0 int32, p1 int32, p2 int32) int32 {
	err := m.impl.pathCreateDirectory(wasiFd(p0), list{pointer: pointer(p1), length: int32(p2)})
	res := int32(wasiErrnoSuccess)
	if err != wasiErrnoSuccess {
		res = int32(err)
	}
	return res
}

// Return the attributes of a file or directory.
// Note: This is similar to `stat` in POSIX.
func (m *wasiSnapshotPreview1) wasiPathFilestatGet(p0 int32, p1 int32, p2 int32, p3 int32, p4 int32) int32 {
	rv, err := m.impl.pathFilestatGet(wasiFd(p0), wasiLookupflags(p1), list{pointer: pointer(p2), length: int32(p3)})
	res := int32(wasiErrnoSuccess)
	if err == wasiErrnoSuccess {
		rv.store(m.mem(), uint32(p4), 0)
	} else {
		res = int32(err)
	}
	return res
}

// Adjust the timestamps of a file or directory.
// Note: This is similar to `utimensat` in POSIX.
func (m *wasiSnapshotPreview1) wasiPathFilestatSetTimes(p0 int32, p1 int32, p2 int32, p3 int32, p4 int64, p5 int64, p6 int32) int32 {
	err := m.impl.pathFilestatSetTimes(wasiFd(p0), wasiLookupflags(p1), list{pointer: pointer(p2), length: int32(p3)}, uint64(p4), uint64(p5), wasiFstflags(p6))
	res := int32(wasiErrnoSuccess)
	if err != wasiErrnoSuccess {
		res = int32(err)
	}
	return res
}

// Create a hard link.
// Note: This is similar to `linkat` in POSIX.
func (m *wasiSnapshotPreview1) wasiPathLink(p0 int32, p1 int32, p2 int32, p3 int32, p4 int32, p5 int32, p6 int32) int32 {
	err := m.impl.pathLink(wasiFd(p0), wasiLookupflags(p1), list{pointer: pointer(p2), length: int32(p3)}, wasiFd(p4), list{pointer: pointer(p5), length: int32(p6)})
	res := int32(wasiErrnoSuccess)
	if err != wasiErrnoSuccess {
		res = int32(err)
	}
	return res
}

// Open a file or directory.
// The returned file descriptor is not guaranteed to be the lowest-numbered
// file descriptor not currently open; it is randomized to prevent
// applications from depending on making assumptions about indexes, since this
// is error-prone in multi-threaded contexts. The returned file descriptor is
// guaranteed to be less than 2**31.
// Note: This is similar to `openat` in POSIX.
func (m *wasiSnapshotPreview1) wasiPathOpen(p0 int32, p1 int32, p2 int32, p3 int32, p4 int32, p5 int64, p6 int64, p7 int32, p8 int32) int32 {
	rv, err := m.impl.pathOpen(wasiFd(p0), wasiLookupflags(p1), list{pointer: pointer(p2), length: int32(p3)}, wasiOflags(p4), wasiRights(p5), wasiRights(p6), wasiFdflags(p7))
	res := int32(wasiErrnoSuccess)
	if err == wasiErrnoSuccess {
		m.mem().PutUint32(uint32(rv), uint32(p8), 0)
	} else {
		res = int32(err)
	}
	return res
}

// Read the contents of a symbolic link.
// Note: This is similar to `readlinkat` in POSIX.
func (m *wasiSnapshotPreview1) wasiPathReadlink(p0 int32, p1 int32, p2 int32, p3 int32, p4 int32, p5 int32) int32 {
	rv, err := m.impl.pathReadlink(wasiFd(p0), list{pointer: pointer(p1), length: int32(p2)}, pointer(p3), uint32(p4))
	res := int32(wasiErrnoSuccess)
	if err == wasiErrnoSuccess {
		m.mem().PutUint32(uint32(rv), uint32(p5), 0)
	} else {
		res = int32(err)
	}
	return res
}

// Remove a directory.
// Return `errno::notempty` if the directory is not empty.
// Note: This is similar to `unlinkat(fd, path, AT_REMOVEDIR)` in POSIX.
func (m *wasiSnapshotPreview1) wasiPathRemoveDirectory(p0 int32, p1 int32, p2 int32) int32 {
	err := m.impl.pathRemoveDirectory(wasiFd(p0), list{pointer: pointer(p1), length: int32(p2)})
	res := int32(wasiErrnoSuccess)
	if err != wasiErrnoSuccess {
		res = int32(err)
	}
	return res
}

// Rename a file or directory.
// Note: This is similar to `renameat` in POSIX.
func (m *wasiSnapshotPreview1) wasiPathRename(p0 int32, p1 int32, p2 int32, p3 int32, p4 int32, p5 int32) int32 {
	err := m.impl.pathRename(wasiFd(p0), list{pointer: pointer(p1), length: int32(p2)}, wasiFd(p3), list{pointer: pointer(p4), length: int32(p5)})
	res := int32(wasiErrnoSuccess)
	if err != wasiErrnoSuccess {
		res = int32(err)
	}
	return res
}

// Create a symbolic link.
// Note: This is similar to `symlinkat` in POSIX.
func (m *wasiSnapshotPreview1) wasiPathSymlink(p0 int32, p1 int32, p2 int32, p3 int32, p4 int32) int32 {
	err := m.impl.pathSymlink(list{pointer: pointer(p0), length: int32(p1)}, wasiFd(p2), list{pointer: pointer(p3), length: int32(p4)})
	res := int32(wasiErrnoSuccess)
	if err != wasiErrnoSuccess {
		res = int32(err)
	}
	return res
}

// Unlink a file.
// Return `errno::isdir` if the path refers to a directory.
// Note: This is similar to `unlinkat(fd, path, 0)` in POSIX.
func (m *wasiSnapshotPreview1) wasiPathUnlinkFile(p0 int32, p1 int32, p2 int32) int32 {
	err := m.impl.pathUnlinkFile(wasiFd(p0), list{pointer: pointer(p1), length: int32(p2)})
	res := int32(wasiErrnoSuccess)
	if err != wasiErrnoSuccess {
		res = int32(err)
	}
	return res
}

// Concurrently poll for the occurrence of a set of events.
func (m *wasiSnapshotPreview1) wasiPollOneoff(p0 int32, p1 int32, p2 int32, p3 int32) int32 {
	rv, err := m.impl.pollOneoff(pointer(p0), pointer(p1), uint32(p2))
	res := int32(wasiErrnoSuccess)
	if err == wasiErrnoSuccess {
		m.mem().PutUint32(uint32(rv), uint32(p3), 0)
	} else {
		res = int32(err)
	}
	return res
}

// Terminate the process normally. An exit code of 0 indicates successful
// termination of the program. The meanings of other values is dependent on
// the environment.
func (m *wasiSnapshotPreview1) wasiProcExit(p0 int32) {
	m.impl.procExit(uint32(p0))
}

// Send a signal to the process of the calling thread.
// Note: This is similar to `raise` in POSIX.
func (m *wasiSnapshotPreview1) wasiProcRaise(p0 int32) int32 {
	err := m.impl.procRaise(uint8(p0))
	res := int32(wasiErrnoSuccess)
	if err != wasiErrnoSuccess {
		res = int32(err)
	}
	return res
}

// Temporarily yield execution of the calling thread.
// Note: This is similar to `sched_yield` in POSIX.
func (m *wasiSnapshotPreview1) wasiSchedYield() int32 {
	err := m.impl.schedYield()
	res := int32(wasiErrnoSuccess)
	if err != wasiErrnoSuccess {
		res = int32(err)
	}
	return res
}

// Write high-quality random data into a buffer.
// This function blocks when the implementation is unable to immediately
// provide sufficient high-quality random data.
// This function may execute slowly, so when large mounts of random data are
// required, it's advisable to use this function to seed a pseudo-random
// number generator, rather than to provide the random data directly.
func (m *wasiSnapshotPreview1) wasiRandomGet(p0 int32, p1 int32) int32 {
	err := m.impl.randomGet(pointer(p0), uint32(p1))
	res := int32(wasiErrnoSuccess)
	if err != wasiErrnoSuccess {
		res = int32(err)
	}
	return res
}

// Receive a message from a socket.
// Note: This is similar to `recv` in POSIX, though it also supports reading
// the data into multiple buffers in the manner of `readv`.
func (m *wasiSnapshotPreview1) wasiSockRecv(p0 int32, p1 int32, p2 int32, p3 int32, p4 int32, p5 int32) int32 {
	rv0, rv1, err := m.impl.sockRecv(wasiFd(p0), list{pointer: pointer(p1), length: int32(p2)}, wasiRiflags(p3))
	res := int32(wasiErrnoSuccess)
	if err == wasiErrnoSuccess {
		m.mem().PutUint16(uint16(rv1), uint32(p5), 0)
		m.mem().PutUint32(uint32(rv0), uint32(p4), 0)
	} else {
		res = int32(err)
	}
	return res
}

// Send a message on a socket.
// Note: This is similar to `send` in POSIX, though it also supports writing
// the data from multiple buffers in the manner of `writev`.
func (m *wasiSnapshotPreview1) wasiSockSend(p0 int32, p1 int32, p2 int32, p3 int32, p4 int32) int32 {
	rv, err := m.impl.sockSend(wasiFd(p0), list{pointer: pointer(p1), length: int32(p2)}, uint16(p3))
	res := int32(wasiErrnoSuccess)
	if err == wasiErrnoSuccess {
		m.mem().PutUint32(uint32(rv), uint32(p4), 0)
	} else {
		res = int32(err)
	}
	return res
}

// Shut down socket send and receive channels.
// Note: This is similar to `shutdown` in POSIX.
func (m *wasiSnapshotPreview1) wasiSockShutdown(p0 int32, p1 int32) int32 {
	err := m.impl.sockShutdown(wasiFd(p0), wasiSdflags(p1))
	res := int32(wasiErrnoSuccess)
	if err != wasiErrnoSuccess {
		res = int32(err)
	}
	return res
}
