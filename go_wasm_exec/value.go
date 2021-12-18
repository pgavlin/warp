package go_wasm_exec

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"unsafe"
)

type Value struct {
	v reflect.Value
}

// Equal reports whether v and w are equal according to JavaScript's === operator.
func (v Value) Equal(w Value) bool {
	return v.v == w.v
}

// Undefined returns the JavaScript value "undefined".
func Undefined() Value {
	return Value{}
}

// IsUndefined reports whether v is the JavaScript value "undefined".
func (v Value) IsUndefined() bool {
	return !v.v.IsValid()
}

// Null returns the JavaScript value "null".
func Null() Value {
	return Value{v: reflect.ValueOf((*byte)(nil))}
}

// IsNull reports whether v is the JavaScript value "null".
func (v Value) IsNull() bool {
	return v.v.Kind() == reflect.Ptr && v.v.IsNil()
}

// IsNaN reports whether v is the JavaScript value "NaN".
func (v Value) IsNaN() bool {
	return v.v.Kind() == reflect.Float64 && math.IsNaN(v.v.Float())
}

func (v Value) Object() (Object, bool) {
	if !v.v.IsValid() || !v.v.CanInterface() {
		return nil, false
	}
	o, ok := v.v.Interface().(Object)
	return o, ok
}

func (v Value) Array() ([]Value, bool) {
	o, ok := v.Object()
	if !ok {
		return nil, false
	}
	if arr, ok := o.(arrayObject); ok {
		return arr.v, true
	}
	return nil, false
}

func (v Value) Uint8Array() ([]byte, bool) {
	o, ok := v.Object()
	if !ok {
		return nil, false
	}
	if arr, ok := o.(uint8ArrayObject); ok {
		return arr.v, true
	}
	return nil, false
}

func (v Value) Function() (Function, bool) {
	if !v.v.IsValid() || !v.v.CanInterface() {
		return nil, false
	}
	f, ok := v.v.Interface().(Function)
	return f, ok
}

// Float returns the value v as a float64.
// It panics if v is not a JavaScript number.
func (v Value) Float() float64 {
	return v.v.Float()
}

// Int returns the value v truncated to an int.
// It panics if v is not a JavaScript number.
func (v Value) Int() int {
	return int(v.Float())
}

// Bool returns the value v as a bool.
// It panics if v is not a JavaScript boolean.
func (v Value) Bool() bool {
	return v.v.Bool()
}

// Truthy returns the JavaScript "truthiness" of the value v. In JavaScript,
// false, 0, "", null, undefined, and NaN are "falsy", and everything else is
// "truthy". See https://developer.mozilla.org/en-US/docs/Glossary/Truthy.
func (v Value) Truthy() bool {
	switch v.Type() {
	case TypeUndefined, TypeNull:
		return false
	case TypeBoolean:
		return v.Bool()
	case TypeNumber:
		return !v.IsNaN() && v.Float() != 0
	case TypeString:
		return v.String() != ""
	case TypeSymbol, TypeFunction, TypeObject:
		return true
	default:
		panic("bad type")
	}
}

// String returns the value v as a string.
// String is a special case because of Go's String method convention. Unlike the other getters,
// it does not panic if v's Type is not TypeString. Instead, it returns a string of the form "<T>"
// or "<T: V>" where T is v's type and V is a string representation of v's value.
func (v Value) String() string {
	switch v.Type() {
	case TypeString:
		return v.v.Interface().(stringObject).v
	case TypeUndefined:
		return "<undefined>"
	case TypeNull:
		return "<null>"
	case TypeBoolean:
		return fmt.Sprintf("<boolean: %v>", v.Bool())
	case TypeNumber:
		return fmt.Sprintf("<number: %v>", v.Float())
	case TypeSymbol:
		return "<symbol>"
	case TypeObject:
		return "<object>"
	case TypeFunction:
		return "<function>"
	default:
		panic("bad type")
	}
}

func (v Value) Get(property string) Value {
	if o, ok := v.Object(); ok {
		return o.Get(property)
	}
	return Undefined()
}

func (v Value) Set(property string, value Value) {
	if o, ok := v.Object(); ok {
		o.Set(property, value)
	}
}

func (v Value) Delete(property string) {
	if o, ok := v.Object(); ok {
		o.Delete(property)
	}
}

func (v Value) Index(i int) Value {
	if o, ok := v.Object(); ok {
		return o.Index(i)
	}
	return Undefined()
}

