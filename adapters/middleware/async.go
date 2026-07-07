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

// AsyncAdapter provides non-blocking logging with background writer
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
	closing       int32 // atomic flag to indicate closing state
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

// Write writes a log entry asynchronously
func (a *AsyncAdapter) Write(entry *types.LogEntry) error {
	if entry == nil {
		return ErrAsyncNilEntry
	}

	// Check if context is already cancelled (adapter closed)
	select {
	case <-a.ctx.Done():
		return ErrAsyncClosed
	default:
	}

	// Try to write to buffer
	select {
	case a.buffer <- entry:
		return nil
	default:
		// Buffer is full, check if context was cancelled during this operation
		select {
		case <-a.ctx.Done():
			return ErrAsyncClosed
		default:
			return ErrAsyncBufferFull
		}
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

// Flush flushes the buffered entries
func (a *AsyncAdapter) Flush() error {
	// Give background writer a moment to process any buffered entries
	time.Sleep(50 * time.Millisecond)

	// Signal background writer to flush current batch
	select {
	case a.flushChan <- struct{}{}:
		// Wait for background writer to complete the flush (with timeout)
		select {
		case <-a.doneChan:
			// Background writer completed flush
		case <-time.After(100 * time.Millisecond):
			// Timeout - proceed with manual flush of remaining entries
		}
	default:
		// If flushChan is full, background writer is already processing
	}

	// Drain any remaining entries from buffer and write them directly
	var entries []*types.LogEntry

	for {
		select {
		case entry := <-a.buffer:
			entries = append(entries, entry)
		default:
			// Buffer is empty
			goto writeBatch
		}
	}

writeBatch:
	// Write all entries to base adapter
	for _, entry := range entries {
		if err := a.base.Write(entry); err != nil {
			return ErrAsyncWriteEntry
		}
	}

	// Check if there were any errors from background writer
	if err := a.getLastError(); err != nil {
		return ErrAsyncWriteEntry
	}

	return a.base.Flush()
}

// Close closes the async adapter
func (a *AsyncAdapter) Close() error {
	// Set closing flag to prevent race conditions
	if !atomic.CompareAndSwapInt32(&a.closing, 0, 1) {
		// Already closing/closed
		return nil
	}

	// Signal shutdown
	a.cancel()

	// Close buffer channel to prevent new writes and signal background writer to exit
	close(a.buffer)

	// Wait for background writer to finish
	a.wg.Wait()

	// Drain remaining entries from buffer
	var remainingEntries []*types.LogEntry
	for entry := range a.buffer {
		remainingEntries = append(remainingEntries, entry)
	}

	// Write remaining entries to base adapter
	for _, entry := range remainingEntries {
		// Silently continue on errors to maintain zero-allocation compliance
		_ = a.base.Write(entry)
	}

	// Close base adapter
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

// backgroundWriter processes log entries in the background
func (a *AsyncAdapter) backgroundWriter() {
	defer a.wg.Done()

	ticker := time.NewTicker(a.flushInterval)
	defer ticker.Stop()

	var batch []*types.LogEntry

	for {
		select {
		case entry, ok := <-a.buffer:
			if !ok {
				// Buffer closed, flush remaining entries
				if len(batch) > 0 {
					a.flushBatch(batch)
				}
				return
			}

			batch = append(batch, entry)

			// Flush if batch is full
			if len(batch) >= a.batchSize {
				a.flushBatch(batch)
				batch = nil
			}

		case <-a.flushChan:
			// Force flush current batch
			if len(batch) > 0 {
				a.flushBatch(batch)
				batch = nil
			}
			// Signal that flush is complete (only if not closing)
			if atomic.LoadInt32(&a.closing) == 0 {
				select {
				case a.doneChan <- struct{}{}:
				default:
				}
			}

		case <-ticker.C:
			// Periodic flush
			if len(batch) > 0 {
				a.flushBatch(batch)
				batch = nil
			}

		case <-a.ctx.Done():
			// Context cancelled, flush remaining entries
			if len(batch) > 0 {
				a.flushBatch(batch)
			}
			return
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
