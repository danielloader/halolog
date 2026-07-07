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
// Package pool provides object pooling
// @author Admilson B. F. Cossa

package pool

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-gen-ecosystem/halolog/types"
)

// waitFor polls cond until it returns true or the timeout elapses.
// Used to synchronise with the asynchronous worker goroutines without
// relying on brittle fixed sleeps.
func waitFor(t *testing.T, timeout time.Duration, cond func() bool) bool {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return true
		}
		time.Sleep(time.Millisecond)
	}
	return cond()
}

// TestWorkerPool_SubmitProcessesEntry verifies that a submitted entry is
// actually delivered to the configured writer and then released.
func TestWorkerPool_SubmitProcessesEntry(t *testing.T) {
	wp := newWorkerPool(2)
	defer wp.Close()

	var processed atomic.Int64
	var gotMessage atomic.Value
	wp.SetWriter(func(e *types.LogEntry) {
		gotMessage.Store(e.Message)
		processed.Add(1)
	})

	entry := AcquireEntry()
	entry.Message = "hello-worker"
	entry.Level = types.InfoLevel

	if ok := wp.Submit(entry); !ok {
		t.Fatal("Submit returned false for a healthy pool")
	}

	if !waitFor(t, time.Second, func() bool { return processed.Load() == 1 }) {
		t.Fatalf("expected 1 processed entry, got %d", processed.Load())
	}
	if got := gotMessage.Load(); got != "hello-worker" {
		t.Errorf("writer saw message %q, want %q", got, "hello-worker")
	}
}

// TestWorkerPool_SetWriterGetWriter verifies the writer round-trips through
// the atomic pointer and can be replaced.
func TestWorkerPool_SetWriterGetWriter(t *testing.T) {
	wp := newWorkerPool(1)
	defer wp.Close()

	if wp.GetWriter() != nil {
		t.Error("GetWriter should be nil before SetWriter")
	}

	called := make(chan string, 1)
	wp.SetWriter(func(e *types.LogEntry) { called <- e.Message })

	if wp.GetWriter() == nil {
		t.Fatal("GetWriter returned nil after SetWriter")
	}

	// Replace the writer and ensure the new one is used.
	wp.SetWriter(func(e *types.LogEntry) { called <- "replaced:" + e.Message })

	entry := AcquireEntry()
	entry.Message = "msg"
	if !wp.Submit(entry) {
		t.Fatal("Submit failed")
	}

	select {
	case got := <-called:
		if got != "replaced:msg" {
			t.Errorf("got %q, want %q", got, "replaced:msg")
		}
	case <-time.After(time.Second):
		t.Fatal("writer was never invoked")
	}
}

// TestWorkerPool_SubmitBeforeWriterFails asserts Submit refuses entries when
// no writer is configured, so the caller can fall back to synchronous writes.
func TestWorkerPool_SubmitBeforeWriterFails(t *testing.T) {
	wp := newWorkerPool(1)
	defer wp.Close()

	entry := AcquireEntry()
	entry.Message = "no-writer"
	if wp.Submit(entry) {
		t.Error("Submit should return false when no writer is set")
	}
	ReleaseEntry(entry)
}

// TestWorkerPool_SubmitAfterCloseFails asserts a stopped pool rejects entries.
func TestWorkerPool_SubmitAfterCloseFails(t *testing.T) {
	wp := newWorkerPool(1)
	wp.SetWriter(func(e *types.LogEntry) {})
	wp.Close()

	if !wp.isStopped() {
		t.Fatal("pool should report stopped after Close")
	}

	entry := AcquireEntry()
	if wp.Submit(entry) {
		t.Error("Submit should return false after Close")
	}
	ReleaseEntry(entry)
}

// TestWorkerPool_CloseIdempotent asserts Close can be safely called multiple
// times without panicking on the double close of channels.
func TestWorkerPool_CloseIdempotent(t *testing.T) {
	wp := newWorkerPool(2)
	wp.SetWriter(func(e *types.LogEntry) {})

	wp.Close()
	// Second and third Close must be no-ops, not panics.
	wp.Close()
	wp.Close()

	if !wp.isStopped() {
		t.Error("pool should remain stopped")
	}
}

