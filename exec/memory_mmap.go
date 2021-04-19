//go:build (!memtrace && !js && !plan9 && !windows && !armbe && !arm64be && !ppc && !ppc64 && !mips && !mips64 && !s390x)
// +build !memtrace,!js,!plan9,!windows,!armbe,!arm64be,!ppc,!ppc64,!mips,!mips64,!s390x

package exec

import (
	"fmt"
	"reflect"
	"runtime/debug"
	"syscall"
	"unsafe"
)

var ErrLimitExceeded = fmt.Errorf("memory limit exceeded")

// Memory is a WASM linear memory.
type Memory struct {
	min, max uint32
	start    uintptr
	size     uintptr
}

//go:linkname mmap runtime.mmap
func mmap(addr unsafe.Pointer, n uintptr, prot, flags, fd int32, off uint32) (unsafe.Pointer, int)

// NewMemory creates a new linear memory with the given limits.
func NewMemory(min, max uint32) Memory {
	debug.SetPanicOnFault(true)

	m := Memory{
		min: min,
		max: max,
	}
	if max > 0 {
		// Reserve twice the maximum allocation (8Gb). This allows us to safely use 64-bit addresses
		// and unmapped pages for bounds checks.
		pages, err := mmap(nil, 1<<33, syscall.PROT_NONE, syscall.MAP_ANON|syscall.MAP_PRIVATE, 0, 0)
		if err != 0 {
			panic(syscall.Errno(uintptr(err)))
		}

		m.start = uintptr(pages)
		if err := m.grow(min); err != nil {
			panic(err)
		}
	}
	return m
}

func (m *Memory) grow(pages uint32) error {
	end := m.start + m.size
	size := uintptr(pages) * 65536
	delta := size - m.size
	if delta == 0 {
		return nil
	}

	_, err := mmap(unsafe.Pointer(end), delta, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_ANON|syscall.MAP_PRIVATE|syscall.MAP_FIXED, 0, 0)
	if err != 0 {
		return syscall.Errno(uintptr(err))
	}
	m.size = uintptr(pages) * 65536
	return nil
}

// Limits returns the minimum and maximum size of the memory in pages.
func (m *Memory) Limits() (min, max uint32) {
	return m.min, m.max
}

// Size returns the current size of the memory in pages.
func (m *Memory) Size() uint32 {
	return uint32(m.size / 65536)
}

// Grow grows the memory by the given number of pages. It returns the old size of the memory in pages and an error if
// growing the memory by the requested amount would exceed the memory's maximum size.
func (m *Memory) Grow(pages uint32) (uint32, error) {
	currentSize := m.Size()
	newSize := currentSize + pages
	if newSize > m.max || newSize > 65536 {
		return currentSize, ErrLimitExceeded
	}
	return currentSize, m.grow(newSize)
}

// Bytes returns the memory's bytes.
func (m *Memory) Bytes() []byte {
	var s []byte
	header := (*reflect.SliceHeader)(unsafe.Pointer(&s))
	header.Data = m.start
	header.Len = int(m.size)
	header.Cap = int(m.size)
	return s
}

func (m *Memory) Start() uintptr {
	if m == nil {
		return 0
	}
	return m.start
}

// Byte returns the byte stored at the given effective address.
func (m *Memory) Byte(base, offset uint32) byte {
	p := (*byte)(unsafe.Pointer(m.start + uintptr(base) + uintptr(offset)))
	return *p
}

// Uint8 returns the byte stored at the given effective address.
func (m *Memory) Uint8(base, offset uint32) byte {
	p := (*byte)(unsafe.Pointer(m.start + uintptr(base) + uintptr(offset)))
	return *p
}

// PutByte writes the given byte to the given effective address.
func (m *Memory) PutByte(v byte, base, offset uint32) {
	p := (*byte)(unsafe.Pointer(m.start + uintptr(base) + uintptr(offset)))
	*p = v
}

// PutUint8 writes the given byte to the given effective address.
func (m *Memory) PutUint8(v byte, base, offset uint32) {
	p := (*byte)(unsafe.Pointer(m.start + uintptr(base) + uintptr(offset)))
	*p = v
}

// Uint16 returns the uint16 stored at the given effective address.
func (m *Memory) Uint16(base, offset uint32) uint16 {
	p := (*uint16)(unsafe.Pointer(m.start + uintptr(base) + uintptr(offset)))
	return *p
}

// PutUint16 writes the given uint16 to the given effective address.
func (m *Memory) PutUint16(v uint16, base, offset uint32) {
	p := (*uint16)(unsafe.Pointer(m.start + uintptr(base) + uintptr(offset)))
	*p = v
}

// Uint32 returns the uint32 stored at the given effective address.
func (m *Memory) Uint32(base, offset uint32) uint32 {
	p := (*uint32)(unsafe.Pointer(m.start + uintptr(base) + uintptr(offset)))
	return *p
}