func (v Value) SetIndex(i int, value Value) {
	if o, ok := v.Object(); ok {
		o.SetIndex(i, value)
	}
}

func (v Value) Call(method string, args []Value) (Value, error) {
	if o, ok := v.Object(); ok {
		return o.Call(method, args)
	}
	return v.Get(method).Invoke(args)
}

func (v Value) Invoke(args []Value) (Value, error) {
	if f, ok := v.Function(); ok {
		return f.Invoke(args)
	}
	return Undefined(), fmt.Errorf("%v is not a function", v.Type())
}

func (v Value) New(args []Value) (Value, error) {
	if f, ok := v.Function(); ok {
		return f.Invoke(args)
	}
	return Undefined(), fmt.Errorf("%v is not a function", v.Type())
}

func (v Value) Length() int {
	if o, ok := v.Object(); ok {
		return o.Length()
	}
	return 0
}

func (v Value) IsInstance(class Value) bool {
	if c, ok := class.Function(); ok {
		if o, ok := v.Object(); ok {
			return o.Class() == c
		}
	}
	return false
}

// ValueOf returns x as a JavaScript value:
//
//  | Go                     | JavaScript             |
//  | ---------------------- | ---------------------- |
//  | js.Value               | [its value]            |
//  | js.Func                | function               |
//  | nil                    | null                   |
//  | bool                   | boolean                |
//  | integers and floats    | number                 |
//  | string                 | string                 |
//  | []interface{}          | new array              |
//  | map[string]interface{} | new object             |
//
// Panics if x is not one of the expected types.
func ValueOf(x interface{}) Value {
	switch x := x.(type) {
	case Value: // should precede Wrapper to avoid a loop
		return x
	case Object, Function:
		return Value{reflect.ValueOf(x)}
	case nil:
		return Null()
	case bool:
		return Value{reflect.ValueOf(x)}
	case int:
		return floatValue(float64(x))
	case int8:
		return floatValue(float64(x))
	case int16:
		return floatValue(float64(x))
	case int32:
		return floatValue(float64(x))
	case int64:
		return floatValue(float64(x))
	case uint:
		return floatValue(float64(x))
	case uint8:
		return floatValue(float64(x))
	case uint16:
		return floatValue(float64(x))
	case uint32:
		return floatValue(float64(x))
	case uint64:
		return floatValue(float64(x))
	case uintptr:
		return floatValue(float64(x))
	case unsafe.Pointer:
		return floatValue(float64(uintptr(x)))
	case float32:
		return floatValue(float64(x))
	case float64:
		return floatValue(x)
	case string:
		return stringValue(x)
	case []Value:
		return arrayValue(x)
	case map[string]Value:
		return objectValue(x)
	case func([]Value) (Value, error):
		return functionValue(x)
	case []byte:
		return uint8ArrayValue(x)
	case error:
		return errorValue(x)
	default:
		panic("ValueOf: invalid value")
	}
}

func floatValue(x float64) Value {
	return Value{reflect.ValueOf(x)}
}

// Type represents the JavaScript type of a Value.
type Type int

const (
	TypeUndefined Type = iota
	TypeNull
	TypeBoolean
	TypeNumber
	TypeString
	TypeSymbol
	TypeObject
	TypeFunction
)

func (t Type) String() string {
	switch t {
	case TypeUndefined:
		return "undefined"
	case TypeNull:
		return "null"
	case TypeBoolean:
		return "boolean"
	case TypeNumber:
		return "number"
	case TypeString:
		return "string"
	case TypeSymbol:
		return "symbol"
	case TypeObject:
		return "object"
	case TypeFunction:
		return "function"
	default:
		panic("bad type")
	}
}

// Type returns the JavaScript type of the value v. It is similar to JavaScript's typeof operator,
// except that it returns TypeNull instead of TypeObject for null.
func (v Value) Type() Type {
	if !v.v.IsValid() {
		return TypeUndefined
	}

	if v.v.CanInterface() {
		i := v.v.Interface()
		switch i.(type) {
		case Function:
			return TypeFunction
		case Object:
			if _, ok := i.(stringObject); ok {
				return TypeString
			}
			return TypeObject
		}
	}

	switch v.v.Kind() {
	case reflect.Ptr:
		return TypeNull
	case reflect.Bool:
		return TypeBoolean
	case reflect.Float64:
		return TypeNumber
	default:
		return TypeUndefined
	}

}

