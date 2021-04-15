package golang

import (
	"fmt"
	"io"
	"math"
	"strconv"

	"github.com/pgavlin/warp/compiler/wax"
	"github.com/pgavlin/warp/wasm"
	"github.com/pgavlin/warp/wasm/code"
)

const flagsUntyped = wax.FlagsBackend

func comma(i int) string {
	if i > 0 {
		return ", "
	}
	return ""
}

func goType(t wasm.ValueType) string {
	switch t {
	case wax.ValueTypeBool:
		return "bool"
	case wasm.ValueTypeI32:
		return "int32"
	case wasm.ValueTypeI64:
		return "int64"
	case wasm.ValueTypeF32:
		return "float32"
	case wasm.ValueTypeF64:
		return "float64"
	default:
		panic("unknown value type")
	}
}

func (m *moduleCompiler) emitFunctionSignature(w io.Writer, sig wasm.FunctionSig, indirect bool) error {
	if err := printf(w, "(m *%sInstance, t *exec.Thread", m.name); err != nil {
		return err
	}
	if indirect {
		if err := printf(w, ", tableidx uint32"); err != nil {
			return err
		}
	}
	for i, t := range sig.ParamTypes {
		if err := printf(w, ", v%d %s", i, goType(t)); err != nil {
			return err
		}
	}
	if err := printf(w, ")"); err != nil {
		return err
	}
	switch len(sig.ReturnTypes) {
	case 0:
		// OK
	case 1:
		if err := printf(w, " (r0 %s)", goType(sig.ReturnTypes[0])); err != nil {
			return err
		}
	default:
		if err := printf(w, " ("); err != nil {
			return err
		}
		for i, t := range sig.ReturnTypes {
			if err := printf(w, "%vr%d %s", comma(i), i, goType(t)); err != nil {
				return err
			}
		}
		if err := printf(w, ")"); err != nil {
			return err
		}
	}
	return nil
}

type functionCompiler struct {
	wax.Function

	m     *moduleCompiler
	index int
}

func (f *functionCompiler) compile(m *moduleCompiler, index int, typeIndex uint32, signature wasm.FunctionSig, body wasm.FunctionBody) {
	f.m = m
	f.index = index
	f.Function = wax.NewFunction(typeIndex, signature, body, m, f)

	s := f.Scope(f.m)

	codeBody, err := code.Decode(body.Code, s, f.Signature.ReturnTypes)
	if err != nil {
		panic(err)
	}

	// Compile the function body into expression trees.
	for ip, instr := range codeBody.Instructions {
		defs := f.ImportInstruction(ip, instr, s)
		for _, d := range defs {
			markUntypedExpressions(d.Expression)
		}
	}
	if len(f.Stack) > 0 {
		f.ImportInstruction(0, code.Return(), s)
	} else {
		// We need a terminal return for Leave().
		f.Body = append(f.Body, &wax.Def{Expression: &wax.Expression{Function: &f.Function, Instr: code.Return()}})
	}
}

func (f *functionCompiler) FormatExpression(fs fmt.State, verb rune, x *wax.Expression) {
	if err := f.emitExpression(fs, x, 0); err != nil {
		panic(err)
	}
}

func (f *functionCompiler) FormatUse(fs fmt.State, verb rune, u *wax.Use) {
	switch verb {
	case 'u', 'd', 'U', 'D':
		// OK
	default:
		panic(fmt.Errorf("unsupported verb %v", string(verb)))
	}

	if u.IsTemp() {
		mustPrintf(fs, "t%d", u.Temp)
		return
	}

	x := u.X
	switch verb {
	case 'D':
		width, ok := fs.Width()
		if !ok {
			break
		}

		switch x.Instr.Opcode {
		case code.OpI32Const:
			switch width {
			case 1:
				mustPrintf(fs, "%d", int8(x.Instr.I32()))
			case 2:
				mustPrintf(fs, "%d", int16(x.Instr.I32()))
			default:
				mustPrintf(fs, "%d", x.Instr.I32())
			}
			return
		case code.OpI64Const:
			switch width {
			case 1:
				mustPrintf(fs, "%d", int8(x.Instr.I64()))
			case 2:
				mustPrintf(fs, "%d", int16(x.Instr.I64()))
			case 4:
				mustPrintf(fs, "%d", int32(x.Instr.I64()))
			default:
				mustPrintf(fs, "%d", x.Instr.I64())
			}
			return
		}
	case 'U':
		width, ok := fs.Width()
		if !ok {
			break
		}

		switch x.Instr.Opcode {
		case code.OpI32Const:
			switch width {
			case 1:
				mustPrintf(fs, "%d", uint8(x.Instr.I32()))
			case 2:
				mustPrintf(fs, "%d", uint16(x.Instr.I32()))
			default:
				mustPrintf(fs, "%d", uint32(x.Instr.I32()))
			}
			return
		case code.OpI64Const:
			switch width {
			case 1:
				mustPrintf(fs, "%d", uint8(x.Instr.I64()))
			case 2:
				mustPrintf(fs, "%d", uint16(x.Instr.I64()))
			case 4:
				mustPrintf(fs, "%d", uint32(x.Instr.I64()))
			default:
				mustPrintf(fs, "%d", uint64(x.Instr.I64()))
			}
			return
		}
	}

	if isUntyped(x.Flags) && verb != 'u' {
		err := f.emitUntyped(fs, u)
		if err != nil {
			panic(err)
		}
		return
	}

	parentPrecedence, ok := fs.Precision()
	if !ok {
		parentPrecedence = 0
	}
	if err := f.emitExpression(fs, x, parentPrecedence); err != nil {
		panic(err)
	}
}

func (f *functionCompiler) FormatUses(fs fmt.State, verb rune, uses wax.Uses) {
	if verb != 'u' && verb != 'd' {
		panic(fmt.Errorf("unsupported verb %v", string(verb)))
	}

	for i, u := range uses {
		if i > 0 {
			if err := printf(fs, ", "); err != nil {
				panic(err)
			}
		}
		f.FormatUse(fs, verb, u)
	}
}

