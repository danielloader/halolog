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
