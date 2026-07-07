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
	"math/bits"
	"sync/atomic"
)

// cacheLinePad is a conservative cache-line size used to pad the hot cursors so
// producers (tail) and the consumer (head) do not false-share a cache line.
const cacheLinePad = 64

// ringSlot is one cell of the bounded queue. seq sequences ownership between
// producers and the consumer; data is the payload, filled and read in place to
// avoid copying large payloads more than once.
type ringSlot[T any] struct {
	seq  atomic.Uint64
	data T
}

// ring is a bounded, multi-producer/single-consumer lock-free queue based on
// Dmitry Vyukov's bounded MPMC algorithm. Capacity is a power of two so the
// position maps to a slot with a mask instead of a modulo. Enqueue and dequeue
// never block and never allocate; a full ring reports failure to the caller,
// which applies the configured overflow policy.
type ring[T any] struct {
	mask  uint64
	_     [cacheLinePad - 8]byte
	tail  atomic.Uint64 // next position a producer will claim
	_     [cacheLinePad - 8]byte
	head  atomic.Uint64 // next position the consumer will read
	_     [cacheLinePad - 8]byte
	slots []ringSlot[T]
}

// newRing creates a ring whose capacity is capacity rounded up to a power of two
// (minimum 2). Each slot's sequence is seeded to its index so slot i is first
// claimable at producer position i.
func newRing[T any](capacity int) *ring[T] {
	n := roundUpPow2(capacity)
	r := &ring[T]{mask: uint64(n - 1), slots: make([]ringSlot[T], n)}
	for i := range r.slots {
		r.slots[i].seq.Store(uint64(i))
	}
	return r
}

// roundUpPow2 returns the smallest power of two >= n, with a floor of 2.
func roundUpPow2(n int) int {
	if n < 2 {
		return 2
	}
	if n&(n-1) == 0 {
		return n
	}
	return 1 << bits.Len(uint(n))
}

// Cap returns the ring capacity (a power of two).
func (r *ring[T]) Cap() int { return len(r.slots) }

// enqueue claims the next free slot and invokes fill to populate it in place,
// then publishes it. It returns false without calling fill if the ring is full.
// Safe for concurrent producers.
func (r *ring[T]) enqueue(fill func(*T)) bool {
	for {
		pos := r.tail.Load()
		slot := &r.slots[pos&r.mask]
		seq := slot.seq.Load()
		diff := int64(seq) - int64(pos)
		switch {
		case diff == 0:
			// Slot is free at this position; try to claim it.
			if r.tail.CompareAndSwap(pos, pos+1) {
				fill(&slot.data)
				slot.seq.Store(pos + 1) // publish to the consumer
				return true
			}
		case diff < 0:
			// The slot the tail points at has not been consumed yet: ring full.
			return false
		default:
			// Another producer advanced tail; reload and retry.
		}
	}
}

// dequeue reads the next ready slot in place via consume, then frees it for a
// future lap. It returns false without calling consume if the ring is empty.
// This queue is single-consumer: dequeue must be called from one goroutine only.
func (r *ring[T]) dequeue(consume func(*T)) bool {
	for {
		pos := r.head.Load()
		slot := &r.slots[pos&r.mask]
		seq := slot.seq.Load()
		diff := int64(seq) - int64(pos+1)
		switch {
		case diff == 0:
			if r.head.CompareAndSwap(pos, pos+1) {
				consume(&slot.data)
				slot.seq.Store(pos + r.mask + 1) // free slot for the next lap
				return true
			}
		case diff < 0:
			return false // empty
		default:
			// Not yet published or already taken; retry.
		}
	}
}