func (f *functionCompiler) FormatDef(fs fmt.State, verb rune, d *wax.Def) {
	switch verb {
	case 'g':
		label, ok := fs.Width()
		if !ok {
			label = 0
		}
		dest := d.BranchTargets[label]

		uses := d.Uses
		if d.Instr.Opcode != code.OpBr {
			uses = uses[:len(uses)-1]
		}

		f.emitBranchDefs(fs, dest, uses)
		if dest.Entry.Instr.Opcode == code.OpLoop {
			mustPrintf(fs, "continue l%d", dest.Label)
		} else {
			mustPrintf(fs, "break l%d", dest.Label)
		}
	default:
		panic(fmt.Errorf("unsupported verb %v", string(verb)))
	}
}

func (f *functionCompiler) emit(w io.Writer) error {
	// Emit the function signature.
	if err := printf(w, "func %s", f.m.functionName(uint32(f.index))); err != nil {
		return err
	}
	if err := f.m.emitFunctionSignature(w, f.Signature, false); err != nil {
		return err
	}
	if err := printf(w, "{\n\tt.Enter()\n"); err != nil {
		return err
	}

	for i, t := range f.Locals[len(f.Signature.ParamTypes):] {
		if f.UsedLocals[len(f.Signature.ParamTypes)+i] {
			if err := printf(w, "v%d := %s(0)\n", len(f.Signature.ParamTypes)+i, goType(t)); err != nil {
				return err
			}
		}
	}

	for _, x := range f.Body {
		if err := f.emitDef(w, x); err != nil {
			return err
		}
	}

	return printf(w, "}\n")
}

func (m *moduleCompiler) emitImportedFunction(w io.Writer, index uint32, sig wasm.FunctionSig) error {
	// Emit the function signature.
	if err := printf(w, "func %s", m.functionName(index)); err != nil {
		return err
	}
	if err := m.emitFunctionSignature(w, sig, false); err != nil {
		return err
	}
	if err := printf(w, "{\n"); err != nil {
		return err
	}

	if err := printf(w, "a := [...]uint64{"); err != nil {
		return err
	}
	for i, t := range sig.ParamTypes {
		var err error
		switch t {
		case wasm.ValueTypeI32, wasm.ValueTypeI64:
			err = printf(w, "%vuint64(v%d)", comma(i), i)
		case wasm.ValueTypeF32:
			err = printf(w, "%vuint64(math.Float32bits(v%d))", comma(i), i)
		case wasm.ValueTypeF64:
			err = printf(w, "%vmath.Float64bits(v%d)", comma(i), i)
		default:
			panic("unknown value type")
		}
		if err != nil {
			return err
		}
	}
	if err := printf(w, "}\n"); err != nil {
		return err
	}
	if err := printf(w, "var r [%d]uint64\n", len(sig.ReturnTypes)); err != nil {
		return err
	}
	if err := printf(w, "m.importedFunctions[%d].UncheckedCall(t, a[:], r[:])\n", index); err != nil {
		return err
	}
	if len(sig.ReturnTypes) > 0 {
		if err := printf(w, "return "); err != nil {
			return err
		}
		for i, t := range sig.ReturnTypes {
			var err error
			switch t {
			case wasm.ValueTypeI32:
				err = printf(w, "%vint32(r[%d])", comma(i), i)
			case wasm.ValueTypeI64:
				err = printf(w, "%vint64(r[%d])", comma(i), i)
			case wasm.ValueTypeF32:
				err = printf(w, "%vmath.Float32frombits(uint32(r[%d]))", comma(i), i)
			case wasm.ValueTypeF64:
				err = printf(w, "%vmath.Float64frombits(r[%d])", comma(i), i)
			default:
				panic("unknown value type")
			}
			if err != nil {
				return err
			}
		}
	}

	return printf(w, "}\n")
}

func printf(w io.Writer, str string, args ...interface{}) error {
	_, err := fmt.Fprintf(w, str, args...)
	return err
}

func mustPrintf(w io.Writer, str string, args ...interface{}) {
	err := printf(w, str, args...)
	if err != nil {
		panic(err)
	}
}

func (f *functionCompiler) emitUntyped(w io.Writer, u *wax.Use) error {
	switch u.Type {
	case wax.ValueTypeBool:
		return printf(w, "bool(%x)", u.X)
	case wasm.ValueTypeI32:
		return printf(w, "int32(%x)", u.X)
	case wasm.ValueTypeI64:
		return printf(w, "int64(%x)", u.X)
	case wasm.ValueTypeF32:
		return printf(w, "float32(%x)", u.X)
	case wasm.ValueTypeF64:
		return printf(w, "float64(%x)", u.X)
	default:
		panic("unreachable")
	}
}

func (f *functionCompiler) emitDeclareTemps(w io.Writer, temp int, types []wasm.ValueType) error {
	if len(types) == 0 {
		return nil
	}

	for i := range types {
		if err := printf(w, "%vt%d", comma(i), temp+i); err != nil {
			return err
		}
	}
	if err := printf(w, " := "); err != nil {
		return err
	}
	for i, t := range types {
		if err := printf(w, "%v%v(0)", comma(i), goType(t)); err != nil {
			return err
		}
	}
	return printf(w, "\n")
}

func (f *functionCompiler) emitDeclareAssignTemps(w io.Writer, temp int, uses wax.Uses) error {
	if len(uses) == 0 {
		return nil
	}

	for i := range uses {
		if err := printf(w, "%vt%d", comma(i), temp+i); err != nil {
			return err
		}
	}
	return printf(w, " := %d\n", uses)
}

func (f *functionCompiler) emitAssignTemps(w io.Writer, temp int, uses wax.Uses) error {
	if len(uses) == 0 {
		return nil
	}

	for i := range uses {
		if err := printf(w, "%vt%d", comma(i), temp+i); err != nil {
			return err
		}
	}
	return printf(w, " = %d\n", uses)
}

func (f *functionCompiler) emitBlockIns(w io.Writer, b *wax.Block, uses wax.Uses) error {
	return f.emitDeclareAssignTemps(w, b.InTemp, uses)
}

func (f *functionCompiler) emitBranchDefs(w io.Writer, b *wax.Block, uses wax.Uses) error {
	temp := b.OutTemp
	if b.Entry.Instr.Opcode == code.OpLoop {
		temp = b.InTemp
	}
	return f.emitAssignTemps(w, temp, uses)
}

