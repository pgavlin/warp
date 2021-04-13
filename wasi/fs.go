package wasi

import (
	"io"
	"os"
	"path/filepath"
	"time"
)

const (
	// The application has no advice to give on its behavior with respect to the specified data.
	AdviceNormal = wasiAdviceNormal
	// The application expects to access the specified data sequentially from lower offsets to higher offsets.
	AdviceSequential = wasiAdviceSequential
	// The application expects to access the specified data in a random order.
	AdviceRandom = wasiAdviceRandom
	// The application expects to access the specified data in the near future.
	AdviceWillneed = wasiAdviceWillneed
	// The application expects that it will not access the specified data in the near future.
	AdviceDontneed = wasiAdviceDontneed
	// The application expects to access the specified data once and then not reuse it thereafter.
	AdviceNoreuse = wasiAdviceNoreuse

	// Append mode: Data written to the file is always appended to the file's end.
	F_Append = wasiFdflagsAppend
	// Write according to synchronized I/O data integrity completion. Only the data stored in the file is synchronized.
	F_Dsync = wasiFdflagsDsync
	// Non-blocking mode.
	F_Nonblock = wasiFdflagsNonblock
	// Synchronized read I/O operations.
	F_Rsync = wasiFdflagsRsync
	// Write according to synchronized I/O file integrity completion. In
	// addition to synchronizing the data stored in the file, the implementation
	// may also synchronously update the file's metadata.
	F_Sync = wasiFdflagsSync

	// Create file if it does not exist.
	O_Create = wasiOflagsCreat
	// Fail if not a directory.
	O_Directory = wasiOflagsDirectory
	// Fail if file already exists.
	O_Excl = wasiOflagsExcl
	// Truncate file to size 0.
	O_Trunc = wasiOflagsTrunc

	// The given timeout has expired or deadline has been reached.
	SubscriptionTimer = 0
	// The given file has data available for reading.
	SubscriptionRead = 1
	// The given file has data available for writing.
	SubscriptionWrite = 2

	// The peer of this socket has closed or disconnected.
	EventHangup = wasiEventrwflagsFdReadwriteHangup

	// ErrnoIO indicates that an I/O error occurred during Poll.
	ErrnoIO = wasiErrnoIo
)

type FileStat struct {
	Dev        uint64
	Inode      uint64
	Mode       os.FileMode
	LinkCount  uint64
	Size       uint64
	AccessTime time.Time
	ModTime    time.Time
	ChangeTime time.Time
}

type FDStat struct {
	FileStat

	Flags int
}

type Subscription struct {
	Kind     int
	File     File
	Timeout  time.Duration
	Deadline time.Time
	Userdata uint64
}

type Event struct {
	Kind      int
	Error     int
	Available uint
	Flags     int
	Userdata  uint64
}

type FS interface {
	OpenDirectory(path string) (Directory, error)
	Link(sourceDir Directory, sourceName string, targetDir Directory, targetName string) error
	Poll(subscriptions []Subscription) ([]Event, error)
	Rename(sourceDir Directory, sourceName string, targetDir Directory, targetName string) error
	Symlink(sourceDir Directory, sourceName string, targetDir Directory, targetName string) error
}

type Directory interface {
	File

	FileStat(path string, followSymlinks bool) (FileStat, error)
	Mkdir(path string) error
	Open(path string, oflags, fflags int) (File, error)
	ReadDir(n int) ([]os.DirEntry, error)
	ReadLink(path string) (string, error)
	Rmdir(path string) error
	SetFileTimes(path string, accessTime *time.Time, modTime *time.Time, followSymlinks bool) error
	UnlinkFile(path string) error
}

type File interface {
	Advise(offset, length uint64, advice int) error
	Close() error
	Datasync() error
	Pread(buffers [][]byte, offset int64) (uint32, error)
	Pwrite(buffers [][]byte, offset int64) (uint32, error)
	Readv(buffers [][]byte) (uint32, error)
	Seek(offset int64, whence int) (int64, error)
	SetFlags(flags int) error
	SetSize(size uint64) error
	SetTimes(accessTime *time.Time, modTime *time.Time) error
	Stat() (FDStat, error)
	Sync() error
	Writev(buffers [][]byte) (uint32, error)
}

