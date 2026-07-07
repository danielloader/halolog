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

	"github.com/go-gen-ecosystem/halolog/types"
)

// Global worker pool cache to eliminate allocations
// We keep one active pool per worker count to avoid repeated allocations
var (
	activePools = make(map[int]*WorkerPool) // Currently active pools
	poolsMu     sync.RWMutex                // Protects activePools

	// Pre-allocated worker pools for common worker counts (1, 2, 4, 8, 16, 32)
	// This eliminates allocations for EnableAsync by reusing pre-created pools
	preallocatedPools = make(map[int]*WorkerPool)
	preallocOnce      sync.Once
)

// WorkerPool manages async log entry processing with graceful shutdown.
// It provides non-blocking submission and guarantees all entries are
// processed before shutdown completes.
type WorkerPool struct {
	// Hot path - cache line isolated
	_      [64]byte
	taskCh chan *types.LogEntry
	_      [64 - 8]byte

	// Shutdown coordination
	stopCh  chan struct{}
	stopped atomic.Bool

	// Worker management
	wgWorkers sync.WaitGroup

	// Writer function - set once before use
	writer atomic.Pointer[func(*types.LogEntry)]
}

// InitWorkerPools creates pre-allocated worker pools for common worker counts.
// This eliminates allocations in EnableAsync by reusing pre-created pools.
func InitWorkerPools() {
	preallocOnce.Do(func() {
		commonCounts := []int{1, 2, 4, 8, 16, 32}
		for _, count := range commonCounts {
			wp := &WorkerPool{
				taskCh: make(chan *types.LogEntry, count*64),
				stopCh: make(chan struct{}),
			}
			// Start workers
			wp.wgWorkers.Add(count)
			for i := 0; i < count; i++ {
				go wp.workerLoop()
			}
			preallocatedPools[count] = wp
		}
	})
}

// GetPreallocatedPool returns a pre-allocated pool for the worker count if available.
// This provides zero-allocation pool retrieval for common worker counts.
func GetPreallocatedPool(workerCount int) *WorkerPool {
	InitWorkerPools()
	return preallocatedPools[workerCount]
}

// CloseAllWorkerPools gracefully shuts down all active and pre-allocated worker pools.
// This is used primarily in tests to ensure clean shutdown and prevent goroutine leaks.
// Safe to call multiple times.
func CloseAllWorkerPools() {
	poolsMu.Lock()
	defer poolsMu.Unlock()

	// Shutdown all active pools
	for _, pool := range activePools {
		if pool != nil && !pool.isStopped() {
			pool.Close()
		}
	}

	// Shutdown all pre-allocated pools
	for _, pool := range preallocatedPools {
		if pool != nil && !pool.isStopped() {
			pool.Close()
		}
	}

	// Clear the maps
	activePools = make(map[int]*WorkerPool)
	preallocatedPools = make(map[int]*WorkerPool)

	// Reset the once so pools can be recreated if needed
	preallocOnce = sync.Once{}
}

// GetActiveWorkerPool returns an existing active pool for the worker count or creates a new one.
// This eliminates allocations by avoiding duplicate pool creation for the same worker count.
func GetActiveWorkerPool(workerCount int) *WorkerPool {
	if workerCount <= 0 {
		workerCount = 1
	}

	// Try to get existing active pool first (zero-allocation path)
	poolsMu.RLock()
	if existing, exists := activePools[workerCount]; exists && !existing.isStopped() {
		poolsMu.RUnlock()
		return existing
	}
	poolsMu.RUnlock()

	// Try to get pre-allocated pool (zero-allocation path for common counts)
	if prealloc := GetPreallocatedPool(workerCount); prealloc != nil {
		poolsMu.Lock()
		activePools[workerCount] = prealloc
		poolsMu.Unlock()
		return prealloc
	}

	// Fallback: Create new pool (this will allocate, but avoids duplicate creation)
	wp := newWorkerPool(workerCount)

	// Track it as active
	poolsMu.Lock()
	activePools[workerCount] = wp
	poolsMu.Unlock()

	return wp
}

// StopActiveWorkerPool stops and removes a worker pool from the active tracking.
// This is called when we want to disable async logging.
func StopActiveWorkerPool(workerCount int) {
	poolsMu.Lock()
	if wp, exists := activePools[workerCount]; exists {
		wp.Close()
		delete(activePools, workerCount)
	}
	poolsMu.Unlock()
}

