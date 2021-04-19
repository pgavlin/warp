package exec

import (
	"runtime"
	"strings"
)

// A Trap represents a WASM trap.
type Trap string

func (t Trap) Error() string {
	return string(t)
}

// TrapGeneric is produced for failures with no associated information.
var TrapGeneric = Trap("")

// TrapInvalidUndefinedElement indicates an attempt to access a table with an index that is out of bounds.
var TrapUndefinedElement = Trap("undefined element")

// TrapUninitializedElement indicates an attempt to use an uninitialized table element.
var TrapUninitializedElement = Trap("uninitialized element")

// TrapIndirectCallTypeMismatch indicates a mismatch between the exepected and actual signature of a function.
var TrapIndirectCallTypeMismatch = Trap("indirect call type mismatch")

// TrapOutOfBoundsMemoryAccess indicates an out-of-bounds memory access.
var TrapOutOfBoundsMemoryAccess = Trap("out of bounds memory access")

// TrapIntegerOverflow indicates an integer overflow.
var TrapIntegerOverflow = Trap("integer overflow")

// TrapInvalidConversionToInteger indicates an invalid converstion from a floating-point value to an
// integer.
var TrapInvalidConversionToInteger = Trap("invalid conversion to integer")

// TrapIntegerDivideByZero indicates an attempt to divide by zero.
var TrapIntegerDivideByZero = Trap("integer divide by zero")

// TrapCallStackExhausted indicates call stack exhaustion.
var TrapCallStackExhausted = Trap("call stack exhausted")

// TrapUnreachable indicates execution of unreachable code.
var TrapUnreachable = Trap("unreachable")

// TranslateRuntimeError is a utility function that translates between Go runtime errors and
// WASM traps.
func TranslateRuntimeError(err runtime.Error) (Trap, bool) {
	switch {
	case err == nil:
		return "", false
	case strings.HasPrefix(err.Error(), "runtime error: index out of range"):
		return TrapOutOfBoundsMemoryAccess, true
	case strings.HasPrefix(err.Error(), "runtime error: slice bounds out of range"):
		return TrapOutOfBoundsMemoryAccess, true
	case strings.HasPrefix(err.Error(), "runtime error: invalid memory address or nil pointer dereference"):
		return TrapOutOfBoundsMemoryAccess, true
	case strings.HasPrefix(err.Error(), "runtime error: integer divide by zero"):
		return TrapIntegerDivideByZero, true
	default:
		return "", false
	}
}

// TranslateRecover is a utility function that translates the result of a call to recover() into nothing,
// a trap, or a panic. This function should be called like so:
//
//     defer func() { exec.TranslateRecover(recover()) }()
//
func TranslateRecover(x interface{}) {
	if x != nil {
		err, _ := x.(runtime.Error)
		if trap, ok := TranslateRuntimeError(err); ok {
			panic(trap)
		}
		panic(x)
	}
}
