// Copyright 2025 Admilson B. F. Cossa
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// Package race behavioural coverage tests.
//
// @author Admilson B. F. Cossa

package race

import (
	"context"
	"sync"
	"testing"
	"unsafe"

	"github.com/go-gen-ecosystem/halolog/types"
)

// newRP builds a fully wired RacePrevention instance identical to the one the
// global initializer produces, but private to a test so state is isolated.
func newRP() *RacePrevention {
	rp := &RacePrevention{
		raceDetectors: []RaceDetector{
			NewBasicRaceDetector(),
			NewMemoryOrderDetector(),
		},
	}
	rp.tlsPool.New = func() interface{} {
		return &ThreadState{threadID: 1}
	}
	return rp
}

// -----------------------------------------------------------------------------
// BasicRaceDetector
// -----------------------------------------------------------------------------

func TestBasicRaceDetector(t *testing.T) {
	d := NewBasicRaceDetector()
	var x int
	ptr := unsafe.Pointer(&x)

	// No prior access -> no race.
	if d.CheckRace(1, "write", ptr) {
		t.Fatal("unexpected race on first check")
	}
	if got := d.GetRaceCount(); got != 0 {
		t.Fatalf("race count = %d, want 0", got)
	}

	// Record a write from thread 1.
	d.RecordAccess(1, "write", ptr)

	// Same thread, same location -> not a race (ThreadID matches).
	if d.CheckRace(1, "write", ptr) {
		t.Fatal("same-thread access flagged as race")
	}

	// Different thread writing the same location -> race.
	if !d.CheckRace(2, "write", ptr) {
		t.Fatal("concurrent write not detected as race")
	}
	if got := d.GetRaceCount(); got != 1 {
		t.Fatalf("race count = %d, want 1", got)
	}

	// read-after-write from another thread -> race.
	if !d.CheckRace(3, "read", ptr) {
		t.Fatal("read-after-write cross-thread not detected")
	}

	// Append a second record then reset.
	d.RecordAccess(2, "read", ptr)
	d.Reset()
	if got := d.GetRaceCount(); got != 0 {
		t.Fatalf("post-reset race count = %d, want 0", got)
	}
	if d.CheckRace(2, "write", ptr) {
		t.Fatal("records not cleared after Reset")
	}
}

func TestBasicRaceDetectorReadReadNoRace(t *testing.T) {
	d := NewBasicRaceDetector()
	var x int
	ptr := unsafe.Pointer(&x)

	d.RecordAccess(1, "read", ptr)
	// read/read across threads is not a conflict.
	if d.CheckRace(2, "read", ptr) {
		t.Fatal("read/read flagged as race")
	}
}

// -----------------------------------------------------------------------------
// MemoryOrderDetector
// -----------------------------------------------------------------------------

func TestMemoryOrderDetector(t *testing.T) {
	d := NewMemoryOrderDetector()
	var x int
	ptr := unsafe.Pointer(&x)

	// Non-barrier access with no prior barrier -> no violation.
	if d.CheckRace(1, "read", ptr) {
		t.Fatal("unexpected violation with no barrier recorded")
	}

	// Record a barrier, then a subsequent access is at-or-after the barrier
	// timestamp, so no violation is reported.
	d.RecordAccess(1, "barrier", ptr)
	if d.CheckRace(1, "read", ptr) {
		t.Fatal("access after barrier wrongly flagged")
	}
	if got := d.GetRaceCount(); got != 0 {
		t.Fatalf("violations = %d, want 0", got)
	}

	// Non-barrier RecordAccess is ignored (no stored barrier for a fresh ptr).
	var y int
	yptr := unsafe.Pointer(&y)
	d.RecordAccess(1, "read", yptr)
	if d.CheckRace(1, "read", yptr) {
		t.Fatal("non-barrier record created a barrier")
	}

	d.Reset()
	if got := d.GetRaceCount(); got != 0 {
		t.Fatalf("post-reset violations = %d, want 0", got)
	}
}