// newWorkerPool creates and starts a worker pool with the specified worker count.
// Buffer size is tuned for throughput: 64 entries per worker.
// NOTE: This function allocates channels. Use getCachedWorkerPool for zero-allocation reuse.
func newWorkerPool(workerCount int) *WorkerPool {
	if workerCount <= 0 {
		workerCount = 1
	}

	wp := &WorkerPool{
		taskCh: make(chan *types.LogEntry, workerCount*64),
		stopCh: make(chan struct{}),
	}

	// Start workers
	wp.wgWorkers.Add(workerCount)
	for i := 0; i < workerCount; i++ {
		go wp.workerLoop()
	}

	return wp
}

// SetWriter injects the writer function for processing entries.
// Must be called before submitting entries. Thread-safe.
func (wp *WorkerPool) SetWriter(w func(*types.LogEntry)) {
	wp.writer.Store(&w)
}

// GetWriter returns the current writer function. Thread-safe.
func (wp *WorkerPool) GetWriter() func(*types.LogEntry) {
	ptr := wp.writer.Load()
	if ptr == nil {
		return nil
	}
	return *ptr
}

// workerLoop processes entries until shutdown, then drains remaining entries.
func (wp *WorkerPool) workerLoop() {
	defer wp.wgWorkers.Done()

	for {
		select {
		case <-wp.stopCh:
			// Shutdown signaled - drain remaining entries
			wp.drain()
			return

		case entry, ok := <-wp.taskCh:
			if !ok {
				// Channel closed - exit
				return
			}
			wp.processEntry(entry)
		}
	}
}

// processEntry writes an entry and returns it to the pool.
// Extracted for clarity and potential future instrumentation.
func (wp *WorkerPool) processEntry(entry *types.LogEntry) {
	writer := wp.GetWriter()
	if writer != nil {
		writer(entry)
	}
	ReleaseEntry(entry)
}

// drain processes all remaining entries in the channel.
// Called during shutdown to ensure no entries are lost.
func (wp *WorkerPool) drain() {
	for {
		select {
		case entry, ok := <-wp.taskCh:
			if !ok {
				return
			}
			wp.processEntry(entry)
		default:
			// Channel empty
			return
		}
	}
}

// Submit attempts non-blocking entry submission.
// Returns true if submitted, false if pool is stopped or buffer full.
// The caller is responsible for fallback handling on false return.
func (wp *WorkerPool) Submit(entry *types.LogEntry) bool {
	// Fast path: check if stopped
	if wp.stopped.Load() {
		return false
	}

	// Fast path: check if writer is set
	if wp.GetWriter() == nil {
		return false
	}

	// Non-blocking send
	select {
	case wp.taskCh <- entry:
		return true
	default:
		// Buffer full - back-pressure
		return false
	}
}

// Close gracefully shuts down the pool.
// Blocks until all workers have finished processing remaining entries.
// Safe to call multiple times.
func (wp *WorkerPool) Close() {
	// Prevent double-close panic
	if wp.stopped.Swap(true) {
		return // Already stopped
	}

	// Signal workers to begin shutdown
	close(wp.stopCh)

	// Wait for all workers to finish draining
	wp.wgWorkers.Wait()

	// Close task channel after workers are done
	// This is safe because no more submits will succeed (stopped=true)
	close(wp.taskCh)
}

// Metrics methods for observability

// GetBufferUsage returns current buffer utilization (0.0-1.0).
func (wp *WorkerPool) GetBufferUsage() float64 {
	if wp.taskCh == nil {
		return 0
	}
	return float64(len(wp.taskCh)) / float64(cap(wp.taskCh))
}

// GetBufferSize returns the buffer capacity.
func (wp *WorkerPool) GetBufferSize() int {
	if wp.taskCh == nil {
		return 0
	}
	return cap(wp.taskCh)
}

// GetBufferCurrent returns the current number of items in buffer.
func (wp *WorkerPool) GetBufferCurrent() int {
	if wp.taskCh == nil {
		return 0
	}
	return len(wp.taskCh)
}

// isStopped returns true if the pool has been stopped.
func (wp *WorkerPool) isStopped() bool {
	return wp.stopped.Load()
}