// TestWorkerPool_CloseDrainsPendingEntries verifies graceful shutdown processes
// entries already buffered in the channel before returning.
func TestWorkerPool_CloseDrainsPendingEntries(t *testing.T) {
	// Single worker + a gate so we can queue entries while the worker is busy,
	// then confirm Close drains them all.
	wp := newWorkerPool(1)

	var processed atomic.Int64
	release := make(chan struct{})
	var once sync.Once
	wp.SetWriter(func(e *types.LogEntry) {
		// Block the first entry until we've queued the rest.
		once.Do(func() { <-release })
		processed.Add(1)
	})

	const total = 20
	for i := 0; i < total; i++ {
		entry := AcquireEntry()
		entry.Message = "drain"
		if !wp.Submit(entry) {
			// Buffer is generously sized (64), so this should not happen.
			ReleaseEntry(entry)
			t.Fatalf("Submit unexpectedly failed at i=%d", i)
		}
	}

	// Unblock the worker and close; Close must wait for the full drain.
	close(release)
	wp.Close()

	if processed.Load() != total {
		t.Errorf("Close drained %d entries, want %d", processed.Load(), total)
	}
}

// TestWorkerPool_SubmitBackpressure fills the buffer while the sole worker is
// blocked, forcing Submit to return false (back-pressure path).
func TestWorkerPool_SubmitBackpressure(t *testing.T) {
	wp := newWorkerPool(1) // buffer capacity = 1*64 = 64
	defer wp.Close()

	block := make(chan struct{})
	started := make(chan struct{}, 1)
	wp.SetWriter(func(e *types.LogEntry) {
		select {
		case started <- struct{}{}:
		default:
		}
		<-block // hold the worker so the buffer fills
	})

	// First submit gets picked up by the worker (which then blocks on <-block).
	first := AcquireEntry()
	first.Message = "block-me"
	if !wp.Submit(first) {
		close(block)
		t.Fatal("first Submit failed")
	}
	<-started // ensure the worker is now blocked

	// Now flood until Submit returns false because the buffer is saturated.
	sawBackpressure := false
	for i := 0; i < cap(wp.taskCh)+10; i++ {
		e := AcquireEntry()
		e.Message = "flood"
		if !wp.Submit(e) {
			ReleaseEntry(e)
			sawBackpressure = true
			break
		}
	}
	close(block) // let the worker drain

	if !sawBackpressure {
		t.Error("expected Submit to report back-pressure when buffer is full")
	}
}

// TestWorkerPool_BufferMetrics exercises the observability accessors on a live
// pool with a known configuration.
func TestWorkerPool_BufferMetrics(t *testing.T) {
	wp := newWorkerPool(2) // buffer capacity = 2*64 = 128
	defer wp.Close()

	if got := wp.GetBufferSize(); got != 128 {
		t.Errorf("GetBufferSize = %d, want 128", got)
	}
	if got := wp.GetBufferCurrent(); got != 0 {
		t.Errorf("GetBufferCurrent = %d, want 0 on a fresh pool", got)
	}
	if got := wp.GetBufferUsage(); got != 0 {
		t.Errorf("GetBufferUsage = %v, want 0 on a fresh pool", got)
	}

	// Block the workers so a queued entry stays in the buffer long enough to
	// observe a non-zero usage.
	block := make(chan struct{})
	busy := make(chan struct{}, 2)
	wp.SetWriter(func(e *types.LogEntry) {
		busy <- struct{}{}
		<-block
	})

	// Saturate both workers first.
	for i := 0; i < 2; i++ {
		e := AcquireEntry()
		e.Message = "occupy"
		if !wp.Submit(e) {
			close(block)
			t.Fatal("failed to occupy workers")
		}
	}
	<-busy
	<-busy

	// This one should sit in the buffer.
	queued := AcquireEntry()
	queued.Message = "queued"
	if !wp.Submit(queued) {
		close(block)
		t.Fatal("failed to queue extra entry")
	}

	if !waitFor(t, time.Second, func() bool { return wp.GetBufferCurrent() >= 1 }) {
		t.Errorf("GetBufferCurrent = %d, want >= 1 while workers blocked", wp.GetBufferCurrent())
	}
	if usage := wp.GetBufferUsage(); usage <= 0 || usage > 1 {
		t.Errorf("GetBufferUsage = %v, want in (0,1]", usage)
	}

	close(block)
}

