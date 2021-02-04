package code

import "github.com/pgavlin/warp/wasm"

type Scope interface {
	GetLocalType(localidx uint32) (wasm.ValueType, bool)
	GetGlobalType(globalidx uint32) (wasm.GlobalVar, bool)
	GetFunctionSignature(funcidx uint32) (wasm.FunctionSig, bool)
	GetType(typeidx uint32) (wasm.FunctionSig, bool)

	HasTable(tableidx uint32) bool
	HasMemory(memoryidx uint32) bool
}

var UnknownTypes = []wasm.ValueType{}

var UnknownScope = unknownScope(0)

type unknownScope int

func (unknownScope) GetLocalType(localidx uint32) (wasm.ValueType, bool) {
	return wasm.ValueTypeT, true
}

func (unknownScope) GetGlobalType(globalidx uint32) (wasm.GlobalVar, bool) {
	return wasm.GlobalVar{Type: wasm.ValueTypeT}, true
}

func (unknownScope) GetFunctionSignature(funcidx uint32) (wasm.FunctionSig, bool) {
	return wasm.FunctionSig{ParamTypes: UnknownTypes, ReturnTypes: UnknownTypes}, true
}

func (unknownScope) GetType(typeidx uint32) (wasm.FunctionSig, bool) {
	return wasm.FunctionSig{ParamTypes: UnknownTypes, ReturnTypes: UnknownTypes}, true
}

func (unknownScope) HasTable(tableidx uint32) bool {
	return true
}

func (unknownScope) HasMemory(memoryidx uint32) bool {
	return true
}
