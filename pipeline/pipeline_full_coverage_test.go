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
// Package pipeline tests exercise the FullPipeline feature-rich path.
// @author Admilson B. F. Cossa

package pipeline

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-gen-ecosystem/halolog/types"
)

// stubSampler is a controllable Sampler for testing the sampling branch.
type stubSampler struct {
	sample atomic.Bool
	calls  atomic.Int64
	rate   float64
}

func (s *stubSampler) ShouldSample(entry *types.LogEntry) bool {
	s.calls.Add(1)
	return s.sample.Load()
}
func (s *stubSampler) GetRate() float64     { return s.rate }
func (s *stubSampler) SetRate(rate float64) { s.rate = rate }

// stubMasker is a controllable PIIMasker that records Apply invocations and
// rewrites the message so we can prove Apply ran.
type stubMasker struct {
	applied atomic.Int64
}

func (m *stubMasker) Apply(entry *types.LogEntry) {
	m.applied.Add(1)
	entry.Message = "MASKED:" + entry.Message
}
func (m *stubMasker) MaskField(f *types.TypedFieldData) *types.TypedFieldData { return f }
func (m *stubMasker) MaskFields(f []types.TypedFieldData) []types.TypedFieldData {
	return f
}
func (m *stubMasker) MaskString(input string) string  { return input }
func (m *stubMasker) AddRule(_, _, _ string) error    { return nil }
func (m *stubMasker) RemoveRule(_ string) error       { return nil }
func (m *stubMasker) AddPattern(_, _, _ string) error { return nil }
func (m *stubMasker) RemovePattern(_ string)          {}
func (m *stubMasker) GetPatterns() []string           { return nil }
func (m *stubMasker) Clone() types.PIIMasker          { return m }

// TestNewFullPipeline_Defaults verifies construction wires up registries and a
// non-nil async queue, and defaults all features off.
func TestNewFullPipeline_Defaults(t *testing.T) {
	var sink []captured
	fp := NewFullPipeline([]types.Adapter{newCapturingAdapter(&sink)}, nil)

	if fp.SensitiveFields == nil {
		t.Fatal("expected SensitiveFields registry to be initialized")
	}
	if fp.ColorRegistry == nil {
		t.Fatal("expected ColorRegistry to be initialized")
	}
	if fp.AsyncQueue == nil {
		t.Fatal("expected AsyncQueue to be initialized")
	}
	if fp.HasMasking || fp.HasSampling || fp.HasAsync || fp.HasColors {
		t.Fatal("expected all feature flags off by default")
	}
}

// TestFullPipeline_WriteSync verifies the synchronous no-feature path delivers
// the entry and increments ProcessedCount.
func TestFullPipeline_WriteSync(t *testing.T) {
	var sink []captured
	fp := NewFullPipeline([]types.Adapter{newCapturingAdapter(&sink)}, nil)
	cc := newTestClock(t)

	fields := []types.TypedFieldData{{Key: "route", Value: "/health"}}
	fp.Write(cc, types.InfoLevel, "req", fields, len(fields))

	if len(sink) != 1 {
		t.Fatalf("expected 1 write, got %d", len(sink))
	}
	if sink[0].numStat != 1 {
		t.Fatalf("expected 1 field delivered, got %d", sink[0].numStat)
	}
	if fp.ProcessedCount.Load() != 1 {
		t.Fatalf("expected ProcessedCount=1, got %d", fp.ProcessedCount.Load())
	}
}

// TestFullPipeline_WriteMasking verifies the masking branch runs the masker and
// mutates the entry before it reaches the adapter.
func TestFullPipeline_WriteMasking(t *testing.T) {
	var masked string
	var got int
	adapter := &types.FuncAdapter{
		WriteFunc: func(e *types.LogEntry) error {
			masked = e.Message
			got++
			return nil
		},
	}
	fp := NewFullPipeline([]types.Adapter{adapter}, nil)
	m := &stubMasker{}
	fp.HasMasking = true
	fp.Masker = m
	cc := newTestClock(t)

	fp.Write(cc, types.InfoLevel, "secret", nil, 0)

	if got != 1 {
		t.Fatalf("expected 1 write, got %d", got)
	}
	if m.applied.Load() != 1 {
		t.Fatalf("expected masker Apply called once, got %d", m.applied.Load())
	}
	if masked != "MASKED:secret" {
		t.Fatalf("expected masked message, got %q", masked)
	}
}

