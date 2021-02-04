package code

import "github.com/pgavlin/warp/wasm"

type StaticScope struct {
	module *wasm.Module

	ImportedFunctions []uint32
	ImportedGlobals   []wasm.GlobalVar

	Tables   int
	Memories int

	Locals []wasm.ValueType
}

func NewStaticScope(m *wasm.Module) *StaticScope {
	s := StaticScope{module: m}

	if m.Import != nil {
		for _, i := range m.Import.Entries {
			switch i := i.Type.(type) {
			case wasm.FuncImport:
				s.ImportedFunctions = append(s.ImportedFunctions, i.Type)
			case wasm.TableImport:
				s.Tables++
			case wasm.MemoryImport:
				s.Memories++
			case wasm.GlobalVarImport:
				s.ImportedGlobals = append(s.ImportedGlobals, i.Type)
			}
		}
	}
	if m.Table != nil {
		s.Tables += len(m.Table.Entries)
	}
	if m.Memory != nil {
		s.Memories += len(m.Memory.Entries)
	}

	return &s
}

func (s *StaticScope) GetLocalType(localidx uint32) (wasm.ValueType, bool) {
	if localidx >= uint32(len(s.Locals)) {
		return 0, false
	}
	return s.Locals[int(localidx)], true
}

func (s *StaticScope) GetGlobalType(globalidx uint32) (wasm.GlobalVar, bool) {
	if globalidx < uint32(len(s.ImportedGlobals)) {
		return s.ImportedGlobals[int(globalidx)], true
	}
	globalidx -= uint32(len(s.ImportedGlobals))
	if s.module.Global == nil || globalidx >= uint32(len(s.module.Global.Globals)) {
		return wasm.GlobalVar{}, false
	}
	return s.module.Global.Globals[int(globalidx)].Type, true
}

func (s *StaticScope) GetFunctionSignature(funcidx uint32) (wasm.FunctionSig, bool) {
	if funcidx < uint32(len(s.ImportedFunctions)) {
		return s.GetType(s.ImportedFunctions[int(funcidx)])
	}
	funcidx -= uint32(len(s.ImportedFunctions))
	if s.module.Function == nil || funcidx >= uint32(len(s.module.Function.Types)) {
		return wasm.FunctionSig{}, false
	}
	return s.GetType(s.module.Function.Types[int(funcidx)])
}

func (s *StaticScope) GetType(typeidx uint32) (wasm.FunctionSig, bool) {
	if s.module.Types == nil || typeidx >= uint32(len(s.module.Types.Entries)) {
		return wasm.FunctionSig{}, false
	}
	return s.module.Types.Entries[int(typeidx)], true
}

func (s *StaticScope) SetFunction(sig wasm.FunctionSig, body wasm.FunctionBody) {
	s.Locals = s.Locals[:0]

	s.Locals = append(s.Locals, sig.ParamTypes...)
	for _, l := range body.Locals {
		for i := uint32(0); i < l.Count; i++ {
			s.Locals = append(s.Locals, l.Type)
		}
	}
}

func (s *StaticScope) HasTable(tableidx uint32) bool {
	return tableidx < uint32(s.Tables)
}

func (s *StaticScope) HasMemory(memoryidx uint32) bool {
	return memoryidx < uint32(s.Memories)
}