// -----------------------------------------------------------------------------
// Global initializer + context wrappers
// -----------------------------------------------------------------------------

func TestInitializeRacePrevention(t *testing.T) {
	GlobalRacePrevention = nil
	InitializeRacePrevention()
	if GlobalRacePrevention == nil {
		t.Fatal("global not initialized")
	}
	// Idempotent: second call is a no-op and keeps the same instance.
	first := GlobalRacePrevention
	InitializeRacePrevention()
	if GlobalRacePrevention != first {
		t.Fatal("re-initialization replaced the global instance")
	}
}

func TestSafeFieldAccessWithContext(t *testing.T) {
	GlobalRacePrevention = nil // force lazy init inside the wrapper
	fd := &types.TypedFieldData{Key: "k", Value: "v", Type: types.TypedFieldString}

	got := SafeFieldAccessWithContext(context.Background(), fd, "read")
	if got == nil {
		t.Fatal("returned nil field data")
	}
	if got.Key != "k" {
		t.Fatalf("key = %q, want k", got.Key)
	}
	if GlobalRacePrevention == nil {
		t.Fatal("wrapper did not lazily initialize the global")
	}
}

func TestSafePipelineTransitionWithContext(t *testing.T) {
	GlobalRacePrevention = nil
	ok := SafePipelineTransitionWithContext(context.Background(), "old", "new")
	if !ok {
		t.Fatal("first transition should succeed")
	}
	if GlobalRacePrevention == nil {
		t.Fatal("wrapper did not lazily initialize the global")
	}
}

// -----------------------------------------------------------------------------
// RacePrevention methods (nil receiver + happy path)
// -----------------------------------------------------------------------------

func TestSafeFieldAccessNilReceiver(t *testing.T) {
	var rp *RacePrevention
	fd := &types.TypedFieldData{Key: "k"}
	if got := rp.SafeFieldAccess(fd, "read"); got != fd {
		t.Fatal("nil receiver must return the same field data unchanged")
	}
}

func TestSafeFieldAccessHappyPath(t *testing.T) {
	rp := newRP()
	fd := &types.TypedFieldData{Key: "user", Value: "bob", Type: types.TypedFieldString}

	got := rp.SafeFieldAccess(fd, "read")
	if got == nil || got.Key != "user" {
		t.Fatalf("unexpected field data: %+v", got)
	}

	// A second, conflicting write from a different recorded access can trip the
	// detector; regardless, the returned copy must preserve the payload.
	rp.recordAccess(999, "write", unsafe.Pointer(fd))
	got2 := rp.SafeFieldAccess(fd, "write")
	if got2 == nil || got2.Key != "user" {
		t.Fatalf("defensive/normal copy lost data: %+v", got2)
	}

	stats := rp.GetRaceStatistics()
	if stats.SyncOperations == 0 {
		t.Fatal("SyncOperations should have advanced")
	}
	if stats.BarrierOperations == 0 {
		t.Fatal("BarrierOperations should have advanced")
	}
}

func TestSafePipelineTransition(t *testing.T) {
	rp := newRP()
	if !rp.SafePipelineTransition("a", "b") {
		t.Fatal("first CAS transition should succeed")
	}
	stats := rp.GetRaceStatistics()
	if stats.LockFreeOperations == 0 {
		t.Fatal("LockFreeOperations should have advanced")
	}

	// Nil receiver returns true (no protection).
	var nilRP *RacePrevention
	if !nilRP.SafePipelineTransition("a", "b") {
		t.Fatal("nil receiver transition should return true")
	}
}

