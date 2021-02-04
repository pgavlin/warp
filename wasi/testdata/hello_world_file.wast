(module
	;; Import the required WASI functions
	(import "wasi_snapshot_preview1" "fd_close" (func $fd_close (param i32) (result i32)))
	(import "wasi_snapshot_preview1" "fd_write" (func $fd_write (param i32 i32 i32 i32) (result i32)))
	(import "wasi_snapshot_preview1" "path_open" (func $path_open (param i32 i32 i32 i32 i32 i64 i64 i32 i32) (result i32)))
	(import "wasi_snapshot_preview1" "proc_exit" (func $proc_exit (param i32)))

	(memory 1)
	(export "memory" (memory 0))

	;; Write 'hello.txthello world\n' to memory at an offset of 8 bytes
	;; Note the trailing newline which is required for the text to appear
	(data (i32.const 8) "hello.txthello world\n")

	(func $exit (param $code i32)
		(call $proc_exit (local.get $code))
		unreachable
	)

	(func $main (export "_start")
		(local $fd i32)

		;; Open a new file in the first preopen. Exit the process if the open fails.
		(if (call $path_open
			(i32.const 3)    ;; fd - 3 for the first preopen
			(i32.const 0)    ;; dirflags - 0
			(i32.const 8)    ;; path - pointer to the start of 'hello.txt'
			(i32.const 9)    ;; path - length of 'hello.txt'
			(i32.const 0x09) ;; oflags - oflags::creat | oflags::trunc
			(i64.const 0x40) ;; fs_rights_base - rights::fd_write
			(i64.const 0)    ;; fs_rights_inherit - 0
			(i32.const 0)    ;; fdflags - 0
			(i32.const 0))   ;; ret_fd - pointer to linear memory
		(then (call $exit (i32.const -1))))

		;; Save the file descriptor.
		(local.set $fd (i32.load (i32.const 0)))

		;; Create a new io vector within linear memory.
		(i32.store (i32.const 0) (i32.const 17))  ;; iov.iov_base - This is a pointer to the start of the 'hello world\n' string
		(i32.store (i32.const 4) (i32.const 12))  ;; iov.iov_len - The length of the 'hello world\n' string

		(if (call $fd_write
			(local.get $fd) ;; file_descriptor
			(i32.const 0)   ;; *iovs - The pointer to the iov array, which is stored at memory location 0
			(i32.const 1)   ;; iovs_len - We're printing 1 string stored in an iov - so one.
			(i32.const 21)) ;; nwritten - A place in memory to store the number of bytes written
		(then (call $exit (i32.const -2))))

		;; Close the file.
		(if (call $fd_close (local.get $fd)) (then (call $exit (i32.const -3))))

		;; Exit.
		(call $exit (i32.const 0))
	)
)
