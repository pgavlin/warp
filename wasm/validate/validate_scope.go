package validate

import (
	"github.com/pgavlin/warp/wasm"
	"github.com/pgavlin/warp/wasm/code"
)

func (v *validator) GetLocalType(localidx uint32) (wasm.ValueType, bool) {
	if localidx >= uint32(len(v.locals)) {
		return 0, false
	}
	return v.locals[int(localidx)], true
}

func (v *validator) GetGlobalType(globalidx uint32) (wasm.GlobalVar, bool) {
	if globalidx < uint32(len(v.importedGlobals)) {
		return v.importedGlobals[int(globalidx)], true
	}
	globalidx -= uint32(len(v.importedGlobals))
	if v.module.Global == nil || globalidx >= uint32(len(v.module.Global.Globals)) {
		return wasm.GlobalVar{}, false
	}
	return v.module.Global.Globals[int(globalidx)].Type, true
}

func (v *validator) GetFunctionSignature(funcidx uint32) (wasm.FunctionSig, bool) {
	if funcidx < uint32(len(v.importedFunctions)) {
		return v.GetType(v.importedFunctions[int(funcidx)])
	}
	funcidx -= uint32(len(v.importedFunctions))
	if v.module.Function == nil || funcidx >= uint32(len(v.module.Function.Types)) {
		return wasm.FunctionSig{}, false
	}
	return v.GetType(v.module.Function.Types[int(funcidx)])
}

func (v *validator) GetType(typeidx uint32) (wasm.FunctionSig, bool) {
	if v.module.Types == nil || typeidx >= uint32(len(v.module.Types.Entries)) {
		return wasm.FunctionSig{}, false
	}
	return v.module.Types.Entries[int(typeidx)], true
}

func (v *validator) SetFunction(sig wasm.FunctionSig, body wasm.FunctionBody) {
	v.locals = v.locals[:0]

	v.locals = append(v.locals, sig.ParamTypes...)
	for _, l := range body.Locals {
		for i := uint32(0); i < l.Count; i++ {
			v.locals = append(v.locals, l.Type)
		}
	}
}

func (v *validator) HasTable(tableidx uint32) bool {
	return tableidx < uint32(v.tables)
}

func (v *validator) HasMemory(memoryidx uint32) bool {
	return memoryidx < uint32(v.memories)
}

func (v *validator) globalScope() code.Scope {
	return globalScope{importedGlobals: v.importedGlobals}
}

type globalScope struct {
	importedGlobals []wasm.GlobalVar
}

func (s globalScope) GetLocalType(localidx uint32) (wasm.ValueType, bool) {
	return 0, false
}

func (s globalScope) GetGlobalType(globalidx uint32) (wasm.GlobalVar, bool) {
	if globalidx < uint32(len(s.importedGlobals)) {
		return s.importedGlobals[int(globalidx)], true
	}
	return wasm.GlobalVar{}, false
}

func (s globalScope) GetFunctionSignature(funcidx uint32) (wasm.FunctionSig, bool) {
	return wasm.FunctionSig{}, false
}

func (s globalScope) GetType(typeidx uint32) (wasm.FunctionSig, bool) {
	return wasm.FunctionSig{}, false
}

func (s globalScope) HasTable(tableidx uint32) bool {
	return false
}

func (s globalScope) HasMemory(memoryidx uint32) bool {
	return false
}
