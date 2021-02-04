package interpreter

import (
	"fmt"
	"io"
	"os"
)

type dumper struct {
	w  io.Writer
	fn *function

	ipToLabelidx map[int]int
}

func (m *machine) dumpFcode(fn *function) {
	d := dumper{w: os.Stderr, fn: fn}
	d.dumpFcode(fn.fcode)
}

func (d *dumper) dumpTuple(end, n int) {
	if n > 0 {
		fmt.Fprintf(d.w, " v[%v:%v]", end-n, end)
	}
}

func (d *dumper) dumpOp(ip int, fi *finstruction, op string, ndef int) {
	if idx, ok := d.ipToLabelidx[ip]; ok {
		l := d.fn.labels[idx]
		fmt.Fprintf(d.w, "l%v", idx)
		d.dumpTuple(d.fn.numLocals+l.stackHeight+l.arity, l.arity)
		fmt.Fprintf(d.w, ":\n")
	}

	fmt.Fprintf(d.w, "\t")
	if ndef != 0 {
		switch fi.opcode {
		case fopGlobalSet, fopGlobalSetI:
			fmt.Fprintf(d.w, "g%d", fi.Globalidx())
		case fopI32Store, fopI64Store, fopF32Store, fopF64Store, fopI32Store8, fopI32Store16, fopI64Store8, fopI64Store16, fopI64Store32:
			fmt.Fprintf(d.w, "*(v%d + %d)", fi.dest, fi.Offset())
		case fopI32StoreI, fopI64StoreI, fopF32StoreI, fopF64StoreI, fopI32Store8I, fopI32Store16I, fopI64Store8I, fopI64Store16I, fopI64Store32I:
			fmt.Fprintf(d.w, "*0x%08x", fi.Offset())
		default:
			for i := 0; i < ndef; i++ {
				if i > 0 {
					fmt.Fprintf(d.w, ", ")
				}
				fmt.Fprintf(d.w, "v%d", fi.dest+uint32(i))
			}
		}

		fmt.Fprintf(d.w, " = ")
	}

	fmt.Fprintf(d.w, "%s", op)
}

func (d *dumper) dumpBranch(ip int, fi *finstruction, op string, isBr bool) {
	d.dumpOp(ip, fi, op, 0)

	if !isBr {
		fmt.Fprintf(d.w, " v%v,", fi.src1)
	}
	fmt.Fprintf(d.w, " l%v", fi.Labelidx())
	d.dumpTuple(d.fn.numLocals+fi.StackHeight(), d.fn.labels[fi.Labelidx()].arity)
}

func (d *dumper) dumpCall(ip int, fi *finstruction, op string, isCalli bool) {
	nparams, nresults := 0, 0
	if !isCalli {
		funcidx := fi.Funcidx()
		if funcidx < uint32(len(d.fn.module.importedFunctions)) {
			sig := d.fn.module.importedFunctions[funcidx].GetSignature()
			nparams, nresults = len(sig.ParamTypes), len(sig.ReturnTypes)
		} else {
			callee := &d.fn.module.functions[funcidx-uint32(len(d.fn.module.importedFunctions))]
			nparams, nresults = len(callee.signature.ParamTypes), len(callee.signature.ReturnTypes)
		}
	} else {
		sig := d.fn.module.types[fi.Typeidx()]
		nparams, nresults = len(sig.ParamTypes), len(sig.ReturnTypes)
	}

	d.dumpOp(ip, fi, op, nresults)

	if !isCalli {
		fmt.Fprintf(d.w, " f%v", fi.Funcidx())
	} else {
		fmt.Fprintf(d.w, " v%v, t%v", fi.src1, fi.Typeidx())
	}

	d.dumpTuple(d.fn.numLocals+fi.StackHeight(), nparams)
}

func (d *dumper) dumpLocalGet(ip int, fi *finstruction) {
	d.dumpOp(ip, fi, "local.get", 1)
	fmt.Fprintf(d.w, " v%v", fi.src1)
}

