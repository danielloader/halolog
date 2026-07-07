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
// Package middleware concurrency regression tests.
// Author: Admilson B. F. Cossa

package middleware

import (
	"sync"
	"testing"
	"time"

	"github.com/go-gen-ecosystem/halolog/types"
)

// countingAdapter records how many distinct entries it received.
type countingAdapter struct {
	mu    sync.Mutex
	count int
}

func (c *countingAdapter) Name() string { return "counting" }
func (c *countingAdapter) Write(entry *types.LogEntry) error {
	c.mu.Lock()
	c.count++
	c.mu.Unlock()
	return nil
}
func (c *countingAdapter) Flush() error                      { return nil }
func (c *countingAdapter) SetFormatter(f types.Formatter)    {}
func (c *countingAdapter) WriteZero(e *types.LogEntry) error { return c.Write(e) }
func (c *countingAdapter) Close() error                      { return nil }
func (c *countingAdapter) Health() error                     { return nil }
func (c *countingAdapter) total() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.count
}

// TestAsyncAdapter_WriteDuringClose_NoPanic reproduces the critical
// "send on closed channel" defect: many goroutines call Write() while another
// calls Close(). Before the fix, Close() closed the buffer channel while a
// concurrent Write() send was in flight, panicking and crashing the process.
// After the fix the channel is never closed and Write must simply return an
// error (or succeed) without panicking.
func TestAsyncAdapter_WriteDuringClose_NoPanic(t *testing.T) {
	for iteration := 0; iteration < 50; iteration++ {
		mock := &countingAdapter{}
		adapter := NewAsyncAdapter(mock, &AsyncAdapterOptions{
			BufferSize:    16, // small buffer to force contention on the send path
			BatchSize:     4,
			FlushInterval: time.Millisecond,
		})

		var wg sync.WaitGroup
		const writers = 8
		for w := 0; w < writers; w++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				entry := &types.LogEntry{Level: types.InfoLevel, Message: "x"}
				for i := 0; i < 200; i++ {
					// A send-on-closed-channel would panic here and crash the test binary.
					_ = adapter.Write(entry)
				}
			}()
		}

		// Close concurrently with the writers to hit the shutdown window. The
		// closed channel lets us deterministically wait for Close to finish
		// before asserting post-close behaviour, without weakening the race:
		// the concurrent Write/Close overlap (the panic check) still happens.
		closed := make(chan struct{})
		go func() {
			time.Sleep(50 * time.Microsecond)
			_ = adapter.Close()
			close(closed)
		}()

		wg.Wait()
		<-closed // ensure Close has fully returned (closing flag is set)

		// After Close, further writes must be rejected (never panic).
		if err := adapter.Write(&types.LogEntry{Level: types.InfoLevel, Message: "after"}); err != ErrAsyncClosed {
			t.Fatalf("iteration %d: expected ErrAsyncClosed after Close, got %v", iteration, err)
		}
	}
}

// TestAsyncAdapter_FlushDeliversAllEntries verifies that Flush routes draining
// exclusively through the single background writer, so no entry is lost,
// duplicated, or reordered between Flush and the writer. Before the fix, Flush
// drained the buffer concurrently with the writer, producing a nondeterministic
// count.
func TestAsyncAdapter_FlushDeliversAllEntries(t *testing.T) {
	const total = 5000
	mock := &countingAdapter{}
	adapter := NewAsyncAdapter(mock, &AsyncAdapterOptions{
		BufferSize:    total + 100,
		BatchSize:     64,
		FlushInterval: time.Hour, // disable periodic flush; only explicit Flush drains
	})

	for i := 0; i < total; i++ {
		if err := adapter.Write(&types.LogEntry{Level: types.InfoLevel, Message: "m"}); err != nil {
			t.Fatalf("write %d failed: %v", i, err)
		}
	}

	if err := adapter.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	if got := mock.total(); got != total {
		t.Fatalf("expected exactly %d delivered entries after Flush, got %d", total, got)
	}

	if err := adapter.Close(); err != nil {
		t.Fatalf("close failed: %v", err)
	}
}

// TestAsyncAdapter_CloseDrainsRemaining verifies that entries still buffered at
// Close time are delivered to the base adapter (no silent drop on shutdown).
func TestAsyncAdapter_CloseDrainsRemaining(t *testing.T) {
	const total = 300
	mock := &countingAdapter{}
	adapter := NewAsyncAdapter(mock, &AsyncAdapterOptions{
		BufferSize:    total + 10,
		BatchSize:     1000,      // larger than total so nothing auto-flushes
		FlushInterval: time.Hour, // no periodic flush
	})

	for i := 0; i < total; i++ {
		if err := adapter.Write(&types.LogEntry{Level: types.InfoLevel, Message: "m"}); err != nil {
			t.Fatalf("write %d failed: %v", i, err)
		}
	}

	if err := adapter.Close(); err != nil {
		t.Fatalf("close failed: %v", err)
	}

	if got := mock.total(); got != total {
		t.Fatalf("expected %d entries drained on Close, got %d", total, got)
	}
}
