// @author Admilson B. F. Cossa

package benchmarks

import (
	"io"
	"testing"

	jsonfmt "github.com/go-gen-ecosystem/halolog/adapters/formatters/json"
	"github.com/go-gen-ecosystem/halolog/adapters/outputs/asyncring"
	"github.com/go-gen-ecosystem/halolog/adapters/outputs/console"
	"github.com/go-gen-ecosystem/halolog/core"
	"github.com/go-gen-ecosystem/halolog/types"
)

// BenchmarkAsyncCallerLatency compares the CALLER cost of one structured log call
// when serialization happens synchronously on the caller (console+JSON to
// io.Discard) versus asynchronously via the lock-free ring (the caller only
// copies into a ring slot; a background goroutine serializes). The async number
// is caller latency, not throughput.
func BenchmarkAsyncCallerLatency(b *testing.B) {
	b.Run("Sync_JSON", func(b *testing.B) {
		l := core.NewLogger(core.Config{
			Component: "bench", Level: types.InfoLevel,
			Adapters: []types.Adapter{console.NewWithWriter(io.Discard, jsonfmt.NewJsonFormatter())},
		})
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			l.WithField("user", "alice").WithField("code", 200).Info("request handled")
		}
	})

	b.Run("AsyncRing", func(b *testing.B) {
		ring, err := asyncring.New(asyncring.Options{
			Writer: io.Discard, Formatter: jsonfmt.NewJsonFormatter(),
			Capacity: 8192, OnFull: asyncring.Drop,
		})
		if err != nil {
			b.Fatal(err)
		}
		defer func() { _ = ring.Close() }()
		l := core.NewLogger(core.Config{
			Component: "bench", Level: types.InfoLevel,
			Adapters: []types.Adapter{ring},
		})
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			l.WithField("user", "alice").WithField("code", 200).Info("request handled")
		}
	})
}
