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

package text

import (
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-gen-ecosystem/halolog/types"
)

// --- Configuration ---

const (
	maxIntCache     = 1000 // Cache line numbers 0-999
	maxMessageSize  = 1024 // Hard limit to prevent OOM/GC spikes
	stackBufferSize = 2048 // Stack allocation size for zero-allocation formatting
)

var (
	// Pre-computed small integers (0-999) to avoid runtime strconv
	smallInts [maxIntCache]string

	// Global "Wall Clock" - Updated by a background ticker.
	// We use an atomic pointer to a byte array to avoid locks/contention on the
	// zero-allocation hot path (Format only does an atomic load + append).
	globalTimeBytes atomic.Pointer[[]byte]

	// clockOnce ensures the wall-clock ticker goroutine is started at most once,
	// lazily, on first text-formatter construction — never at package init, so
	// importing this package (e.g. transitively) does not spawn a daemon
	// goroutine for a formatter that is never used.
	clockOnce sync.Once
	// clockStop signals the wall-clock goroutine to exit (stoppable daemon).
	clockStop = make(chan struct{})
	// clockStopped guards StopClock against a double close.
	clockStopped atomic.Bool
)

func init() {
	// Initialize the integer cache and seed the wall clock deterministically so
	// the very first Format call (before the ticker has ticked) still emits a
	// valid timestamp and never nil-dereferences globalTimeBytes. This is pure
	// CPU work with no goroutines — safe to do at package init.
	for i := 0; i < maxIntCache; i++ {
		smallInts[i] = strconv.Itoa(i)
	}
	updateTimeCache(time.Now())
}

// startClock lazily launches the stoppable wall-clock ticker. It is invoked on
// first text-formatter construction (via NewTextFormatter) rather than at
// package init, and runs at most once for the process lifetime.
func startClock() {
	clockOnce.Do(func() {
		go func() {
			for {
				now := time.Now()
				// Align to the next second boundary for cleaner logs.
				timer := time.NewTimer(time.Until(now.Truncate(time.Second).Add(time.Second)))
				select {
				case <-timer.C:
					updateTimeCache(time.Now())
				case <-clockStop:
					timer.Stop()
					return
				}
			}
		}()
	})
}

// StopClock tears down the background wall-clock ticker goroutine started by the
// text formatter. It is safe to call multiple times and from multiple
// goroutines. After StopClock, the cached timestamp stops advancing; construct
// a new formatter to restart it is NOT supported (the once has fired) — StopClock
// is intended for clean process shutdown / test teardown only.
func StopClock() {
	if clockStopped.CompareAndSwap(false, true) {
		close(clockStop)
	}
}

func updateTimeCache(t time.Time) {
	// Format: "[2025-11-24 12:00:00] "
	// Allocation here is irrelevant (once per second)
	b := make([]byte, 0, 24)
	b = append(b, '[')
	b = t.AppendFormat(b, "2006-01-02 15:04:05")
	b = append(b, ']', ' ')
	globalTimeBytes.Store(&b)
}

// --- The Formatter ---

// Formatter is a zero-allocation formatter that renders log entries as human-readable text.
type Formatter struct {
	// Padding prevents False Sharing (Cache Line thrashing)
	// Ensures this struct sits on its own cache line in the heap
	_ [64]byte
}

// NewTextFormatter creates a new text formatter and lazily starts the shared,
// stoppable wall-clock ticker on first use (see startClock / StopClock).
func NewTextFormatter() *Formatter {
	startClock()
	return &Formatter{}
}

// Format - Optimized for direct-to-destination writing
//
//go:noinline
func (f *Formatter) Format(entry *types.LogEntry, dst []byte) []byte {
	// 1. Fast Nil Check
	if entry == nil {
		return dst
	}

	// 2. Timestamp (Wall Clock Copy) - <2ns
	// Atomic load is cheap on x86/ARM (no lock)
	tBytes := *globalTimeBytes.Load()
	dst = append(dst, tBytes...)

	// 3. Level (Jump Table) - <2ns
	switch entry.Level {
	case types.InfoLevel:
		dst = append(dst, "INFO  - "...)
	case types.ErrorLevel:
		dst = append(dst, "ERROR - "...)
	case types.DebugLevel:
		dst = append(dst, "DEBUG - "...)
	case types.WarnLevel:
		dst = append(dst, "WARN  - "...)
	case types.TraceLevel:
		dst = append(dst, "TRACE - "...)
	case types.FatalLevel:
		dst = append(dst, "FATAL - "...)
	default:
		dst = append(dst, "UNK   - "...)
	}

	// 4. Message (Safety Enforced) - Memcpy speed
	msg := entry.Message
	if len(msg) > maxMessageSize {
		// Truncate efficiently without allocation
		dst = append(dst, msg[:maxMessageSize]...)
		dst = append(dst, "..."...)
	} else {
		dst = append(dst, msg...)
	}

	// 5. Caller Info (Hot Path Conditional)
	// We only pay the cost if File is present
	if entry.File != "" {
		dst = f.appendCaller(dst, entry.File, entry.Line)
	}

	// 6. Newline
	return append(dst, '\n')
}

// appendCaller - Extracted to keep Format small (I-Cache friendly)
func (f *Formatter) appendCaller(dst []byte, file string, line int) []byte {
	dst = append(dst, " ["...)

	// Fast Basename Extraction (No allocation)
	// Scan backwards for slash
	lastSlash := -1
	for i := len(file) - 1; i >= 0; i-- {
		if file[i] == '/' {
			lastSlash = i
			break
		}
	}
	dst = append(dst, file[lastSlash+1:]...)
	dst = append(dst, ':')

	// Fast Integer Append (Lookup Table)
	if line >= 0 && line < maxIntCache {
		dst = append(dst, smallInts[line]...)
	} else {
		dst = strconv.AppendInt(dst, int64(line), 10)
	}

	return append(dst, ']')
}

// --- Metrics & Control (Optional) ---

// Metrics is a stub retained for API compatibility.
type Metrics struct{} // Stub for API compatibility

// CopyMetrics copies formatter metrics into m. It is a no-op stub retained for API compatibility.
func (f *Formatter) CopyMetrics(m *Metrics) {}

// EstimatedSize returns an estimated buffer size for pre-allocation
func (f *Formatter) EstimatedSize() int {
	return 2048 // Default 2KB buffer size for text formatter
}

// Reset resets any internal state (for pooling)
func (f *Formatter) Reset() {
	// Stateless formatter - no-op
}

// GLOBAL LEVEL - used for zero-cost disabled path
var globalLevel atomicUint64 = atomicUint64{value: 2} // INFO

type atomicUint64 struct{ value uint64 }

func (a *atomicUint64) Load() uint64   { return atomic.LoadUint64(&a.value) }
func (a *atomicUint64) Store(v uint64) { atomic.StoreUint64(&a.value, v) }

// SetGlobalLevel sets the global minimum log level used by the zero-cost disabled path.
func SetGlobalLevel(level uint8) { globalLevel.Store(uint64(level)) }