func TestSafeEntryPoolAccess(t *testing.T) {
	rp := newRP()
	entry := &types.LogEntry{Message: "m", Component: "c"}

	got := rp.SafeEntryPoolAccess(entry, "read")
	if got == nil || got.Message != "m" {
		t.Fatalf("unexpected entry: %+v", got)
	}

	// Force a detected race by pre-recording a conflicting cross-thread write.
	rp.recordAccess(424242, "write", unsafe.Pointer(entry))
	got2 := rp.SafeEntryPoolAccess(entry, "write")
	if got2 == nil || got2.Message != "m" {
		t.Fatalf("defensive entry copy lost data: %+v", got2)
	}

	// Nil receiver returns entry unchanged.
	var nilRP *RacePrevention
	if got := nilRP.SafeEntryPoolAccess(entry, "read"); got != entry {
		t.Fatal("nil receiver must return same entry")
	}
}

func TestSafeStrategyAccess(t *testing.T) {
	rp := newRP()
	rp.SafeStrategyAccess(7, "read")
	rp.SafeStrategyAccess(7, "write")
	if rp.GetRaceStatistics().SyncOperations == 0 {
		t.Fatal("strategy access should advance SyncOperations")
	}

	var nilRP *RacePrevention
	nilRP.SafeStrategyAccess(1, "read") // must not panic
}

func TestSafePipelineExecution(t *testing.T) {
	rp := newRP()
	rp.SafePipelineExecution(7, "read", "pipeline")
	rp.SafePipelineExecution(7, "write", "pipeline")
	if rp.GetRaceStatistics().SyncOperations == 0 {
		t.Fatal("pipeline execution should advance SyncOperations")
	}

	var nilRP *RacePrevention
	nilRP.SafePipelineExecution(1, "read", nil) // must not panic
}

// TestSafePipelineTransitionConcurrent drives many goroutines through the
// CAS-based state transition. Contention exercises the CAS-failure branch and
// validates the lock-free path is race-free under `go test -race`.
func TestSafePipelineTransitionConcurrent(t *testing.T) {
	rp := newRP()

	const goroutines = 32
	const iters = 50

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for g := 0; g < goroutines; g++ {
		go func() {
			defer wg.Done()
			for i := 0; i < iters; i++ {
				// Return value may be true or false depending on CAS contention;
				// both outcomes are valid — we only assert it never panics or
				// corrupts state (checked below and by the race detector).
				rp.SafePipelineTransition("old", "new")
			}
		}()
	}
	wg.Wait()

	// Version advances only on successful transitions; there must be at least
	// one success across all the work performed.
	if rp.version.Load() == 0 {
		t.Fatal("no successful pipeline transition recorded")
	}
	if rp.GetRaceStatistics().LockFreeOperations == 0 {
		t.Fatal("LockFreeOperations should have advanced under concurrency")
	}
}

// TestSafePipelineExecutionConcurrent runs concurrent pipeline executions to
// exercise the race-detection branch under contention and validate safety.
func TestSafePipelineExecutionConcurrent(t *testing.T) {
	rp := newRP()

	var wg sync.WaitGroup
	wg.Add(8)
	for g := 0; g < 8; g++ {
		go func(id int) {
			defer wg.Done()
			for i := 0; i < 40; i++ {
				op := "read"
				if i%2 == 0 {
					op = "write"
				}
				rp.SafePipelineExecution(uint64(id), op, "shared-pipeline")
			}
		}(g)
	}
	wg.Wait()

	if rp.GetRaceStatistics().SyncOperations == 0 {
		t.Fatal("SyncOperations should have advanced under concurrency")
	}
}

// -----------------------------------------------------------------------------
// Strategy registry
// -----------------------------------------------------------------------------

