package interpreter

import (
	"fmt"
	"math"
	"runtime"

	"github.com/pgavlin/warp/exec"
	"github.com/pgavlin/warp/wasm"
	"github.com/pgavlin/warp/wasm/code"
)

type functionKind int

const (
	functionKindBytecode = iota
	functionKindVirtual
	functionKindCountingICode
	functionKindICode
	functionKindFCode
)

// A function holds a decoded WASM function.
type function struct {
	module       *module            // The function's module.
	index        uint32             // The function's index.
	signature    wasm.FunctionSig   // The function signature.
	localEntries []wasm.LocalEntry  // The raw local entries for the function.
	numLocals    int                // The total number of locals for the function.
	metrics      code.Metrics       // Metrics for this function's body.
	kind         functionKind       // The kind of body the function has.
	invokeCount  int32              // The number of invocations of this function.
	bytecode     []byte             // The raw bytecode for the function. Discarded after decoding.
	icode        []code.Instruction // The decoded body of the function. Discarded after compiling to fcode.
	fcode        []finstruction     // The compiled body of the function.
	labels       []label            // The function's labels.
	switches     []switchTable      // The function's switch tables.
}

func (fn *function) blockType(instr *code.Instruction) (ins []wasm.ValueType, outs []wasm.ValueType) {
	return fn.module.blockType(instr)
}

func (fn *function) blockArity(instr *code.Instruction, isLoop bool) int {
	return fn.module.blockArity(instr, isLoop)
}

func (f *function) GetSignature() wasm.FunctionSig {
	return f.signature
}

func (f *function) Call(thread *exec.Thread, args ...interface{}) []interface{} {
	if len(args) != len(f.signature.ParamTypes) {
		panic(fmt.Errorf("expected %v args; got %v", len(f.signature.ParamTypes), len(args)))
	}

	rawArgs, rawReturns := make([]uint64, len(args)), make([]uint64, len(f.signature.ReturnTypes))
	for i, v := range args {
		paramType := f.signature.ParamTypes[i]

		switch v := v.(type) {
		case int32:
			if paramType != wasm.ValueTypeI32 {
				panic(fmt.Errorf("cannot assign int32 argument to a parameter of type %v", paramType))
			}
			rawArgs[i] = uint64(v)
		case int64:
			if paramType != wasm.ValueTypeI64 {
				panic(fmt.Errorf("cannot assign int64 argument to a parameter of type %v", paramType))
			}
			rawArgs[i] = uint64(v)
		case float32:
			if paramType != wasm.ValueTypeF32 {
				panic(fmt.Errorf("cannot assign float32 argument to a parameter of type %v", paramType))
			}
			rawArgs[i] = uint64(math.Float32bits(v))
		case float64:
			if paramType != wasm.ValueTypeF64 {
				panic(fmt.Errorf("cannot assign float64 argument to a parameter of type %v", paramType))
			}
			rawArgs[i] = math.Float64bits(v)
		default:
			panic(fmt.Errorf("cannot assign %T argument to a parameter of type %v", v, f.signature.ParamTypes[i]))
		}
	}

	f.UncheckedCall(thread, rawArgs, rawReturns)

	returns := make([]interface{}, len(f.signature.ReturnTypes))
	for i, t := range f.signature.ReturnTypes {
		switch t {
		case wasm.ValueTypeI32:
			returns[i] = int32(rawReturns[i])
		case wasm.ValueTypeI64:
			returns[i] = int64(rawReturns[i])
		case wasm.ValueTypeF32:
			returns[i] = math.Float32frombits(uint32(rawReturns[i]))
		case wasm.ValueTypeF64:
			returns[i] = math.Float64frombits(rawReturns[i])
		default:
			panic("unreachable")
		}
	}
	return returns
}

func (f *function) UncheckedCall(thread *exec.Thread, args, returns []uint64) {
	var m machine
	m.init(thread)

	maxStack := len(args)
	if len(returns) > maxStack {
		maxStack = len(returns)
	}

	caller := function{
		metrics: code.Metrics{MaxStackDepth: maxStack, MaxNesting: 1},
		kind:    functionKindVirtual,
	}

	frame := m.push(&caller)

	defer func() {
		if x := recover(); x != nil {
			err, _ := x.(runtime.Error)
			if trap, ok := exec.TranslateRuntimeError(err); ok {
				frame.trap(trap)
			}
			panic(x)
		}
	}()

	frame.pushn(args)
	frame.invokeDirect(f)
	frame.popn(returns)
}
