package exec

import (
	"reflect"

	"github.com/pgavlin/warp/wasm"
)

func goKind(valueType wasm.ValueType) reflect.Kind {
	switch valueType {
	case wasm.ValueTypeI32:
		return reflect.Int32
	case wasm.ValueTypeI64:
		return reflect.Int64
	case wasm.ValueTypeF32:
		return reflect.Float32
	case wasm.ValueTypeF64:
		return reflect.Float64
	default:
		return reflect.Invalid
	}
}

func wasmType(kind reflect.Kind) wasm.ValueType {
	switch kind {
	case reflect.Int32, reflect.Uint32:
		return wasm.ValueTypeI32
	case reflect.Int64, reflect.Uint64:
		return wasm.ValueTypeI64
	case reflect.Float32:
		return wasm.ValueTypeF32
	case reflect.Float64:
		return wasm.ValueTypeF64
	default:
		return 0
	}
}