func (f *functionCompiler) emitBlockOuts(w io.Writer, b *wax.Block, uses wax.Uses) error {
	return f.emitAssignTemps(w, b.OutTemp, uses)
}

func (f *functionCompiler) load(x *wax.Expression, loadWidth int) string {
	switch {
	case x.Instr.Offset() == 0:
		return fmt.Sprintf("m.mem0.Uint%vAt(uint32(%4U))", loadWidth, x.Uses[0])
	case isConst0(x.Uses[0]):
		return fmt.Sprintf("m.mem0.Uint%vAt(%d)", loadWidth, x.Instr.Offset())
	}
	return fmt.Sprintf("m.mem0.Uint%v(uint32(%4U), %d)", loadWidth, x.Uses[0], x.Instr.Offset())
}

func (f *functionCompiler) emitStore(w io.Writer, x *wax.Def, storeWidth int, value string) error {
	switch {
	case x.Instr.Offset() == 0:
		return printf(w, "m.mem0.PutUint%vAt(%s, uint32(%4U))\n", storeWidth, value, x.Uses[0])
	case isConst0(x.Uses[0]):
		return printf(w, "m.mem0.PutUint%vAt(%s, %d)\n", storeWidth, value, x.Instr.Offset())
	}
	return printf(w, "m.mem0.PutUint%v(%s, uint32(%4U), %d)\n", storeWidth, value, x.Uses[0], x.Instr.Offset())
}

func (f *functionCompiler) emitDef(w io.Writer, x *wax.Def) error {
	switch x.Instr.Opcode {
	case code.OpUnreachable:
		return printf(w, "panic(exec.TrapUnreachable)\n")

	case code.OpBlock, code.OpLoop:
		if err := f.emitBlockIns(w, x.Block, x.Uses); err != nil {
			return err
		}
		if err := f.emitDeclareTemps(w, x.Block.OutTemp, x.Block.Outs); err != nil {
			return err
		}
		if x.Block.BranchTarget {
			return printf(w, "l%d: for {\n", x.Block.Label)
		}
		return nil
	case code.OpIf:
		ins, cond := x.Uses[:len(x.Uses)-1], x.Uses[len(x.Uses)-1]
		if err := f.emitBlockIns(w, x.Block, ins); err != nil {
			return err
		}
		if x.Block.Else == nil {
			uses := make([]*wax.Use, len(ins))
			for i := range ins {
				uses[i] = &wax.Use{Function: &f.Function, Temp: x.Block.InTemp + i}
			}
			if err := f.emitDeclareAssignTemps(w, x.Block.OutTemp, uses); err != nil {
				return err
			}
		} else {
			if err := f.emitDeclareTemps(w, x.Block.OutTemp, x.Block.Outs); err != nil {
				return err
			}
		}
		if !x.Block.BranchTarget {
			return printf(w, "if %u {\n", cond)
		}
		return printf(w, "l%d: for {\nif %u {\n", x.Block.Label, cond)
	case code.OpElse:
		if err := f.emitBlockOuts(w, x.Block, x.Uses); err != nil {
			return err
		}
		return printf(w, "} else {\n")
	case code.OpEnd:
		if x.Block != nil {
			if err := f.emitBlockOuts(w, x.Block, x.Uses); err != nil {
				return err
			}
			if x.Block.Entry.Instr.Opcode == code.OpIf {
				if err := printf(w, "}\n"); err != nil {
					return err
				}
			}
			if x.Block.BranchTarget {
				if !x.Block.Unreachable || x.Block.Entry.Instr.Opcode == code.OpIf {
					if err := printf(w, "break\n"); err != nil {
						return err
					}
				}
				return printf(w, "}\n")
			}
		}
		return nil

	case code.OpBr:
		return printf(w, "%g\n", x)
	case code.OpBrIf:
		if err := printf(w, "if %u {\n %g }\n", x.Uses[len(x.Uses)-1], x); err != nil {
			return err
		}
		if len(x.Types) != 0 {
			for i := range x.Types {
				if err := printf(w, "%vt%d", comma(i), x.Temp+i); err != nil {
					return err
				}
			}
			if err := printf(w, " := %d\n", x.Uses[:len(x.Uses)-1]); err != nil {
				return err
			}
		}
		return nil
	case code.OpBrTable:
		if err := printf(w, "switch %u {\n", x.Uses[len(x.Uses)-1]); err != nil {
			return err
		}
		for i := range x.BranchTargets[:len(x.BranchTargets)-1] {
			if err := printf(w, "case %v:\n%[1]*[2]g\n", i, x); err != nil {
				return err
			}
		}
		return printf(w, "default:\n%[1]*[2]g\n}\n", len(x.BranchTargets)-1, x)

	case code.OpCall:
		if len(x.Types) > 0 {
			for i := range x.Types {
				if err := printf(w, "%vt%d", comma(i), x.Temp+i); err != nil {
					return err
				}
			}
			if err := printf(w, " := "); err != nil {
				return err
			}
		}
		return printf(w, "%s(m, t%v%u)\n", f.m.functionName(x.Instr.Funcidx()), comma(len(x.Uses)), x.Uses)

	case code.OpCallIndirect:
		typeidx := x.Instr.Typeidx()
		sig := f.m.module.Types.Entries[typeidx]

		if len(x.Types) > 0 {
			for i := range x.Types {
				if err := printf(w, "%vt%d", comma(i), x.Temp+i); err != nil {
					return err
				}
			}
			if err := printf(w, " := "); err != nil {
				return err
			}
		}
		tableidx := x.Uses[len(x.Uses)-1]
		uses := x.Uses[:len(x.Uses)-1]
		return printf(w, "%sCallIndirect(m, t, uint32(%4U)%v%u)\n", f.m.functionTypeName(sig), tableidx, comma(len(uses)), uses)

	case code.OpReturn:
		if len(x.Uses) > 0 {
			for i := range x.Uses {
				if err := printf(w, "%vr%d", comma(i), i); err != nil {
					return err
				}
			}
			if err := printf(w, " = %u\n", x.Uses); err != nil {
				return err
			}
		}
		return printf(w, "t.Leave()\nreturn\n")

	case code.OpDrop:
		return printf(w, "_ = %u\n", x.Uses[0])

	case code.OpSelect:
		return printf(w, "var t%d %s\nif %u {\nt%d = %u\n} else {\nt%d = %u\n}\n", x.Temp, goType(x.Types[0]), x.Uses[2], x.Temp, x.Uses[0], x.Temp, x.Uses[1])

	case code.OpLocalSet:
		localidx := int(x.Instr.Localidx())
		if f.UsedLocals[localidx] {
			return printf(w, "v%d = %u\n", localidx, x.Uses[0])
		}
		return printf(w, "_ = %u\n", x.Uses[0])

	case code.OpLocalTee:
		localidx := int(x.Instr.Localidx())
		if err := printf(w, "v%d = %u\n", localidx, x.Uses[0]); err != nil {
			return err
		}
		return printf(w, "t%d := v%d\n", x.Temp, localidx)

	case code.OpGlobalSet:
		globalidx := x.Instr.Globalidx()
		if globalidx < uint32(len(f.m.importedGlobals)) || f.m.exportedGlobals[globalidx] {
			if err := printf(w, "m.g%d", globalidx); err != nil {
				return err
			}
			switch f.m.globalType(globalidx) {
			case wasm.ValueTypeI32:
				return printf(w, ".SetI32(%u)\n", x.Uses[0])
			case wasm.ValueTypeI64:
				return printf(w, ".SetI64(%u)\n", x.Uses[0])
			case wasm.ValueTypeF32:
				return printf(w, ".SetF32(%u)\n", x.Uses[0])
			case wasm.ValueTypeF64:
				return printf(w, ".SetF64(%u)\n", x.Uses[0])
			default:
				panic("unexpected global type")
			}
		}
		return printf(w, "m.g%d = %u\n", globalidx, x.Uses[0])

	case code.OpI32Store:
		return f.emitStore(w, x, 32, fmt.Sprintf("uint32(%4U)", x.Uses[1]))
	case code.OpI64Store:
		return f.emitStore(w, x, 64, fmt.Sprintf("uint64(%8U)", x.Uses[1]))
	case code.OpF32Store:
		return f.emitStore(w, x, 32, fmt.Sprintf("math.Float32bits(%u)", x.Uses[1]))
	case code.OpF64Store:
		return f.emitStore(w, x, 64, fmt.Sprintf("math.Float64bits(%u)", x.Uses[1]))

	case code.OpI32Store8:
		return f.emitStore(w, x, 8, fmt.Sprintf("uint8(%1U)", x.Uses[1]))
	case code.OpI32Store16:
		return f.emitStore(w, x, 16, fmt.Sprintf("uint16(%2U)", x.Uses[1]))

	case code.OpI64Store8:
		return f.emitStore(w, x, 8, fmt.Sprintf("uint8(%1U)", x.Uses[1]))
	case code.OpI64Store16:
		return f.emitStore(w, x, 16, fmt.Sprintf("uint16(%2U)", x.Uses[1]))
	case code.OpI64Store32:
		return f.emitStore(w, x, 32, fmt.Sprintf("uint32(%4U)", x.Uses[1]))

	case code.OpMemoryGrow:
		return printf(w, "var t%d int32\nif sz, err := m.mem0.Grow(uint32(%4U)); err != nil {\nt%d = -1\n} else {\nt%d = int32(sz)\n}\n", x.Temp, x.Uses[0], x.Temp, x.Temp)

	default:
		return printf(w, "t%d := %d\n", x.Temp, wax.UseExpression(x.Types[0], x.Expression))
	}
}