type ErrFile int

var _ = File(ErrFile(0))

func (ErrFile) Advise(offset, length uint64, advice int) error {
	return os.ErrInvalid
}

func (ErrFile) Close() error {
	return nil
}

func (ErrFile) Datasync() error {
	return os.ErrInvalid
}

func (ErrFile) Pread(buffers [][]byte, offset int64) (uint32, error) {
	return 0, os.ErrInvalid
}

func (ErrFile) Pwrite(buffers [][]byte, offset int64) (uint32, error) {
	return 0, os.ErrInvalid
}

func (ErrFile) Readv(buffers [][]byte) (uint32, error) {
	return 0, os.ErrInvalid
}

func (ErrFile) Seek(offset int64, whence int) (int64, error) {
	return 0, os.ErrInvalid
}

func (ErrFile) SetFlags(flags int) error {
	return os.ErrInvalid
}

func (ErrFile) SetSize(size uint64) error {
	return os.ErrInvalid
}

func (ErrFile) SetTimes(accessTime *time.Time, modTime *time.Time) error {
	return os.ErrInvalid
}

func (ErrFile) Stat() (FDStat, error) {
	return FDStat{}, os.ErrInvalid
}

func (ErrFile) Sync() error {
	return os.ErrInvalid
}

func (ErrFile) Writev(buffers [][]byte) (uint32, error) {
	return 0, os.ErrInvalid
}

func Pread(r io.ReaderAt, buffers [][]byte, offset int64) (uint32, error) {
	read := uint32(0)
	for _, b := range buffers {
		n, err := r.ReadAt(b, offset)
		read, offset = read+uint32(n), offset+int64(n)

		if err != nil {
			return read, err
		}
	}
	return read, nil
}

func Pwrite(w io.WriterAt, buffers [][]byte, offset int64) (uint32, error) {
	written := uint32(0)
	for _, b := range buffers {
		n, err := w.WriteAt(b, offset)
		written, offset = written+uint32(n), offset+int64(n)

		if err != nil {
			return written, err
		}
	}
	return written, nil
}

func Readv(r io.Reader, buffers [][]byte) (uint32, error) {
	read := uint32(0)
	for _, b := range buffers {
		n, err := r.Read(b)
		read += uint32(n)

		if err != nil {
			return read, err
		}
	}
	return read, nil
}

func Writev(w io.Writer, buffers [][]byte) (uint32, error) {
	written := uint32(0)
	for _, b := range buffers {
		n, err := w.Write(b)
		written += uint32(n)

		if err != nil {
			return written, err
		}
	}
	return written, nil
}

type readerFile struct {
	ErrFile
	r io.Reader
}

func NewReader(r io.Reader) File {
	return &readerFile{r: r}
}

func (f *readerFile) Readv(buffers [][]byte) (uint32, error) {
	return Readv(f.r, buffers)
}

type writerFile struct {
	ErrFile
	w io.Writer
}

func NewWriter(w io.Writer) File {
	return &writerFile{w: w}
}

func (f *writerFile) Writev(buffers [][]byte) (uint32, error) {
	return Writev(f.w, buffers)
}

type osFile struct {
	ErrFile
	f     *os.File
	flags int
}

func NewFile(f *os.File, flags int) File {
	return &osFile{f: f, flags: flags}
}

func (f *osFile) Close() error {
	return f.f.Close()
}

func (f *osFile) Pread(buffers [][]byte, offset int64) (uint32, error) {
	return Pread(f.f, buffers, offset)
}

func (f *osFile) Pwrite(buffers [][]byte, offset int64) (uint32, error) {
	return Pwrite(f.f, buffers, offset)
}

func (f *osFile) Readv(buffers [][]byte) (uint32, error) {
	return Readv(f.f, buffers)
}

func (f *osFile) Seek(offset int64, whence int) (int64, error) {
	return f.f.Seek(offset, whence)
}

func (f *osFile) SetSize(size uint64) error {
	return f.f.Truncate(int64(size))
}

func (f *osFile) Stat() (FDStat, error) {
	info, err := f.f.Stat()
	if err != nil {
		return FDStat{}, err
	}
	return FDStat{
		FileStat: fileStat(info),
		Flags:    f.flags,
	}, nil
}