// TestFullPipeline_SamplingDrops verifies that a sampler returning false causes
// the entry to be dropped (no adapter write) and released back to the pool.
func TestFullPipeline_SamplingDrops(t *testing.T) {
	var writes int
	adapter := &types.FuncAdapter{
		WriteFunc: func(e *types.LogEntry) error { writes++; return nil },
	}
	fp := NewFullPipeline([]types.Adapter{adapter}, nil)
	s := &stubSampler{}
	s.sample.Store(false) // drop everything
	fp.HasSampling = true
	fp.Sampler = s
	cc := newTestClock(t)

	fp.Write(cc, types.InfoLevel, "dropme", nil, 0)

	if writes != 0 {
		t.Fatalf("expected entry dropped, but adapter wrote %d times", writes)
	}
	if s.calls.Load() != 1 {
		t.Fatalf("expected sampler consulted once, got %d", s.calls.Load())
	}
	// Dropped entries are not counted as processed.
	if fp.ProcessedCount.Load() != 0 {
		t.Fatalf("expected ProcessedCount=0 for dropped log, got %d", fp.ProcessedCount.Load())
	}
}

// TestFullPipeline_SamplingKeeps verifies a sampler returning true passes the
// entry through to the adapter.
func TestFullPipeline_SamplingKeeps(t *testing.T) {
	var writes int
	adapter := &types.FuncAdapter{
		WriteFunc: func(e *types.LogEntry) error { writes++; return nil },
	}
	fp := NewFullPipeline([]types.Adapter{adapter}, nil)
	s := &stubSampler{}
	s.sample.Store(true)
	fp.HasSampling = true
	fp.Sampler = s
	cc := newTestClock(t)

	// Include fields to exercise the sampleEntry field-copy branch (<=32 fields).
	fields := []types.TypedFieldData{{Key: "a", Value: 1}, {Key: "b", Value: 2}}
	fp.Write(cc, types.InfoLevel, "keepme", fields, len(fields))

	if writes != 1 {
		t.Fatalf("expected 1 write, got %d", writes)
	}
	if fp.ProcessedCount.Load() != 1 {
		t.Fatalf("expected ProcessedCount=1, got %d", fp.ProcessedCount.Load())
	}
}

// TestFullPipeline_BuiltinSampling exercises shouldSample's built-in rate path
// (no custom sampler) across boundary rates.
func TestFullPipeline_BuiltinSampling(t *testing.T) {
	var sink []captured
	fp := NewFullPipeline([]types.Adapter{newCapturingAdapter(&sink)}, nil)

	// rate <= 0 -> sample everything (true).
	fp.SetSampleRate(0)
	if !fp.shouldSample(&types.LogEntry{}) {
		t.Fatal("rate=0 should sample everything")
	}

	// rate >= 100 -> sample everything (true).
	fp.SetSampleRate(100)
	if !fp.shouldSample(&types.LogEntry{}) {
		t.Fatal("rate=100 should sample everything")
	}

	// rate = 50 -> deterministic modulo decision; every 2nd call sampled.
	fp.SetSampleRate(50)
	fp.SampleCount.Store(0)
	first := fp.shouldSample(&types.LogEntry{})  // count=1 -> 1%2 != 0 -> false
	second := fp.shouldSample(&types.LogEntry{}) // count=2 -> 2%2 == 0 -> true
	if first {
		t.Fatal("expected first (count=1) not sampled at rate=50")
	}
	if !second {
		t.Fatal("expected second (count=2) sampled at rate=50")
	}
}

// TestFullPipeline_SetSampleRateClamping verifies rate is clamped to [0,100].
func TestFullPipeline_SetSampleRateClamping(t *testing.T) {
	var sink []captured
	fp := NewFullPipeline([]types.Adapter{newCapturingAdapter(&sink)}, nil)

	fp.SetSampleRate(-5)
	if fp.SampleRate.Load() != 0 {
		t.Fatalf("expected clamp to 0, got %d", fp.SampleRate.Load())
	}
	fp.SetSampleRate(250)
	if fp.SampleRate.Load() != 100 {
		t.Fatalf("expected clamp to 100, got %d", fp.SampleRate.Load())
	}
	fp.SetSampleRate(37)
	if fp.SampleRate.Load() != 37 {
		t.Fatalf("expected 37, got %d", fp.SampleRate.Load())
	}
}

// TestFullPipeline_Colors verifies the color feature branch of applyFeatures is
// safe (delegated to formatters, no entry mutation).
func TestFullPipeline_Colors(t *testing.T) {
	var writes int
	adapter := &types.FuncAdapter{
		WriteFunc: func(e *types.LogEntry) error { writes++; return nil },
	}
	fp := NewFullPipeline([]types.Adapter{adapter}, nil)
	fp.HasColors = true
	cc := newTestClock(t)

	fp.Write(cc, types.InfoLevel, "colored", nil, 0)
	if writes != 1 {
		t.Fatalf("expected 1 write with colors on, got %d", writes)
	}
}

