//go:build memtrace
// +build memtrace

package exec

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
)

var ErrLimitExceeded = fmt.Errorf("memory limit exceeded")

// Memory is a WASM linear memory.
type Memory struct {
	min, max uint32
	bytes    []byte
}

// NewMemory creates a new linear memory with the given limits.
func NewMemory(min, max uint32) Memory {
	return Memory{
		min:   min,
		max:   max,
		bytes: make([]byte, min*65536),
	}
}

// Limits returns the minimum and maximum size of the memory in pages.
func (m *Memory) Limits() (min, max uint32) {
	return m.min, m.max
}

// Size returns the current size of the memory in pages.
func (m *Memory) Size() uint32 {
	return uint32(len(m.bytes) / 65536)
}

// Grow grows the memory by the given number of pages. It returns the old size of the memory in pages and an error if
// growing the memory by the requested amount would exceed the memory's maximum size.
func (m *Memory) Grow(pages uint32) (uint32, error) {
	currentSize := m.Size()
	newSize := currentSize + pages
	if newSize > m.max || newSize > 65536 {
		return currentSize, ErrLimitExceeded
	}
	newBytes := make([]byte, int(newSize*65536))
	copy(newBytes, m.bytes)
	m.bytes = newBytes
	return currentSize, nil
}

// Bytes returns the memory's bytes.
func (m *Memory) Bytes() []byte {
	return m.bytes
}

func (m *Memory) Start() uintptr {
	panic("Start() is not supported when tracing memory accesses")
}

func effectiveAddress(base, offset uint32) int {
	sum64 := uint64(base) + uint64(offset)
	return int(uint32(sum64) | uint32(sum64>>1)&0x80000000)
}

// Byte returns the byte stored at the given offset.
func (m *Memory) Byte(base, offset uint32) byte {
	addr := effectiveAddress(base, offset)
	v := m.bytes[addr]
	fmt.Fprintf(os.Stderr, "0x%08x -> 0x%02x\n", addr, v)
	return v
}

// Uint8 returns the byte stored at the given offset.
func (m *Memory) Uint8(base, offset uint32) byte {
	addr := effectiveAddress(base, offset)
	v := m.bytes[addr]
	fmt.Fprintf(os.Stderr, "0x%08x -> 0x%02x\n", addr, v)
	return v
}

// PutByte writes the given byte to the given offset.
func (m *Memory) PutByte(v byte, base, offset uint32) {
	addr := effectiveAddress(base, offset)
	fmt.Fprintf(os.Stderr, "0x%08x <- 0x%02x\n", addr, v)
	m.bytes[addr] = v
}

// PutUint8 writes the given byte to the given offset.
func (m *Memory) PutUint8(v byte, base, offset uint32) {
	addr := effectiveAddress(base, offset)
	fmt.Fprintf(os.Stderr, "0x%08x <- 0x%02x\n", addr, v)
	m.bytes[addr] = v
}

// Uint16 returns the uint16 stored at the given offset.
func (m *Memory) Uint16(base, offset uint32) uint16 {
	addr := effectiveAddress(base, offset)
	v := binary.LittleEndian.Uint16(m.bytes[addr:])
	fmt.Fprintf(os.Stderr, "0x%08x -> 0x%04x\n", addr, v)
	return v
}

// PutUint16 writes the given uint16 to the given offset.
func (m *Memory) PutUint16(v uint16, base, offset uint32) {
	addr := effectiveAddress(base, offset)
	fmt.Fprintf(os.Stderr, "0x%08x <- 0x%04x\n", addr, v)
	binary.LittleEndian.PutUint16(m.bytes[addr:], v)
}

// Uint32 returns the uint32 stored at the given offset.
func (m *Memory) Uint32(base, offset uint32) uint32 {
	addr := effectiveAddress(base, offset)
	v := binary.LittleEndian.Uint32(m.bytes[addr:])
	fmt.Fprintf(os.Stderr, "0x%08x -> 0x%08x\n", addr, v)
	return v
}

