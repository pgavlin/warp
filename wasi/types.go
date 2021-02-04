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
	"github.com/pgavlin/warp/exec"
)

type wasiSize = uint32

// Non-negative file size or length of a region within a file.
type wasiFilesize = uint64

// Timestamp in nanoseconds.
type wasiTimestamp = uint64

// Identifiers for clocks.
type wasiClockid = uint32

// The clock measuring real time. Time value zero corresponds with
// 1970-01-01T00:00:00Z.
const wasiClockidRealtime = 0

// The store-wide monotonic clock, which is defined as a clock measuring
// real time, whose value cannot be adjusted and which cannot have negative
// clock jumps. The epoch of this clock is undefined. The absolute time
// value of this clock therefore has no meaning.
const wasiClockidMonotonic = 1

// The CPU-time clock associated with the current process.
const wasiClockidProcessCputimeId = 2

// The CPU-time clock associated with the current thread.
const wasiClockidThreadCputimeId = 3

// Error codes returned by functions.
// Not all of these error codes are returned by the functions provided by this
// API; some are used in higher-level library layers, and others are provided
// merely for alignment with POSIX.
type wasiErrno = uint16

// No error occurred. System call completed successfully.
const wasiErrnoSuccess = 0

// Argument list too long.
const wasiErrno2big = 1

// Permission denied.
const wasiErrnoAcces = 2

// Address in use.
const wasiErrnoAddrinuse = 3

// Address not available.
const wasiErrnoAddrnotavail = 4

// Address family not supported.
const wasiErrnoAfnosupport = 5

// Resource unavailable, or operation would block.
const wasiErrnoAgain = 6

// Connection already in progress.
const wasiErrnoAlready = 7

// Bad file descriptor.
const wasiErrnoBadf = 8

// Bad message.
const wasiErrnoBadmsg = 9

// Device or resource busy.
const wasiErrnoBusy = 10

// Operation canceled.
const wasiErrnoCanceled = 11

// No child processes.
const wasiErrnoChild = 12

// Connection aborted.
const wasiErrnoConnaborted = 13

// Connection refused.
const wasiErrnoConnrefused = 14

// Connection reset.
const wasiErrnoConnreset = 15

// Resource deadlock would occur.
const wasiErrnoDeadlk = 16

// Destination address required.
const wasiErrnoDestaddrreq = 17

// Mathematics argument out of domain of function.
const wasiErrnoDom = 18

// Reserved.
const wasiErrnoDquot = 19

// File exists.
const wasiErrnoExist = 20

// Bad address.
const wasiErrnoFault = 21

// File too large.
const wasiErrnoFbig = 22

// Host is unreachable.
const wasiErrnoHostunreach = 23

// Identifier removed.
const wasiErrnoIdrm = 24

// Illegal byte sequence.
const wasiErrnoIlseq = 25

// Operation in progress.
const wasiErrnoInprogress = 26

// Interrupted function.
const wasiErrnoIntr = 27

// Invalid argument.
const wasiErrnoInval = 28

// I/O error.
const wasiErrnoIo = 29

// Socket is connected.
const wasiErrnoIsconn = 30

// Is a directory.
const wasiErrnoIsdir = 31

// Too many levels of symbolic links.
const wasiErrnoLoop = 32

// File descriptor value too large.
const wasiErrnoMfile = 33

// Too many links.
const wasiErrnoMlink = 34

// Message too large.
const wasiErrnoMsgsize = 35

// Reserved.
const wasiErrnoMultihop = 36

// Filename too long.
const wasiErrnoNametoolong = 37

// Network is down.
const wasiErrnoNetdown = 38

// Connection aborted by network.
const wasiErrnoNetreset = 39

// Network unreachable.
const wasiErrnoNetunreach = 40

// Too many files open in system.
const wasiErrnoNfile = 41

// No buffer space available.
const wasiErrnoNobufs = 42

// No such device.
const wasiErrnoNodev = 43

// No such file or directory.
const wasiErrnoNoent = 44

// Executable file format error.
const wasiErrnoNoexec = 45

// No locks available.
const wasiErrnoNolck = 46

// Reserved.
const wasiErrnoNolink = 47

// Not enough space.
const wasiErrnoNomem = 48

// No message of the desired type.
const wasiErrnoNomsg = 49

// Protocol not available.
const wasiErrnoNoprotoopt = 50

// No space left on device.
const wasiErrnoNospc = 51

// Function not supported.
const wasiErrnoNosys = 52

