package go_wasm_exec

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/pgavlin/warp/exec"
	"github.com/pgavlin/warp/wasi"
	"github.com/pgavlin/warp/wasm"
)

type Options struct {
	Env  map[string]string
	Args []string

	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
	FS     wasi.FS

	Global Object

	Debug    bool
	Trace    io.Writer
	Resolver exec.ModuleResolver
}

type ExitError struct {
	code int
}

func (e *ExitError) Code() int {
	return e.code
}

func (e *ExitError) Error() string {
	return fmt.Sprintf("exit status %d", e.code)
}

func Run(name string, def exec.ModuleDefinition, options *Options) error {
	if options == nil {
		options = &Options{}
	}
	options.Args = append([]string{name}, options.Args...)

	fs := wasi.NewFS()
	stdin, stdout, stderr := wasi.NewFile(os.Stdin, 0), wasi.NewFile(os.Stdout, wasi.F_Append), wasi.NewFile(os.Stderr, wasi.F_Append)
	if options.FS != nil {
		fs = options.FS
	}
	if options.Stdin != nil {
		stdin = wasi.NewReader(options.Stdin)
	}
	if options.Stdout != nil {
		stdout = wasi.NewWriter(options.Stdout)
	}
	if options.Stderr != nil {
		stderr = wasi.NewWriter(options.Stderr)
	}

	if options.Global == nil {
		options.Global = NewGlobal(NewFS(stdin, stdout, stderr, fs))
	}

	resolver := exec.ModuleResolver(exec.MapResolver{})
	if options.Resolver != nil {
		resolver = options.Resolver
	}
	store := exec.NewStore(resolver)

	wasm := &wasmExec{
		env:    options.Env,
		args:   options.Args,
		stdout: stdout,
		stderr: stderr,
		global: options.Global,
	}
	store.RegisterModule(wasm.Name(), wasm)

	mod, err := store.InstantiateModuleDefinition("", def)
	if err != nil {
		return err
	}

	if err = wasm.init(mod); err != nil {
		return err
	}

	t := exec.NewThread(0)
	wasm.thread = &t
	code := wasm.main()
	if code != 0 {
		return &ExitError{code: code}
	}
	return nil
}

func Main(def exec.ModuleDefinition) {
	if err := MainErr(def); err != nil {
		if exit, ok := err.(*ExitError); ok {
			os.Exit(exit.code)
		}
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(-1)
	}
}

func MainErr(def exec.ModuleDefinition) error {
	env := map[string]string{}
	for _, v := range os.Environ() {
		kvp := strings.SplitN(v, "=", 2)
		env[kvp[0]] = kvp[1]
	}

	err := Run(os.Args[0], def, &Options{
		Env:  env,
		Args: os.Args[1:],
	})
	if err != nil {
		if _, ok := err.(*ExitError); ok {
			return err
		}
		return fmt.Errorf("error running program: %w", err)
	}
	return nil
}

type wasmExec struct {
	env    map[string]string
	args   []string
	stdout wasi.File
	stderr wasi.File
	global Object

	mem     *exec.Memory
	runF    exec.Function
	getspF  exec.Function
	resumeF exec.Function

	thread     *exec.Thread
	timeOrigin time.Time
	events     chan chan bool

	pendingEvent          Value
	scheduledTimeouts     map[uint32]*time.Timer
	nextCallbackTimeoutID uint32

	values      []Value
	goRefCounts []int
	ids         map[Value]uint32
	idPool      []uint32
	exited      bool

	exitCode int
}

func (m *wasmExec) Class() Function {
	return ObjectClass
}

func (m *wasmExec) Get(property string) Value {
	if property == "_pendingEvent" {
		return m.pendingEvent
	}
	return Undefined()
}

func (m *wasmExec) Set(property string, value Value) {
	if property == "_pendingEvent" {
		m.pendingEvent = value
	}
}

func (m *wasmExec) Delete(property string) {
}

func (m *wasmExec) Index(i int) Value {
	return Undefined()
}

