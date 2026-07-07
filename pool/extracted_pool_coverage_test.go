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
	"testing"

	"github.com/go-gen-ecosystem/halolog/types"
)

// TestExtractedPerPState_SetLevelSetTimestamp exercises the cached-level and
// cached-timestamp setters used by the logger hot path.
func TestExtractedPerPState_SetLevelSetTimestamp(t *testing.T) {
	pool := NewExtractedPool()
	state := pool.GetPerPState()
	defer pool.PutPerPState(state)

	state.SetLevel(types.WarnLevel)
	if state.level != types.WarnLevel {
		t.Errorf("SetLevel: got %v, want %v", state.level, types.WarnLevel)
	}

	const ts int64 = 1_700_000_000_123456789
	state.SetTimestamp(ts)
	if state.timestamp != ts {
		t.Errorf("SetTimestamp: got %d, want %d", state.timestamp, ts)
	}

	// Overwrite to confirm the setters are not one-shot.
	state.SetLevel(types.ErrorLevel)
	state.SetTimestamp(0)
	if state.level != types.ErrorLevel || state.timestamp != 0 {
		t.Errorf("setters did not overwrite: level=%v ts=%d", state.level, state.timestamp)
	}
}

// TestExtractedPool_PutPerPStateNil verifies the nil guard in PutPerPState.
func TestExtractedPool_PutPerPStateNil(t *testing.T) {
	pool := NewExtractedPool()
	pool.PutPerPState(nil) // must not panic
}

// TestExtractedPool_FallbackWhenAllSlotsBusy forces the slot-miss branch of
// GetPerPState by holding every fast-path slot for a single P, then verifies
// the fallback state (poolIndex == -1) round-trips correctly through
// PutPerPState back into the fallback sync.Pool.
func TestExtractedPool_FallbackWhenAllSlotsBusy(t *testing.T) {
	// Single P with a single slot => trivially exhaustible fast path.
	pool := &ExtractedPool{
		shards:     make([]ExtractedPerPState, 1),
		numP:       1,
		slotsPerP:  1,
		totalSlots: 1,
		fallbackPool: sync.Pool{
			New: func() interface{} { return &ExtractedPerPState{poolIndex: -1} },
		},
	}
	pool.shards[0].poolIndex = 0

	// Acquire the only fast-path slot and hold it.
	fast := pool.GetPerPState()
	if fast.poolIndex != 0 {
		t.Fatalf("first acquisition should be a fast-path slot, poolIndex=%d", fast.poolIndex)
	}

	// The next acquisition cannot find a free fast-path slot and must fall back.
	fallback := pool.GetPerPState()
	if fallback == nil {
		t.Fatal("fallback acquisition returned nil")
	}
	if fallback.poolIndex != -1 {
		t.Errorf("expected fallback state (poolIndex=-1), got %d", fallback.poolIndex)
	}

	if pool.slotMisses.Load() == 0 {
		t.Error("expected slotMisses to be incremented on fallback")
	}
	if pool.fastPathHits.Load() == 0 {
		t.Error("expected fastPathHits to be incremented on the first acquisition")
	}

	// Returning the fallback state must route into the fallback pool, not flip
	// an inUse flag on a real shard.
	pool.PutPerPState(fallback)

	// Returning the fast-path slot must release it for reuse.
	pool.PutPerPState(fast)

	reused := pool.GetPerPState()
	if reused.poolIndex != 0 {
		t.Errorf("after releasing the fast slot, expected it back (poolIndex=0), got %d", reused.poolIndex)
	}
	pool.PutPerPState(reused)
}

// TestExtractedFieldBuilder_OverflowIgnoresExtraFields verifies WithField
// silently drops writes once the static buffer (capacity 16) is full, matching
// the bounded hot-path contract.
func TestExtractedFieldBuilder_OverflowIgnoresExtraFields(t *testing.T) {
	pool := NewExtractedPool()
	builder := NewExtractedFieldBuilder(pool)
	defer builder.Reset()

	// Write more than the static capacity of 16.
	const attempts = 25
	for i := 0; i < attempts; i++ {
		builder.WithField("k", i)
	}

	entry := builder.GetEntry()
	if entry == nil {
		t.Fatal("expected non-nil entry after writing fields")
	}
	if entry.StaticFieldCount != 16 {
		t.Errorf("StaticFieldCount = %d, want 16 (bounded capacity)", entry.StaticFieldCount)
	}
	// The first 16 writes must be preserved in order.
	for i := 0; i < 16; i++ {
		if entry.StaticFields[i].Value != i {
			t.Errorf("StaticFields[%d].Value = %v, want %d", i, entry.StaticFields[i].Value, i)
		}
	}
}

// TestExtractedFieldBuilder_ResetIsIdempotent verifies Reset is safe to call
// repeatedly and before any field has been written (state == nil path).
func TestExtractedFieldBuilder_ResetIsIdempotent(t *testing.T) {
	pool := NewExtractedPool()
	builder := NewExtractedFieldBuilder(pool)

	// Reset before any WithField: state is nil, must be a no-op.
	builder.Reset()

	builder.WithField("a", 1)
	if builder.GetEntry() == nil {
		t.Fatal("entry should be non-nil after WithField")
	}
	builder.Reset()
	if builder.GetEntry() != nil {
		t.Error("GetEntry should be nil after Reset")
	}
	// Second Reset must remain a no-op.
	builder.Reset()
}

// TestExtractedFieldBuilder_ReacquiresAfterReset verifies a builder can be
// reused after Reset: a fresh WithField re-acquires per-P state and starts a
// clean entry.
func TestExtractedFieldBuilder_ReacquiresAfterReset(t *testing.T) {
	pool := NewExtractedPool()
	builder := NewExtractedFieldBuilder(pool)

	builder.WithField("first", 1).WithField("second", 2)
	if got := builder.GetEntry().StaticFieldCount; got != 2 {
		t.Fatalf("first cycle field count = %d, want 2", got)
	}
	builder.Reset()

	builder.WithField("third", 3)
	e := builder.GetEntry()
	if e == nil {
		t.Fatal("entry nil after re-acquire")
	}
	if e.StaticFieldCount != 1 {
		t.Errorf("second cycle field count = %d, want 1 (fresh entry)", e.StaticFieldCount)
	}
	if e.StaticFields[0].Key != "third" {
		t.Errorf("re-acquired entry field[0].Key = %q, want %q", e.StaticFields[0].Key, "third")
	}
	builder.Reset()
}

// TestExtractedPool_WarmupTouchesShards verifies Warmup resets shard field
// counts and seeds the fallback pool without disturbing acquisition.
func TestExtractedPool_WarmupTouchesShards(t *testing.T) {
	pool := NewExtractedPool()

	// Dirty a shard, then warm up and confirm it is reset.
	pool.shards[0].Entry.StaticFieldCount = 9
	pool.Warmup(8)
	if pool.shards[0].Entry.StaticFieldCount != 0 {
		t.Errorf("Warmup did not reset shard StaticFieldCount, got %d",
			pool.shards[0].Entry.StaticFieldCount)
	}

	// Pool remains usable after warmup.
	state := pool.GetPerPState()
	if state == nil {
		t.Fatal("pool unusable after Warmup")
	}
	pool.PutPerPState(state)
}
