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
// Package adapters provides output adapters
// Author: Admilson B. F. Cossa

package middleware

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-gen-ecosystem/halolog/types"
	"github.com/go-gen-ecosystem/halolog/utils"
)

// Pre-allocated errors for zero-allocation design
var (
	ErrAsyncNilEntry   = errors.New("cannot write nil entry")
	ErrAsyncClosed     = errors.New("async adapter is closed")
	ErrAsyncBufferFull = errors.New("async buffer is full")
	ErrAsyncWriteEntry = errors.New("failed to write entry")
)

// AsyncAdapter provides non-blocking logging with a single background writer.
//
// Concurrency model:
//   - The backgroundWriter goroutine is the SOLE consumer of a.buffer. No other
//     method drains the buffer, which guarantees log records are never split,
//     duplicated, or reordered between competing consumers.
//   - a.buffer is never closed. Shutdown is signalled exclusively via ctx
//     cancellation. sendMu (an RWMutex) makes the "is-closing check + channel
//     send" atomic with respect to Close(): senders hold the read lock and
//     re-check the closing flag before sending; Close() takes the write lock so
//     no send can be in flight while it flips the flag. This eliminates the
//     "send on closed channel" panic entirely (the channel is simply never
//     closed) while still refusing writes after Close begins.
type AsyncAdapter struct {
	base          types.Adapter
	buffer        chan *types.LogEntry
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
	batchSize     int
	flushInterval time.Duration
	flushChan     chan struct{}
	doneChan      chan struct{}
	lastError     error
	errorMu       sync.Mutex

	// sendMu guards concurrent sends on a.buffer against Close.
	// Writers take RLock; Close takes Lock. Combined with the closing flag it
	// makes the (check-closing, send) pair atomic w.r.t. shutdown.
	sendMu  sync.RWMutex
	closing int32 // atomic flag to indicate closing state
}

// AsyncAdapterOptions contains configuration options for AsyncAdapter
type AsyncAdapterOptions struct {
	BufferSize    int
	BatchSize     int
	FlushInterval time.Duration
}

// NewAsyncAdapter creates a new async adapter wrapper
func NewAsyncAdapter(base types.Adapter, options *AsyncAdapterOptions) *AsyncAdapter {
	// Set default values
	bufferSize := 1000
	batchSize := 100
	flushInterval := 1 * time.Second

	if options != nil {
		// Override defaults with provided values
		if options.BufferSize > 0 {
			bufferSize = options.BufferSize
		}
		if options.BatchSize > 0 {
			batchSize = options.BatchSize
		}
		if options.FlushInterval > 0 {
			flushInterval = options.FlushInterval
		}
	}

	ctx, cancel := context.WithCancel(context.Background())

	adapter := &AsyncAdapter{
		base:          base,
		buffer:        make(chan *types.LogEntry, bufferSize),
		ctx:           ctx,
		cancel:        cancel,
		batchSize:     batchSize,
		flushInterval: flushInterval,
		flushChan:     make(chan struct{}),
		doneChan:      make(chan struct{}, 1),
	}

	// Start background writer
	adapter.wg.Add(1)
	go adapter.backgroundWriter()

	return adapter
}

// Name returns the name of this adapter
func (a *AsyncAdapter) Name() string {
	return utils.FormatWithPrefixAndSuffix("AsyncAdapter(", a.base.Name(), ")")
}

// setLastError stores the last error encountered by the background writer
func (a *AsyncAdapter) setLastError(err error) {
	if err != nil {
		a.errorMu.Lock()
		a.lastError = err
		a.errorMu.Unlock()
	}
}

// getLastError retrieves and clears the last error
func (a *AsyncAdapter) getLastError() error {
	a.errorMu.Lock()
	err := a.lastError
	a.lastError = nil
	a.errorMu.Unlock()
	return err
}

// Write writes a log entry asynchronously.
//
// The send on a.buffer is performed under sendMu.RLock after re-checking the
// closing flag, so it cannot race a concurrent Close(): the channel is never
// closed, and Close cannot flip the closing flag while any RLock is held.
func (a *AsyncAdapter) Write(entry *types.LogEntry) error {
	if entry == nil {
		return ErrAsyncNilEntry
	}

	// Hold the read lock for the whole (check + send) so Close (which takes the
	// write lock) cannot interleave. Many writers proceed concurrently.
	a.sendMu.RLock()
	defer a.sendMu.RUnlock()

	// Re-check closing under the lock. Once Close sets this, no send occurs.
	if atomic.LoadInt32(&a.closing) != 0 {
		return ErrAsyncClosed
	}

	// Try to write to buffer without blocking.
	select {
	case a.buffer <- entry:
		return nil
	default:
		return ErrAsyncBufferFull
	}
}

// WriteZero writes a zero-allocation log entry to the async buffer
func (a *AsyncAdapter) WriteZero(entry *types.LogEntry) error {
	if entry == nil {
		return ErrAsyncNilEntry
	}

	// LogEntry is already unified, use it directly
	return a.Write(entry)
}

