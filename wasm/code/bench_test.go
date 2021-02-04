package code

import (
	"testing"

	"github.com/pgavlin/warp/bench/flate"
	"github.com/pgavlin/warp/wasm"
)

func decodeFunctions(tb testing.TB, m *wasm.Module) {
	s := NewStaticScope(m)
	for funcidx, body := range m.Code.Bodies {
		sig := m.Types.Entries[m.Function.Types[funcidx]]
		s.SetFunction(sig, body)

		if _, err := Decode(body.Code, s, sig.ReturnTypes); err != nil {
			tb.Fatal(err)
		}
	}
}

func TestFlate(t *testing.T) {
	decodeFunctions(t, flate.Module)
}

func BenchmarkFlate(b *testing.B) {
	for i := 0; i < b.N; i++ {
		decodeFunctions(b, flate.Module)
	}
}