func (m *wasmExec) SetIndex(i int, value Value) {
}

func (m *wasmExec) Call(method string, args []Value) (Value, error) {
	if method == "_makeFuncWrapper" {
		return ValueOf(func(funcArgs []Value) (Value, error) {
			event := ValueOf(map[string]Value{
				"id":   args[0],
				"this": funcArgs[0],
				"args": ValueOf(funcArgs[1:]),
			})
			m.pendingEvent = event

			done := make(chan bool)
			m.events <- done
			<-done

			return event.Get("result"), nil
		}), nil
	}
	return Undefined().Invoke(args)
}

func (m *wasmExec) Length() int {
	return 0
}

func (m *wasmExec) init(mod exec.Module) (err error) {
	m.mem, err = mod.GetMemory("mem")
	if err != nil {
		return err
	}

	m.runF, err = mod.GetFunction("run")
	if err != nil {
		return err
	}
	if !m.runF.GetSignature().Equals(wasm.FunctionSig{
		ParamTypes: []wasm.ValueType{wasm.ValueTypeI32, wasm.ValueTypeI32},
	}) {
		return fmt.Errorf("run must be of type (func (param i32 i32))")
	}

	m.getspF, err = mod.GetFunction("getsp")
	if err != nil {
		return err
	}
	if !m.getspF.GetSignature().Equals(wasm.FunctionSig{
		ReturnTypes: []wasm.ValueType{wasm.ValueTypeI32},
	}) {
		return fmt.Errorf("getsp must be of type (func (result i32))")
	}

	m.resumeF, err = mod.GetFunction("resume")
	if err != nil {
		return err
	}
	if !m.resumeF.GetSignature().Equals(wasm.FunctionSig{}) {
		return fmt.Errorf("resume must be of type (func)")
	}

	m.events = make(chan chan bool)
	m.scheduledTimeouts = map[uint32]*time.Timer{}

	m.values = []Value{
		ValueOf(math.NaN()),
		ValueOf(0),
		Null(),
		ValueOf(true),
		ValueOf(false),
		ValueOf(m.global),
		ValueOf(m),
	}
	m.goRefCounts = []int{-1, -1, -1, -1, -1, -1, -1}
	m.ids = map[Value]uint32{}
	for id, v := range m.values {
		m.ids[v] = uint32(id)
	}

	return nil
}

func (m *wasmExec) Name() string {
	return "go"
}

func (m *wasmExec) GetTable(name string) (*exec.Table, error) {
	return nil, errors.New("unknown table")
}

func (m *wasmExec) GetMemory(name string) (*exec.Memory, error) {
	return nil, errors.New("unknown memory")
}

func (m *wasmExec) GetGlobal(name string) (*exec.Global, error) {
	return nil, errors.New("unknown global")
}

