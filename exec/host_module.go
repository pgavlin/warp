package exec

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"unicode"
	"unicode/utf8"

	"github.com/pgavlin/warp/wasm"
)

type hostModuleDefinition struct {
	instantiate reflect.Value
}

// instantiate must be a func() T. instantiate will be called to instantiate a host module.
func NewHostModuleDefinition(instantiate interface{}) ModuleDefinition {
	f := reflect.ValueOf(instantiate)

	type_ := f.Type()
	if type_.Kind() != reflect.Func || type_.NumIn() != 0 || type_.NumOut() != 2 || !type_.Out(1).ConvertibleTo(reflect.TypeOf((*error)(nil)).Elem()) {
		panic(errors.New("instantiate must be a func() (T, error)"))
	}

	return hostModuleDefinition{instantiate: f}
}

func (def hostModuleDefinition) Allocate(name string) (AllocatedModule, error) {
	v := def.instantiate.Call(nil)
	if err, ok := v[1].Interface().(error); ok && err != nil {
		return nil, err
	}
	return newHostModule(name, v[0]), nil
}

type HostFunction struct {
	module Module
	index  uint32
	sig    wasm.FunctionSig

	method reflect.Value
}

func NewHostFunction(module Module, index uint32, method reflect.Value) *HostFunction {
	t := method.Type()

	params := make([]wasm.ValueType, t.NumIn())
	for i, n := 0, t.NumIn(); i < n; i++ {
		vt := wasmType(t.In(i).Kind())
		if vt == 0 {
			panic(fmt.Errorf("cannot export method with parameter type %v", t.In(i)))
		}
		params[i] = vt
	}

	returns := make([]wasm.ValueType, t.NumOut())
	for i, n := 0, t.NumOut(); i < n; i++ {
		vt := wasmType(t.Out(i).Kind())
		if vt == 0 {
			panic(fmt.Errorf("cannot export method with return type %v", t.Out(i)))
		}
		returns[i] = vt
	}

	return &HostFunction{
		module: module,
		index:  index,
		sig: wasm.FunctionSig{
			Form:        0x60,
			ParamTypes:  params,
			ReturnTypes: returns,
		},
		method: method,
	}
}

func (f *HostFunction) GetSignature() wasm.FunctionSig {
	return f.sig
}

func (f *HostFunction) Call(thread *Thread, args ...interface{}) []interface{} {
	vargs := make([]reflect.Value, len(args))
	for i, v := range args {
		vargs[i] = reflect.ValueOf(v)
	}

	vreturns := f.method.Call(vargs)

	returns := make([]interface{}, len(vreturns))
	for i, v := range vreturns {
		returns[i] = v.Interface()
	}

	return returns
}

func (f *HostFunction) UncheckedCall(thread *Thread, args, returns []uint64) {
	if len(args) != len(f.sig.ParamTypes) {
		panic(fmt.Errorf("expected %v args; got %v", len(f.sig.ParamTypes), len(args)))
	}

	t := f.method.Type()

	vargs := make([]reflect.Value, len(args))
	for i, v := range args {
		t := t.In(i)

		var av reflect.Value
		switch f.sig.ParamTypes[i] {
		case wasm.ValueTypeI32, wasm.ValueTypeI64:
			switch t.Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				av = reflect.ValueOf(args[i]).Convert(t)
			default:
				panic("invalid argument type")
			}
		case wasm.ValueTypeF32:
			av = reflect.ValueOf(math.Float32frombits(uint32(v))).Convert(t)
		case wasm.ValueTypeF64:
			av = reflect.ValueOf(math.Float64frombits(v)).Convert(t)
		default:
			panic("unreachable")
		}
		vargs[i] = av
	}

	vreturns := f.method.Call(vargs)

	for i, v := range vreturns {
		switch f.sig.ReturnTypes[i] {
		case wasm.ValueTypeI32:
			if v.Kind() == reflect.Uint32 {
				returns[i] = v.Uint()
			} else {
				returns[i] = uint64(int32(v.Int()))
			}
		case wasm.ValueTypeI64:
			if v.Kind() == reflect.Uint64 {
				returns[i] = v.Uint()
			} else {
				returns[i] = uint64(v.Int())
			}
		case wasm.ValueTypeF32:
			returns[i] = uint64(math.Float32bits(float32(v.Float())))
		case wasm.ValueTypeF64:
			returns[i] = math.Float64bits(v.Float())
		default:
			panic("unreachable")
		}
	}
}

func (f *HostFunction) Func() interface{} {
	return f.method.Interface()
}

type hostModule struct {
	value reflect.Value
	name  string

	exports map[string]interface{}
}

func NewHostModule(name string, v interface{}) Module {
	return newHostModule(name, reflect.ValueOf(v))
}

var tableType = reflect.TypeOf((*Table)(nil)).Elem()
var memoryType = reflect.TypeOf((*Memory)(nil)).Elem()
var globalType = reflect.TypeOf((*Global)(nil)).Elem()

func isExported(n string) bool {
	r, _ := utf8.DecodeRuneInString(n)
	return unicode.IsUpper(r)
}

func exportName(n string) string {
	runes := []rune(n)
	runes[0] = unicode.ToLower(runes[0])
	return string(runes)
}

func newHostModule(name string, v reflect.Value) *hostModule {
	m := hostModule{
		value:   v,
		name:    name,
		exports: map[string]interface{}{},
	}

	value := v
	for value.Kind() == reflect.Ptr {
		value = value.Elem()
	}

	t := value.Type()
	for i, n := 0, t.NumField(); i < n; i++ {
		f := t.Field(i)
		if !isExported(f.Name) {
			continue
		}

		var fv interface{}
		switch f.Type {
		case tableType:
			fv = value.Field(i).Addr().Interface().(*Table)
		case memoryType:
			fv = value.Field(i).Addr().Interface().(*Memory)
		case globalType:
			fv = value.Field(i).Addr().Interface().(*Global)
		default:
			continue
		}
		m.exports[exportName(f.Name)] = fv
	}

	t = v.Type()

	index := uint32(0)
	for i, n := 0, t.NumMethod(); i < n; i++ {
		if name := t.Method(i).Name; isExported(name) {
			m.exports[exportName(name)] = NewHostFunction(&m, index, v.Method(i))
			index++
		}
	}

	return &m
}

func (m *hostModule) Instantiate(imports ImportResolver) (Module, error) {
	return m, nil
}

func (m *hostModule) Name() string {
	return m.name
}

func (m *hostModule) GetFunction(name string) (Function, error) {
	if f, ok := m.exports[name].(Function); ok {
		return f, nil
	}
	return nil, errors.New("unknown function")
}

func (m *hostModule) GetTable(name string) (*Table, error) {
	if f, ok := m.exports[name].(*Table); ok {
		return f, nil
	}
	return nil, errors.New("unknown table")
}

func (m *hostModule) GetMemory(name string) (*Memory, error) {
	if f, ok := m.exports[name].(*Memory); ok {
		return f, nil
	}
	return nil, errors.New("unknown memory")
}

func (m *hostModule) GetGlobal(name string) (*Global, error) {
	if f, ok := m.exports[name].(*Global); ok {
		return f, nil
	}
	return nil, errors.New("unknown global")
}