type Object interface {
	Class() Function
	Get(property string) Value
	Set(property string, value Value)
	Delete(property string)
	Index(i int) Value
	SetIndex(i int, value Value)
	Call(method string, args []Value) (Value, error)
	Length() int
}

type Function interface {
	Object

	Invoke(args []Value) (Value, error)
	New(args []Value) (Value, error)
}

type stringObject struct {
	v string
}

func stringValue(s string) Value {
	return Value{reflect.ValueOf(stringObject{s})}
}

func (o stringObject) Class() Function {
	return StringClass
}

func (o stringObject) Get(property string) Value {
	i, err := strconv.ParseInt(property, 10, 0)
	if err == nil && i >= 0 && i < int64(len(o.v)) {
		return ValueOf(o.v[i])
	}
	return Undefined()
}

func (o stringObject) Set(property string, value Value) {
}

func (o stringObject) Delete(property string) {
}

func (o stringObject) Index(i int) Value {
	if i >= 0 && i < len(o.v) {
		return ValueOf(o.v[i])
	}
	return Undefined()
}

func (o stringObject) SetIndex(i int, value Value) {
}

func (o stringObject) Call(method string, args []Value) (Value, error) {
	return o.Get(method).Invoke(args)
}

func (o stringObject) Length() int {
	return len(o.v)
}

type arrayObject struct {
	v []Value
}

func arrayValue(x []Value) Value {
	return Value{reflect.ValueOf(arrayObject{x})}
}

func (o arrayObject) Class() Function {
	return ArrayClass
}

func (o arrayObject) Get(property string) Value {
	i, err := strconv.ParseInt(property, 10, 0)
	if err == nil && i >= 0 && i < int64(len(o.v)) {
		return ValueOf(o.v[i])
	}
	return Undefined()
}

func (o arrayObject) Set(property string, value Value) {
	i, err := strconv.ParseInt(property, 10, 0)
	if err == nil && i >= 0 && i < int64(len(o.v)) {
		o.v[i] = value
	}
}

func (o arrayObject) Delete(property string) {
	o.Set(property, Undefined())
}

func (o arrayObject) Index(i int) Value {
	if i >= 0 && i < len(o.v) {
		return ValueOf(o.v[i])
	}
	return Undefined()
}

func (o arrayObject) SetIndex(i int, value Value) {
	if i >= 0 && i < len(o.v) {
		o.v[i] = value
	}
}

func (o arrayObject) Call(method string, args []Value) (Value, error) {
	return o.Get(method).Invoke(args)
}

func (o arrayObject) Length() int {
	return len(o.v)
}

type mapObject struct {
	c Function
	v map[string]Value
}

func NewObject(class Function, properties map[string]Value) Object {
	return mapObject{c: class, v: properties}
}

func objectValue(x map[string]Value) Value {
	return Value{reflect.ValueOf(NewObject(ObjectClass, x))}
}

func (o mapObject) Class() Function {
	return o.c
}

func (o mapObject) Get(property string) Value {
	return ValueOf(o.v[property])
}

func (o mapObject) Set(property string, value Value) {
	o.v[property] = value
}

func (o mapObject) Delete(property string) {
	delete(o.v, property)
}

func (o mapObject) Index(i int) Value {
	return ValueOf(o.v[strconv.FormatInt(int64(i), 10)])
}

func (o mapObject) SetIndex(i int, value Value) {
	o.v[strconv.FormatInt(int64(i), 10)] = value
}

func (o mapObject) Call(method string, args []Value) (Value, error) {
	return o.Get(method).Invoke(args)
}

func (o mapObject) Length() int {
	return len(o.v)
}

type functionObject func(args []Value) (Value, error)

func functionValue(f func(args []Value) (Value, error)) Value {
	return ValueOf(functionObject(f))
}

func (o functionObject) Class() Function {
	return FunctionClass
}

func (o functionObject) Get(property string) Value {
	return Undefined()
}

func (o functionObject) Set(property string, value Value) {
}

func (o functionObject) Delete(property string) {
}

func (o functionObject) Index(i int) Value {
	return Undefined()
}

func (o functionObject) SetIndex(i int, value Value) {
}

func (o functionObject) Call(method string, args []Value) (Value, error) {
	return o.Get(method).Invoke(args)
}

func (o functionObject) Invoke(args []Value) (Value, error) {
	return o(args)
}