func (m *wasmExec) GetFunction(name string) (exec.Function, error) {
	switch name {
	case "runtime.wasmExit":
		return exec.NewHostFunction(m, 0, reflect.ValueOf(m.wasmExit)), nil
	case "runtime.wasmWrite":
		return exec.NewHostFunction(m, 1, reflect.ValueOf(m.wasmWrite)), nil
	case "runtime.resetMemoryDataView":
		return exec.NewHostFunction(m, 2, reflect.ValueOf(m.resetMemoryDataView)), nil
	case "runtime.nanotime1":
		return exec.NewHostFunction(m, 3, reflect.ValueOf(m.nanotime1)), nil
	case "runtime.walltime1":
		return exec.NewHostFunction(m, 4, reflect.ValueOf(m.walltime1)), nil
	case "runtime.scheduleTimeoutEvent":
		return exec.NewHostFunction(m, 5, reflect.ValueOf(m.scheduleTimeoutEvent)), nil
	case "runtime.clearTimeoutEvent":
		return exec.NewHostFunction(m, 6, reflect.ValueOf(m.clearTimeoutEvent)), nil
	case "runtime.getRandomData":
		return exec.NewHostFunction(m, 7, reflect.ValueOf(m.getRandomData)), nil
	case "syscall/js.finalizeRef":
		return exec.NewHostFunction(m, 8, reflect.ValueOf(m.finalizeRef)), nil
	case "syscall/js.stringVal":
		return exec.NewHostFunction(m, 9, reflect.ValueOf(m.stringVal)), nil
	case "syscall/js.valueGet":
		return exec.NewHostFunction(m, 10, reflect.ValueOf(m.valueGet)), nil
	case "syscall/js.valueSet":
		return exec.NewHostFunction(m, 11, reflect.ValueOf(m.valueSet)), nil
	case "syscall/js.valueDelete":
		return exec.NewHostFunction(m, 12, reflect.ValueOf(m.valueDelete)), nil
	case "syscall/js.valueIndex":
		return exec.NewHostFunction(m, 13, reflect.ValueOf(m.valueIndex)), nil
	case "syscall/js.valueSetIndex":
		return exec.NewHostFunction(m, 14, reflect.ValueOf(m.valueSetIndex)), nil
	case "syscall/js.valueCall":
		return exec.NewHostFunction(m, 15, reflect.ValueOf(m.valueCall)), nil
	case "syscall/js.valueInvoke":
		return exec.NewHostFunction(m, 16, reflect.ValueOf(m.valueInvoke)), nil
	case "syscall/js.valueNew":
		return exec.NewHostFunction(m, 17, reflect.ValueOf(m.valueNew)), nil
	case "syscall/js.valueLength":
		return exec.NewHostFunction(m, 18, reflect.ValueOf(m.valueLength)), nil
	case "syscall/js.valuePrepareString":
		return exec.NewHostFunction(m, 19, reflect.ValueOf(m.valuePrepareString)), nil
	case "syscall/js.valueLoadString":
		return exec.NewHostFunction(m, 20, reflect.ValueOf(m.valueLoadString)), nil
	case "syscall/js.valueInstanceOf":
		return exec.NewHostFunction(m, 21, reflect.ValueOf(m.valueInstanceOf)), nil
	case "syscall/js.copyBytesToGo":
		return exec.NewHostFunction(m, 22, reflect.ValueOf(m.copyBytesToGo)), nil
	case "syscall/js.copyBytesToJS":
		return exec.NewHostFunction(m, 23, reflect.ValueOf(m.copyBytesToJS)), nil
	case "debug":
		return exec.NewHostFunction(m, 24, reflect.ValueOf(m.debug)), nil
	default:
		return nil, errors.New("unknown function")
	}
}

func (m *wasmExec) getsp() uint32 {
	var newSP [1]uint64
	m.getspF.UncheckedCall(m.thread, nil, newSP[:])
	return uint32(newSP[0])
}

func (m *wasmExec) loadValue(addr uint32) Value {
	f := m.mem.Float64At(addr)
	if f == 0 {
		return Undefined()
	}
	if !math.IsNaN(f) {
		return ValueOf(f)
	}
	id := m.mem.Uint32At(addr)
	return ValueOf(m.values[id])
}

func (m *wasmExec) storeValue(addr uint32, v Value) {
	const nan = 0x7ff8000000000000

	if v.IsUndefined() {
		m.mem.PutFloat64At(0, addr)
		return
	}

	if v.Type() == TypeNumber && v.Float() != 0 {
		f := v.Float()
		if math.IsNaN(f) {
			m.mem.PutUint64At(nan, addr)
			return
		}
		m.mem.PutFloat64At(f, addr)
		return
	}

	id, ok := m.ids[v]
	if !ok {
		if len(m.idPool) == 0 {
			id, m.values, m.goRefCounts = uint32(len(m.values)), append(m.values, v), append(m.goRefCounts, 0)
		} else {
			id, m.idPool = m.idPool[len(m.idPool)-1], m.idPool[:len(m.idPool)-1]
			m.values[id], m.goRefCounts[id] = v, 0
		}
		m.ids[v] = id
	}
	m.goRefCounts[id]++

	typeFlag := 0
	switch v.Type() {
	case TypeObject:
		typeFlag = 1
	case TypeString:
		typeFlag = 2
	case TypeSymbol:
		typeFlag = 3
	case TypeFunction:
		typeFlag = 4
	}
	m.mem.PutUint64At(nan|uint64(typeFlag)<<32|uint64(id), addr)
}

