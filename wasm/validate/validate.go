package validate

import (
	"github.com/pgavlin/warp/wasm"
	"github.com/pgavlin/warp/wasm/code"
)

type validator struct {
	module       *wasm.Module
	validateCode bool

	importedFunctions []uint32
	importedGlobals   []wasm.GlobalVar

	tables   int
	memories int

	locals []wasm.ValueType
}

func ValidateModule(m *wasm.Module, validateCode bool) error {
	v := validator{
		module:       m,
		validateCode: validateCode,
	}

	if v.module.Import != nil {
		for _, i := range v.module.Import.Entries {
			switch i := i.Type.(type) {
			case wasm.FuncImport:
				v.importedFunctions = append(v.importedFunctions, i.Type)
			case wasm.TableImport:
				v.tables++
			case wasm.MemoryImport:
				v.memories++
			case wasm.GlobalVarImport:
				v.importedGlobals = append(v.importedGlobals, i.Type)
			}
		}
	}
	if v.module.Table != nil {
		v.tables += len(v.module.Table.Entries)
	}
	if v.module.Memory != nil {
		v.memories += len(v.module.Memory.Entries)
	}

	return v.validateModule()
}

func (v *validator) validateModule() error {
	if err := v.validateTypes(); err != nil {
		return err
	}
	if err := v.validateFunctions(); err != nil {
		return err
	}
	if err := v.validateTables(); err != nil {
		return err
	}
	if err := v.validateMemories(); err != nil {
		return err
	}
	if err := v.validateGlobals(); err != nil {
		return err
	}
	if err := v.validateElements(); err != nil {
		return err
	}
	if err := v.validateData(); err != nil {
		return err
	}
	if err := v.validateStart(); err != nil {
		return err
	}
	if err := v.validateImports(); err != nil {
		return err
	}
	if err := v.validateExports(); err != nil {
		return err
	}
	return nil
}

func (v *validator) validateTypes() error {
	// no-op: all types are valid by definition
	return nil
}

func (v *validator) validateFunctions() error {
	var types []uint32
	if v.module.Function != nil {
		types = v.module.Function.Types
	}

	var bodies []wasm.FunctionBody
	if v.module.Code != nil {
		bodies = v.module.Code.Bodies
	}

	if len(types) != len(bodies) {
		return wasm.ValidationError("function and code section have inconsistent lengths")
	}

	for i, typeidx := range types {
		sig, ok := v.GetType(typeidx)
		if !ok {
			return wasm.ValidationError("unknown type")
		}

		if !v.validateCode {
			continue
		}

		body := bodies[i]

		v.SetFunction(sig, body)
		_, err := code.Decode(body.Code, v, sig.ReturnTypes)
		if err != nil {
			return err
		}
	}

	return nil
}

func (v *validator) validateLimits(limits wasm.ResizableLimits) error {
	if limits.Flags != 0 && limits.Initial > limits.Maximum {
		return wasm.ValidationError("size minimum must not be greater than maximum")
	}
	return nil
}

func (v *validator) validateTables() error {
	if v.module.Table == nil || len(v.module.Table.Entries) == 0 {
		return nil
	}
	if v.tables > 1 {
		return wasm.ValidationError("multiple tables")
	}
	return v.validateLimits(v.module.Table.Entries[0].Limits)
}

func (v *validator) validateMemories() error {
	if v.module.Memory == nil || len(v.module.Memory.Entries) == 0 {
		return nil
	}
	if v.memories > 1 {
		return wasm.ValidationError("multiple memories")
	}

	limits := v.module.Memory.Entries[0].Limits
	if err := v.validateLimits(limits); err != nil {
		return err
	}
	if limits.Initial > 65536 || limits.Flags != 0 && limits.Maximum > 65536 {
		return wasm.ValidationError("memory size must be at most 65536 pages (4GiB)")
	}
	return nil
}