// The socket is not connected.
const wasiErrnoNotconn = 53

// Not a directory or a symbolic link to a directory.
const wasiErrnoNotdir = 54

// Directory not empty.
const wasiErrnoNotempty = 55

// State not recoverable.
const wasiErrnoNotrecoverable = 56

// Not a socket.
const wasiErrnoNotsock = 57

// Not supported, or operation not supported on socket.
const wasiErrnoNotsup = 58

// Inappropriate I/O control operation.
const wasiErrnoNotty = 59

// No such device or address.
const wasiErrnoNxio = 60

// Value too large to be stored in data type.
const wasiErrnoOverflow = 61

// Previous owner died.
const wasiErrnoOwnerdead = 62

// Operation not permitted.
const wasiErrnoPerm = 63

// Broken pipe.
const wasiErrnoPipe = 64

// Protocol error.
const wasiErrnoProto = 65

// Protocol not supported.
const wasiErrnoProtonosupport = 66

// Protocol wrong type for socket.
const wasiErrnoPrototype = 67

// Result too large.
const wasiErrnoRange = 68

// Read-only file system.
const wasiErrnoRofs = 69

// Invalid seek.
const wasiErrnoSpipe = 70

// No such process.
const wasiErrnoSrch = 71

// Reserved.
const wasiErrnoStale = 72

// Connection timed out.
const wasiErrnoTimedout = 73

// Text file busy.
const wasiErrnoTxtbsy = 74

// Cross-device link.
const wasiErrnoXdev = 75

// Extension: Capabilities insufficient.
const wasiErrnoNotcapable = 76

// File descriptor rights, determining which actions may be performed.
type wasiRights = uint64

// The right to invoke `fd_datasync`.
// If `path_open` is set, includes the right to invoke
// `path_open` with `fdflags::dsync`.
const wasiRightsFdDatasync = 1 << 0

// The right to invoke `fd_read` and `sock_recv`.
// If `rights::fd_seek` is set, includes the right to invoke `fd_pread`.
const wasiRightsFdRead = 1 << 1

// The right to invoke `fd_seek`. This flag implies `rights::fd_tell`.
const wasiRightsFdSeek = 1 << 2

// The right to invoke `fd_fdstat_set_flags`.
const wasiRightsFdFdstatSetFlags = 1 << 3

// The right to invoke `fd_sync`.
// If `path_open` is set, includes the right to invoke
// `path_open` with `fdflags::rsync` and `fdflags::dsync`.
const wasiRightsFdSync = 1 << 4

// The right to invoke `fd_seek` in such a way that the file offset
// remains unaltered (i.e., `whence::cur` with offset zero), or to
// invoke `fd_tell`.
const wasiRightsFdTell = 1 << 5

// The right to invoke `fd_write` and `sock_send`.
// If `rights::fd_seek` is set, includes the right to invoke `fd_pwrite`.
const wasiRightsFdWrite = 1 << 6

// The right to invoke `fd_advise`.
const wasiRightsFdAdvise = 1 << 7

// The right to invoke `fd_allocate`.
const wasiRightsFdAllocate = 1 << 8

// The right to invoke `path_create_directory`.
const wasiRightsPathCreateDirectory = 1 << 9

// If `path_open` is set, the right to invoke `path_open` with `oflags::creat`.
const wasiRightsPathCreateFile = 1 << 10

// The right to invoke `path_link` with the file descriptor as the
// source directory.
const wasiRightsPathLinkSource = 1 << 11

// The right to invoke `path_link` with the file descriptor as the
// target directory.
const wasiRightsPathLinkTarget = 1 << 12

// The right to invoke `path_open`.
const wasiRightsPathOpen = 1 << 13

// The right to invoke `fd_readdir`.
const wasiRightsFdReaddir = 1 << 14

// The right to invoke `path_readlink`.
const wasiRightsPathReadlink = 1 << 15

// The right to invoke `path_rename` with the file descriptor as the source directory.
const wasiRightsPathRenameSource = 1 << 16

// The right to invoke `path_rename` with the file descriptor as the target directory.
const wasiRightsPathRenameTarget = 1 << 17

// The right to invoke `path_filestat_get`.
const wasiRightsPathFilestatGet = 1 << 18

// The right to change a file's size (there is no `path_filestat_set_size`).
// If `path_open` is set, includes the right to invoke `path_open` with `oflags::trunc`.
const wasiRightsPathFilestatSetSize = 1 << 19

