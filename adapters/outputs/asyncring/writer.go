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
	"io"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/go-gen-ecosystem/halolog/types"
)

// initialWriterBuf is the starting size of the writer's reusable serialization
// buffer; it grows as needed and is reused for the process lifetime.
const initialWriterBuf = 4096

// bgWriter is the single consumer of the ring. It serializes ready records into
// a reused buffer and flushes them to the destination writer in batches. It owns
// the buffer and the scratch entry, so serialization allocates nothing steady
// state. Only run() touches those fields, so no synchronization is needed there.
type bgWriter struct {
	ring      *ring[entryPayload]
	w         io.Writer
	formatter types.Formatter
	batchSize int
	interval  time.Duration

	buf     []byte
	scratch types.LogEntry
	consume func(*entryPayload) // bound once to avoid per-dequeue method values

	stop    chan struct{}
	done    chan struct{}
	flushCh chan chan struct{} // Flush requests carry an ack channel closed when drained

	// inflight points at the adapter's in-flight producer counter so finalDrain
	// can wait for producers that already passed the closing check.
	inflight *atomic.Int64

	writeErrors atomic.Uint64
}

func newBGWriter(r *ring[entryPayload], o Options) *bgWriter {
	bw := &bgWriter{
		ring:      r,
		w:         o.Writer,
		formatter: o.Formatter,
		batchSize: o.BatchSize,
		interval:  o.FlushInterval,
		buf:       make([]byte, 0, initialWriterBuf),
		stop:      make(chan struct{}),
		done:      make(chan struct{}),
		flushCh:   make(chan chan struct{}),
	}
	bw.consume = bw.serializeOne
	return bw
}

func (bw *bgWriter) start() { go bw.run() }

// run drains the ring while there is work, and otherwise parks until the flush
// interval elapses or a stop signal arrives. On stop it performs a final drain
// so no buffered record is lost.
func (bw *bgWriter) run() {
	defer close(bw.done)

	timer := time.NewTimer(bw.interval)
	defer timer.Stop()

	for {
		if bw.drainBatch() > 0 {
			bw.flush()
			continue
		}

		// Idle: re-arm the timer safely, then wait for work or shutdown.
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
		timer.Reset(bw.interval)

		select {
		case <-bw.stop:
			bw.finalDrain()
			return
		case ack := <-bw.flushCh:
			bw.drainAll()
			close(ack)
		case <-timer.C:
		}
	}
}

// requestFlush asks the writer to drain everything currently queued and flush it,
// blocking until it acknowledges or the writer stops.
func (bw *bgWriter) requestFlush() {
	ack := make(chan struct{})
	select {
	case bw.flushCh <- ack:
		select {
		case <-ack:
		case <-bw.done:
		}
	case <-bw.done:
	}
}

// drainBatch serializes up to batchSize ready records and returns how many it
// serialized (0 when the ring is empty).
func (bw *bgWriter) drainBatch() int {
	n := 0
	for n < bw.batchSize && bw.ring.dequeue(bw.consume) {
		n++
	}
	return n
}

// serializeOne renders one payload into the shared buffer with newline
// termination. Called only from run() via dequeue, in place, before the slot is
// freed.
func (bw *bgWriter) serializeOne(p *entryPayload) {
	p.into(&bw.scratch)
	start := len(bw.buf)
	bw.buf = bw.formatter.Format(&bw.scratch, bw.buf)
	if len(bw.buf) == start || bw.buf[len(bw.buf)-1] != '\n' {
		bw.buf = append(bw.buf, '\n')
	}
}

// flush writes the accumulated buffer to the destination and resets it.
func (bw *bgWriter) flush() {
	if len(bw.buf) == 0 {
		return
	}
	if _, err := bw.w.Write(bw.buf); err != nil {
		bw.writeErrors.Add(1)
	}
	bw.buf = bw.buf[:0]
}

// drainAll serializes and flushes every record currently in the ring.
func (bw *bgWriter) drainAll() {
	for bw.drainBatch() > 0 {
		bw.flush()
	}
	bw.flush()
}

// finalDrain empties the ring completely on shutdown without losing a record
// enqueued by a producer that had already passed the closing check.
//
// Ordering is critical and must be check-THEN-drain, not drain-then-check. A
// producer publishes into the ring BEFORE it decrements inflight
// (adapter.enqueue), so if we drained first and then observed inflight==0, a
// producer could publish in the window between the final drain and the load, and
// that record would be lost. Instead we first observe inflight==0 — at which
// point every counted producer has already published (its publish is ordered
// before its decrement, and all decrements precede the observed-zero load), and
// no new producer can publish because Close set closing=true before signalling
// stop, so any later producer observes closing and drops — and only THEN perform
// a final drain, which is guaranteed to see all those publishes.
func (bw *bgWriter) finalDrain() {
	for {
		if bw.inflight == nil || bw.inflight.Load() == 0 {
			bw.drainAll()
			return
		}
		bw.drainAll()
		runtime.Gosched()
	}
}
