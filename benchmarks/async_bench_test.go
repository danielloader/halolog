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

// BenchmarkAsyncCallerLatency measures the CALLER cost of one 8-field structured
// log call under three regimes. It is deliberately kept OUT of the head-to-head
// serialization tables (BenchmarkDefaults / BenchmarkNoTimestamp) because async
// logging does not make serialization cheaper — it moves it to a background
// goroutine. The honest async story needs both a burst and a saturated number:
//
//   - Sync_JSON            full JSON serialization on the calling goroutine (the baseline).
//   - AsyncRing_Burst      OnFull=Drop, large ring: what a caller pays while the ring
//     has slack. FAST, but a Go bench runs millions of iterations
//     against one consumer, so the ring saturates and records are
//     dropped — the reported "drops" custom metric shows how many.
//     This number is only representative of BURSTY, below-drain-rate
//     logging; it is NOT a serialization-throughput number.
//   - AsyncRing_Blocking   OnFull=Block: the caller spins for a slot, so under sustained
//     max-rate logging its latency rises toward the consumer's
//     per-record serialization cost. "drops" must be 0 here — every
//     record is serialized. This is the honest sustained-load figure.
func BenchmarkAsyncCallerLatency(b *testing.B) {
	b.Run("Sync_JSON", func(b *testing.B) {
		l := core.NewLogger(core.Config{Component: "bench", Level: types.InfoLevel,
			Adapters: []types.Adapter{console.NewWithWriter(io.Discard, jsonfmt.NewJsonFormatter())}})
		benchLoop(b, func() { haloWithField(l) })
	})

	b.Run("AsyncRing_Burst", func(b *testing.B) {
		ring, err := asyncring.New(asyncring.Options{Writer: io.Discard,
			Formatter: jsonfmt.NewJsonFormatter(), Capacity: 65536, OnFull: asyncring.Drop})
		if err != nil {
			b.Fatal(err)
		}
		l := core.NewLogger(core.Config{Component: "bench", Level: types.InfoLevel,
			Adapters: []types.Adapter{ring}})
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			haloWithField(l)
		}
		b.StopTimer()
		b.ReportMetric(float64(ring.Dropped()), "drops")
		_ = ring.Close()
	})

	b.Run("AsyncRing_Blocking", func(b *testing.B) {
		ring, err := asyncring.New(asyncring.Options{Writer: io.Discard,
			Formatter: jsonfmt.NewJsonFormatter(), Capacity: 8192, OnFull: asyncring.Block})
		if err != nil {
			b.Fatal(err)
		}
		l := core.NewLogger(core.Config{Component: "bench", Level: types.InfoLevel,
			Adapters: []types.Adapter{ring}})
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			haloWithField(l)
		}
		b.StopTimer()
		b.ReportMetric(float64(ring.Dropped()), "drops")
		_ = ring.Close()
	})
}