// The right to invoke `path_filestat_set_times`.
const wasiRightsPathFilestatSetTimes = 1 << 20

// The right to invoke `fd_filestat_get`.
const wasiRightsFdFilestatGet = 1 << 21

// The right to invoke `fd_filestat_set_size`.
const wasiRightsFdFilestatSetSize = 1 << 22

// The right to invoke `fd_filestat_set_times`.
const wasiRightsFdFilestatSetTimes = 1 << 23

// The right to invoke `path_symlink`.
const wasiRightsPathSymlink = 1 << 24

// The right to invoke `path_remove_directory`.
const wasiRightsPathRemoveDirectory = 1 << 25

// The right to invoke `path_unlink_file`.
const wasiRightsPathUnlinkFile = 1 << 26

// If `rights::fd_read` is set, includes the right to invoke `poll_oneoff` to subscribe to `eventtype::fd_read`.
// If `rights::fd_write` is set, includes the right to invoke `poll_oneoff` to subscribe to `eventtype::fd_write`.
const wasiRightsPollFdReadwrite = 1 << 27

// The right to invoke `sock_shutdown`.
const wasiRightsSockShutdown = 1 << 28

// A file descriptor handle.
type wasiFd = handle

// A region of memory for scatter/gather reads.
type wasiIovec struct {
	// The address of the buffer to be filled.
	buf pointer
	// The length of the buffer to be filled.
	bufLen wasiSize
}

func (v *wasiIovec) layout() (uint32, uint32) {
	return 8, 4
}

func (v *wasiIovec) store(mem *exec.Memory, addr, offset uint32) {
	base := addr + offset
	mem.PutUint32(uint32(v.buf), uint32(base), 0)
	mem.PutUint32(uint32(v.bufLen), uint32(base), 4)
}

func (v *wasiIovec) load(mem *exec.Memory, addr, offset uint32) {
	base := addr + offset
	v.buf = pointer(mem.Uint32(uint32(base), 0))
	v.bufLen = wasiSize(mem.Uint32(uint32(base), 4))
}

// A region of memory for scatter/gather writes.
type wasiCiovec struct {
	// The address of the buffer to be written.
	buf pointer
	// The length of the buffer to be written.
	bufLen wasiSize
}

func (v *wasiCiovec) layout() (uint32, uint32) {
	return 8, 4
}

func (v *wasiCiovec) store(mem *exec.Memory, addr, offset uint32) {
	base := addr + offset
	mem.PutUint32(uint32(v.buf), uint32(base), 0)
	mem.PutUint32(uint32(v.bufLen), uint32(base), 4)
}

func (v *wasiCiovec) load(mem *exec.Memory, addr, offset uint32) {
	base := addr + offset
	v.buf = pointer(mem.Uint32(uint32(base), 0))
	v.bufLen = wasiSize(mem.Uint32(uint32(base), 4))
}

type wasiIovecArray list

func (v *wasiIovecArray) elementSize() uint32 {
	return 8
}

func (l *wasiIovecArray) storeIndex(mem *exec.Memory, index int, value wasiIovec) {
	addr := uint32(l.pointer) + 8*uint32(index)
	value.store(mem, uint32(addr), 0)
}

func (l *wasiIovecArray) loadIndex(mem *exec.Memory, index int) wasiIovec {
	var value wasiIovec
	addr := uint32(l.pointer) + uint32(index)*8
	value.load(mem, uint32(addr), 0)
	return value
}

type wasiCiovecArray list

func (v *wasiCiovecArray) elementSize() uint32 {
	return 8
}

func (l *wasiCiovecArray) storeIndex(mem *exec.Memory, index int, value wasiCiovec) {
	addr := uint32(l.pointer) + 8*uint32(index)
	value.store(mem, uint32(addr), 0)
}

func (l *wasiCiovecArray) loadIndex(mem *exec.Memory, index int) wasiCiovec {
	var value wasiCiovec
	addr := uint32(l.pointer) + uint32(index)*8
	value.load(mem, uint32(addr), 0)
	return value
}

// Relative offset within a file.
type wasiFiledelta = int64

// The position relative to which to set the offset of the file descriptor.
type wasiWhence = uint8

// Seek relative to start-of-file.
const wasiWhenceSet = 0

// Seek relative to current position.
const wasiWhenceCur = 1

// Seek relative to end-of-file.
const wasiWhenceEnd = 2