func (d *dumper) dumpLoad(ip int, fi *finstruction, op string) {
	d.dumpOp(ip, fi, op, 1)

	if fi.flags&ifSrc1Frame == 0 {
		fmt.Fprintf(d.w, " *0x%08x", fi.Offset())
	} else {
		fmt.Fprintf(d.w, " *(v%v + %v)", fi.src1, fi.Offset())
	}
}

func (d *dumper) dumpStore(ip int, fi *finstruction, op string) {
	d.dumpOp(ip, fi, op, 1)
	fmt.Fprintf(d.w, " v%v", fi.src1)
}

func (d *dumper) dumpBinOp(ip int, fi *finstruction, op string) {
	d.dumpOp(ip, fi, op, 1)

	fmt.Fprintf(d.w, " v%v, ", fi.src1)
	if fi.flags&ifSrc2Frame != 0 {
		fmt.Fprintf(d.w, "v%v", uint32(fi.src2))
	} else {
		fmt.Fprintf(d.w, "0x%016x", fi.src2)
	}
}

func (d *dumper) dumpUnOp(ip int, fi *finstruction, op string) {
	d.dumpOp(ip, fi, op, 1)

	if fi.flags&ifSrc1Frame != 0 {
		fmt.Fprintf(d.w, " v%v", fi.src1)
	} else {
		fmt.Fprintf(d.w, " 0x%016x", fi.src2)
	}
}