func (m *wasmExec) loadSlice(addr uint32) []byte {
	array, length := m.mem.Uint64(addr, 0), m.mem.Uint64(addr, 8)
	return m.mem.Bytes()[array : array+length]
}

func (m *wasmExec) loadSliceOfValues(addr uint32) []Value {
	array, length := m.mem.Uint64(addr, 0), m.mem.Uint64(addr, 8)
	a := make([]Value, int(length))
	for i := range a {
		a[i] = m.loadValue(uint32(array) + uint32(i)*8)
	}
	return a
}

func (m *wasmExec) loadString(addr uint32) string {
	saddr, length := m.mem.Uint64(addr, 0), m.mem.Uint64(addr, 8)
	return string(m.mem.Bytes()[saddr : saddr+length])
}

func (m *wasmExec) storeString(addr uint32, v string) {
	copy(m.mem.Bytes()[addr:addr+uint32(len(v))], v)
}

// Go's SP does not change as long as no Go code is running. Some operations (e.g. calls, getters and setters)
// may synchronously trigger a Go event handler. This makes Go code get executed in the middle of the imported
// function. A goroutine can switch to a new stack if the current stack is too small (see morestack function).
// This changes the SP, thus we have to update the SP used by the imported function.

// func wasmExit(code int32)
func (m *wasmExec) wasmExit(sp uint32) {
	code := int32(m.mem.Uint32(sp, 8))
	m.exited = true
	m.exitCode = int(code)
	close(m.events)
}

// func wasmWrite(fd uintptr, p unsafe.Pointer, n int32)
func (m *wasmExec) wasmWrite(sp uint32) {
	fd := m.mem.Uint64(sp, 8)
	p := m.mem.Uint64(sp, 16)
	n := m.mem.Uint32(sp, 24)

	b := m.mem.Bytes()[p : p+uint64(n)]
	switch fd {
	case 1:
		m.stdout.Writev([][]byte{b})
	case 2:
		m.stderr.Writev([][]byte{b})
	}
}

// func resetMemoryDataView()
func (m *wasmExec) resetMemoryDataView(sp uint32) {
}

// func nanotime1() int64
func (m *wasmExec) nanotime1(sp uint32) {
	t := m.timeOrigin.Add(time.Now().Sub(m.timeOrigin)).UnixNano()
	m.mem.PutUint64(uint64(t), sp, 8)
}

// func walltime1() (sec int64, nsec int32)
func (m *wasmExec) walltime1(sp uint32) {
	t := time.Now()
	m.mem.PutUint64(uint64(int64(t.Second())), sp, 8)
	m.mem.PutUint32(uint32(int32(t.Nanosecond())), sp, 16)
}

// func scheduleTimeoutEvent(delay int64) int32
func (m *wasmExec) scheduleTimeoutEvent(sp uint32) {
	id := m.nextCallbackTimeoutID
	m.nextCallbackTimeoutID++

	duration := time.Duration(m.mem.Uint64(sp, 8)) * time.Millisecond
	m.scheduledTimeouts[id] = time.AfterFunc(duration, func() {
		for {
			done := make(chan bool)
			m.events <- done
			<-done

			if _, ok := m.scheduledTimeouts[id]; !ok {
				return
			}

			// for some reason Go failed to register the timeout event, log and try again
			// (temporary workaround for https://github.com/golang/go/issues/28975)
			log.Printf("scheduleTimeoutEvent: missed timeout event")
		}
	})
	m.mem.PutUint32(id, sp, 16)
}