// A reference to the offset of a directory entry.
//
// The value 0 signifies the start of the directory.
type wasiDircookie = uint64

// The type for the `dirent::d_namlen` field of `dirent` struct.
type wasiDirnamlen = uint32

// File serial number that is unique within its file system.
type wasiInode = uint64

// The type of a file descriptor or file.
type wasiFiletype = uint8

// The type of the file descriptor or file is unknown or is different from any of the other types specified.
const wasiFiletypeUnknown = 0

// The file descriptor or file refers to a block device inode.
const wasiFiletypeBlockDevice = 1

// The file descriptor or file refers to a character device inode.
const wasiFiletypeCharacterDevice = 2

// The file descriptor or file refers to a directory inode.
const wasiFiletypeDirectory = 3

// The file descriptor or file refers to a regular file inode.
const wasiFiletypeRegularFile = 4

// The file descriptor or file refers to a datagram socket.
const wasiFiletypeSocketDgram = 5

// The file descriptor or file refers to a byte-stream socket.
const wasiFiletypeSocketStream = 6

// The file refers to a symbolic link inode.
const wasiFiletypeSymbolicLink = 7

// A directory entry.
type wasiDirent struct {
	// The offset of the next directory entry stored in this directory.
	dNext wasiDircookie
	// The serial number of the file referred to by this directory entry.
	dIno wasiInode
	// The length of the name of the directory entry.
	dNamlen wasiDirnamlen
	// The type of the file referred to by this directory entry.
	dType wasiFiletype
}

func (v *wasiDirent) layout() (uint32, uint32) {
	return 24, 8
}

func (v *wasiDirent) store(mem *exec.Memory, addr, offset uint32) {
	base := addr + offset
	mem.PutUint64(uint64(v.dNext), uint32(base), 0)
	mem.PutUint64(uint64(v.dIno), uint32(base), 8)
	mem.PutUint32(uint32(v.dNamlen), uint32(base), 16)
	mem.PutByte(byte(v.dType), uint32(base), 20)
}

func (v *wasiDirent) load(mem *exec.Memory, addr, offset uint32) {
	base := addr + offset
	v.dNext = wasiDircookie(mem.Uint64(uint32(base), 0))
	v.dIno = wasiInode(mem.Uint64(uint32(base), 8))
	v.dNamlen = wasiDirnamlen(mem.Uint32(uint32(base), 16))
	v.dType = wasiFiletype(mem.Byte(uint32(base), 20))
}

// File or memory access pattern advisory information.
type wasiAdvice = uint8

// The application has no advice to give on its behavior with respect to the specified data.
const wasiAdviceNormal = 0

// The application expects to access the specified data sequentially from lower offsets to higher offsets.
const wasiAdviceSequential = 1

// The application expects to access the specified data in a random order.
const wasiAdviceRandom = 2

// The application expects to access the specified data in the near future.
const wasiAdviceWillneed = 3

// The application expects that it will not access the specified data in the near future.
const wasiAdviceDontneed = 4

// The application expects to access the specified data once and then not reuse it thereafter.
const wasiAdviceNoreuse = 5

// File descriptor flags.
type wasiFdflags = uint16

// Append mode: Data written to the file is always appended to the file's end.
const wasiFdflagsAppend = 1 << 0

// Write according to synchronized I/O data integrity completion. Only the data stored in the file is synchronized.
const wasiFdflagsDsync = 1 << 1

// Non-blocking mode.
const wasiFdflagsNonblock = 1 << 2

// Synchronized read I/O operations.
const wasiFdflagsRsync = 1 << 3

// Write according to synchronized I/O file integrity completion. In
// addition to synchronizing the data stored in the file, the implementation
// may also synchronously update the file's metadata.
const wasiFdflagsSync = 1 << 4

// File descriptor attributes.
type wasiFdstat struct {
	// File type.
	fsFiletype wasiFiletype
	// File descriptor flags.
	fsFlags wasiFdflags
	// Rights that apply to this file descriptor.
	fsRightsBase wasiRights
	// Maximum set of rights that may be installed on new file descriptors that
	// are created through this file descriptor, e.g., through `path_open`.
	fsRightsInheriting wasiRights
}

func (v *wasiFdstat) layout() (uint32, uint32) {
	return 24, 8
}

func (v *wasiFdstat) store(mem *exec.Memory, addr, offset uint32) {
	base := addr + offset
	mem.PutByte(byte(v.fsFiletype), uint32(base), 0)
	mem.PutUint16(uint16(v.fsFlags), uint32(base), 2)
	mem.PutUint64(uint64(v.fsRightsBase), uint32(base), 8)
	mem.PutUint64(uint64(v.fsRightsInheriting), uint32(base), 16)
}

