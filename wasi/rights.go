package wasi

type Rights wasiRights

const (
	FileRights      = RightsFdAdvise | RightsFdAllocate | RightsFdDatasync | RightsFdFdstatSetFlags | RightsFdFilestatGet | RightsFdFilestatSetSize | RightsFdFilestatSetTimes | RightsFdRead | RightsFdSeek | RightsFdSync | RightsFdTell | RightsFdWrite | RightsPollFdReadwrite | RightsSockShutdown
	DirectoryRights = RightsFdReaddir | RightsPathCreateDirectory | RightsPathCreateFile | RightsPathFilestatGet | RightsPathFilestatSetSize | RightsPathFilestatSetTimes | RightsPathLinkSource | RightsPathLinkTarget | RightsPathOpen | RightsPathReadlink | RightsPathRemoveDirectory | RightsPathRenameSource | RightsPathRenameTarget | RightsPathSymlink | RightsPathUnlinkFile
	AllRights       = FileRights | DirectoryRights

	ReadOnlyRights = RightsFdRead | RightsFdSeek | RightsFdTell | RightsFdAdvise | RightsPathOpen | RightsFdReaddir | RightsPathReadlink | RightsPathFilestatGet | RightsFdFilestatGet | RightsPollFdReadwrite

	// The right to invoke `fd_datasync`.
	// If `path_open` is set, includes the right to invoke
	// `path_open` with `fdflags::dsync`.
	RightsFdDatasync = 1 << 0

	// The right to invoke `fd_read` and `sock_recv`.
	// If `rights::fd_seek` is set, includes the right to invoke `fd_pread`.
	RightsFdRead = 1 << 1

	// The right to invoke `fd_seek`. This flag implies `rights::fd_tell`.
	RightsFdSeek = 1 << 2

	// The right to invoke `fd_fdstat_set_flags`.
	RightsFdFdstatSetFlags = 1 << 3

	// The right to invoke `fd_sync`.
	// If `path_open` is set, includes the right to invoke
	// `path_open` with `fdflags::rsync` and `fdflags::dsync`.
	RightsFdSync = 1 << 4

	// The right to invoke `fd_seek` in such a way that the file offset
	// remains unaltered (i.e., `whence::cur` with offset zero), or to
	// invoke `fd_tell`.
	RightsFdTell = 1 << 5

	// The right to invoke `fd_write` and `sock_send`.
	// If `rights::fd_seek` is set, includes the right to invoke `fd_pwrite`.
	RightsFdWrite = 1 << 6

	// The right to invoke `fd_advise`.
	RightsFdAdvise = 1 << 7

	// The right to invoke `fd_allocate`.
	RightsFdAllocate = 1 << 8

	// The right to invoke `path_create_directory`.
	RightsPathCreateDirectory = 1 << 9

	// If `path_open` is set, the right to invoke `path_open` with `oflags::creat`.
	RightsPathCreateFile = 1 << 10

	// The right to invoke `path_link` with the file descriptor as the
	// source directory.
	RightsPathLinkSource = 1 << 11

	// The right to invoke `path_link` with the file descriptor as the
	// target directory.
	RightsPathLinkTarget = 1 << 12

	// The right to invoke `path_open`.
	RightsPathOpen = 1 << 13

	// The right to invoke `fd_readdir`.
	RightsFdReaddir = 1 << 14

	// The right to invoke `path_readlink`.
	RightsPathReadlink = 1 << 15

	// The right to invoke `path_rename` with the file descriptor as the source directory.
	RightsPathRenameSource = 1 << 16

	// The right to invoke `path_rename` with the file descriptor as the target directory.
	RightsPathRenameTarget = 1 << 17

	// The right to invoke `path_filestat_get`.
	RightsPathFilestatGet = 1 << 18

	// The right to change a file's size (there is no `path_filestat_set_size`).
	// If `path_open` is set, includes the right to invoke `path_open` with `oflags::trunc`.
	RightsPathFilestatSetSize = 1 << 19

	// The right to invoke `path_filestat_set_times`.
	RightsPathFilestatSetTimes = 1 << 20

	// The right to invoke `fd_filestat_get`.
	RightsFdFilestatGet = 1 << 21

	// The right to invoke `fd_filestat_set_size`.
	RightsFdFilestatSetSize = 1 << 22

	// The right to invoke `fd_filestat_set_times`.
	RightsFdFilestatSetTimes = 1 << 23

	// The right to invoke `path_symlink`.
	RightsPathSymlink = 1 << 24

	// The right to invoke `path_remove_directory`.
	RightsPathRemoveDirectory = 1 << 25

	// The right to invoke `path_unlink_file`.
	RightsPathUnlinkFile = 1 << 26

	// If `rights::fd_read` is set, includes the right to invoke `poll_oneoff` to subscribe to `eventtype::fd_read`.
	// If `rights::fd_write` is set, includes the right to invoke `poll_oneoff` to subscribe to `eventtype::fd_write`.
	RightsPollFdReadwrite = 1 << 27

	// The right to invoke `sock_shutdown`.
	RightsSockShutdown = 1 << 28
)