// TestFullPipeline_Async verifies the async path: entries are queued and drained
// by a worker, then a clean shutdown via Reset.
func TestFullPipeline_Async(t *testing.T) {
	var mu sync.Mutex
	var messages []string
	adapter := &types.FuncAdapter{
		WriteFunc: func(e *types.LogEntry) error {
			mu.Lock()
			messages = append(messages, e.Message)
			mu.Unlock()
			return nil
		},
	}
	fp := NewFullPipeline([]types.Adapter{adapter}, nil)
	fp.HasAsync = true
	fp.StartAsyncWorkers(2)
	cc := newTestClock(t)

	const n = 20
	for i := 0; i < n; i++ {
		fp.Write(cc, types.InfoLevel, "async-msg", nil, 0)
	}

	// Wait until all queued entries drain (or timeout).
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		count := len(messages)
		mu.Unlock()
		if count >= n {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	mu.Lock()
	got := len(messages)
	mu.Unlock()
	if got != n {
		t.Fatalf("expected %d async writes drained, got %d", n, got)
	}
	if fp.ProcessedCount.Load() != int64(n) {
		t.Fatalf("expected ProcessedCount=%d, got %d", n, fp.ProcessedCount.Load())
	}

	// Reset must stop workers and drain cleanly without hanging.
	fp.Reset()
	if len(fp.Adapters) != 0 {
		t.Fatalf("expected adapters cleared after Reset, got %d", len(fp.Adapters))
	}
	if fp.ProcessedCount.Load() != 0 {
		t.Fatalf("expected metrics reset after Reset, got processed=%d", fp.ProcessedCount.Load())
	}
}

// TestFullPipeline_ProcessFieldsOverflow drives processFields' dynamic overflow
// branch through a real Write.
func TestFullPipeline_ProcessFieldsOverflow(t *testing.T) {
	var dynamicCount int
	adapter := &types.FuncAdapter{
		WriteFunc: func(e *types.LogEntry) error {
			dynamicCount = len(e.Fields)
			return nil
		},
	}
	fp := NewFullPipeline([]types.Adapter{adapter}, nil)
	cc := newTestClock(t)

	const n = 40 // exceeds static capacity
	fields := make([]types.TypedFieldData, n)
	for i := range fields {
		fields[i] = types.TypedFieldData{Key: "k", Value: i}
	}
	fp.Write(cc, types.InfoLevel, "overflow", fields, n)

	if dynamicCount != n {
		t.Fatalf("expected %d overflow fields, got %d", n, dynamicCount)
	}
}

// TestFullPipeline_GetMetrics verifies the metrics snapshot reflects processed
// counts and queue size.
func TestFullPipeline_GetMetrics(t *testing.T) {
	var sink []captured
	fp := NewFullPipeline([]types.Adapter{newCapturingAdapter(&sink)}, nil)
	cc := newTestClock(t)

	fp.Write(cc, types.InfoLevel, "m1", nil, 0)
	fp.Write(cc, types.InfoLevel, "m2", nil, 0)

	m := fp.GetMetrics()
	if m.ProcessedCount != 2 {
		t.Fatalf("expected ProcessedCount=2, got %d", m.ProcessedCount)
	}
	if m.QueueSize != 0 {
		t.Fatalf("expected empty queue, got %d", m.QueueSize)
	}
}

// TestFullPipeline_ProcessAsyncQueueFullFallback verifies processAsync falls back
// to synchronous processing when the async queue is full.
func TestFullPipeline_ProcessAsyncQueueFullFallback(t *testing.T) {
	var writes int32
	adapter := &types.FuncAdapter{
		WriteFunc: func(e *types.LogEntry) error {
			atomic.AddInt32(&writes, 1)
			return nil
		},
	}
	fp := NewFullPipeline([]types.Adapter{adapter}, nil)

	// Replace the queue with a tiny full one so the default branch is taken.
	fp.AsyncQueue = make(chan *types.LogEntry, 1)
	fp.AsyncQueue <- &types.LogEntry{} // fill it (no workers running)

	// processAsync should fall back to processSync (which writes + releases).
	entry := &types.LogEntry{Message: "fallback"}
	fp.processAsync(entry)

	if atomic.LoadInt32(&writes) != 1 {
		t.Fatalf("expected sync fallback to write once, got %d", atomic.LoadInt32(&writes))
	}
}

// TestIsFullPathEligible verifies the eligibility predicate.
func TestIsFullPathEligible(t *testing.T) {
	testCases := []struct {
		name     string
		masking  bool
		sampling bool
		async    bool
		colors   bool
		want     bool
	}{
		{name: "no_features", want: false},
		{name: "masking", masking: true, want: true},
		{name: "sampling", sampling: true, want: true},
		{name: "async", async: true, want: true},
		{name: "colors", colors: true, want: true},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := IsFullPathEligible(tc.masking, tc.sampling, tc.async, tc.colors)
			if got != tc.want {
				t.Fatalf("IsFullPathEligible=%v, want %v", got, tc.want)
			}
		})
	}
}