func (f *functionCompiler) emitExpression(w io.Writer, x *wax.Expression, parentPrecedence int) error {
	if x.IsPseudo() {
		return f.emitPseudoOp(w, x, parentPrecedence)
	}

	switch x.Instr.Opcode {
	case code.OpLocalGet, code.OpLocalTee:
		return printf(w, "v%d", x.Instr.Localidx())

	case code.OpGlobalGet:
		globalidx := x.Instr.Globalidx()
		if globalidx < uint32(len(f.m.importedGlobals)) || f.m.exportedGlobals[globalidx] {
			if err := printf(w, "m.g%d", globalidx); err != nil {
				return err
			}
			switch f.m.globalType(globalidx) {
			case wasm.ValueTypeI32:
				return printf(w, ".GetI32()")
			case wasm.ValueTypeI64:
				return printf(w, ".GetI64()")
			case wasm.ValueTypeF32:
				return printf(w, ".GetF32()")
			case wasm.ValueTypeF64:
				return printf(w, ".GetF64()")
			default:
				panic("unexpected global type")
			}
		}
		return printf(w, "m.g%d", globalidx)

	case code.OpI32Load:
		return printf(w, "int32(%s)", f.load(x, 32))
	case code.OpI64Load:
		return printf(w, "int64(%s)", f.load(x, 64))
	case code.OpF32Load:
		return printf(w, "math.Float32frombits(%s)", f.load(x, 32))
	case code.OpF64Load:
		return printf(w, "math.Float64frombits(%s)", f.load(x, 64))

	case code.OpI32Load8S:
		return printf(w, "int32(int8(%s))", f.load(x, 8))
	case code.OpI32Load8U:
		return printf(w, "int32(%s)", f.load(x, 8))
	case code.OpI32Load16S:
		return printf(w, "int32(int16(%s))", f.load(x, 16))
	case code.OpI32Load16U:
		return printf(w, "int32(%s)", f.load(x, 16))

	case code.OpI64Load8S:
		return printf(w, "int64(int8(%s))", f.load(x, 8))
	case code.OpI64Load8U:
		return printf(w, "int64(%s)", f.load(x, 8))
	case code.OpI64Load16S:
		return printf(w, "int64(int16(%s))", f.load(x, 16))
	case code.OpI64Load16U:
		return printf(w, "int64(%s)", f.load(x, 16))
	case code.OpI64Load32S:
		return printf(w, "int64(int32(%s))", f.load(x, 32))
	case code.OpI64Load32U:
		return printf(w, "int64(%s)", f.load(x, 32))

	case code.OpMemorySize:
		return printf(w, "int32(m.mem0.Size())")

	case code.OpI32Const:
		return printf(w, "%d", x.Instr.I32())
	case code.OpI64Const:
		return printf(w, "%d", x.Instr.I64())
	case code.OpF32Const:
		return printf(w, "%s", f32Const(x.Instr.F32()))
	case code.OpF64Const:
		return printf(w, "%s", f64Const(x.Instr.F64()))

	case code.OpI32Eqz:
		return printBinaryExpression(w, 3, parentPrecedence, "%.3u == 0", x.Uses[0])
	case code.OpI32Eq:
		return printBinaryExpression(w, 3, parentPrecedence, "%.3u == %.3u", x.Uses[0], x.Uses[1])
	case code.OpI32Ne:
		return printBinaryExpression(w, 3, parentPrecedence, "%.3u != %.3u", x.Uses[0], x.Uses[1])
	case code.OpI32LtS:
		return printBinaryExpression(w, 3, parentPrecedence, "%.3u < %.3u", x.Uses[0], x.Uses[1])
	case code.OpI32LtU:
		return printBinaryExpression(w, 3, parentPrecedence, "uint32(%4U) < uint32(%4U)", x.Uses[0], x.Uses[1])
	case code.OpI32GtS:
		return printBinaryExpression(w, 3, parentPrecedence, "%.3u > %.3u", x.Uses[0], x.Uses[1])
	case code.OpI32GtU:
		return printBinaryExpression(w, 3, parentPrecedence, "uint32(%4U) > uint32(%4U)", x.Uses[0], x.Uses[1])
	case code.OpI32LeS:
		return printBinaryExpression(w, 3, parentPrecedence, "%.3u <= %.3u", x.Uses[0], x.Uses[1])
	case code.OpI32LeU:
		return printBinaryExpression(w, 3, parentPrecedence, "uint32(%4U) <= uint32(%4U)", x.Uses[0], x.Uses[1])
	case code.OpI32GeS:
		return printBinaryExpression(w, 3, parentPrecedence, "%.3u >= %.3u", x.Uses[0], x.Uses[1])
	case code.OpI32GeU:
		return printBinaryExpression(w, 3, parentPrecedence, "uint32(%4U) >= uint32(%4U)", x.Uses[0], x.Uses[1])

	case code.OpI64Eqz:
		return printBinaryExpression(w, 3, parentPrecedence, "%u == 0", x.Uses[0])
	case code.OpI64Eq:
		return printBinaryExpression(w, 3, parentPrecedence, "%.3u == %.3u", x.Uses[0], x.Uses[1])
	case code.OpI64Ne:
		return printBinaryExpression(w, 3, parentPrecedence, "%.3u != %.3u", x.Uses[0], x.Uses[1])
	case code.OpI64LtS:
		return printBinaryExpression(w, 3, parentPrecedence, "%.3u < %.3u", x.Uses[0], x.Uses[1])
	case code.OpI64LtU:
		return printBinaryExpression(w, 3, parentPrecedence, "uint64(%8U) < uint64(%8U)", x.Uses[0], x.Uses[1])
	case code.OpI64GtS:
		return printBinaryExpression(w, 3, parentPrecedence, "%.3u > %.3u", x.Uses[0], x.Uses[1])
	case code.OpI64GtU:
		return printBinaryExpression(w, 3, parentPrecedence, "uint64(%8U) > uint64(%8U)", x.Uses[0], x.Uses[1])
	case code.OpI64LeS:
		return printBinaryExpression(w, 3, parentPrecedence, "%.3u <= %.3u", x.Uses[0], x.Uses[1])
	case code.OpI64LeU:
		return printBinaryExpression(w, 3, parentPrecedence, "uint64(%8U) <= uint64(%8U)", x.Uses[0], x.Uses[1])
	case code.OpI64GeS:
		return printBinaryExpression(w, 3, parentPrecedence, "%.3u >= %.3u", x.Uses[0], x.Uses[1])
	case code.OpI64GeU:
		return printBinaryExpression(w, 3, parentPrecedence, "uint64(%8U) >= uint64(%8U)", x.Uses[0], x.Uses[1])

	case code.OpF32Eq:
		return printBinaryExpression(w, 3, parentPrecedence, "%.3u == %.3u", x.Uses[0], x.Uses[1])
	case code.OpF32Ne:
		return printBinaryExpression(w, 3, parentPrecedence, "%.3u != %.3u", x.Uses[0], x.Uses[1])
	case code.OpF32Lt:
		return printBinaryExpression(w, 3, parentPrecedence, "%.3u < %.3u", x.Uses[0], x.Uses[1])
	case code.OpF32Gt:
		return printBinaryExpression(w, 3, parentPrecedence, "%.3u > %.3u", x.Uses[0], x.Uses[1])
	case code.OpF32Le:
		return printBinaryExpression(w, 3, parentPrecedence, "%.3u <= %.3u", x.Uses[0], x.Uses[1])
	case code.OpF32Ge:
		return printBinaryExpression(w, 3, parentPrecedence, "%.3u >= %.3u", x.Uses[0], x.Uses[1])

	case code.OpF64Eq:
		return printBinaryExpression(w, 3, parentPrecedence, "%.3u == %.3u", x.Uses[0], x.Uses[1])
	case code.OpF64Ne:
		return printBinaryExpression(w, 3, parentPrecedence, "%.3u != %.3u", x.Uses[0], x.Uses[1])
	case code.OpF64Lt:
		return printBinaryExpression(w, 3, parentPrecedence, "%.3u < %.3u", x.Uses[0], x.Uses[1])
	case code.OpF64Gt:
		return printBinaryExpression(w, 3, parentPrecedence, "%.3u > %.3u", x.Uses[0], x.Uses[1])
	case code.OpF64Le:
		return printBinaryExpression(w, 3, parentPrecedence, "%.3u <= %.3u", x.Uses[0], x.Uses[1])
	case code.OpF64Ge:
		return printBinaryExpression(w, 3, parentPrecedence, "%.3u >= %.3u", x.Uses[0], x.Uses[1])

	case code.OpI32Clz:
		return printf(w, "int32(bits.LeadingZeros32(uint32(%4U)))", x.Uses[0])
	case code.OpI32Ctz:
		return printf(w, "int32(bits.TrailingZeros32(uint32(%4U)))", x.Uses[0])
	case code.OpI32Popcnt:
		return printf(w, "int32(bits.OnesCount32(uint32(%4U)))", x.Uses[0])
	case code.OpI32Add:
		return printBinaryExpression(w, 4, parentPrecedence, "%.4u + %.4u", x.Uses[0], x.Uses[1])
	case code.OpI32Sub:
		return printBinaryExpression(w, 4, parentPrecedence, "%.4u - %.4u", x.Uses[0], x.Uses[1])
	case code.OpI32Mul:
		return printBinaryExpression(w, 5, parentPrecedence, "%.5u * %.5u", x.Uses[0], x.Uses[1])
	case code.OpI32DivS:
		return printf(w, "exec.I32DivS(%u, %u)", x.Uses[0], x.Uses[1])
	case code.OpI32DivU:
		if x.Uses[1].IsZeroIConst() {
			return printf(w, "exec.I32DivS(%u, %u)", x.Uses[0], x.Uses[1])
		}
		return printf(w, "int32(uint32(%4U) / uint32(%4U))", x.Uses[0], x.Uses[1])
	case code.OpI32RemS:
		return printBinaryExpression(w, 5, parentPrecedence, "%.5u %% %.5u", x.Uses[0], x.Uses[1])
	case code.OpI32RemU:
		return printf(w, "int32(uint32(%4U) %% uint32(%4U))", x.Uses[0], x.Uses[1])
	case code.OpI32And:
		return printBinaryExpression(w, 5, parentPrecedence, "%.5u & %.5u", x.Uses[0], x.Uses[1])
	case code.OpI32Or:
		return printBinaryExpression(w, 4, parentPrecedence, "%.4u | %.4u", x.Uses[0], x.Uses[1])
	case code.OpI32Xor:
		return printBinaryExpression(w, 4, parentPrecedence, "%.4u ^ %.4u", x.Uses[0], x.Uses[1])
	case code.OpI32Shl:
		return printBinaryExpression(w, 5, parentPrecedence, "%.5u << (%.5u & 31)", x.Uses[0], x.Uses[1])
	case code.OpI32ShrS:
		return printBinaryExpression(w, 5, parentPrecedence, "%.5u >> (%.5u & 31)", x.Uses[0], x.Uses[1])
	case code.OpI32ShrU:
		return printf(w, "int32(uint32(%4U) >> (uint32(%4U) & 31))", x.Uses[0], x.Uses[1])
	case code.OpI32Rotl:
		return printf(w, "int32(bits.RotateLeft32(uint32(%4U), int(%u)))", x.Uses[0], x.Uses[1])
	case code.OpI32Rotr:
		return printf(w, "int32(bits.RotateLeft32(uint32(%4U), -int(%u)))", x.Uses[0], x.Uses[1])

	case code.OpI64Clz:
		return printf(w, "int64(bits.LeadingZeros64(uint64(%8U)))", x.Uses[0])
	case code.OpI64Ctz:
		return printf(w, "int64(bits.TrailingZeros64(uint64(%8U)))", x.Uses[0])
	case code.OpI64Popcnt:
		return printf(w, "int64(bits.OnesCount64(uint64(%8U)))", x.Uses[0])
	case code.OpI64Add:
		return printBinaryExpression(w, 4, parentPrecedence, "%.4u + %.4u", x.Uses[0], x.Uses[1])
	case code.OpI64Sub:
		return printBinaryExpression(w, 4, parentPrecedence, "%.4u - %.4u", x.Uses[0], x.Uses[1])
	case code.OpI64Mul:
		return printBinaryExpression(w, 5, parentPrecedence, "%.5u * %.5u", x.Uses[0], x.Uses[1])
	case code.OpI64DivS:
		return printf(w, "exec.I64DivS(%u, %u)", x.Uses[0], x.Uses[1])
	case code.OpI64DivU:
		if x.Uses[1].IsZeroIConst() {
			return printf(w, "exec.I64DivS(%u, %u)", x.Uses[0], x.Uses[1])
		}
		return printf(w, "int64(uint64(%8U) / uint64(%8U))", x.Uses[0], x.Uses[1])
	case code.OpI64RemS:
		return printBinaryExpression(w, 5, parentPrecedence, "%.5u %% %.5u", x.Uses[0], x.Uses[1])
	case code.OpI64RemU:
		return printf(w, "int64(uint64(%8U) %% uint64(%8U))", x.Uses[0], x.Uses[1])
	case code.OpI64And:
		return printBinaryExpression(w, 5, parentPrecedence, "%.5u & %.5u", x.Uses[0], x.Uses[1])
	case code.OpI64Or:
		return printBinaryExpression(w, 4, parentPrecedence, "%.4u | %.4u", x.Uses[0], x.Uses[1])
	case code.OpI64Xor:
		return printBinaryExpression(w, 4, parentPrecedence, "%.4u ^ %.4u", x.Uses[0], x.Uses[1])
	case code.OpI64Shl:
		return printBinaryExpression(w, 5, parentPrecedence, "%.5u << (%.5u & 63)", x.Uses[0], x.Uses[1])
	case code.OpI64ShrS:
		return printBinaryExpression(w, 5, parentPrecedence, "%.5u >> (%.5u & 63)", x.Uses[0], x.Uses[1])
	case code.OpI64ShrU:
		return printf(w, "int64(uint64(%8U) >> (uint64(%8U) & 63))", x.Uses[0], x.Uses[1])
	case code.OpI64Rotl:
		return printf(w, "int64(bits.RotateLeft64(uint64(%8U), int(%u)))", x.Uses[0], x.Uses[1])
	case code.OpI64Rotr:
		return printf(w, "int64(bits.RotateLeft64(uint64(%8U), -int(%u)))", x.Uses[0], x.Uses[1])

	case code.OpF32Abs:
		return printf(w, "float32(math.Abs(float64(%u)))", x.Uses[0])
	case code.OpF32Neg:
		return printf(w, "-%u", x.Uses[0])
	case code.OpF32Ceil:
		return printf(w, "float32(math.Ceil(float64(%u)))", x.Uses[0])
	case code.OpF32Floor:
		return printf(w, "float32(math.Floor(float64(%u)))", x.Uses[0])
	case code.OpF32Trunc:
		return printf(w, "float32(math.Trunc(float64(%u)))", x.Uses[0])
	case code.OpF32Nearest:
		return printf(w, "float32(math.RoundToEven(float64(%u)))", x.Uses[0])
	case code.OpF32Sqrt:
		return printf(w, "float32(math.Sqrt(float64(%u)))", x.Uses[0])
	case code.OpF32Add:
		return printBinaryExpression(w, 4, parentPrecedence, "%.4u + %.4u", x.Uses[0], x.Uses[1])
	case code.OpF32Sub:
		return printBinaryExpression(w, 4, parentPrecedence, "%.4u - %.4u", x.Uses[0], x.Uses[1])
	case code.OpF32Mul:
		return printBinaryExpression(w, 5, parentPrecedence, "%.5u * %.5u", x.Uses[0], x.Uses[1])
	case code.OpF32Div:
		return printBinaryExpression(w, 5, parentPrecedence, "%.5u / %.5u", x.Uses[0], x.Uses[1])
	case code.OpF32Min:
		return printf(w, "float32(exec.Fmin(float64(%u), float64(%u)))", x.Uses[0], x.Uses[1])
	case code.OpF32Max:
		return printf(w, "float32(exec.Fmax(float64(%u), float64(%u)))", x.Uses[0], x.Uses[1])
	case code.OpF32Copysign:
		return printf(w, "float32(math.Copysign(float64(%u), float64(%u)))", x.Uses[0], x.Uses[1])

	case code.OpF64Abs:
		return printf(w, "math.Abs(%u)", x.Uses[0])
	case code.OpF64Neg:
		return printf(w, "-%u", x.Uses[0])
	case code.OpF64Ceil:
		return printf(w, "math.Ceil(%u)", x.Uses[0])
	case code.OpF64Floor:
		return printf(w, "math.Floor(%u)", x.Uses[0])
	case code.OpF64Trunc:
		return printf(w, "math.Trunc(%u)", x.Uses[0])
	case code.OpF64Nearest:
		return printf(w, "math.RoundToEven(%u)", x.Uses[0])
	case code.OpF64Sqrt:
		return printf(w, "math.Sqrt(%u)", x.Uses[0])
	case code.OpF64Add:
		return printBinaryExpression(w, 4, parentPrecedence, "%.4u + %.4u", x.Uses[0], x.Uses[1])
	case code.OpF64Sub:
		return printBinaryExpression(w, 4, parentPrecedence, "%.4u - %.4u", x.Uses[0], x.Uses[1])
	case code.OpF64Mul:
		return printBinaryExpression(w, 5, parentPrecedence, "%.5u * %.5u", x.Uses[0], x.Uses[1])
	case code.OpF64Div:
		return printBinaryExpression(w, 5, parentPrecedence, "%.5u / %.5u", x.Uses[0], x.Uses[1])
	case code.OpF64Min:
		return printf(w, "exec.Fmin(%u, %u)", x.Uses[0], x.Uses[1])
	case code.OpF64Max:
		return printf(w, "exec.Fmax(%u, %u)", x.Uses[0], x.Uses[1])
	case code.OpF64Copysign:
		return printf(w, "math.Copysign(%u, %u)", x.Uses[0], x.Uses[1])

	case code.OpI32WrapI64:
		return printf(w, "int32(%u)", x.Uses[0])
	case code.OpI32TruncF32S:
		return printf(w, "exec.I32TruncS(float64(%u))", x.Uses[0])
	case code.OpI32TruncF32U:
		return printf(w, "int32(exec.I32TruncU(float64(%u)))", x.Uses[0])
	case code.OpI32TruncF64S:
		return printf(w, "exec.I32TruncS(%u)", x.Uses[0])
	case code.OpI32TruncF64U:
		return printf(w, "int32(exec.I32TruncU(%u))", x.Uses[0])

	case code.OpI64ExtendI32S:
		return printf(w, "int64(%u)", x.Uses[0])
	case code.OpI64ExtendI32U:
		return printf(w, "int64(uint32(%4U))", x.Uses[0])
	case code.OpI64TruncF32S:
		return printf(w, "exec.I64TruncS(float64(%u))", x.Uses[0])
	case code.OpI64TruncF32U:
		return printf(w, "int64(exec.I64TruncU(float64(%u)))", x.Uses[0])
	case code.OpI64TruncF64S:
		return printf(w, "exec.I64TruncS(%u)", x.Uses[0])
	case code.OpI64TruncF64U:
		return printf(w, "int64(exec.I64TruncU(%u))", x.Uses[0])

	case code.OpF32ConvertI32S:
		return printf(w, "float32(%u)", x.Uses[0])
	case code.OpF32ConvertI32U:
		return printf(w, "float32(uint32(%4U))", x.Uses[0])
	case code.OpF32ConvertI64S:
		return printf(w, "float32(%u)", x.Uses[0])
	case code.OpF32ConvertI64U:
		return printf(w, "float32(uint64(%8U))", x.Uses[0])
	case code.OpF32DemoteF64:
		return printf(w, "float32(%u)", x.Uses[0])

	case code.OpF64ConvertI32S:
		return printf(w, "float64(%u)", x.Uses[0])
	case code.OpF64ConvertI32U:
		return printf(w, "float64(uint32(%4U))", x.Uses[0])
	case code.OpF64ConvertI64S:
		return printf(w, "float64(%u)", x.Uses[0])
	case code.OpF64ConvertI64U:
		return printf(w, "float64(uint64(%8U))", x.Uses[0])
	case code.OpF64PromoteF32:
		return printf(w, "float64(%u)", x.Uses[0])

	case code.OpI32ReinterpretF32:
		return printf(w, "int32(math.Float32bits(%u))", x.Uses[0])
	case code.OpI64ReinterpretF64:
		return printf(w, "int64(math.Float64bits(%u))", x.Uses[0])
	case code.OpF32ReinterpretI32:
		return printf(w, "math.Float32frombits(uint32(%4U))", x.Uses[0])
	case code.OpF64ReinterpretI64:
		return printf(w, "math.Float64frombits(uint64(%8U))", x.Uses[0])

	case code.OpI32Extend8S:
		return printf(w, "int32(int8(%1D))", x.Uses[0])
	case code.OpI32Extend16S:
		return printf(w, "int32(int16(%2D))", x.Uses[0])
	case code.OpI64Extend8S:
		return printf(w, "int64(int8(%1D))", x.Uses[0])
	case code.OpI64Extend16S:
		return printf(w, "int64(int16(%2D))", x.Uses[0])
	case code.OpI64Extend32S:
		return printf(w, "int64(int32(%4D))", x.Uses[0])

	case code.OpPrefix:
		switch x.Instr.Immediate {
		case code.OpI32TruncSatF32S:
			return printf(w, "exec.I32TruncSatS(float64(%u))", x.Uses[0])
		case code.OpI32TruncSatF32U:
			return printf(w, "int32(exec.I32TruncSatU(float64(%u)))", x.Uses[0])
		case code.OpI32TruncSatF64S:
			return printf(w, "exec.I32TruncSatS(%u)", x.Uses[0])
		case code.OpI32TruncSatF64U:
			return printf(w, "int32(exec.I32TruncSatU(%u))", x.Uses[0])
		case code.OpI64TruncSatF32S:
			return printf(w, "exec.I64TruncSatS(float64(%u))", x.Uses[0])
		case code.OpI64TruncSatF32U:
			return printf(w, "int64(exec.I64TruncSatU(float64(%u)))", x.Uses[0])
		case code.OpI64TruncSatF64S:
			return printf(w, "exec.I64TruncSatS(%u)", x.Uses[0])
		case code.OpI64TruncSatF64U:
			return printf(w, "int64(exec.I64TruncSatU(%u))", x.Uses[0])
		}
	}

	panic(fmt.Errorf("unexpected instruction %#v", x.Instr))
}