// func clearTimeoutEvent(id int32)
func (m *wasmExec) clearTimeoutEvent(sp uint32) {
	id := m.mem.Uint32(sp, 8)
	if timer, ok := m.scheduledTimeouts[id]; ok {
		timer.Stop()
		delete(m.scheduledTimeouts, id)
	}
}

// func getRandomData(r []byte)
func (m *wasmExec) getRandomData(sp uint32) {
	if _, err := rand.Read(m.loadSlice(sp + 8)); err != nil {
		panic(err)
	}
}

// func finalizeRef(v ref)
func (m *wasmExec) finalizeRef(sp uint32) {
	id := m.mem.Uint32(sp, 8)
	if m.goRefCounts[id] == -1 {
		return
	}

	m.goRefCounts[id]--
	if m.goRefCounts[id] == 0 {
		m.values[id] = Undefined()
		m.idPool = append(m.idPool, id)
	}
}

// func stringVal(value string) ref
func (m *wasmExec) stringVal(sp uint32) {
	m.storeValue(sp+24, ValueOf(m.loadString(sp+8)))
}

// func valueGet(v ref, p string) ref
func (m *wasmExec) valueGet(sp uint32) {
	v := m.loadValue(sp + 8)
	p := m.loadString(sp + 16)
	x := v.Get(p)
	sp = m.getsp()
	m.storeValue(sp+32, x)
}

// func valueSet(v ref, p string, x ref)
func (m *wasmExec) valueSet(sp uint32) {
	v := m.loadValue(sp + 8)
	p := m.loadString(sp + 16)
	x := m.loadValue(sp + 32)
	v.Set(p, x)
}

// func valueDelete(v ref, p string)
func (m *wasmExec) valueDelete(sp uint32) {
	v := m.loadValue(sp + 8)
	p := m.loadString(sp + 16)
	v.Delete(p)
}

// func valueIndex(v ref, i int) ref
func (m *wasmExec) valueIndex(sp uint32) {
	v := m.loadValue(sp + 8)
	i := int64(m.mem.Uint64(sp, 16))
	x := v.Index(int(i))
	sp = m.getsp()
	m.storeValue(sp+24, x)
}

// valueSetIndex(v ref, i int, x ref)
func (m *wasmExec) valueSetIndex(sp uint32) {
	v := m.loadValue(sp + 8)
	i := int64(m.mem.Uint64(sp, 16))
	x := m.loadValue(sp + 24)
	v.SetIndex(int(i), x)
}

// func valueCall(v ref, m string, args []ref) (ref, bool)
func (m *wasmExec) valueCall(sp uint32) {
	v := m.loadValue(sp + 8)
	p := m.loadString(sp + 16)
	args := m.loadSliceOfValues(sp + 32)
	result, err := v.Call(p, args)
	sp = m.getsp()
	if err == nil {
		m.storeValue(sp+56, result)
		m.mem.PutByte(1, sp, 64)
	} else {
		m.storeValue(sp+56, ValueOf(err))
		m.mem.PutByte(0, sp, 64)
	}
}

// func valueInvoke(v ref, args []ref) (ref, bool)
func (m *wasmExec) valueInvoke(sp uint32) {
	v := m.loadValue(sp + 8)
	args := m.loadSliceOfValues(sp + 16)
	result, err := v.Invoke(args)
	sp = m.getsp()
	if err == nil {
		m.storeValue(sp+40, result)
		m.mem.PutByte(1, sp, 48)
	} else {
		m.storeValue(sp+40, ValueOf(err))
		m.mem.PutByte(0, sp, 48)
	}
}

// func valueNew(v ref, args []ref) (ref, bool)
func (m *wasmExec) valueNew(sp uint32) {
	v := m.loadValue(sp + 8)
	args := m.loadSliceOfValues(sp + 16)
	result, err := v.New(args)
	sp = m.getsp()
	if err == nil {
		m.storeValue(sp+40, result)
		m.mem.PutByte(1, sp, 48)
	} else {
		m.storeValue(sp+40, ValueOf(err))
		m.mem.PutByte(0, sp, 48)
	}
}