// TestWorkerPool_BufferMetricsNilChannel guards the nil-channel branches of the
// metrics accessors using a zero-value pool (never started).
func TestWorkerPool_BufferMetricsNilChannel(t *testing.T) {
	var wp WorkerPool // taskCh is nil

	if got := wp.GetBufferSize(); got != 0 {
		t.Errorf("GetBufferSize on nil channel = %d, want 0", got)
	}
	if got := wp.GetBufferCurrent(); got != 0 {
		t.Errorf("GetBufferCurrent on nil channel = %d, want 0", got)
	}
	if got := wp.GetBufferUsage(); got != 0 {
		t.Errorf("GetBufferUsage on nil channel = %v, want 0", got)
	}
}

// TestNewWorkerPool_ClampsWorkerCount verifies non-positive counts are clamped
// to a single worker so the pool remains functional.
func TestNewWorkerPool_ClampsWorkerCount(t *testing.T) {
	for _, count := range []int{0, -5} {
		wp := newWorkerPool(count)
		// buffer size must reflect the clamped worker count of 1 (1*64).
		if got := wp.GetBufferSize(); got != 64 {
			t.Errorf("newWorkerPool(%d) buffer size = %d, want 64", count, got)
		}

		done := make(chan struct{}, 1)
		wp.SetWriter(func(e *types.LogEntry) { done <- struct{}{} })
		e := AcquireEntry()
		e.Message = "clamped"
		if !wp.Submit(e) {
			wp.Close()
			t.Fatalf("Submit failed for clamped pool (count=%d)", count)
		}
		select {
		case <-done:
		case <-time.After(time.Second):
			wp.Close()
			t.Fatalf("clamped pool (count=%d) never processed entry", count)
		}
		wp.Close()
	}
}

// TestGetPreallocatedPool_ReturnsCommonCounts verifies the pre-allocation cache
// returns live pools for the documented common worker counts and nil otherwise.
func TestGetPreallocatedPool_ReturnsCommonCounts(t *testing.T) {
	defer CloseAllWorkerPools()

	for _, count := range []int{1, 2, 4, 8, 16, 32} {
		wp := GetPreallocatedPool(count)
		if wp == nil {
			t.Errorf("GetPreallocatedPool(%d) returned nil, want a pool", count)
			continue
		}
		want := count * 64
		if got := wp.GetBufferSize(); got != want {
			t.Errorf("prealloc pool(%d) buffer size = %d, want %d", count, got, want)
		}
	}

	// An uncommon count is not pre-allocated.
	if wp := GetPreallocatedPool(7); wp != nil {
		t.Errorf("GetPreallocatedPool(7) = %v, want nil for uncommon count", wp)
	}
}

// TestGetActiveWorkerPool_ReusesInstances verifies that repeated requests for
// the same worker count return the identical (cached) pool instance.
func TestGetActiveWorkerPool_ReusesInstances(t *testing.T) {
	defer CloseAllWorkerPools()

	a := GetActiveWorkerPool(4)
	if a == nil {
		t.Fatal("GetActiveWorkerPool(4) returned nil")
	}
	b := GetActiveWorkerPool(4)
	if a != b {
		t.Error("GetActiveWorkerPool(4) returned different instances; expected reuse")
	}

	// Zero is clamped to 1 internally and must yield a usable pool.
	z := GetActiveWorkerPool(0)
	if z == nil {
		t.Fatal("GetActiveWorkerPool(0) returned nil")
	}
	if z.GetBufferSize() != 64 {
		t.Errorf("clamped active pool buffer size = %d, want 64", z.GetBufferSize())
	}
}

