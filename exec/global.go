package exec

import (
	"math"

	"github.com/pgavlin/warp/wasm"
)

type Global struct {
	typ       wasm.ValueType
	immutable bool
	value     uint64
}

func NewGlobalI32(immutable bool, value int32) Global {
	return Global{
		typ:       wasm.ValueTypeI32,
		immutable: immutable,
		value:     uint64(value),
	}
}

func NewGlobalI64(immutable bool, value int64) Global {
	return Global{
		typ:       wasm.ValueTypeI64,
		immutable: immutable,
		value:     uint64(value),
	}
}

func NewGlobalF32(immutable bool, value float32) Global {
	return Global{
		typ:       wasm.ValueTypeF32,
		immutable: immutable,
		value:     uint64(math.Float32bits(value)),
	}
}

func NewGlobalF64(immutable bool, value float64) Global {
	return Global{
		typ:       wasm.ValueTypeF64,
		immutable: immutable,
		value:     math.Float64bits(value),
	}
}

func (g *Global) Type() wasm.GlobalVar {
	return wasm.GlobalVar{Type: g.typ, Mutable: !g.immutable}
}

func (g *Global) Get() uint64 {
	return g.value
}

func (g *Global) GetValue() interface{} {
	switch g.typ {
	case wasm.ValueTypeI32:
		return g.GetI32()
	case wasm.ValueTypeI64:
		return g.GetI64()
	case wasm.ValueTypeF32:
		return g.GetF32()
	case wasm.ValueTypeF64:
		return g.GetF64()
	default:
		panic("unreachable")
	}
}

func (g *Global) GetI32() int32 {
	return int32(g.value)
}

func (g *Global) GetI64() int64 {
	return int64(g.value)
}

func (g *Global) GetF32() float32 {
	return math.Float32frombits(uint32(g.value))
}

func (g *Global) GetF64() float64 {
	return math.Float64frombits(g.value)
}

func (g *Global) Set(v uint64) {
	g.value = v
}

func (g *Global) SetValue(v interface{}) {
	switch g.typ {
	case wasm.ValueTypeI32:
		g.SetI32(v.(int32))
	case wasm.ValueTypeI64:
		g.SetI64(v.(int64))
	case wasm.ValueTypeF32:
		g.SetF32(v.(float32))
	case wasm.ValueTypeF64:
		g.SetF64(v.(float64))
	default:
		panic("unreachable")
	}
}

func (g *Global) SetI32(v int32) {
	g.value = uint64(v)
}

func (g *Global) SetI64(v int64) {
	g.value = uint64(v)
}

func (g *Global) SetF32(v float32) {
	g.value = uint64(math.Float32bits(v))
}

func (g *Global) SetF64(v float64) {
	g.value = math.Float64bits(v)
}