// func valueLength(v ref) int
func (m *wasmExec) valueLength(sp uint32) {
	v := m.loadValue(sp + 8)
	l := v.Length()
	sp = m.getsp()
	m.mem.PutUint64(uint64(l), sp, 16)
}

// valuePrepareString(v ref) (ref, int)
func (m *wasmExec) valuePrepareString(sp uint32) {
	v := []byte(m.loadValue(sp + 8).String())
	m.storeValue(sp+16, ValueOf(v))
	m.mem.PutUint64(uint64(len(v)), sp, 24)
}

// valueLoadString(v ref, b []byte)
func (m *wasmExec) valueLoadString(sp uint32) {
	v := m.loadValue(sp + 8).String()
	copy(m.loadSlice(sp+16), v)
}

// func valueInstanceOf(v ref, t ref) bool
func (m *wasmExec) valueInstanceOf(sp uint32) {
	v := m.loadValue(sp + 8)
	t := m.loadValue(sp + 16)
	if t.IsInstance(v) {
		m.mem.PutByte(1, sp, 24)
	} else {
		m.mem.PutByte(0, sp, 24)
	}
}

// func copyBytesToGo(dst []byte, src ref) (int, bool)
func (m *wasmExec) copyBytesToGo(sp uint32) {
	dst := m.loadSlice(sp + 8)
	srcValue := m.loadValue(sp + 32)
	if src, ok := srcValue.Uint8Array(); ok {
		n := copy(dst, src)
		m.mem.PutUint64(uint64(n), sp, 40)
		m.mem.PutByte(1, sp, 48)
		return
	}
	m.mem.PutByte(0, sp, 48)
}

// func copyBytesToJS(dst ref, src []byte) (int, bool)
func (m *wasmExec) copyBytesToJS(sp uint32) {
	dstValue := m.loadValue(sp + 8)
	src := m.loadSlice(sp + 16)
	if dst, ok := dstValue.Uint8Array(); ok {
		n := copy(dst, src)
		m.mem.PutUint64(uint64(n), sp, 40)
		m.mem.PutByte(1, sp, 48)
		return
	}
	m.mem.PutByte(0, sp, 48)
}

func (m *wasmExec) debug(value int32) {
	println(value)
}

func (m *wasmExec) turn() bool {
	if m.exited {
		panic("Go program has already exited")
	}
	m.resumeF.UncheckedCall(m.thread, nil, nil)
	return m.exited
}

func (m *wasmExec) main() int {
	argc := len(m.args)

	// Pass command line arguments and environment variables to WebAssembly by writing them to the linear memory.
	offset := uint32(4096)
	argvPtrs := make([]uint32, 0, len(m.args)+len(m.env)+2)
	for _, arg := range m.args {
		argvPtrs = append(argvPtrs, offset)
		m.storeString(offset, arg)
		offset += uint32(len(arg))
		if pad := offset % 8; pad != 0 {
			offset += 8 - pad
		}
	}
	argvPtrs = append(argvPtrs, 0)

	keys := make([]string, 0, len(m.env))
	for k := range m.env {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		argvPtrs = append(argvPtrs, offset)
		str := k + "=" + m.env[k]
		m.storeString(offset, str)
		offset += uint32(len(str))
		if pad := offset % 8; pad != 0 {
			offset += 8 - pad
		}
	}
	argvPtrs = append(argvPtrs, 0)

	argv := offset
	for _, ptr := range argvPtrs {
		m.mem.PutUint64At(uint64(ptr), offset)
		offset += 8
	}

	go func() {
		args := [2]uint64{uint64(argc), uint64(argv)}
		m.runF.UncheckedCall(m.thread, args[:], nil)
	}()
	for done := range m.events {
		if m.turn() {
			return m.exitCode
		}
		close(done)
	}
	return m.exitCode
}