func (d *dumper) dumpInstruction(ip int, fi *finstruction) {
	switch fi.opcode & 0x1ff {
	case fopUnreachable:
		d.dumpOp(ip, fi, "unreachable", 0)
	case fopIf:
		d.dumpOp(ip, fi, "if", 0)
		fmt.Fprintf(d.w, " v%v", fi.src1)
	case fopElse:
		d.dumpOp(ip, fi, "else", 0)
	case fopBr:
		d.dumpBranch(ip, fi, "br", true)
	case fopBrIf:
		d.dumpBranch(ip, fi, "br_if", false)
	case fopBrTable:
		d.dumpBranch(ip, fi, "br_table", false)
	case fopBrL:
		d.dumpBranch(ip, fi, "br.l", true)
	case fopBrIfL:
		d.dumpBranch(ip, fi, "br_if.l", false)
	case fopBrTableL:
		d.dumpBranch(ip, fi, "br_table.l", false)
	case fopReturn:
		d.dumpOp(ip, fi, "return", 0)
		d.dumpTuple(d.fn.numLocals+fi.StackHeight(), len(d.fn.signature.ReturnTypes))
	case fopCall:
		d.dumpCall(ip, fi, "call", false)
	case fopCallIndirect:
		d.dumpCall(ip, fi, "call", true)
	case fopSelect:
		d.dumpOp(ip, fi, "select", 1)
		fmt.Fprintf(d.w, " v%v, v%v, v%v", fi.src1, fi.Src2(), fi.Src3())
	case fopLocalGet:
		d.dumpLocalGet(ip, fi)
	case fopLocalSet:
		d.dumpOp(ip, fi, "", 1)

		if fi.flags&ifSrc1Frame != 0 {
			fmt.Fprintf(d.w, "copy v%v", fi.src1)
		} else {
			fmt.Fprintf(d.w, "const 0x%016x", fi.src2)
		}
	case fopGlobalGet:
		d.dumpOp(ip, fi, "global.get", 1)
		fmt.Fprintf(d.w, " g%v", fi.src1)
	case fopGlobalSet:
		d.dumpUnOp(ip, fi, "global.set")
	case fopI32Load:
		d.dumpLoad(ip, fi, "i32.load")
	case fopI64Load:
		d.dumpLoad(ip, fi, "i64.load")
	case fopF32Load:
		d.dumpLoad(ip, fi, "f32.load")
	case fopF64Load:
		d.dumpLoad(ip, fi, "f64.load")
	case fopI32Load8S:
		d.dumpLoad(ip, fi, "i32.load8_s")
	case fopI32Load8U:
		d.dumpLoad(ip, fi, "i32.load8_u")
	case fopI32Load16S:
		d.dumpLoad(ip, fi, "i32.load16_s")
	case fopI32Load16U:
		d.dumpLoad(ip, fi, "i32.load16_u")
	case fopI64Load8S:
		d.dumpLoad(ip, fi, "i64.load8_s")
	case fopI64Load8U:
		d.dumpLoad(ip, fi, "i64.load8_u")
	case fopI64Load16S:
		d.dumpLoad(ip, fi, "i64.load16_s")
	case fopI64Load16U:
		d.dumpLoad(ip, fi, "i64.load16_u")
	case fopI64Load32S:
		d.dumpLoad(ip, fi, "i64.load32_s")
	case fopI64Load32U:
		d.dumpLoad(ip, fi, "i64.load32_u")
	case fopI32Store:
		d.dumpStore(ip, fi, "i32.store")
	case fopI64Store:
		d.dumpStore(ip, fi, "i64.store")
	case fopF32Store:
		d.dumpStore(ip, fi, "f32.store")
	case fopF64Store:
		d.dumpStore(ip, fi, "f64.store")
	case fopI32Store8:
		d.dumpStore(ip, fi, "i32.store8")
	case fopI32Store16:
		d.dumpStore(ip, fi, "i32.store16")
	case fopI64Store8:
		d.dumpStore(ip, fi, "i64.store8")
	case fopI64Store16:
		d.dumpStore(ip, fi, "i64.store16")
	case fopI64Store32:
		d.dumpStore(ip, fi, "i64.store32")
	case fopMemorySize:
		d.dumpOp(ip, fi, "memory.size", 1)
	case fopMemoryGrow:
		d.dumpUnOp(ip, fi, "memory.grow")
	case fopI64Const:
		d.dumpOp(ip, fi, "const", 1)
		fmt.Fprintf(d.w, " 0x%016x", fi.src2)
	case fopI32Eqz:
		d.dumpUnOp(ip, fi, "i32.eqz")
	case fopI32Eq:
		d.dumpBinOp(ip, fi, "i32.eq")
	case fopI32Ne:
		d.dumpBinOp(ip, fi, "i32.ne")
	case fopI32LtS:
		d.dumpBinOp(ip, fi, "i32.lt_s")
	case fopI32LtU:
		d.dumpBinOp(ip, fi, "i32.lt_u")
	case fopI32GtS:
		d.dumpBinOp(ip, fi, "i32.gt_s")
	case fopI32GtU:
		d.dumpBinOp(ip, fi, "i32.gt_u")
	case fopI32LeS:
		d.dumpBinOp(ip, fi, "i32.le_s")
	case fopI32LeU:
		d.dumpBinOp(ip, fi, "i32.le_u")
	case fopI32GeS:
		d.dumpBinOp(ip, fi, "i32.ge_s")
	case fopI32GeU:
		d.dumpBinOp(ip, fi, "i32.ge_u")
	case fopI64Eqz:
		d.dumpUnOp(ip, fi, "i64.eqz")
	case fopI64Eq:
		d.dumpBinOp(ip, fi, "i64.eq")
	case fopI64Ne:
		d.dumpBinOp(ip, fi, "i64.ne")
	case fopI64LtS:
		d.dumpBinOp(ip, fi, "i64.lt_s")
	case fopI64LtU:
		d.dumpBinOp(ip, fi, "i64.lt_u")
	case fopI64GtS:
		d.dumpBinOp(ip, fi, "i64.gt_s")
	case fopI64GtU:
		d.dumpBinOp(ip, fi, "i64.gt_u")
	case fopI64LeS:
		d.dumpBinOp(ip, fi, "i64.le_s")
	case fopI64LeU:
		d.dumpBinOp(ip, fi, "i64.le_u")
	case fopI64GeS:
		d.dumpBinOp(ip, fi, "i64.ge_s")
	case fopI64GeU:
		d.dumpBinOp(ip, fi, "i64.ge_u")
	case fopF32Eq:
		d.dumpBinOp(ip, fi, "f32.eq")
	case fopF32Ne:
		d.dumpBinOp(ip, fi, "f32.ne")
	case fopF32Lt:
		d.dumpBinOp(ip, fi, "f32.lt")
	case fopF32Gt:
		d.dumpBinOp(ip, fi, "f32.gt")
	case fopF32Le:
		d.dumpBinOp(ip, fi, "f32.le")
	case fopF32Ge:
		d.dumpBinOp(ip, fi, "f32.ge")
	case fopF64Eq:
		d.dumpBinOp(ip, fi, "f64.eq")
	case fopF64Ne:
		d.dumpBinOp(ip, fi, "f64.ne")
	case fopF64Lt:
		d.dumpBinOp(ip, fi, "f64.lt")
	case fopF64Gt:
		d.dumpBinOp(ip, fi, "f64.gt")
	case fopF64Le:
		d.dumpBinOp(ip, fi, "f64.le")
	case fopF64Ge:
		d.dumpBinOp(ip, fi, "f64.ge")
	case fopI32Clz:
		d.dumpUnOp(ip, fi, "i32.clz")
	case fopI32Ctz:
		d.dumpUnOp(ip, fi, "i32.ctz")
	case fopI32Popcnt:
		d.dumpUnOp(ip, fi, "i32.popcnt")
	case fopI32Add:
		d.dumpBinOp(ip, fi, "i32.add")
	case fopI32Sub:
		d.dumpBinOp(ip, fi, "i32.sub")
	case fopI32Mul:
		d.dumpBinOp(ip, fi, "i32.mul")
	case fopI32DivS:
		d.dumpBinOp(ip, fi, "i32.div_s")
	case fopI32DivU:
		d.dumpBinOp(ip, fi, "i32.div_u")
	case fopI32RemS:
		d.dumpBinOp(ip, fi, "i32.rem_s")
	case fopI32RemU:
		d.dumpBinOp(ip, fi, "i32.rem_u")
	case fopI32And:
		d.dumpBinOp(ip, fi, "i32.and")
	case fopI32Or:
		d.dumpBinOp(ip, fi, "i32.or")
	case fopI32Xor:
		d.dumpBinOp(ip, fi, "i32.xor")
	case fopI32Shl:
		d.dumpBinOp(ip, fi, "i32.shl")
	case fopI32ShrS:
		d.dumpBinOp(ip, fi, "i32.shr_s")
	case fopI32ShrU:
		d.dumpBinOp(ip, fi, "i32.shr_u")
	case fopI32Rotl:
		d.dumpBinOp(ip, fi, "i32.rotl")
	case fopI32Rotr:
		d.dumpBinOp(ip, fi, "i32.rotr")
	case fopI64Clz:
		d.dumpUnOp(ip, fi, "i64.clz")
	case fopI64Ctz:
		d.dumpUnOp(ip, fi, "i64.ctz")
	case fopI64Popcnt:
		d.dumpUnOp(ip, fi, "i64.popcnt")
	case fopI64Add:
		d.dumpBinOp(ip, fi, "i64.add")
	case fopI64Sub:
		d.dumpBinOp(ip, fi, "i64.sub")
	case fopI64Mul:
		d.dumpBinOp(ip, fi, "i64.mul")
	case fopI64DivS:
		d.dumpBinOp(ip, fi, "i64.div_s")
	case fopI64DivU:
		d.dumpBinOp(ip, fi, "i64.div_u")
	case fopI64RemS:
		d.dumpBinOp(ip, fi, "i64.rem_s")
	case fopI64RemU:
		d.dumpBinOp(ip, fi, "i64.rem_u")
	case fopI64And:
		d.dumpBinOp(ip, fi, "i64.and")
	case fopI64Or:
		d.dumpBinOp(ip, fi, "i64.or")
	case fopI64Xor:
		d.dumpBinOp(ip, fi, "i64.xor")
	case fopI64Shl:
		d.dumpBinOp(ip, fi, "i64.shl")
	case fopI64ShrS:
		d.dumpBinOp(ip, fi, "i64.shr_s")
	case fopI64ShrU:
		d.dumpBinOp(ip, fi, "i64.shr_u")
	case fopI64Rotl:
		d.dumpBinOp(ip, fi, "i64.rotl")
	case fopI64Rotr:
		d.dumpBinOp(ip, fi, "i64.rotr")
	case fopF32Abs:
		d.dumpUnOp(ip, fi, "f32.abs")
	case fopF32Neg:
		d.dumpUnOp(ip, fi, "f32.neg")
	case fopF32Ceil:
		d.dumpUnOp(ip, fi, "f32.ceil")
	case fopF32Floor:
		d.dumpUnOp(ip, fi, "f32.floor")
	case fopF32Trunc:
		d.dumpUnOp(ip, fi, "f32.trunc")
	case fopF32Nearest:
		d.dumpUnOp(ip, fi, "f32.nearest")
	case fopF32Sqrt:
		d.dumpUnOp(ip, fi, "f32.sqrt")
	case fopF32Add:
		d.dumpBinOp(ip, fi, "f32.add")
	case fopF32Sub:
		d.dumpBinOp(ip, fi, "f32.sub")
	case fopF32Mul:
		d.dumpBinOp(ip, fi, "f32.mul")
	case fopF32Div:
		d.dumpBinOp(ip, fi, "f32.div")
	case fopF32Min:
		d.dumpBinOp(ip, fi, "f32.min")
	case fopF32Max:
		d.dumpBinOp(ip, fi, "f32.max")
	case fopF32Copysign:
		d.dumpBinOp(ip, fi, "f32.copysign")
	case fopF64Abs:
		d.dumpUnOp(ip, fi, "f64.abs")
	case fopF64Neg:
		d.dumpUnOp(ip, fi, "f64.neg")
	case fopF64Ceil:
		d.dumpUnOp(ip, fi, "f64.ceil")
	case fopF64Floor:
		d.dumpUnOp(ip, fi, "f64.floor")
	case fopF64Trunc:
		d.dumpUnOp(ip, fi, "f64.trunc")
	case fopF64Nearest:
		d.dumpUnOp(ip, fi, "f64.nearest")
	case fopF64Sqrt:
		d.dumpUnOp(ip, fi, "f64.sqrt")
	case fopF64Add:
		d.dumpBinOp(ip, fi, "f64.add")
	case fopF64Sub:
		d.dumpBinOp(ip, fi, "f64.sub")
	case fopF64Mul:
		d.dumpBinOp(ip, fi, "f64.mul")
	case fopF64Div:
		d.dumpBinOp(ip, fi, "f64.div")
	case fopF64Min:
		d.dumpBinOp(ip, fi, "f64.min")
	case fopF64Max:
		d.dumpBinOp(ip, fi, "f64.max")
	case fopF64Copysign:
		d.dumpBinOp(ip, fi, "f64.copysign")
	case fopI32WrapI64:
		d.dumpUnOp(ip, fi, "i32.wrap_i64")
	case fopI32TruncF32S:
		d.dumpUnOp(ip, fi, "i32.trunc_f32_s")
	case fopI32TruncF32U:
		d.dumpUnOp(ip, fi, "i32.trunc_f32_u")
	case fopI32TruncF64S:
		d.dumpUnOp(ip, fi, "i32.trunc_f64_s")
	case fopI32TruncF64U:
		d.dumpUnOp(ip, fi, "i32.trunc_f64_u")
	case fopI64ExtendI32S:
		d.dumpUnOp(ip, fi, "i64.extend_i32_s")
	case fopI64ExtendI32U:
		d.dumpUnOp(ip, fi, "i64.extend_i32_u")
	case fopI64TruncF32S:
		d.dumpUnOp(ip, fi, "i64.trunc_f32_s")
	case fopI64TruncF32U:
		d.dumpUnOp(ip, fi, "i64.trunc_f32_u")
	case fopI64TruncF64S:
		d.dumpUnOp(ip, fi, "i64.trunc_f64_s")
	case fopI64TruncF64U:
		d.dumpUnOp(ip, fi, "i64.trunc_f64_u")
	case fopF32ConvertI32S:
		d.dumpUnOp(ip, fi, "f32.convert_i32_s")
	case fopF32ConvertI32U:
		d.dumpUnOp(ip, fi, "f32.convert_i32_u")
	case fopF32ConvertI64S:
		d.dumpUnOp(ip, fi, "f32.convert_i64_s")
	case fopF32ConvertI64U:
		d.dumpUnOp(ip, fi, "f32.convert_i64_u")
	case fopF32DemoteF64:
		d.dumpUnOp(ip, fi, "f32.demote_f64")
	case fopF64ConvertI32S:
		d.dumpUnOp(ip, fi, "f64.convert_i32_s")
	case fopF64ConvertI32U:
		d.dumpUnOp(ip, fi, "f64.convert_i32_u")
	case fopF64ConvertI64S:
		d.dumpUnOp(ip, fi, "f64.convert_i64_s")
	case fopF64ConvertI64U:
		d.dumpUnOp(ip, fi, "f64.convert_i64_u")
	case fopF64PromoteF32:
		d.dumpUnOp(ip, fi, "f64.promote_f32")
	case fopI32ReinterpretF32:
		d.dumpUnOp(ip, fi, "i32.reinterpret_f32")
	case fopI64ReinterpretF64:
		d.dumpUnOp(ip, fi, "i64.reinterpret_f64")
	case fopF32ReinterpretI32:
		d.dumpUnOp(ip, fi, "f32.reinterpret_i32")
	case fopF64ReinterpretI64:
		d.dumpUnOp(ip, fi, "f64.reinterpret_i64")
	case fopI32Extend8S:
		d.dumpUnOp(ip, fi, "i32.extend8_s")
	case fopI32Extend16S:
		d.dumpUnOp(ip, fi, "i32.extend16_s")
	case fopI64Extend8S:
		d.dumpUnOp(ip, fi, "i64.extend8_s")
	case fopI64Extend16S:
		d.dumpUnOp(ip, fi, "i64.extend16_s")
	case fopI64Extend32S:
		d.dumpUnOp(ip, fi, "i64.extend32_s")
	case fopI32TruncSatF32S:
		d.dumpUnOp(ip, fi, "i32.trunc_sat_f32_s")
	case fopI32TruncSatF32U:
		d.dumpUnOp(ip, fi, "i32.trunc_sat_f32_u")
	case fopI32TruncSatF64S:
		d.dumpUnOp(ip, fi, "i32.trunc_sat_f64_s")
	case fopI32TruncSatF64U:
		d.dumpUnOp(ip, fi, "i32.trunc_sat_f64_u")
	case fopI64TruncSatF32S:
		d.dumpUnOp(ip, fi, "i64.trunc_sat_f32_s")
	case fopI64TruncSatF32U:
		d.dumpUnOp(ip, fi, "i64.trunc_sat_f32_u")
	case fopI64TruncSatF64S:
		d.dumpUnOp(ip, fi, "i64.trunc_sat_f64_s")
	case fopI64TruncSatF64U:
		d.dumpUnOp(ip, fi, "i64.trunc_sat_f64_u")

	case fopBrIfI32Eqz:
		d.dumpBranch(ip, fi, "br_if.i32.eqz", false)
	case fopBrIfI32Eq:
		d.dumpBranch(ip, fi, "br_if.i32.eq", false)
	case fopBrIfI32Ne:
		d.dumpBranch(ip, fi, "br_if.i32.ne", false)
	case fopBrIfI32LtS:
		d.dumpBranch(ip, fi, "br_if.i32.lt_s", false)
	case fopBrIfI32LtU:
		d.dumpBranch(ip, fi, "br_if.i32.lt_u", false)
	case fopBrIfI32GtS:
		d.dumpBranch(ip, fi, "br_if.i32.gt_s", false)
	case fopBrIfI32GtU:
		d.dumpBranch(ip, fi, "br_if.i32.gt_u", false)
	case fopBrIfI32LeS:
		d.dumpBranch(ip, fi, "br_if.i32.le_s", false)
	case fopBrIfI32LeU:
		d.dumpBranch(ip, fi, "br_if.i32.le_u", false)
	case fopBrIfI32GeS:
		d.dumpBranch(ip, fi, "br_if.i32.ge_s", false)
	case fopBrIfI32GeU:
		d.dumpBranch(ip, fi, "br_if.i32.ge_u", false)

	case fopBrIfI64Eqz:
		d.dumpBranch(ip, fi, "br_if.i64.eqz", false)
	case fopBrIfI64Eq:
		d.dumpBranch(ip, fi, "br_if.i64.eq", false)
	case fopBrIfI64Ne:
		d.dumpBranch(ip, fi, "br_if.i64.ne", false)
	case fopBrIfI64LtS:
		d.dumpBranch(ip, fi, "br_if.i64.lt_s", false)
	case fopBrIfI64LtU:
		d.dumpBranch(ip, fi, "br_if.i64.lt_u", false)
	case fopBrIfI64GtS:
		d.dumpBranch(ip, fi, "br_if.i64.gt_s", false)
	case fopBrIfI64GtU:
		d.dumpBranch(ip, fi, "br_if.i64.gt_u", false)
	case fopBrIfI64LeS:
		d.dumpBranch(ip, fi, "br_if.i64.le_s", false)
	case fopBrIfI64LeU:
		d.dumpBranch(ip, fi, "br_if.i64.le_u", false)
	case fopBrIfI64GeS:
		d.dumpBranch(ip, fi, "br_if.i64.ge_s", false)
	case fopBrIfI64GeU:
		d.dumpBranch(ip, fi, "br_if.i64.ge_u", false)

	case fopBrIfF32Eq:
		d.dumpBranch(ip, fi, "br_if.f32.eq", false)
	case fopBrIfF32Ne:
		d.dumpBranch(ip, fi, "br_if.f32.ne", false)
	case fopBrIfF32Lt:
		d.dumpBranch(ip, fi, "br_if.f32.lt", false)
	case fopBrIfF32Gt:
		d.dumpBranch(ip, fi, "br_if.f32.gt", false)
	case fopBrIfF32Le:
		d.dumpBranch(ip, fi, "br_if.f32.le", false)
	case fopBrIfF32Ge:
		d.dumpBranch(ip, fi, "br_if.f32.ge", false)

	case fopBrIfF64Eq:
		d.dumpBranch(ip, fi, "br_if.f64.eq", false)
	case fopBrIfF64Ne:
		d.dumpBranch(ip, fi, "br_if.f64.ne", false)
	case fopBrIfF64Lt:
		d.dumpBranch(ip, fi, "br_if.f64.lt", false)
	case fopBrIfF64Gt:
		d.dumpBranch(ip, fi, "br_if.f64.gt", false)
	case fopBrIfF64Le:
		d.dumpBranch(ip, fi, "br_if.f64.le", false)
	case fopBrIfF64Ge:
		d.dumpBranch(ip, fi, "br_if.f64.ge", false)
	}
	fmt.Fprintf(d.w, "\n")
}

func (d *dumper) dumpFcode(body []finstruction) {
	d.ipToLabelidx = map[int]int{}
	for i := range d.fn.labels {
		d.ipToLabelidx[d.fn.labels[i].continuation[0]] = i
	}

	fmt.Fprintf(d.w, "// locals: %d, maxstack: %d\n", d.fn.numLocals, d.fn.metrics.MaxStackDepth)
	fmt.Fprintf(d.w, "f%v:", d.fn.index)
	d.dumpTuple(len(d.fn.signature.ParamTypes), len(d.fn.signature.ParamTypes))
	fmt.Fprintf(d.w, "\n")
	for ip := range body {
		d.dumpInstruction(ip, &body[ip])
	}
	fmt.Fprintf(d.w, "\n")
}

func dumpFinstruction(fn *function, fi *finstruction) {
	d := dumper{fn: fn}
	d.dumpInstruction(0, fi)
}