// PutUint32 writes the given uint32 to the given effective address.
func (m *Memory) PutUint32(v uint32, base, offset uint32) {
	p := (*uint32)(unsafe.Pointer(m.start + uintptr(base) + uintptr(offset)))
	*p = v
}

// Uint64 returns the uint64 stored at the given effective address.
func (m *Memory) Uint64(base, offset uint32) uint64 {
	p := (*uint64)(unsafe.Pointer(m.start + uintptr(base) + uintptr(offset)))
	return *p
}

// PutUint64 writes the given uint64 to the given effective address.
func (m *Memory) PutUint64(v uint64, base, offset uint32) {
	p := (*uint64)(unsafe.Pointer(m.start + uintptr(base) + uintptr(offset)))
	*p = v
}

// Float32 returns the float32 stored at the given effective address.
func (m *Memory) Float32(base, offset uint32) float32 {
	p := (*float32)(unsafe.Pointer(m.start + uintptr(base) + uintptr(offset)))
	return *p
}

// PutFloat32 writes the given float32 to the given effective address.
func (m *Memory) PutFloat32(v float32, base, offset uint32) {
	p := (*float32)(unsafe.Pointer(m.start + uintptr(base) + uintptr(offset)))
	*p = v
}

// Float64 returns the float64 stored at the given effective address.
func (m *Memory) Float64(base, offset uint32) float64 {
	p := (*float64)(unsafe.Pointer(m.start + uintptr(base) + uintptr(offset)))
	return *p
}

// PutFloat64 writes the given float64 to the given effective address.
func (m *Memory) PutFloat64(v float64, base, offset uint32) {
	p := (*float64)(unsafe.Pointer(m.start + uintptr(base) + uintptr(offset)))
	*p = v
}

// ByteAt returns the byte stored at the given offset.
func (m *Memory) ByteAt(offset uint32) byte {
	p := (*byte)(unsafe.Pointer(m.start + uintptr(offset)))
	return *p
}

// PutByteAt writes the given byte to the given offset.
func (m *Memory) PutByteAt(v byte, offset uint32) {
	p := (*byte)(unsafe.Pointer(m.start + uintptr(offset)))
	*p = v
}

// Uint8At returns the byte stored at the given offset.
func (m *Memory) Uint8At(offset uint32) byte {
	p := (*byte)(unsafe.Pointer(m.start + uintptr(offset)))
	return *p
}

// PutUint8At writes the given byte to the given offset.
func (m *Memory) PutUint8At(v byte, offset uint32) {
	p := (*byte)(unsafe.Pointer(m.start + uintptr(offset)))
	*p = v
}

// Uint16At returns the uint16 stored at the given offset.
func (m *Memory) Uint16At(offset uint32) uint16 {
	p := (*uint16)(unsafe.Pointer(m.start + uintptr(offset)))
	return *p
}

// PutUint16At writes the given uint16 to the given offset.
func (m *Memory) PutUint16At(v uint16, offset uint32) {
	p := (*uint16)(unsafe.Pointer(m.start + uintptr(offset)))
	*p = v
}

// Uint32At returns the uint32 stored at the given offset.
func (m *Memory) Uint32At(offset uint32) uint32 {
	p := (*uint32)(unsafe.Pointer(m.start + uintptr(offset)))
	return *p
}

// PutUint32At writes the given uint32 to the given offset.
func (m *Memory) PutUint32At(v uint32, offset uint32) {
	p := (*uint32)(unsafe.Pointer(m.start + uintptr(offset)))
	*p = v
}

// Uint64 returns the uint64 stored at the given offset.
func (m *Memory) Uint64At(offset uint32) uint64 {
	p := (*uint64)(unsafe.Pointer(m.start + uintptr(offset)))
	return *p
}

// PutUint64 writes the given uint64 to the given offset.
func (m *Memory) PutUint64At(v uint64, offset uint32) {
	p := (*uint64)(unsafe.Pointer(m.start + uintptr(offset)))
	*p = v
}

// Float32At returns the float32 stored at the given offset.
func (m *Memory) Float32At(offset uint32) float32 {
	p := (*float32)(unsafe.Pointer(m.start + uintptr(offset)))
	return *p
}

// PutFloat32At writes the given float32 to the given offset.
func (m *Memory) PutFloat32At(v float32, offset uint32) {
	p := (*float32)(unsafe.Pointer(m.start + uintptr(offset)))
	*p = v
}

// Float64At returns the float64 stored at the given offset.
func (m *Memory) Float64At(offset uint32) float64 {
	p := (*float64)(unsafe.Pointer(m.start + uintptr(offset)))
	return *p
}

// PutFloat64At writes the given float64 to the given offset.
func (m *Memory) PutFloat64At(v float64, offset uint32) {
	p := (*float64)(unsafe.Pointer(m.start + uintptr(offset)))
	*p = v
}