// Flush flushes the buffered entries by delegating entirely to the single
// backgroundWriter, which owns a.buffer. Flush never drains the buffer itself,
// so it cannot compete with the writer for entries (no lost/duplicated/reordered
// records). It signals the writer to drain everything currently queued and
// waits for the writer to acknowledge completion.
func (a *AsyncAdapter) Flush() error {
	// If already closing/closed, the writer may be gone; flush the base directly.
	if atomic.LoadInt32(&a.closing) != 0 {
		if err := a.getLastError(); err != nil {
			return ErrAsyncWriteEntry
		}
		return a.base.Flush()
	}

	// Ask the writer to drain the buffer and flush the base adapter, then wait
	// for its acknowledgement. Use a bounded wait so a stuck base adapter cannot
	// hang the caller forever.
	select {
	case a.flushChan <- struct{}{}:
		select {
		case <-a.doneChan:
			// Writer completed the drain + base flush.
		case <-a.ctx.Done():
			// Shutting down; fall through to error check.
		case <-time.After(5 * time.Second):
			// Writer did not acknowledge in time; report as a write failure.
			return ErrAsyncWriteEntry
		}
	case <-a.ctx.Done():
		// Adapter is shutting down.
	}

	if err := a.getLastError(); err != nil {
		return ErrAsyncWriteEntry
	}
	return a.base.Flush()
}

// Close closes the async adapter. It is safe to call multiple times.
//
// Close never closes a.buffer. It flips the closing flag under sendMu.Lock so
// that no Write send can be in flight, cancels the context to stop the writer,
// waits for the writer to drain and exit, then closes the base adapter.
func (a *AsyncAdapter) Close() error {
	// Take the write lock so all in-flight sends complete and no new send can
	// start while we mark the adapter closing.
	a.sendMu.Lock()
	if !atomic.CompareAndSwapInt32(&a.closing, 0, 1) {
		// Already closing/closed.
		a.sendMu.Unlock()
		return nil
	}
	// Signal shutdown. The writer drains the buffer on ctx.Done() before exiting.
	a.cancel()
	a.sendMu.Unlock()

	// Wait for background writer to finish draining and exit.
	a.wg.Wait()

	// Close base adapter.
	return a.base.Close()
}

// SetFormatter sets the formatter for the base adapter (no-op for compatibility)
func (a *AsyncAdapter) SetFormatter(formatter types.Formatter) {
	// No-op: formatter compatibility for interface compliance
}

// Health checks the health of the async adapter
func (a *AsyncAdapter) Health() error {
	// Check if the adapter is still running
	select {
	case <-a.ctx.Done():
		return fmt.Errorf("async adapter is stopped")
	default:
		// Check base adapter health
		if a.base != nil {
			return a.base.Health()
		}
		return nil
	}
}

// backgroundWriter is the SOLE consumer of a.buffer. It batches entries and
// flushes them to the base adapter on batch-full, periodic tick, explicit
// Flush, or shutdown.
func (a *AsyncAdapter) backgroundWriter() {
	defer a.wg.Done()

	ticker := time.NewTicker(a.flushInterval)
	defer ticker.Stop()

	var batch []*types.LogEntry

	for {
		select {
		case entry := <-a.buffer:
			batch = append(batch, entry)

			// Flush if batch is full
			if len(batch) >= a.batchSize {
				a.flushBatch(batch)
				batch = batch[:0]
			}

		case <-a.flushChan:
			// Drain everything currently queued, then flush the batch and the
			// base adapter, so Flush() observes a fully-drained buffer.
			batch = a.drainInto(batch)
			a.flushBatch(batch)
			batch = batch[:0]
			if err := a.base.Flush(); err != nil {
				a.setLastError(err)
			}
			// Acknowledge completion (non-blocking; doneChan is buffered).
			select {
			case a.doneChan <- struct{}{}:
			default:
			}

		case <-ticker.C:
			// Periodic flush
			if len(batch) > 0 {
				a.flushBatch(batch)
				batch = batch[:0]
			}

		case <-a.ctx.Done():
			// Context cancelled: drain any remaining entries and flush before
			// exiting so no queued record is dropped on shutdown.
			batch = a.drainInto(batch)
			if len(batch) > 0 {
				a.flushBatch(batch)
			}
			return
		}
	}
}

// drainInto pulls every entry currently buffered into batch without blocking
// and returns the extended batch. Only the backgroundWriter calls this, so it
// remains the single consumer of a.buffer.
func (a *AsyncAdapter) drainInto(batch []*types.LogEntry) []*types.LogEntry {
	for {
		select {
		case entry := <-a.buffer:
			batch = append(batch, entry)
		default:
			return batch
		}
	}
}

// flushBatch writes a batch of entries to the base adapter
func (a *AsyncAdapter) flushBatch(batch []*types.LogEntry) {
	if len(batch) == 0 {
		return
	}

	for _, entry := range batch {
		if err := a.base.Write(entry); err != nil {
			a.setLastError(err)
			return // Stop processing on first error
		}
	}

	// Flush base adapter periodically
	if err := a.base.Flush(); err != nil {
		a.setLastError(err)
	}
}