func (o functionObject) New(args []Value) (Value, error) {
	return o(args)
}

func (o functionObject) Length() int {
	return 0
}

type uint8ArrayObject struct {
	v []byte
}

func uint8ArrayValue(x []byte) Value {
	return Value{reflect.ValueOf(uint8ArrayObject{x})}
}

func (o uint8ArrayObject) Class() Function {
	return Uint8ArrayClass
}

func (o uint8ArrayObject) Get(property string) Value {
	i, err := strconv.ParseInt(property, 10, 0)
	if err == nil && i >= 0 && i < int64(len(o.v)) {
		return ValueOf(o.v[i])
	}
	return Undefined()
}

func (o uint8ArrayObject) Set(property string, value Value) {
	i, err := strconv.ParseInt(property, 10, 0)
	if err == nil && i >= 0 && i < int64(len(o.v)) {
		o.v[i] = byte(value.Int())
	}
}

func (o uint8ArrayObject) Delete(property string) {
	o.Set(property, Undefined())
}

func (o uint8ArrayObject) Index(i int) Value {
	if i >= 0 && i < len(o.v) {
		return ValueOf(o.v[i])
	}
	return Undefined()
}

func (o uint8ArrayObject) SetIndex(i int, value Value) {
	if i >= 0 && i < len(o.v) {
		o.v[i] = byte(value.Int())
	}
}

func (o uint8ArrayObject) Call(method string, args []Value) (Value, error) {
	return o.Get(method).Invoke(args)
}

func (o uint8ArrayObject) Length() int {
	return len(o.v)
}

type errorObject struct {
	v error
}

func errorValue(x error) Value {
	return ValueOf(errorObject{x})
}

func (o errorObject) Class() Function {
	return ErrorClass
}

func (o errorObject) Get(property string) Value {
	if property == "message" {
		return ValueOf(o.v.Error())
	}
	return Undefined()
}

func (o errorObject) Set(property string, value Value) {
}

func (o errorObject) Delete(property string) {
}

func (o errorObject) Index(i int) Value {
	return Undefined()
}

func (o errorObject) SetIndex(i int, value Value) {
}

func (o errorObject) Call(method string, args []Value) (Value, error) {
	return o.Get(method).Invoke(args)
}

func (o errorObject) Length() int {
	return 0
}

type objectClass int

func (o objectClass) Class() Function {
	return FunctionClass
}

func (o objectClass) Get(property string) Value {
	return Undefined()
}

func (o objectClass) Set(property string, value Value) {
}

func (o objectClass) Delete(property string) {
}

func (o objectClass) Index(i int) Value {
	return Undefined()
}

func (o objectClass) SetIndex(i int, value Value) {
}

func (o objectClass) Call(method string, args []Value) (Value, error) {
	return o.Get(method).Invoke(args)
}

func (o objectClass) Invoke(args []Value) (Value, error) {
	return o.New(args)
}

func (o objectClass) New(args []Value) (Value, error) {
	return ValueOf(NewObject(o, map[string]Value{})), nil
}

func (o objectClass) Length() int {
	return 0
}

func newString(args []Value) (Value, error) {
	return ValueOf(""), nil
}

func newArray(args []Value) (Value, error) {
	if len(args) == 1 && args[0].Type() == TypeNumber {
		l := args[0].Int()
		if l >= 0 && l <= (2<<31)-1 {
			v := make([]Value, l)
			return ValueOf(v), nil
		}
	}

	v := make([]Value, len(args))
	return ValueOf(v), nil
}

func newFunction(args []Value) (Value, error) {
	return Undefined(), errors.New("Function() is not supported")
}

func newUint8Array(args []Value) (Value, error) {
	if len(args) == 1 && args[0].Type() == TypeNumber {
		l := args[0].Int()
		if l >= 0 && l <= (2<<31)-1 {
			v := make([]byte, l)
			return ValueOf(v), nil
		}
	}

	return ValueOf([]byte{}), nil
}

func newError(args []Value) (Value, error) {
	return ValueOf(errors.New(args[0].String())), nil
}

var ObjectClass Function = objectClass(0)
var ArrayClass Function = functionObject(newArray)
var StringClass Function = functionObject(newString)
var FunctionClass Function = functionObject(newFunction)
var Uint8ArrayClass Function = functionObject(newUint8Array)
var ErrorClass Function = functionObject(newError)