// PutUint32 writes the given uint32 to the given offset.
func (m *Memory) PutUint32(v uint32, base, offset uint32) {
	addr := effectiveAddress(base, offset)
	fmt.Fprintf(os.Stderr, "0x%08x <- 0x%08x\n", addr, v)
	binary.LittleEndian.PutUint32(m.bytes[addr:], v)
}

// Uint64 returns the uint64 stored at the given offset.
func (m *Memory) Uint64(base, offset uint32) uint64 {
	addr := effectiveAddress(base, offset)
	v := binary.LittleEndian.Uint64(m.bytes[addr:])
	fmt.Fprintf(os.Stderr, "0x%08x -> 0x%016x\n", addr, v)
	return v
}

// PutUint64 writes the given uint64 to the given offset.
func (m *Memory) PutUint64(v uint64, base, offset uint32) {
	addr := effectiveAddress(base, offset)
	fmt.Fprintf(os.Stderr, "0x%08x <- 0x%016x\n", addr, v)
	binary.LittleEndian.PutUint64(m.bytes[addr:], v)
}

// Float32 returns the float32 stored at the given offset.
func (m *Memory) Float32(base, offset uint32) float32 {
	return math.Float32frombits(m.Uint32(base, offset))
}

// PutFloat32 writes the given float32 to the given offset.
func (m *Memory) PutFloat32(v float32, base, offset uint32) {
	m.PutUint32(math.Float32bits(v), base, offset)
}

// Float64 returns the float64 stored at the given offset.
func (m *Memory) Float64(base, offset uint32) float64 {
	return math.Float64frombits(m.Uint64(base, offset))
}

// PutFloat64 writes the given float64 to the given offset.
func (m *Memory) PutFloat64(v float64, base, offset uint32) {
	m.PutUint64(math.Float64bits(v), base, offset)
}

// ByteAt returns the byte stored at the given offset.
func (m *Memory) ByteAt(offset uint32) byte {
	return m.Byte(offset, 0)
}

// Uint8At returns the byte stored at the given offset.
func (m *Memory) Uint8At(offset uint32) byte {
	return m.Uint8(offset, 0)
}

// PutByte writes the given byte to the given offset.
func (m *Memory) PutByteAt(v byte, offset uint32) {
	m.PutByte(v, offset, 0)
}

// PutUint8 writes the given byte to the given offset.
func (m *Memory) PutUint8At(v byte, offset uint32) {
	m.PutUint8(v, offset, 0)
}

// Uint16 returns the uint16 stored at the given offset.
func (m *Memory) Uint16At(offset uint32) uint16 {
	return m.Uint16(offset, 0)
}

// PutUint16 writes the given uint16 to the given offset.
func (m *Memory) PutUint16At(v uint16, offset uint32) {
	m.PutUint16(v, offset, 0)
}

// Uint32 returns the uint32 stored at the given offset.
func (m *Memory) Uint32At(offset uint32) uint32 {
	return m.Uint32(offset, 0)
}

// PutUint32 writes the given uint32 to the given offset.
func (m *Memory) PutUint32At(v uint32, offset uint32) {
	m.PutUint32(v, offset, 0)
}

// Uint64 returns the uint64 stored at the given offset.
func (m *Memory) Uint64At(offset uint32) uint64 {
	return m.Uint64(offset, 0)
}

// PutUint64 writes the given uint64 to the given offset.
func (m *Memory) PutUint64At(v uint64, offset uint32) {
	m.PutUint64(v, offset, 0)
}

// Float32 returns the float32 stored at the given offset.
func (m *Memory) Float32At(offset uint32) float32 {
	return m.Float32(offset, 0)
}

// PutFloat32 writes the given float32 to the given offset.
func (m *Memory) PutFloat32At(v float32, offset uint32) {
	m.PutFloat32(v, offset, 0)
}

// Float64 returns the float64 stored at the given offset.
func (m *Memory) Float64At(offset uint32) float64 {
	return m.Float64(offset, 0)
}

// PutFloat64 writes the given float64 to the given offset.
func (m *Memory) PutFloat64At(v float64, offset uint32) {
	m.PutFloat64(v, offset, 0)
}
