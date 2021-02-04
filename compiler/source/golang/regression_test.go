package golang

import (
	"testing"
)

func TestShiftConstant(t *testing.T) {
	testModule(t, ShiftConstant, "main")
}

var ShiftConstant = mustParseModule(`(module
	(memory 1)

	(func $repro (export "repro") (param i32 i32)
		(i64.store
			(local.get 1)
			(i64.xor
				(i64.shl (i64.const -1) (i64.extend_i32_u (local.get 0)))
				(i64.const -1)
			)
		)
	)

	(func (export "main")
		(call $repro (i32.const 0) (i32.const 0))
	)
)`)