func TestStrategyRegistration(t *testing.T) {
	rp := newRP()

	id := rp.GenerateStrategyID()
	if id == 0 {
		t.Fatal("generated strategy id should be non-zero")
	}
	// Two consecutive IDs must differ (monotonic version + time based).
	if id2 := rp.GenerateStrategyID(); id2 == id {
		t.Fatal("consecutive strategy ids collided")
	}

	rp.RegisterStrategy(id)
	rp.UnregisterStrategy(id)
	// Unregister again (count already 0) exercises the guarded-decrement branch.
	rp.UnregisterStrategy(id)

	// Nil receiver: all strategy ops are safe no-ops.
	var nilRP *RacePrevention
	nilRP.RegisterStrategy(1)
	nilRP.UnregisterStrategy(1)
	if got := nilRP.GenerateStrategyID(); got != 0 {
		t.Fatalf("nil receiver GenerateStrategyID = %d, want 0", got)
	}
}

// -----------------------------------------------------------------------------
// Statistics / Reset / defensive copies
// -----------------------------------------------------------------------------

func TestGetRaceStatisticsNil(t *testing.T) {
	var rp *RacePrevention
	if got := rp.GetRaceStatistics(); got != (RacePreventionStats{}) {
		t.Fatalf("nil receiver stats = %+v, want zero value", got)
	}
}

func TestResetClearsState(t *testing.T) {
	rp := newRP()

	// Generate some activity across detectors and thread states.
	rp.SafeFieldAccess(&types.TypedFieldData{Key: "k"}, "read")
	rp.SafeStrategyAccess(1, "write")
	if rp.getActiveThreadCount() == 0 {
		t.Fatal("expected at least one active thread after access")
	}

	rp.Reset()

	stats := rp.GetRaceStatistics()
	if stats.SyncOperations != 0 || stats.LockFreeOperations != 0 ||
		stats.BarrierOperations != 0 || stats.LastAccessTime != 0 {
		t.Fatalf("metrics not cleared after Reset: %+v", stats)
	}
	if stats.ActiveThreads != 0 {
		t.Fatalf("active threads = %d after Reset, want 0", stats.ActiveThreads)
	}

	// Nil receiver Reset is a safe no-op.
	var nilRP *RacePrevention
	nilRP.Reset()
}

func TestCreateDefensiveCopyNil(t *testing.T) {
	rp := newRP()
	if got := rp.createDefensiveCopy(nil); got != nil {
		t.Fatal("defensive copy of nil field data should be nil")
	}
	if got := rp.createDefensiveEntryCopy(nil); got != nil {
		t.Fatal("defensive copy of nil entry should be nil")
	}

	src := &types.TypedFieldData{Key: "k", Value: 1, Type: types.TypedFieldInt64}
	cp := rp.createDefensiveCopy(src)
	if cp == src {
		t.Fatal("defensive copy must be a distinct pointer")
	}
	if cp.Key != src.Key || cp.Value != src.Value || cp.Type != src.Type {
		t.Fatalf("defensive copy lost data: %+v", cp)
	}

	entry := &types.LogEntry{Message: "hi"}
	ecp := rp.createDefensiveEntryCopy(entry)
	if ecp == entry {
		t.Fatal("defensive entry copy must be a distinct pointer")
	}
	if ecp.Message != "hi" {
		t.Fatalf("defensive entry copy lost data: %+v", ecp)
	}
}

// -----------------------------------------------------------------------------
// Thread-state caching
// -----------------------------------------------------------------------------

func TestGetThreadStateReuse(t *testing.T) {
	rp := newRP()
	// Detectors + thread map wired; two calls in the same goroutine may hit
	// distinct routineID() values (time-based), so we just assert non-nil and
	// that accessCount tracking works when the same state is updated.
	s := rp.getThreadState()
	if s == nil {
		t.Fatal("thread state was nil")
	}
	before := s.accessCount.Load()
	rp.updateThreadState(s, "read")
	rp.updateThreadState(s, "write")
	if s.accessCount.Load() != before+2 {
		t.Fatalf("accessCount = %d, want %d", s.accessCount.Load(), before+2)
	}
	if s.readCount.Load() != 1 || s.writeCount.Load() != 1 {
		t.Fatalf("read/write counts = %d/%d, want 1/1", s.readCount.Load(), s.writeCount.Load())
	}
}
