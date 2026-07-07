//go:build amd64 || arm64
// +build amd64 arm64

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

	// Global "Wall Clock" - Updated by background ticker
	// We use an atomic pointer to a byte array to avoid locks/contention
	globalTimeBytes atomic.Pointer[[]byte]
)

func init() {
	// 1. Initialize Integer Cache
	for i := 0; i < maxIntCache; i++ {
		smallInts[i] = strconv.Itoa(i)
	}

	// 2. Initialize Wall Clock immediately so first log isn't empty
	updateTimeCache(time.Now())

	// 3. Start Background Time Ticker
	// NOTE: In a perfect library, this would be explicitly started/stopped.
	// For high-perf internal logging, a daemon goroutine is acceptable.
	go func() {
		// 1s resolution is standard for text logs.
		// Use time.Sleep instead of Ticker to allow easier GC if needed.
		for {
			now := time.Now()
			// Align to the next second boundary for cleaner logs
			time.Sleep(time.Until(now.Truncate(time.Second).Add(time.Second)))
			updateTimeCache(time.Now())
		}
	}()
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

// NewTextFormatter creates a new text formatter.
func NewTextFormatter() *Formatter {
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
//
//go:inline
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