func (f *functionCompiler) emitPseudoOp(w io.Writer, x *wax.Expression, parentPrecedence int) error {
	switch x.Instr.Opcode {
	case wax.PseudoBoolConst:
		if wax.BoolConst(x) {
			return printf(w, "true")
		}
		return printf(w, "false")
	case wax.PseudoI32ConvertBool:
		return printf(w, "m.i32Bool(%u)", x.Uses[0])
	}

	panic(fmt.Errorf("unexpected pseudo instruction %#v", x.Instr))
}

func printBinaryExpression(w io.Writer, precedence, parentPrecedence int, fmtStr string, args ...interface{}) (err error) {
	if precedence <= parentPrecedence {
		if err := printf(w, "("); err != nil {
			return err
		}
		defer func() {
			if err == nil {
				err = printf(w, ")")
			}
		}()
	}
	return printf(w, fmtStr, args...)
}

func f32Const(v float32) string {
	v64 := float64(v)
	switch {
	case math.IsInf(v64, 0) || math.IsNaN(v64) || v >= math.MaxFloat32 || v <= -math.MaxFloat32 || math.Signbit(v64) && v == 0:
		return fmt.Sprintf("math.Float32frombits(0x%x)", math.Float32bits(v))
	default:
		return strconv.FormatFloat(v64, 'g', -1, 32)
	}
}