func (v *validator) validateGlobals() error {
	if v.module.Global == nil {
		return nil
	}

	scope := v.globalScope()
	for _, g := range v.module.Global.Globals {
		if err := v.validateInitExpr(g.Init, g.Type.Type, scope); err != nil {
			return err
		}
	}

	return nil
}

func (v *validator) validateElements() error {
	if v.module.Elements == nil {
		return nil
	}
	for _, elem := range v.module.Elements.Entries {
		if elem.Index >= uint32(v.tables) {
			return wasm.ValidationError("unknown table")
		}
		if err := v.validateInitExpr(elem.Offset, wasm.ValueTypeI32, v); err != nil {
			return err
		}
		for _, funcidx := range elem.Elems {
			if _, ok := v.GetFunctionSignature(funcidx); !ok {
				return wasm.ValidationError("unknown function")
			}
		}
	}
	return nil
}

func (v *validator) validateData() error {
	if v.module.Data == nil {
		return nil
	}
	for _, data := range v.module.Data.Entries {
		if data.Index >= uint32(v.memories) {
			return wasm.ValidationError("unknown memory")
		}
		if err := v.validateInitExpr(data.Offset, wasm.ValueTypeI32, v); err != nil {
			return err
		}
	}
	return nil
}

func (v *validator) validateStart() error {
	if v.module.Start == nil {
		return nil
	}
	sig, ok := v.GetFunctionSignature(v.module.Start.Index)
	if !ok {
		return wasm.ValidationError("unknown function")
	}
	if len(sig.ParamTypes) != 0 || len(sig.ReturnTypes) != 0 {
		return wasm.ValidationError("start function")
	}
	return nil
}

func (v *validator) validateImports() error {
	if v.module.Import == nil {
		return nil
	}
	for _, i := range v.module.Import.Entries {
		switch i := i.Type.(type) {
		case wasm.FuncImport:
			if _, ok := v.GetFunctionSignature(i.Type); !ok {
				return wasm.ValidationError("unknown type")
			}
		case wasm.TableImport:
			if err := v.validateLimits(i.Type.Limits); err != nil {
				return err
			}
		case wasm.MemoryImport:
			if err := v.validateLimits(i.Type.Limits); err != nil {
				return err
			}
		case wasm.GlobalVarImport:
			// OK
		}
	}
	return nil
}

func (v *validator) validateExports() error {
	if v.module.Export == nil {
		return nil
	}

	names := map[string]bool{}
	for _, e := range v.module.Export.Entries {
		if names[e.FieldStr] {
			return wasm.ValidationError("duplicate export name")
		}
		names[e.FieldStr] = true

		switch e.Kind {
		case wasm.ExternalFunction:
			if _, ok := v.GetFunctionSignature(e.Index); !ok {
				return wasm.ValidationError("unknown function")
			}
		case wasm.ExternalTable:
			if e.Index >= uint32(v.tables) {
				return wasm.ValidationError("unknown table")
			}
		case wasm.ExternalMemory:
			if e.Index >= uint32(v.memories) {
				return wasm.ValidationError("unknown memory")
			}
		case wasm.ExternalGlobal:
			if _, ok := v.GetGlobalType(e.Index); !ok {
				return wasm.ValidationError("unknown global")
			}
		}
	}
	return nil
}

func (v *validator) validateInitExpr(expr []byte, expected wasm.ValueType, scope code.Scope) error {
	decoded, err := code.Decode(expr, scope, []wasm.ValueType{expected})
	if err != nil {
		return err
	}
	for _, instr := range decoded.Instructions {
		switch instr.Opcode {
		case code.OpI32Const, code.OpI64Const, code.OpF32Const, code.OpF64Const, code.OpEnd:
			// OK
		case code.OpGlobalGet:
			if v.importedGlobals[int(instr.Globalidx())].Mutable {
				return wasm.ValidationError("constant expression required")
			}
		default:
			return wasm.ValidationError("constant expression required")
		}
	}
	return nil
}
