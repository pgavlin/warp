package exec

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"reflect"

	"github.com/pgavlin/warp/wasm"
	"github.com/pgavlin/warp/wasm/code"
	"github.com/pgavlin/warp/wasm/leb128"
)

type InvalidGlobalIndexError uint32

func (e InvalidGlobalIndexError) Error() string {
	return fmt.Sprintf("wasm: Invalid index to global index space: %#x", uint32(e))
}

type InvalidValueTypeInitExprError struct {
	Wanted reflect.Kind
	Got    reflect.Kind
}

func (e InvalidValueTypeInitExprError) Error() string {
	return fmt.Sprintf("wasm: Wanted initializer expression to return %v value, got %v", e.Wanted, e.Got)
}

// EvalConstantExpression executes the given (encoded) constant expression in the context of the given imports.
func EvalConstantExpression(imports []*Global, expr []byte) (interface{}, error) {
	var stack []uint64
	var topType wasm.ValueType

	if len(expr) == 0 {
		return nil, wasm.ErrEmptyInitExpr
	}

	for {
		if len(expr) == 0 {
			return nil, io.ErrUnexpectedEOF
		}
		opcode := expr[0]
		expr = expr[1:]

		switch opcode {
		case code.OpI32Const:
			v, sz, err := leb128.GetVarint32(expr)
			if err != nil {
				return nil, err
			}
			expr = expr[sz:]
			stack = append(stack, uint64(v))
			topType = wasm.ValueTypeI32
		case code.OpI64Const:
			v, sz, err := leb128.GetVarint64(expr)
			if err != nil {
				return nil, err
			}
			expr = expr[sz:]
			stack = append(stack, uint64(v))
			topType = wasm.ValueTypeI64
		case code.OpF32Const:
			if len(expr) < 4 {
				return nil, io.ErrUnexpectedEOF
			}
			v := binary.LittleEndian.Uint32(expr)
			expr = expr[4:]
			stack = append(stack, uint64(v))
			topType = wasm.ValueTypeF32
		case code.OpF64Const:
			if len(expr) < 8 {
				return nil, io.ErrUnexpectedEOF
			}
			v := binary.LittleEndian.Uint64(expr)
			expr = expr[8:]
			stack = append(stack, v)
			topType = wasm.ValueTypeF64
		case code.OpGlobalGet:
			index, sz, err := leb128.GetVarUint32(expr)
			if err != nil {
				return nil, err
			}
			expr = expr[sz:]

			if index > uint32(len(imports)) {
				return nil, InvalidGlobalIndexError(index)
			}
			global := imports[int(index)]
			stack = append(stack, global.value)
			topType = global.typ
		case code.OpEnd:
			if len(stack) == 0 {
				return nil, nil
			}

			v := stack[len(stack)-1]
			switch topType {
			case wasm.ValueTypeI32:
				return int32(v), nil
			case wasm.ValueTypeI64:
				return int64(v), nil
			case wasm.ValueTypeF32:
				return math.Float32frombits(uint32(v)), nil
			case wasm.ValueTypeF64:
				return math.Float64frombits(uint64(v)), nil
			default:
				panic("unreachable")
			}
		default:
			return nil, wasm.InvalidInitExprOpError(opcode)
		}
	}
}