func f64Const(v float64) string {
	switch {
	case math.IsInf(v, 0) || math.IsNaN(v) || v >= math.MaxFloat64 || v <= -math.MaxFloat64 || math.Signbit(v) && v == 0:
		return fmt.Sprintf("math.Float64frombits(0x%x)", math.Float64bits(v))
	default:
		return strconv.FormatFloat(v, 'g', -1, 64)
	}
}

func isConst(u *wax.Use) bool {
	if u.IsTemp() {
		return false
	}
	switch u.X.Instr.Opcode {
	case code.OpI32Const, code.OpI64Const, code.OpF32Const, code.OpF64Const:
		return true
	default:
		return false
	}
}

func isConst0(u *wax.Use) bool {
	if u.IsTemp() {
		return false
	}
	switch u.X.Instr.Opcode {
	case code.OpI32Const:
		return u.X.Instr.I32() == 0
	case code.OpI64Const:
		return u.X.Instr.I64() == 0
	case code.OpF32Const:
		return u.X.Instr.F32() == 0
	case code.OpF64Const:
		return u.X.Instr.F64() == 0
	default:
		return false
	}
}

func isUntyped(f wax.Flags) bool {
	return f&flagsUntyped != 0
}

func markUntypedExpressions(x *wax.Expression) {
	for _, u := range x.Uses {
		if !u.IsTemp() {
			markUntypedExpressions(u.X)

			if isUntyped(u.X.Flags) {
				u.AllFlags |= flagsUntyped
			}
		}
	}

	switch x.Instr.Opcode {
	case code.OpI32Const, code.OpI64Const, code.OpF32Const, code.OpF64Const:
		x.Flags |= flagsUntyped

	case code.OpI32Shl, code.OpI32ShrS, code.OpI64Shl, code.OpI64ShrS:
		// This rule is a little weird. From the spec:
		//
		//     The right operand in a shift expression must have integer type or be an untyped constant representable by a
		//     value of type uint. If the left operand of a non-constant shift expression is an untyped constant, it is
		//     first implicitly converted to the type it would assume if the shift expression were replaced by its left
		//     operand alone.
		//
		// Thus, we mark the entire shift operand as untyped if its left operand is untyped. This will force a
		// conversion during codegen.
		if isUntyped(x.Uses[0].AllFlags) {
			x.Flags |= flagsUntyped
		}

	case code.OpI32Add, code.OpI64Add, code.OpI32Sub, code.OpI64Sub,
		code.OpI32Mul, code.OpI64Mul, code.OpI32RemS, code.OpI64RemS,
		code.OpI32And, code.OpI64And, code.OpI32Or, code.OpI64Or, code.OpI32Xor, code.OpI64Xor,
		code.OpF32Add, code.OpF64Add, code.OpF32Sub, code.OpF64Sub,
		code.OpF32Mul, code.OpF64Mul, code.OpF32Div, code.OpF64Div:
		if isUntyped(x.Uses[0].AllFlags) && isUntyped(x.Uses[1].AllFlags) {
			x.Flags |= flagsUntyped
		}

	case code.OpF32Neg, code.OpF64Neg:
		if isUntyped(x.Uses[0].AllFlags) {
			x.Flags |= flagsUntyped
		}
	}
}
