package dump

import (
	"encoding/csv"
	"io"

	"github.com/jszwec/csvutil"
	"github.com/pgavlin/warp/wasm"
	"github.com/pgavlin/warp/wasm/code"
)

// rows:
// - function
//     - import, export, in/out, nlocals, max stack, max nesting, # labels, # instructions, instruction breakdown

func dumpStats(w io.Writer, m *wasm.Module, n *names) error {
	type row struct {
		Function         string `csv:"function"`
		Funcidx          int    `csv:"funcidx"`
		In               int    `csv:"in"`
		Out              int    `csv:"out"`
		LocalCount       int    `csv:"local count"`
		MaxStack         int    `csv:"max stack"`
		MaxNesting       int    `csv:"max nesting"`
		LabelCount       int    `csv:"label count"`
		InstructionCount int    `csv:"instruction count"`
		Unreachable      int    `csv:"unreachable"`
		Nop              int    `csv:"nop"`
		Block            int    `csv:"block"`
		BlockUnit        int    `csv:"unit block"`
		BlockZero        int    `csv:"zero block"`
		Loop             int    `csv:"loop"`
		LoopUnit         int    `csv:"unit loop"`
		LoopZero         int    `csv:"zero loop"`
		If               int    `csv:"if"`
		IfUnit           int    `csv:"unit if"`
		IfZero           int    `csv:"zero if"`
		Else             int    `csv:"else"`
		Br               int    `csv:"br"`
		BrIf             int    `csv:"br_if"`
		BrTable          int    `csv:"br_table"`
		Return           int    `csv:"return"`
		Call             int    `csv:"call"`
		CallIndirect     int    `csv:"call_indirect"`
		Drop             int    `csv:"drop"`
		Select           int    `csv:"select"`
		LocalGet         int    `csv:"local.get"`
		LocalSet         int    `csv:"local.set"`
		LocalTee         int    `csv:"local.tee"`
		GlobalGet        int    `csv:"global.get"`
		GlobalSet        int    `csv:"global.set"`
		Load             int    `csv:"load"`
		LoadSmall        int    `csv:"load small offset"`
		Store            int    `csv:"store"`
		StoreSmall       int    `csv:"store small offset"`
		MemorySize       int    `csv:"memory.size"`
		MemoryGrow       int    `csv:"memory.grow"`
		I32Const         int    `csv:"i32.const"`
		I64Const         int    `csv:"i64.const"`
		F32Const         int    `csv:"f32.const"`
		F64Const         int    `csv:"f64.const"`
		I32SmallConst    int    `csv:"i32.small.const"`
		I64SmallConst    int    `csv:"i64.small.const"`
		F32SmallConst    int    `csv:"f32.small.const"`
		F64SmallConst    int    `csv:"f64.small.const"`
		I32Compare       int    `csv:"i32 compare"`
		I64Compare       int    `csv:"i64 compare"`
		F32Compare       int    `csv:"f32 compare"`
		F64Compare       int    `csv:"f64 compare"`
		I32Arith         int    `csv:"i32 arith"`
		I64Arith         int    `csv:"i64 arith"`
		F32Arith         int    `csv:"f32 arith"`
		F64Arith         int    `csv:"f64 arith"`
		I32Convert       int    `csv:"i32 convert"`
		I64Convert       int    `csv:"i64 convert"`
		F32Convert       int    `csv:"f32 convert"`
		F64Convert       int    `csv:"f64 convert"`
	}

	csvWriter := csv.NewWriter(w)
	defer csvWriter.Flush()

	encoder := csvutil.NewEncoder(csvWriter)

	s := code.NewStaticScope(m)
	for idx, body := range m.Code.Bodies {
		sig := m.Types.Entries[m.Function.Types[idx]]
		s.SetFunction(sig, body)

		decoded, err := code.Decode(body.Code, s, sig.ReturnTypes)
		if err != nil {
			return err
		}

		name, _ := n.FunctionName("", uint32(idx+len(s.ImportedFunctions)))
		r := row{
			Function:         name,
			Funcidx:          idx + len(s.ImportedFunctions),
			In:               len(sig.ParamTypes),
			Out:              len(sig.ReturnTypes),
			LocalCount:       len(s.Locals),
			MaxStack:         decoded.Metrics.MaxStackDepth,
			MaxNesting:       decoded.Metrics.MaxNesting,
			LabelCount:       decoded.Metrics.LabelCount,
			InstructionCount: len(decoded.Instructions),
		}
		for _, instr := range decoded.Instructions {
			switch instr.Opcode {
			case code.OpUnreachable:
				r.Unreachable++

			case code.OpNop:
				r.Nop++

			case code.OpBlock:
				ins, outs, _ := instr.BlockType(s)
				if len(ins) == 0 && len(outs) == 0 {
					r.BlockUnit++
				}
				if instr.StackHeight() == 0 {
					r.BlockZero++
				}
				r.Block++
			case code.OpLoop:
				ins, outs, _ := instr.BlockType(s)
				if len(ins) == 0 && len(outs) == 0 {
					r.LoopUnit++
				}
				if instr.StackHeight() == 0 {
					r.LoopZero++
				}
				r.Loop++
			case code.OpIf:
				ins, outs, _ := instr.BlockType(s)
				if len(ins) == 0 && len(outs) == 0 {
					r.IfUnit++
				}
				if instr.StackHeight() == 0 {
					r.IfZero++
				}
				r.If++
			case code.OpElse:
				r.Else++
			case code.OpEnd:
				// ignore

			case code.OpBr:
				r.Br++
			case code.OpBrIf:
				r.BrIf++
			case code.OpBrTable:
				r.BrTable++

			case code.OpReturn:
				r.Return++

			case code.OpCall:
				r.Call++
			case code.OpCallIndirect:
				r.CallIndirect++

			case code.OpDrop:
				r.Drop++
			case code.OpSelect:
				r.Select++

			case code.OpLocalGet:
				r.LocalGet++
			case code.OpLocalSet:
				r.LocalSet++
			case code.OpLocalTee:
				r.LocalTee++
			case code.OpGlobalGet:
				r.GlobalGet++
			case code.OpGlobalSet:
				r.GlobalSet++

			case code.OpI32Load:
				if instr.Offset() < 256 {
					r.LoadSmall++
				}
				r.Load++
			case code.OpI64Load:
				if instr.Offset() < 256 {
					r.LoadSmall++
				}
				r.Load++
			case code.OpF32Load:
				if instr.Offset() < 256 {
					r.LoadSmall++
				}
				r.Load++
			case code.OpF64Load:
				if instr.Offset() < 256 {
					r.LoadSmall++
				}
				r.Load++

			case code.OpI32Load8S:
				if instr.Offset() < 256 {
					r.LoadSmall++
				}
				r.Load++
			case code.OpI32Load8U:
				if instr.Offset() < 256 {
					r.LoadSmall++
				}
				r.Load++
			case code.OpI32Load16S:
				if instr.Offset() < 256 {
					r.LoadSmall++
				}
				r.Load++
			case code.OpI32Load16U:
				if instr.Offset() < 256 {
					r.LoadSmall++
				}
				r.Load++

			case code.OpI64Load8S:
				if instr.Offset() < 256 {
					r.LoadSmall++
				}
				r.Load++
			case code.OpI64Load8U:
				if instr.Offset() < 256 {
					r.LoadSmall++
				}
				r.Load++
			case code.OpI64Load16S:
				if instr.Offset() < 256 {
					r.LoadSmall++
				}
				r.Load++
			case code.OpI64Load16U:
				if instr.Offset() < 256 {
					r.LoadSmall++
				}
				r.Load++
			case code.OpI64Load32S:
				if instr.Offset() < 256 {
					r.LoadSmall++
				}
				r.Load++
			case code.OpI64Load32U:
				if instr.Offset() < 256 {
					r.LoadSmall++
				}
				r.Load++

			case code.OpI32Store:
				if instr.Offset() < 256 {
					r.StoreSmall++
				}
				r.Store++
			case code.OpI64Store:
				if instr.Offset() < 256 {
					r.StoreSmall++
				}
				r.Store++
			case code.OpF32Store:
				if instr.Offset() < 256 {
					r.StoreSmall++
				}
				r.Store++
			case code.OpF64Store:
				if instr.Offset() < 256 {
					r.StoreSmall++
				}
				r.Store++

			case code.OpI32Store8:
				if instr.Offset() < 256 {
					r.StoreSmall++
				}
				r.Store++
			case code.OpI32Store16:
				if instr.Offset() < 256 {
					r.StoreSmall++
				}
				r.Store++

			case code.OpI64Store8:
				if instr.Offset() < 256 {
					r.StoreSmall++
				}
				r.Store++
			case code.OpI64Store16:
				if instr.Offset() < 256 {
					r.StoreSmall++
				}
				r.Store++
			case code.OpI64Store32:
				if instr.Offset() < 256 {
					r.StoreSmall++
				}
				r.Store++

			case code.OpMemorySize:
				r.MemorySize++
			case code.OpMemoryGrow:
				r.MemoryGrow++

			case code.OpI32Const:
				if instr.I32() >= -128 && instr.I32() < 128 {
					r.I32SmallConst++
				}
				r.I32Const++
			case code.OpI64Const:
				if instr.I64() >= -128 && instr.I64() < 128 {
					r.I64SmallConst++
				}
				r.I64Const++
			case code.OpF32Const:
				if instr.F32() == 0 {
					r.F32SmallConst++
				}
				r.F32Const++
			case code.OpF64Const:
				if instr.F64() == 0 {
					r.F64SmallConst++
				}
				r.F64Const++

			case code.OpI32Eqz:
				r.I32Compare++
			case code.OpI32Eq:
				r.I32Compare++
			case code.OpI32Ne:
				r.I32Compare++
			case code.OpI32LtS:
				r.I32Compare++
			case code.OpI32LtU:
				r.I32Compare++
			case code.OpI32GtS:
				r.I32Compare++
			case code.OpI32GtU:
				r.I32Compare++
			case code.OpI32LeS:
				r.I32Compare++
			case code.OpI32LeU:
				r.I32Compare++
			case code.OpI32GeS:
				r.I32Compare++
			case code.OpI32GeU:
				r.I32Compare++

			case code.OpI64Eqz:
				r.I64Compare++
			case code.OpI64Eq:
				r.I64Compare++
			case code.OpI64Ne:
				r.I64Compare++
			case code.OpI64LtS:
				r.I64Compare++
			case code.OpI64LtU:
				r.I64Compare++
			case code.OpI64GtS:
				r.I64Compare++
			case code.OpI64GtU:
				r.I64Compare++
			case code.OpI64LeS:
				r.I64Compare++
			case code.OpI64LeU:
				r.I64Compare++
			case code.OpI64GeS:
				r.I64Compare++
			case code.OpI64GeU:
				r.I64Compare++

			case code.OpF32Eq:
				r.F32Compare++
			case code.OpF32Ne:
				r.F32Compare++
			case code.OpF32Lt:
				r.F32Compare++
			case code.OpF32Gt:
				r.F32Compare++
			case code.OpF32Le:
				r.F32Compare++
			case code.OpF32Ge:
				r.F32Compare++

			case code.OpF64Eq:
				r.F64Compare++
			case code.OpF64Ne:
				r.F64Compare++
			case code.OpF64Lt:
				r.F64Compare++
			case code.OpF64Gt:
				r.F64Compare++
			case code.OpF64Le:
				r.F64Compare++
			case code.OpF64Ge:
				r.F64Compare++

			case code.OpI32Clz:
				r.I32Arith++
			case code.OpI32Ctz:
				r.I32Arith++
			case code.OpI32Popcnt:
				r.I32Arith++
			case code.OpI32Add:
				r.I32Arith++
			case code.OpI32Sub:
				r.I32Arith++
			case code.OpI32Mul:
				r.I32Arith++
			case code.OpI32DivS:
				r.I32Arith++
			case code.OpI32DivU:
				r.I32Arith++
			case code.OpI32RemS:
				r.I32Arith++
			case code.OpI32RemU:
				r.I32Arith++
			case code.OpI32And:
				r.I32Arith++
			case code.OpI32Or:
				r.I32Arith++
			case code.OpI32Xor:
				r.I32Arith++
			case code.OpI32Shl:
				r.I32Arith++
			case code.OpI32ShrS:
				r.I32Arith++
			case code.OpI32ShrU:
				r.I32Arith++
			case code.OpI32Rotl:
				r.I32Arith++
			case code.OpI32Rotr:
				r.I32Arith++

			case code.OpI64Clz:
				r.I64Arith++
			case code.OpI64Ctz:
				r.I64Arith++
			case code.OpI64Popcnt:
				r.I64Arith++
			case code.OpI64Add:
				r.I64Arith++
			case code.OpI64Sub:
				r.I64Arith++
			case code.OpI64Mul:
				r.I64Arith++
			case code.OpI64DivS:
				r.I64Arith++
			case code.OpI64DivU:
				r.I64Arith++
			case code.OpI64RemS:
				r.I64Arith++
			case code.OpI64RemU:
				r.I64Arith++
			case code.OpI64And:
				r.I64Arith++
			case code.OpI64Or:
				r.I64Arith++
			case code.OpI64Xor:
				r.I64Arith++
			case code.OpI64Shl:
				r.I64Arith++
			case code.OpI64ShrS:
				r.I64Arith++
			case code.OpI64ShrU:
				r.I64Arith++
			case code.OpI64Rotl:
				r.I64Arith++
			case code.OpI64Rotr:
				r.I64Arith++

			case code.OpF32Abs:
				r.F32Arith++
			case code.OpF32Neg:
				r.F32Arith++
			case code.OpF32Ceil:
				r.F32Arith++
			case code.OpF32Floor:
				r.F32Arith++
			case code.OpF32Trunc:
				r.F32Arith++
			case code.OpF32Nearest:
				r.F32Arith++
			case code.OpF32Sqrt:
				r.F32Arith++
			case code.OpF32Add:
				r.F32Arith++
			case code.OpF32Sub:
				r.F32Arith++
			case code.OpF32Mul:
				r.F32Arith++
			case code.OpF32Div:
				r.F32Arith++
			case code.OpF32Min:
				r.F32Arith++
			case code.OpF32Max:
				r.F32Arith++
			case code.OpF32Copysign:
				r.F32Arith++

			case code.OpF64Abs:
				r.F64Arith++
			case code.OpF64Neg:
				r.F64Arith++
			case code.OpF64Ceil:
				r.F64Arith++
			case code.OpF64Floor:
				r.F64Arith++
			case code.OpF64Trunc:
				r.F64Arith++
			case code.OpF64Nearest:
				r.F64Arith++
			case code.OpF64Sqrt:
				r.F64Arith++
			case code.OpF64Add:
				r.F64Arith++
			case code.OpF64Sub:
				r.F64Arith++
			case code.OpF64Mul:
				r.F64Arith++
			case code.OpF64Div:
				r.F64Arith++
			case code.OpF64Min:
				r.F64Arith++
			case code.OpF64Max:
				r.F64Arith++
			case code.OpF64Copysign:
				r.F64Arith++

			case code.OpI32WrapI64:
				r.I32Convert++
			case code.OpI32TruncF32S:
				r.I32Convert++
			case code.OpI32TruncF32U:
				r.I32Convert++
			case code.OpI32TruncF64S:
				r.I32Convert++
			case code.OpI32TruncF64U:
				r.I32Convert++

			case code.OpI64ExtendI32S:
				r.I64Convert++
			case code.OpI64ExtendI32U:
				r.I64Convert++
			case code.OpI64TruncF32S:
				r.I64Convert++
			case code.OpI64TruncF32U:
				r.I64Convert++
			case code.OpI64TruncF64S:
				r.I64Convert++
			case code.OpI64TruncF64U:
				r.I64Convert++

			case code.OpF32ConvertI32S:
				r.F32Convert++
			case code.OpF32ConvertI32U:
				r.F32Convert++
			case code.OpF32ConvertI64S:
				r.F32Convert++
			case code.OpF32ConvertI64U:
				r.F32Convert++
			case code.OpF32DemoteF64:
				r.F32Convert++

			case code.OpF64ConvertI32S:
				r.F64Convert++
			case code.OpF64ConvertI32U:
				r.F64Convert++
			case code.OpF64ConvertI64S:
				r.F64Convert++
			case code.OpF64ConvertI64U:
				r.F64Convert++
			case code.OpF64PromoteF32:
				r.F64Convert++

			case code.OpI32ReinterpretF32:
				r.I32Convert++
			case code.OpI64ReinterpretF64:
				r.I64Convert++
			case code.OpF32ReinterpretI32:
				r.F32Convert++
			case code.OpF64ReinterpretI64:
				r.F64Convert++

			case code.OpI32Extend8S:
				r.I32Convert++
			case code.OpI32Extend16S:
				r.I32Convert++
			case code.OpI64Extend8S:
				r.I64Convert++
			case code.OpI64Extend16S:
				r.I64Convert++
			case code.OpI64Extend32S:
				r.I64Convert++

			case code.OpPrefix:
				switch instr.Immediate {
				case code.OpI32TruncSatF32S:
					r.I32Convert++
				case code.OpI32TruncSatF32U:
					r.I32Convert++
				case code.OpI32TruncSatF64S:
					r.I32Convert++
				case code.OpI32TruncSatF64U:
					r.I32Convert++
				case code.OpI64TruncSatF32S:
					r.I64Convert++
				case code.OpI64TruncSatF32U:
					r.I64Convert++
				case code.OpI64TruncSatF64S:
					r.I64Convert++
				case code.OpI64TruncSatF64U:
					r.I64Convert++
				}
			}
		}

		if err := encoder.Encode(&r); err != nil {
			return err
		}
	}
	return nil
}