func (f *osFile) Sync() error {
	return f.f.Sync()
}

func (f *osFile) Writev(buffers [][]byte) (uint32, error) {
	return Writev(f.f, buffers)
}

type osDirectory struct {
	osFile

	path string
}

func NewDirectory(path string, f *os.File) Directory {
	return &osDirectory{osFile: osFile{f: f}, path: path}
}

func (d *osDirectory) fullpath(name string) string {
	return filepath.Join(d.path, name)
}

func (d *osDirectory) FileStat(name string, followSymlinks bool) (FileStat, error) {
	path := d.fullpath(name)

	var info os.FileInfo
	var err error
	if followSymlinks {
		info, err = os.Lstat(path)
	} else {
		info, err = os.Stat(path)
	}
	if err != nil {
		return FileStat{}, err
	}

	return fileStat(info), nil
}

func (d *osDirectory) Mkdir(name string) error {
	return os.Mkdir(d.fullpath(name), 0700)
}

func (d *osDirectory) Open(name string, oflags, fflags int) (File, error) {
	osFlags := os.O_RDWR
	if oflags&O_Create != 0 {
		osFlags |= os.O_CREATE
	}
	if oflags&O_Excl != 0 {
		osFlags |= os.O_EXCL
	}
	if oflags&O_Trunc != 0 {
		osFlags |= os.O_TRUNC
	}
	if fflags&F_Append != 0 {
		osFlags |= os.O_APPEND
	}
	if fflags&(F_Dsync|F_Rsync|F_Sync) != 0 {
		osFlags |= os.O_SYNC
	}

	path := d.fullpath(name)
	if oflags&O_Directory != 0 {
		f, err := os.OpenFile(path, osFlags, os.ModeDir|0700)
		if err != nil {
			return nil, err
		}
		return NewDirectory(path, f), nil
	}

	f, err := os.OpenFile(path, osFlags, 0600)
	if err != nil {
		return nil, err
	}
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return NewDirectory(path, f), nil
	}
	return NewFile(f, fflags), nil
}

func (d *osDirectory) ReadDir(n int) ([]os.DirEntry, error) {
	return d.f.ReadDir(n)
}

func (d *osDirectory) ReadLink(name string) (string, error) {
	return os.Readlink(d.fullpath(name))
}

func (d *osDirectory) Rmdir(name string) error {
	return os.Remove(d.fullpath(name))
}

func (d *osDirectory) SetFileTimes(mame string, accessTime *time.Time, modTime *time.Time, followSymlinks bool) error {
	return os.ErrInvalid
}

func (d *osDirectory) UnlinkFile(name string) error {
	return os.Remove(d.fullpath(name))
}

func NewFS() FS {
	return newOSFS()
}

func (*osFS) OpenDirectory(path string) (Directory, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	f, err := os.OpenFile(abs, os.O_RDONLY, os.ModeDir)
	if err != nil {
		return nil, err
	}
	return NewDirectory(abs, f), nil
}

func (*osFS) Link(sourceDir Directory, sourceName string, targetDir Directory, targetName string) error {
	source, ok := sourceDir.(*osDirectory)
	if !ok {
		return os.ErrInvalid
	}
	target, ok := targetDir.(*osDirectory)
	if !ok {
		return os.ErrInvalid
	}
	return os.Link(source.fullpath(sourceName), target.fullpath(targetName))
}

func (*osFS) Rename(sourceDir Directory, sourceName string, targetDir Directory, targetName string) error {
	source, ok := sourceDir.(*osDirectory)
	if !ok {
		return os.ErrInvalid
	}
	target, ok := targetDir.(*osDirectory)
	if !ok {
		return os.ErrInvalid
	}
	return os.Rename(source.fullpath(sourceName), target.fullpath(targetName))
}

func (*osFS) Symlink(sourceDir Directory, sourceName string, targetDir Directory, targetName string) error {
	source, ok := sourceDir.(*osDirectory)
	if !ok {
		return os.ErrInvalid
	}
	target, ok := targetDir.(*osDirectory)
	if !ok {
		return os.ErrInvalid
	}
	return os.Symlink(source.fullpath(sourceName), target.fullpath(targetName))
}