// TestGetActiveWorkerPool_UncommonCountCreatesNew verifies an uncommon count
// (not pre-allocated) is created on demand and then cached.
func TestGetActiveWorkerPool_UncommonCountCreatesNew(t *testing.T) {
	defer CloseAllWorkerPools()

	wp := GetActiveWorkerPool(5) // 5 is not a pre-allocated common count
	if wp == nil {
		t.Fatal("GetActiveWorkerPool(5) returned nil")
	}
	if got := wp.GetBufferSize(); got != 5*64 {
		t.Errorf("on-demand pool buffer size = %d, want %d", got, 5*64)
	}

	// It must be routed through and process work.
	done := make(chan struct{}, 1)
	wp.SetWriter(func(e *types.LogEntry) { done <- struct{}{} })
	e := AcquireEntry()
	e.Message = "uncommon"
	if !wp.Submit(e) {
		t.Fatal("Submit failed on on-demand pool")
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("on-demand pool never processed entry")
	}

	// Cached: second call returns same instance.
	if again := GetActiveWorkerPool(5); again != wp {
		t.Error("expected cached instance for count=5 on second call")
	}
}

// TestStopActiveWorkerPool_RemovesFromTracking verifies stopping an active pool
// closes it and evicts it from the active-tracking map.
//
// An *uncommon* worker count is used deliberately: uncommon counts are created
// fresh via newWorkerPool (not cached in the pre-allocated map), so after
// StopActiveWorkerPool evicts the entry a subsequent GetActiveWorkerPool builds
// a brand-new, non-stopped pool. (For common counts such as 2, a re-request
// would return the still-stopped pre-allocated pool, which is a separate design
// characteristic; see the report note.)
func TestStopActiveWorkerPool_RemovesFromTracking(t *testing.T) {
	defer CloseAllWorkerPools()

	const uncommon = 6 // not in {1,2,4,8,16,32}
	first := GetActiveWorkerPool(uncommon)
	if first == nil {
		t.Fatalf("GetActiveWorkerPool(%d) returned nil", uncommon)
	}

	StopActiveWorkerPool(uncommon)
	if !first.isStopped() {
		t.Error("StopActiveWorkerPool did not close the pool")
	}

	// A fresh request for the evicted uncommon count must build a new, live pool.
	second := GetActiveWorkerPool(uncommon)
	if second == nil {
		t.Fatalf("GetActiveWorkerPool(%d) returned nil after stop", uncommon)
	}
	if second == first {
		t.Error("expected a new instance after StopActiveWorkerPool evicted the old one")
	}
	if second.isStopped() {
		t.Error("GetActiveWorkerPool returned a stopped pool after StopActiveWorkerPool")
	}

	// Stopping an unknown count is a harmless no-op.
	StopActiveWorkerPool(999)
}

// TestCloseAllWorkerPools_StopsEverything verifies the bulk shutdown helper
// closes both active and pre-allocated pools and resets tracking state.
func TestCloseAllWorkerPools_StopsEverything(t *testing.T) {
	// Populate both active and pre-allocated caches.
	active := GetActiveWorkerPool(3) // uncommon -> newly created, tracked as active
	prealloc := GetPreallocatedPool(8)
	if active == nil || prealloc == nil {
		t.Fatal("failed to populate pool caches")
	}

	CloseAllWorkerPools()

	if !active.isStopped() {
		t.Error("active pool not stopped by CloseAllWorkerPools")
	}
	if !prealloc.isStopped() {
		t.Error("pre-allocated pool not stopped by CloseAllWorkerPools")
	}

	// Calling again after everything is cleared must not panic.
	CloseAllWorkerPools()
}

// TestInitWorkerPools_Idempotent verifies the sync.Once-guarded initialisation
// can be invoked repeatedly without spawning duplicate pools.
func TestInitWorkerPools_Idempotent(t *testing.T) {
	defer CloseAllWorkerPools()

	InitWorkerPools()
	first := GetPreallocatedPool(16)
	InitWorkerPools() // second call must be a no-op due to sync.Once
	second := GetPreallocatedPool(16)

	if first == nil || second == nil {
		t.Fatal("pre-allocated pool(16) missing after InitWorkerPools")
	}
	if first != second {
		t.Error("InitWorkerPools recreated pools; expected identical instances")
	}
}