func (v *wasiFdstat) load(mem *exec.Memory, addr, offset uint32) {
	base := addr + offset
	v.fsFiletype = wasiFiletype(mem.Byte(uint32(base), 0))
	v.fsFlags = wasiFdflags(mem.Uint16(uint32(base), 2))
	v.fsRightsBase = wasiRights(mem.Uint64(uint32(base), 8))
	v.fsRightsInheriting = wasiRights(mem.Uint64(uint32(base), 16))
}

// Identifier for a device containing a file system. Can be used in combination
// with `inode` to uniquely identify a file or directory in the filesystem.
type wasiDevice = uint64

// Which file time attributes to adjust.
type wasiFstflags = uint16

// Adjust the last data access timestamp to the value stored in `filestat::atim`.
const wasiFstflagsAtim = 1 << 0

// Adjust the last data access timestamp to the time of clock `clockid::realtime`.
const wasiFstflagsAtimNow = 1 << 1

// Adjust the last data modification timestamp to the value stored in `filestat::mtim`.
const wasiFstflagsMtim = 1 << 2

// Adjust the last data modification timestamp to the time of clock `clockid::realtime`.
const wasiFstflagsMtimNow = 1 << 3

// Flags determining the method of how paths are resolved.
type wasiLookupflags = uint32

// As long as the resolved path corresponds to a symbolic link, it is expanded.
const wasiLookupflagsSymlinkFollow = 1 << 0

// Open flags used by `path_open`.
type wasiOflags = uint16

// Create file if it does not exist.
const wasiOflagsCreat = 1 << 0

// Fail if not a directory.
const wasiOflagsDirectory = 1 << 1

// Fail if file already exists.
const wasiOflagsExcl = 1 << 2

// Truncate file to size 0.
const wasiOflagsTrunc = 1 << 3

// Number of hard links to an inode.
type wasiLinkcount = uint64

// File attributes.
type wasiFilestat struct {
	// Device ID of device containing the file.
	dev wasiDevice
	// File serial number.
	ino wasiInode
	// File type.
	filetype wasiFiletype
	// Number of hard links to the file.
	nlink wasiLinkcount
	// For regular files, the file size in bytes. For symbolic links, the length in bytes of the pathname contained in the symbolic link.
	size wasiFilesize
	// Last data access timestamp.
	atim wasiTimestamp
	// Last data modification timestamp.
	mtim wasiTimestamp
	// Last file status change timestamp.
	ctim wasiTimestamp
}

func (v *wasiFilestat) layout() (uint32, uint32) {
	return 64, 8
}

func (v *wasiFilestat) store(mem *exec.Memory, addr, offset uint32) {
	base := addr + offset
	mem.PutUint64(uint64(v.dev), uint32(base), 0)
	mem.PutUint64(uint64(v.ino), uint32(base), 8)
	mem.PutByte(byte(v.filetype), uint32(base), 16)
	mem.PutUint64(uint64(v.nlink), uint32(base), 24)
	mem.PutUint64(uint64(v.size), uint32(base), 32)
	mem.PutUint64(uint64(v.atim), uint32(base), 40)
	mem.PutUint64(uint64(v.mtim), uint32(base), 48)
	mem.PutUint64(uint64(v.ctim), uint32(base), 56)
}

func (v *wasiFilestat) load(mem *exec.Memory, addr, offset uint32) {
	base := addr + offset
	v.dev = wasiDevice(mem.Uint64(uint32(base), 0))
	v.ino = wasiInode(mem.Uint64(uint32(base), 8))
	v.filetype = wasiFiletype(mem.Byte(uint32(base), 16))
	v.nlink = wasiLinkcount(mem.Uint64(uint32(base), 24))
	v.size = wasiFilesize(mem.Uint64(uint32(base), 32))
	v.atim = wasiTimestamp(mem.Uint64(uint32(base), 40))
	v.mtim = wasiTimestamp(mem.Uint64(uint32(base), 48))
	v.ctim = wasiTimestamp(mem.Uint64(uint32(base), 56))
}

// User-provided value that may be attached to objects that is retained when
// extracted from the implementation.
type wasiUserdata = uint64

// Type of a subscription to an event or its occurrence.
type wasiEventtype = uint8

