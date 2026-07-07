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
