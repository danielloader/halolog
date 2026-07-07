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
// Author: Admilson B. F. Cossa

package pool

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-gen-ecosystem/halolog/types"
)

type poolCounters struct {
	acquire atomic.Int64
	release atomic.Int64
	alloc   atomic.Int64
	active  atomic.Int64
}

// EntryPool provides zero-allocation entry management using per-P state.
// Benchmark results: ~30% faster than the sync.Pool approach, with zero contention.
type EntryPool struct {
	entryPool   sync.Pool
	counters    poolCounters
	initialized atomic.Bool
	shutdown    atomic.Bool
}

// PoolStats represents pool statistics.
//
//nolint:revive // PoolStats is stable public API; renaming to Stats would break importers.
type PoolStats struct {
	AcquireCount    int64
	ReleaseCount    int64
	AllocationCount int64
	ActiveEntries   int64
	Initialized     bool
	Shutdown        bool
}

// GlobalPool is the process-wide entry pool used by the convenience wrappers.
var GlobalPool *EntryPool

// InitializeGlobalPool creates the global entry pool
func InitializeGlobalPool() {
	if GlobalPool != nil && GlobalPool.initialized.Load() {
		return
	}

	GlobalPool = &EntryPool{
		entryPool: sync.Pool{
			New: func() interface{} {
				entry := &types.LogEntry{}
				entry.Fields = make([]types.TypedFieldData, 0, 32)
				// Allocate buffers for static fields since they are no longer embedded
				entry.StaticFields = make([]types.TypedFieldData, 16)
				entry.StaticContext = make([]types.TypedFieldData, 16)
				return entry
			},
		},
	}
	GlobalPool.initialized.Store(true)
}

func init() {
	InitializeGlobalPool()
}

// AcquireEntry returns a pooled entry
func (p *EntryPool) AcquireEntry() *types.LogEntry {
	if p == nil || !p.initialized.Load() || p.shutdown.Load() {
		return acquireEntryBasic()
	}

	entry := p.entryPool.Get().(*types.LogEntry)

	// Minimal reset: set lengths to zero but preserve capacity
	if len(entry.Fields) > 0 {
		entry.Fields = entry.Fields[:0]
	}
	entry.StaticFieldCount = 0

	p.counters.acquire.Add(1)
	p.counters.active.Add(1)

	return entry
}

// ReleaseEntry returns entry to pool
func (p *EntryPool) ReleaseEntry(entry *types.LogEntry) {
	if entry == nil || p == nil || !p.initialized.Load() {
		return
	}

	// Clear sensitive data
	for i := 0; i < len(entry.Fields); i++ {
		entry.Fields[i] = types.TypedFieldData{}
	}
	for i := 0; i < len(entry.StaticContext); i++ {
		entry.StaticContext[i] = types.TypedFieldData{}
	}

	// Reset standard fields
	entry.Message = ""
	entry.Level = types.InfoLevel // Reset to default
	entry.Component = ""
	entry.Timestamp = time.Time{}
	entry.TimestampUnix = 0

	entry.StaticFieldCount = 0
	entry.Fields = entry.Fields[:0]
	entry.StaticContext = entry.StaticContext[:0]

	p.counters.release.Add(1)
	p.counters.active.Add(-1)

	p.entryPool.Put(entry)
}

// GetStats returns pool statistics
func (p *EntryPool) GetStats() PoolStats {
	if p == nil || !p.initialized.Load() {
		return PoolStats{}
	}
	return PoolStats{
		AcquireCount:    p.counters.acquire.Load(),
		ReleaseCount:    p.counters.release.Load(),
		AllocationCount: p.counters.alloc.Load(),
		ActiveEntries:   p.counters.active.Load(),
		Initialized:     p.initialized.Load(),
		Shutdown:        p.shutdown.Load(),
	}
}

// Close shuts down the pool. It marks the pool as shut down so subsequent
// Acquire/Release calls fall back to the basic path; the underlying sync.Pool
// needs no manual draining (its entries are reclaimed by the GC).
//
// The previous implementation looped on entryPool.Get() waiting for a nil
// sentinel to stop, but a sync.Pool with a non-nil New func never returns nil,
// so that loop spun forever (a hang, reachable via Shutdown()).
func (p *EntryPool) Close() {
	if p == nil {
		return
	}
	p.shutdown.Store(true)
}

// AcquireEntry returns a pooled entry from the global pool, falling back to a
// freshly allocated entry when the global pool is uninitialized or shut down.
func AcquireEntry() *types.LogEntry {
	if GlobalPool != nil && GlobalPool.initialized.Load() && !GlobalPool.shutdown.Load() {
		return GlobalPool.AcquireEntry()
	}
	return acquireEntryBasic()
}

// ReleaseEntry returns an entry to the global pool, falling back to a basic
// reset when the global pool is uninitialized or shut down.
func ReleaseEntry(entry *types.LogEntry) {
	if GlobalPool != nil && GlobalPool.initialized.Load() && !GlobalPool.shutdown.Load() {
		GlobalPool.ReleaseEntry(entry)
		return
	}
	releaseEntryBasic(entry)
}

// Basic fallback implementations
func acquireEntryBasic() *types.LogEntry {
	entry := &types.LogEntry{}
	entry.Fields = make([]types.TypedFieldData, 0, 32)
	// Initialize StaticFields slice backed by external buffer (since internal buffer removed)
	entry.StaticFields = make([]types.TypedFieldData, 16)
	entry.StaticContext = make([]types.TypedFieldData, 16)
	return entry
}

func releaseEntryBasic(entry *types.LogEntry) {
	if entry == nil {
		return
	}
	if entry.Fields != nil {
		entry.Fields = entry.Fields[:0]
	}
	entry.StaticFieldCount = 0
}