// The time value of clock `subscription_clock::id` has
// reached timestamp `subscription_clock::timeout`.
const wasiEventtypeClock = 0

// File descriptor `subscription_fd_readwrite::file_descriptor` has data
// available for reading. This event always triggers for regular files.
const wasiEventtypeFdRead = 1

// File descriptor `subscription_fd_readwrite::file_descriptor` has capacity
// available for writing. This event always triggers for regular files.
const wasiEventtypeFdWrite = 2

// The state of the file descriptor subscribed to with
// `eventtype::fd_read` or `eventtype::fd_write`.
type wasiEventrwflags = uint16

// The peer of this socket has closed or disconnected.
const wasiEventrwflagsFdReadwriteHangup = 1 << 0

// The contents of an `event` when type is `eventtype::fd_read` or
// `eventtype::fd_write`.
type wasiEventFdReadwrite struct {
	// The number of bytes available for reading or writing.
	nbytes wasiFilesize
	// The state of the file descriptor.
	flags wasiEventrwflags
}

func (v *wasiEventFdReadwrite) layout() (uint32, uint32) {
	return 16, 8
}

func (v *wasiEventFdReadwrite) store(mem *exec.Memory, addr, offset uint32) {
	base := addr + offset
	mem.PutUint64(uint64(v.nbytes), uint32(base), 0)
	mem.PutUint16(uint16(v.flags), uint32(base), 8)
}

func (v *wasiEventFdReadwrite) load(mem *exec.Memory, addr, offset uint32) {
	base := addr + offset
	v.nbytes = wasiFilesize(mem.Uint64(uint32(base), 0))
	v.flags = wasiEventrwflags(mem.Uint16(uint32(base), 8))
}

// An event that occurred.
type wasiEvent struct {
	// User-provided value that got attached to `subscription::userdata`.
	userdata wasiUserdata
	// If non-zero, an error that occurred while processing the subscription request.
	error wasiErrno
	// The type of event that occured
	type_ wasiEventtype
	// The contents of the event, if it is an `eventtype::fd_read` or
	// `eventtype::fd_write`. `eventtype::clock` events ignore this field.
	fdReadwrite wasiEventFdReadwrite
}

func (v *wasiEvent) layout() (uint32, uint32) {
	return 32, 8
}

func (v *wasiEvent) store(mem *exec.Memory, addr, offset uint32) {
	base := addr + offset
	mem.PutUint64(uint64(v.userdata), uint32(base), 0)
	mem.PutUint16(uint16(v.error), uint32(base), 8)
	mem.PutByte(byte(v.type_), uint32(base), 10)
	v.fdReadwrite.store(mem, uint32(base), 16)
}

func (v *wasiEvent) load(mem *exec.Memory, addr, offset uint32) {
	base := addr + offset
	v.userdata = wasiUserdata(mem.Uint64(uint32(base), 0))
	v.error = wasiErrno(mem.Uint16(uint32(base), 8))
	v.type_ = wasiEventtype(mem.Byte(uint32(base), 10))
	v.fdReadwrite.load(mem, uint32(base), 16)
}

// Flags determining how to interpret the timestamp provided in
// `subscription_clock::timeout`.
type wasiSubclockflags = uint16

// If set, treat the timestamp provided in
// `subscription_clock::timeout` as an absolute timestamp of clock
// `subscription_clock::id`. If clear, treat the timestamp
// provided in `subscription_clock::timeout` relative to the
// current time value of clock `subscription_clock::id`.
const wasiSubclockflagsSubscriptionClockAbstime = 1 << 0

// The contents of a `subscription` when type is `eventtype::clock`.
type wasiSubscriptionClock struct {
	// The clock against which to compare the timestamp.
	id wasiClockid
	// The absolute or relative timestamp.
	timeout wasiTimestamp
	// The amount of time that the implementation may wait additionally
	// to coalesce with other events.
	precision wasiTimestamp
	// Flags specifying whether the timeout is absolute or relative
	flags wasiSubclockflags
}

func (v *wasiSubscriptionClock) layout() (uint32, uint32) {
	return 32, 8
}

func (v *wasiSubscriptionClock) store(mem *exec.Memory, addr, offset uint32) {
	base := addr + offset
	mem.PutUint32(uint32(v.id), uint32(base), 0)
	mem.PutUint64(uint64(v.timeout), uint32(base), 8)
	mem.PutUint64(uint64(v.precision), uint32(base), 16)
	mem.PutUint16(uint16(v.flags), uint32(base), 24)
}

