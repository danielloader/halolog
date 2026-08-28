/**
 * @file alloc_guard_test.go
 * @description Guards HaloLog's core promise: the logging hot path allocates
 * zero times per call. If these fail, a change has broken the zero-allocation
 * design — fix the change, do not relax the guard.
 * @author Admilson B. F. Cossa
 */

package core

import (
	"testing"

	jsonfmt "github.com/go-gen-ecosystem/halolog/adapters/formatters/json"
	"github.com/go-gen-ecosystem/halolog/adapters/outputs/discard"
	"github.com/go-gen-ecosystem/halolog/types"
)

func TestZeroAlloc_Info(t *testing.T) {
	logger := New().Adapter(discard.New()).MustBuild()
	if allocs := testing.AllocsPerRun(1000, func() {
		logger.Info("hot path message")
	}); allocs != 0 {
		t.Fatalf("Info must allocate 0 times/op, got %.2f", allocs)
	}
}

func TestZeroAlloc_WithFields(t *testing.T) {
	logger := New().Adapter(discard.New()).MustBuild()
	if allocs := testing.AllocsPerRun(1000, func() {
		logger.WithField("k1", "v1").WithField("k2", 42).Info("hot path")
	}); allocs != 0 {
		t.Fatalf("WithField chain must allocate 0 times/op, got %.2f", allocs)
	}
}

// TestZeroAlloc_WithFieldsRealAdapter guards the hot path against a real adapter.
// The discard adapter short-circuits before entry construction, so a discard-only
// guard cannot catch a regression where dispatch escapes to the heap. A no-op
// adapter that receives the entry exercises the full construct-and-dispatch path.
func TestZeroAlloc_WithFieldsRealAdapter(t *testing.T) {
	logger := NewLogger(Config{
		Component: "guard",
		Level:     types.DebugLevel,
		Adapters:  []types.Adapter{&benchmarkAdapter{}},
	})

	// Single field previously allocated a MinimalFieldEntry that escaped via the
	// adapter interface (2 allocs/op); it must now reuse the pooled entry.
	if allocs := testing.AllocsPerRun(1000, func() {
		logger.WithField("k1", "v1").Info("hot path")
	}); allocs != 0 {
		t.Fatalf("single-field WithField (real adapter) must allocate 0 times/op, got %.2f", allocs)
	}

	// Multi-field pooled path.
	if allocs := testing.AllocsPerRun(1000, func() {
		logger.WithField("k1", "v1").WithField("k2", 42).WithField("k3", "v3").Info("hot path")
	}); allocs != 0 {
		t.Fatalf("multi-field WithField (real adapter) must allocate 0 times/op, got %.2f", allocs)
	}
}

// rawNull is a raw-capable, JSON-direct adapter that counts WriteRaw calls
// without copying the line, so the alloc guard measures the logger alone.
type rawNull struct {
	enc      types.DirectFieldEncoder
	rawCalls int
}

func (r *rawNull) Name() string                            { return "rawnull" }
func (r *rawNull) Write(*types.LogEntry) error             { return nil }
func (r *rawNull) WriteZero(*types.LogEntry) error         { return nil }
func (r *rawNull) Flush() error                            { return nil }
func (r *rawNull) Close() error                            { return nil }
func (r *rawNull) SetFormatter(types.Formatter)            {}
func (r *rawNull) Health() error                           { return nil }
func (r *rawNull) WriteRaw([]byte) error                   { r.rawCalls++; return nil }
func (r *rawNull) DirectEncoder() types.DirectFieldEncoder { return r.enc }

// TestZeroAlloc_MessageDirect guards the message-only direct fast path: with a
// single raw-capable JSON adapter, l.Info(msg) renders header+closer straight
// to bytes through the pooled line buffer — no LogEntry, no allocations.
func TestZeroAlloc_MessageDirect(t *testing.T) {
	sink := &rawNull{enc: jsonfmt.NewJsonFormatter()}
	logger := NewLogger(Config{
		Component: "guard",
		Level:     types.InfoLevel,
		Adapters:  []types.Adapter{sink},
	})
	if logger.rawWriter == nil {
		t.Fatal("expected direct-append eligibility")
	}
	if allocs := testing.AllocsPerRun(1000, func() {
		logger.Info("hot path message")
	}); allocs != 0 {
		t.Fatalf("message-direct path must allocate 0 times/op, got %.2f", allocs)
	}
	if sink.rawCalls == 0 {
		t.Fatal("direct message path did not engage WriteRaw")
	}
}

// TestZeroAlloc_BoundContext guards the child-logger hot path: once the
// context is bound (allocates, once, off the hot path), every line — message
// only or with fields, direct or capture — must stay at 0 allocs/op.
func TestZeroAlloc_BoundContext(t *testing.T) {
	direct := NewLogger(Config{
		Component: "guard",
		Level:     types.DebugLevel,
		Adapters:  []types.Adapter{&benchmarkAdapter{}},
	})
	capture := NewLogger(Config{
		Component: "guard",
		Level:     types.DebugLevel,
		Adapters:  []types.Adapter{&benchmarkAdapter{}, &benchmarkAdapter{}},
	})

	for name, l := range map[string]*Logger{"direct": direct, "capture": capture} {
		child := l.With().WithString("svc", "auth").WithInt("shard", 3).Logger()
		if allocs := testing.AllocsPerRun(1000, func() {
			child.Info("bound hot path")
		}); allocs != 0 {
			t.Fatalf("%s bound message-only must allocate 0 times/op, got %.2f", name, allocs)
		}
		if allocs := testing.AllocsPerRun(1000, func() {
			child.Typed().WithInt("status", 200).Info("bound + field")
		}); allocs != 0 {
			t.Fatalf("%s bound typed line must allocate 0 times/op, got %.2f", name, allocs)
		}
	}
}

// TestZeroAlloc_TypedRealAdapter guards the typed builder's construct-and-dispatch
// path (inline 1-4 fields and the pooled >4 path) against heap escape.
func TestZeroAlloc_TypedRealAdapter(t *testing.T) {
	logger := NewLogger(Config{
		Component: "guard",
		Level:     types.DebugLevel,
		Adapters:  []types.Adapter{&benchmarkAdapter{}},
	})

	if allocs := testing.AllocsPerRun(1000, func() {
		logger.Typed().WithString("k", "v").WithInt("n", 7).Info("hot path")
	}); allocs != 0 {
		t.Fatalf("typed inline path (real adapter) must allocate 0 times/op, got %.2f", allocs)
	}

	if allocs := testing.AllocsPerRun(1000, func() {
		logger.Typed().
			WithString("a", "1").WithString("b", "2").WithString("c", "3").
			WithString("d", "4").WithString("e", "5").Info("hot path")
	}); allocs != 0 {
		t.Fatalf("typed pooled path (real adapter) must allocate 0 times/op, got %.2f", allocs)
	}
}
