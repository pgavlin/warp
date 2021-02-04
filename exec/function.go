package exec

import (
	"github.com/pgavlin/warp/wasm"
)

// Function represents a function exported by a WASM module.
type Function interface {
	// GetSignature returns this function's signature.
	GetSignature() wasm.FunctionSig
	// Call calls the function with the given arguments. If the number and type of the arguments do not match the
	// number and type of the parameters in this function's signature, this method may panic.
	Call(thread *Thread, args ...interface{}) []interface{}
	// UncheckedCall calls the function with the given arguments. This method's behavior is undefined If the number of
	// arguments/returns does not match the number of parameters/results in this function's signature.
	UncheckedCall(thread *Thread, args, returns []uint64)
}