func (v *wasiSubscriptionClock) load(mem *exec.Memory, addr, offset uint32) {
	base := addr + offset
	v.id = wasiClockid(mem.Uint32(uint32(base), 0))
	v.timeout = wasiTimestamp(mem.Uint64(uint32(base), 8))
	v.precision = wasiTimestamp(mem.Uint64(uint32(base), 16))
	v.flags = wasiSubclockflags(mem.Uint16(uint32(base), 24))
}

// The contents of a `subscription` when type is type is
// `eventtype::fd_read` or `eventtype::fd_write`.
type wasiSubscriptionFdReadwrite struct {
	// The file descriptor on which to wait for it to become ready for reading or writing.
	fileDescriptor wasiFd
}

func (v *wasiSubscriptionFdReadwrite) layout() (uint32, uint32) {
	return 4, 4
}

func (v *wasiSubscriptionFdReadwrite) store(mem *exec.Memory, addr, offset uint32) {
	base := addr + offset
	mem.PutUint32(uint32(v.fileDescriptor), uint32(base), 0)
}

func (v *wasiSubscriptionFdReadwrite) load(mem *exec.Memory, addr, offset uint32) {
	base := addr + offset
	v.fileDescriptor = wasiFd(mem.Uint32(uint32(base), 0))
}

// The contents of a `subscription`.
type wasiSubscriptionU struct {
	tag uint8

	clock   wasiSubscriptionClock
	fdRead  wasiSubscriptionFdReadwrite
	fdWrite wasiSubscriptionFdReadwrite
}

func (v *wasiSubscriptionU) layout() (uint32, uint32) {
	return 40, 8
}

func (v *wasiSubscriptionU) store(mem *exec.Memory, addr, offset uint32) {
	base := addr + offset
	mem.PutByte(byte(v.tag), uint32(base), 0)
	switch v.tag {
	case 0:
		v.clock.store(mem, uint32(base), 8)
	case 1:
		v.fdRead.store(mem, uint32(base), 4)
	case 2:
		v.fdWrite.store(mem, uint32(base), 4)
	}
}

func (v *wasiSubscriptionU) load(mem *exec.Memory, addr, offset uint32) {
	base := addr + offset
	v.tag = uint8(mem.Byte(uint32(base), 0))
	switch v.tag {
	case 0:
		v.clock.load(mem, uint32(base), 8)
	case 1:
		v.fdRead.load(mem, uint32(base), 4)
	case 2:
		v.fdWrite.load(mem, uint32(base), 4)
	}
}

// Subscription to an event.
type wasiSubscription struct {
	// User-provided value that is attached to the subscription in the
	// implementation and returned through `event::userdata`.
	userdata wasiUserdata
	// The type of the event to which to subscribe, and its contents
	u wasiSubscriptionU
}

func (v *wasiSubscription) layout() (uint32, uint32) {
	return 48, 8
}

func (v *wasiSubscription) store(mem *exec.Memory, addr, offset uint32) {
	base := addr + offset
	mem.PutUint64(uint64(v.userdata), uint32(base), 0)
	v.u.store(mem, uint32(base), 8)
}

func (v *wasiSubscription) load(mem *exec.Memory, addr, offset uint32) {
	base := addr + offset
	v.userdata = wasiUserdata(mem.Uint64(uint32(base), 0))
	v.u.load(mem, uint32(base), 8)
}

// Exit code generated by a process when exiting.
type wasiExitcode = uint32

// Signal condition.
type wasiSignal = uint8

// No signal. Note that POSIX has special semantics for `kill(pid, 0)`,
// so this value is reserved.
const wasiSignalNone = 0

// Hangup.
// Action: Terminates the process.
const wasiSignalHup = 1

// Terminate interrupt signal.
// Action: Terminates the process.
const wasiSignalInt = 2

// Terminal quit signal.
// Action: Terminates the process.
const wasiSignalQuit = 3

// Illegal instruction.
// Action: Terminates the process.
const wasiSignalIll = 4

// Trace/breakpoint trap.
// Action: Terminates the process.
const wasiSignalTrap = 5

// Process abort signal.
// Action: Terminates the process.
const wasiSignalAbrt = 6

// Access to an undefined portion of a memory object.
// Action: Terminates the process.
const wasiSignalBus = 7

// Erroneous arithmetic operation.
// Action: Terminates the process.
const wasiSignalFpe = 8

