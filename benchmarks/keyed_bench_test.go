// @author Admilson B. F. Cossa

package benchmarks

import (
	"io"
	"testing"

	jsonfmt "github.com/go-gen-ecosystem/halolog/adapters/formatters/json"
	"github.com/go-gen-ecosystem/halolog/adapters/outputs/console"
	"github.com/go-gen-ecosystem/halolog/core"
	"github.com/go-gen-ecosystem/halolog/types"
)

// keyNames and predeclaredKeys back the keyed benchmark; keys are declared once,
// exactly as a real application would.
var keyNames = []string{"k0", "k1", "k2", "k3", "k4", "k5", "k6", "k7", "k8", "k9",
	"k10", "k11", "k12", "k13", "k14", "k15", "k16", "k17", "k18", "k19"}

var predeclaredKeys = func() []*types.FieldKey {
	ks := make([]*types.FieldKey, len(keyNames))
	for i, n := range keyNames {
		ks[i] = &types.FieldKey{Name: n, JSONFragment: jsonfmt.KeyFragment(n)}
	}
	return ks
}()

func newKeyedBenchLogger() *core.Logger {
	return core.NewLogger(core.Config{
		Component: "bench", Level: types.InfoLevel,
		Adapters: []types.Adapter{console.NewWithWriter(io.Discard, jsonfmt.NewJsonFormatter())},
	})
}

// BenchmarkKeyed_TenFields compares the three key strategies at 10 fields, all
// serializing real JSON to io.Discard:
//   - WithField: string key, escaped/cached in the formatter
//   - Keyed:     pre-declared key, fragment copied with no lookup
func BenchmarkKeyed_TenFields(b *testing.B) {
	b.Run("WithField", func(b *testing.B) {
		l := newKeyedBenchLogger()
		l.WithField("k0", 0).Info("warm") // warm the formatter key cache
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			fb := l.WithField(keyNames[0], 0)
			for j := 1; j < 10; j++ {
				fb = fb.WithField(keyNames[j], j)
			}
			fb.Info("msg")
		}
	})
	b.Run("Keyed", func(b *testing.B) {
		l := newKeyedBenchLogger()
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			fb := l.Typed().Int(predeclaredKeys[0], 0)
			for j := 1; j < 10; j++ {
				fb = fb.Int(predeclaredKeys[j], j)
			}
			fb.Info("msg")
		}
	})
}

// BenchmarkKeyIsolation isolates ONLY the key mechanism: both variants use the
// typed builder and identical string values, differing solely in how the key is
// emitted — WithString escapes the key inline; Str copies a pre-escaped fragment.
func BenchmarkKeyIsolation(b *testing.B) {
	b.Run("StringKey", func(b *testing.B) {
		l := newKeyedBenchLogger()
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			fb := l.Typed().WithString(keyNames[0], "value")
			for j := 1; j < 10; j++ {
				fb = fb.WithString(keyNames[j], "value")
			}
			fb.Info("msg")
		}
	})
	b.Run("PreDeclaredKey", func(b *testing.B) {
		l := newKeyedBenchLogger()
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			fb := l.Typed().Str(predeclaredKeys[0], "value")
			for j := 1; j < 10; j++ {
				fb = fb.Str(predeclaredKeys[j], "value")
			}
			fb.Info("msg")
		}
	})
}

// BenchmarkKeyed_TwentyFields is the same at 20 fields.
func BenchmarkKeyed_TwentyFields(b *testing.B) {
	b.Run("WithField", func(b *testing.B) {
		l := newKeyedBenchLogger()
		l.WithField("k0", 0).Info("warm")
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			fb := l.WithField(keyNames[0], 0)
			for j := 1; j < 20; j++ {
				fb = fb.WithField(keyNames[j], j)
			}
			fb.Info("msg")
		}
	})
	b.Run("Keyed", func(b *testing.B) {
		l := newKeyedBenchLogger()
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			fb := l.Typed().Int(predeclaredKeys[0], 0)
			for j := 1; j < 20; j++ {
				fb = fb.Int(predeclaredKeys[j], j)
			}
			fb.Info("msg")
		}
	})
}
