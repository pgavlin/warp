package golang

import (
	"bytes"
	"fmt"
	"io"
	"math"

	"github.com/pgavlin/warp/wasm"
	"github.com/pgavlin/warp/wasm/code"
)

type constExpression struct {
	instr code.Instruction
	uses  []*constExpression
	defs  []wasm.ValueType
}

type constExpressionCompiler struct {
	m      *moduleCompiler
	code   []code.Instruction
	stack  []*constExpression
	result *constExpression
}

func (c *constExpressionCompiler) compile() {
	// Compile the expression body into an expression tree.
	for _, i := range c.code {
		c.compileInstruction(i)
	}
	if len(c.stack) != 0 {
		c.result = c.stack[len(c.stack)-1]
	}
}

func (c *constExpressionCompiler) emit() (interface{}, string) {
	if c.result == nil {
		return int32(0), "0"
	}

	var buf bytes.Buffer
	v, _ := c.emitConstExpression(&buf, c.result)
	return v, buf.String()
}

func (c *constExpressionCompiler) compileInstruction(instr code.Instruction) {
	type VT = wasm.ValueType

	const (
		I32 = wasm.ValueTypeI32
		I64 = wasm.ValueTypeI64
		F32 = wasm.ValueTypeF32
		F64 = wasm.ValueTypeF64
	)

	stackUses, stackDefs := []VT(nil), []VT(nil)
	x := &constExpression{instr: instr}

	switch instr.Opcode {
	case code.OpGlobalGet:
		stackDefs = []VT{c.m.globalType(instr.Globalidx())}
	case code.OpI32Const:
		stackDefs = []VT{I32}
	case code.OpI64Const:
		stackDefs = []VT{I64}
	case code.OpF32Const:
		stackDefs = []VT{F32}
	case code.OpF64Const:
		stackDefs = []VT{F64}
	case code.OpEnd:
	default:
		panic(fmt.Errorf("unexpected instruction %v in constant expression", instr))
	}

	// Pop uses.
	if len(stackUses) > 0 {
		firstUse := len(c.stack) - 1
		for i := len(stackUses); ; {
			i -= len(c.stack[firstUse].defs)
			if i <= 0 {
				break
			}
			firstUse--
		}
		x.uses = append([]*constExpression(nil), c.stack[firstUse:]...)
		c.stack = c.stack[:firstUse]
	}

	// Push defs.
	if len(stackDefs) != 0 {
		c.stack = append(c.stack, x)
	}
	x.defs = stackDefs
}

func (c *constExpressionCompiler) emitConstExpression(w io.Writer, x *constExpression) (interface{}, error) {
	switch x.instr.Opcode {
	case code.OpGlobalGet:
		globalidx := x.instr.Globalidx()
		if globalidx < uint32(len(c.m.importedGlobals)) || c.m.exportedGlobals[globalidx] {
			if err := printf(w, "m.g%d", globalidx); err != nil {
				return nil, err
			}
			switch c.m.globalType(globalidx) {
			case wasm.ValueTypeI32:
				return nil, printf(w, ".GetI32()")
			case wasm.ValueTypeI64:
				return nil, printf(w, ".GetI64()")
			case wasm.ValueTypeF32:
				return nil, printf(w, ".GetF32()")
			case wasm.ValueTypeF64:
				return nil, printf(w, ".GetF64()")
			default:
				panic("unexpected global type")
			}
		}
		return nil, printf(w, "m.g%d", globalidx)
	case code.OpI32Const:
		v := int32(x.instr.Immediate)
		return v, printf(w, "int32(%d)", v)
	case code.OpI64Const:
		v := int64(x.instr.Immediate)
		return v, printf(w, "int64(%d)", v)
	case code.OpF32Const:
		v := math.Float32frombits(uint32(x.instr.Immediate))
		return v, printf(w, "%s", f32Const(v))
	case code.OpF64Const:
		v := math.Float64frombits(x.instr.Immediate)
		return v, printf(w, "%s", f64Const(v))
	case code.OpEnd:
		return nil, nil
	}

	panic("unexpected instruction")
}