// Kill.
// Action: Terminates the process.
const wasiSignalKill = 9

// User-defined signal 1.
// Action: Terminates the process.
const wasiSignalUsr1 = 10

// Invalid memory reference.
// Action: Terminates the process.
const wasiSignalSegv = 11

// User-defined signal 2.
// Action: Terminates the process.
const wasiSignalUsr2 = 12

// Write on a pipe with no one to read it.
// Action: Ignored.
const wasiSignalPipe = 13

// Alarm clock.
// Action: Terminates the process.
const wasiSignalAlrm = 14

// Termination signal.
// Action: Terminates the process.
const wasiSignalTerm = 15

// Child process terminated, stopped, or continued.
// Action: Ignored.
const wasiSignalChld = 16

// Continue executing, if stopped.
// Action: Continues executing, if stopped.
const wasiSignalCont = 17

// Stop executing.
// Action: Stops executing.
const wasiSignalStop = 18

// Terminal stop signal.
// Action: Stops executing.
const wasiSignalTstp = 19

// Background process attempting read.
// Action: Stops executing.
const wasiSignalTtin = 20

// Background process attempting write.
// Action: Stops executing.
const wasiSignalTtou = 21

// High bandwidth data is available at a socket.
// Action: Ignored.
const wasiSignalUrg = 22

// CPU time limit exceeded.
// Action: Terminates the process.
const wasiSignalXcpu = 23

// File size limit exceeded.
// Action: Terminates the process.
const wasiSignalXfsz = 24

// Virtual timer expired.
// Action: Terminates the process.
const wasiSignalVtalrm = 25

// Profiling timer expired.
// Action: Terminates the process.
const wasiSignalProf = 26

// Window changed.
// Action: Ignored.
const wasiSignalWinch = 27

// I/O possible.
// Action: Terminates the process.
const wasiSignalPoll = 28

// Power failure.
// Action: Terminates the process.
const wasiSignalPwr = 29

// Bad system call.
// Action: Terminates the process.
const wasiSignalSys = 30

// Flags provided to `sock_recv`.
type wasiRiflags = uint16

// Returns the message without removing it from the socket's receive queue.
const wasiRiflagsRecvPeek = 1 << 0

// On byte-stream sockets, block until the full amount of data can be returned.
const wasiRiflagsRecvWaitall = 1 << 1

// Flags returned by `sock_recv`.
type wasiRoflags = uint16

// Returned by `sock_recv`: Message data has been truncated.
const wasiRoflagsRecvDataTruncated = 1 << 0

// Flags provided to `sock_send`. As there are currently no flags
// defined, it must be set to zero.
type wasiSiflags = uint16

// Which channels on a socket to shut down.
type wasiSdflags = uint8

// Disables further receive operations.
const wasiSdflagsRd = 1 << 0

// Disables further send operations.
const wasiSdflagsWr = 1 << 1

// Identifiers for preopened capabilities.
type wasiPreopentype = uint8

// A pre-opened directory.
const wasiPreopentypeDir = 0

// The contents of a $prestat when type is `preopentype::dir`.
type wasiPrestatDir struct {
	// The length of the directory name for use with `fd_prestat_dir_name`.
	prNameLen wasiSize
}

func (v *wasiPrestatDir) layout() (uint32, uint32) {
	return 4, 4
}

func (v *wasiPrestatDir) store(mem *exec.Memory, addr, offset uint32) {
	base := addr + offset
	mem.PutUint32(uint32(v.prNameLen), uint32(base), 0)
}

func (v *wasiPrestatDir) load(mem *exec.Memory, addr, offset uint32) {
	base := addr + offset
	v.prNameLen = wasiSize(mem.Uint32(uint32(base), 0))
}

// Information about a pre-opened capability.
type wasiPrestat struct {
	tag uint8

	dir wasiPrestatDir
}

func (v *wasiPrestat) layout() (uint32, uint32) {
	return 8, 4
}

func (v *wasiPrestat) store(mem *exec.Memory, addr, offset uint32) {
	base := addr + offset
	mem.PutByte(byte(v.tag), uint32(base), 0)
	switch v.tag {
	case 0:
		v.dir.store(mem, uint32(base), 4)
	}
}

func (v *wasiPrestat) load(mem *exec.Memory, addr, offset uint32) {
	base := addr + offset
	v.tag = uint8(mem.Byte(uint32(base), 0))
	switch v.tag {
	case 0:
		v.dir.load(mem, uint32(base), 4)
	}
}
