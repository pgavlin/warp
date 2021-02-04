package go_wasm_exec

func NewGlobal(fs Value) Object {
	return NewObject(ObjectClass, map[string]Value{
		"fs":         fs,
		"Object":     ValueOf(ObjectClass),
		"Array":      ValueOf(ArrayClass),
		"Function":   ValueOf(FunctionClass),
		"Uint8Array": ValueOf(Uint8ArrayClass),
	})
}
