package exec

import (
	"io"

	"github.com/pgavlin/warp/wasm"
	"github.com/pgavlin/warp/wasm/trace"
)

// A Frame records a single WASM activation record.
type Frame struct {
	Caller            *Frame
	ModuleName        string
	FunctionIndex     uint32
	FunctionSignature wasm.FunctionSig
	Locals            []uint64
}

// A Thread carries information about a single WASM thread.
type Thread struct {
	active   *Frame
	trace    io.Writer
	debug    bool
	depth    uint
	maxDepth uint
}

// NewThread creates a new thread with the given max depth, if any.
func NewThread(maxDepth uint) Thread {
	if maxDepth == 0 {
		maxDepth = (1 << 32) - 1
	}
	return Thread{maxDepth: maxDepth}
}

// NewDebugThread creates a new, debuggable thread with the given tracer and max depth, if any.
func NewDebugThread(trace io.Writer, maxDepth uint) Thread {
	t := NewThread(maxDepth)
	t.trace, t.debug = trace, true
	return t
}

// Close closes the thread.
func (t *Thread) Close() error {
	if t.trace != nil {
		var entry trace.EndEntry
		return entry.Encode(t.trace)
	}
	return nil
}

// Trace returns the writer for this thread's tracer, if any.
func (t *Thread) Trace() (io.Writer, bool) {
	if t.trace != nil {
		return t.trace, true
	}
	return nil, false
}

// Debug returns true if this thread's frames should be recorded for debugging.
func (t *Thread) Debug() bool {
	return t.debug
}

// MaxDepth returns the maximum call stack depth, if any.
func (t *Thread) MaxDepth() uint {
	return t.maxDepth
}

// Enter pushes a new frame onto the thread's stack. Each call to Enter must be balanced with a call to Leave.
func (t *Thread) Enter() {
	if t.depth >= t.maxDepth {
		panic(TrapCallStackExhausted)
	}
	t.depth++
}

// EnterFrame pushes a new frame onto the thread's stack. Each call to EnterFrame must be balanced with a call to LeaveFrame.
func (t *Thread) EnterFrame(f *Frame) {
	t.Enter()

	f.Caller, t.active = t.active, f

	if t.trace != nil {
		entry := trace.EnterEntry{
			ModuleName:        f.ModuleName,
			FunctionIndex:     f.FunctionIndex,
			FunctionSignature: f.FunctionSignature,
		}
		entry.Encode(t.trace)
	}
}

// Leave pops the top of the thread's stack.
func (t *Thread) Leave() {
	t.depth--
}

// LeaveFrame pops the top of the thread's stack.
func (t *Thread) LeaveFrame() {
	t.Leave()

	t.active = t.active.Caller

	if t.trace != nil {
		leave := trace.LeaveEntry{}
		leave.Encode(t.trace)
	}
}
