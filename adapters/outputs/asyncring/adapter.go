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

// @author Admilson B. F. Cossa

package asyncring

import (
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/go-gen-ecosystem/halolog/types"
)

// RingAdapter is an asynchronous log sink: producers copy each record into a
// bounded lock-free ring and return immediately, while a single background
// goroutine serializes and writes the records off the caller's thread. This
// trades serialization throughput (bounded by the one writer) for very low,
// predictable caller latency.
//
// Unlike a naive async adapter, RingAdapter copies each record into
// ring-owned storage, so the caller's pooled entry is safe to recycle the
// instant Write returns.
type RingAdapter struct {
	ring   *ring[entryPayload]
	bw     *bgWriter
	onFull OnFull

	inflight  atomic.Int64  // producers between entering enqueue and finishing it
	dropped   atomic.Uint64 // records dropped under OnFull=Drop (or during close)
	closing   atomic.Bool
	closeOnce sync.Once
}

// Compile-time guarantee that RingAdapter satisfies the adapter contract.
var _ types.Adapter = (*RingAdapter)(nil)

// New builds a RingAdapter and starts its background writer. It returns an error
// if required options (Writer, Formatter) are missing.
func New(o Options) (*RingAdapter, error) {
	opts, err := o.withDefaults()
	if err != nil {
		return nil, err
	}
	r := newRing[entryPayload](opts.Capacity)
	bw := newBGWriter(r, opts)
	a := &RingAdapter{ring: r, bw: bw, onFull: opts.OnFull}
	bw.inflight = &a.inflight
	bw.start()
	return a, nil
}

// Name identifies the adapter.
func (a *RingAdapter) Name() string { return "asyncring" }

// Write enqueues a record for asynchronous serialization.
func (a *RingAdapter) Write(e *types.LogEntry) error { return a.enqueue(e) }

// WriteZero enqueues a record for asynchronous serialization (zero-allocation
// producer path).
func (a *RingAdapter) WriteZero(e *types.LogEntry) error { return a.enqueue(e) }

// enqueue copies e into a free ring slot. Under OnFull=Drop it never blocks and
// increments the dropped counter when the ring is full; under OnFull=Block it
// spins until a slot frees or the adapter closes.
func (a *RingAdapter) enqueue(e *types.LogEntry) error {
	if e == nil {
		return nil
	}
	a.inflight.Add(1)

	if a.closing.Load() {
		a.dropped.Add(1)
		a.inflight.Add(-1)
		return nil
	}

	if a.ring.enqueue(func(p *entryPayload) { p.copyFrom(e) }) {
		a.inflight.Add(-1)
		return nil
	}

	if a.onFull == Block {
		for !a.ring.enqueue(func(p *entryPayload) { p.copyFrom(e) }) {
			if a.closing.Load() {
				a.dropped.Add(1)
				a.inflight.Add(-1)
				return nil
			}
			runtime.Gosched()
		}
		a.inflight.Add(-1)
		return nil
	}

	a.dropped.Add(1)
	a.inflight.Add(-1)
	return nil
}

// Dropped returns the number of records discarded because the ring was full
// (OnFull=Drop) or because they arrived during close.
func (a *RingAdapter) Dropped() uint64 { return a.dropped.Load() }

// WriteErrors returns how many times the destination writer returned an error.
func (a *RingAdapter) WriteErrors() uint64 { return a.bw.writeErrors.Load() }

// Flush blocks until every record already accepted has been serialized and
// written to the destination. It is a no-op once the adapter is closing.
func (a *RingAdapter) Flush() error {
	if a.closing.Load() {
		return nil
	}
	a.bw.requestFlush()
	return nil
}

// SetFormatter is a no-op: the formatter is fixed at construction via Options so
// the single writer goroutine can read it without synchronization. Build a new
// adapter to change formatters. It exists to satisfy types.Adapter.
func (a *RingAdapter) SetFormatter(types.Formatter) {}

// Health reports the adapter healthy. Overload is surfaced separately via
// Dropped(); destination write failures via WriteErrors().
func (a *RingAdapter) Health() error { return nil }

// Close stops accepting new records, drains everything already accepted, waits
// for in-flight producers, and stops the background writer. It is idempotent and
// safe to call once; further Writes after Close are dropped.
func (a *RingAdapter) Close() error {
	a.closeOnce.Do(func() {
		a.closing.Store(true)
		close(a.bw.stop)
		<-a.bw.done
	})
	return nil
}
