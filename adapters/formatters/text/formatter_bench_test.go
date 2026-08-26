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
	"fmt"
	"strings"
	"testing"

	"github.com/go-gen-ecosystem/halolog/types"
)

// BenchmarkFormatter_SimpleMessage - Baseline performance (Timestamp + Level + Message + Newline)
func BenchmarkFormatter_SimpleMessage(b *testing.B) {
	f := NewTextFormatter()
	entry := &types.LogEntry{
		// Timestamp is not set here as the formatter uses the global wall clock cache
		Level:   types.InfoLevel,
		Message: "Simple log message",
		// Caller info fields are empty to test the shortest path
	}

	dst := make([]byte, 0, stackBufferSize)
	// Calculate size for accurate b.SetBytes()
	size := len(f.Format(entry, dst[:0]))
	b.SetBytes(int64(size))

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Reset slice length to 0 but reuse capacity
		_ = f.Format(entry, dst[:0])
	}
}

// BenchmarkFormatter_WithCaller - Full context (Hot Path: Basename + Fast Line Num)
func BenchmarkFormatter_WithCaller(b *testing.B) {
	f := NewTextFormatter()
	entry := &types.LogEntry{
		Level:   types.InfoLevel,
		Message: "Message with full caller info and low line number",
		// Use a deep path to test the efficient basename extraction logic
		File: "/usr/local/go/src/runtime/internal/sys/handler.go",
		Line: 142, // Line < 1000 uses the fast smallInts cache
	}

	dst := make([]byte, 0, stackBufferSize)
	size := len(f.Format(entry, dst[:0]))
	b.SetBytes(int64(size))

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = f.Format(entry, dst[:0])
	}
}

// BenchmarkFormatter_LargeMessage - Stress test the new Truncation Logic
func BenchmarkFormatter_LargeMessage(b *testing.B) {
	f := NewTextFormatter()

	// Create a message significantly larger than maxMessageSize (1024)
	hugeMessage := strings.Repeat("x", maxMessageSize*2)

	entry := &types.LogEntry{
		Level:   types.InfoLevel,
		Message: hugeMessage,
		File:    "test.go",
		Line:    1,
	}

	dst := make([]byte, 0, stackBufferSize)
	// The formatter truncates, so size will be maxMessageSize + headers
	size := len(f.Format(entry, dst[:0]))
	b.SetBytes(int64(size))

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// This tests the branch that checks length and truncates, ensuring zero allocs
		_ = f.Format(entry, dst[:0])
	}
}

// BenchmarkFormatter_AllLevels - Level formatting cost (tests jump table efficiency)
func BenchmarkFormatter_AllLevels(b *testing.B) {
	levels := []types.LogLevel{
		types.TraceLevel, types.DebugLevel, types.InfoLevel,
		types.WarnLevel, types.ErrorLevel, types.FatalLevel,
	}

	for _, level := range levels {
		b.Run(fmt.Sprintf("level_%s", level), func(b *testing.B) {
			f := NewTextFormatter()
			entry := &types.LogEntry{
				Level:   level,
				Message: "test message",
			}

			dst := make([]byte, 0, stackBufferSize)

			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_ = f.Format(entry, dst[:0])
			}
		})
	}
}

// BenchmarkFormatter_Concurrent - Scalability test (measures contention on globalTimeBytes)
func BenchmarkFormatter_Concurrent(b *testing.B) {
	f := NewTextFormatter()
	entry := &types.LogEntry{
		Level:   types.InfoLevel,
		Message: "concurrent test message to stress the atomic wall clock read",
		File:    "/a/b/c/test.go",
		Line:    42,
	}

	dst := make([]byte, 0, stackBufferSize)
	size := len(f.Format(entry, dst[:0]))
	b.SetBytes(int64(size))

	b.ReportAllocs()
	b.ResetTimer()

	// Run multiple goroutines in parallel, each formatting the same entry
	b.RunParallel(func(pb *testing.PB) {
		// Each goroutine must have its own output buffer to ensure zero allocs locally
		localDst := make([]byte, 0, stackBufferSize)
		for pb.Next() {
			_ = f.Format(entry, localDst[:0])
		}
	})
}

// BenchmarkFormatter_CallerDigitCount - Measures line number performance (cache vs. strconv)
func BenchmarkFormatter_CallerDigitCount(b *testing.B) {
	f := NewTextFormatter()

	// Test the fast path (cached) and the cold path (strconv)
	lines := []int{10, 999, 1000, 10000}

	for _, line := range lines {
		b.Run(fmt.Sprintf("line_%d", line), func(b *testing.B) {
			entry := &types.LogEntry{
				Level:   types.InfoLevel,
				Message: "test",
				File:    "test.go",
				Line:    line,
			}

			dst := make([]byte, 0, stackBufferSize)

			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_ = f.Format(entry, dst[:0])
			}
		})
	}
}

// BenchmarkFormatter_ZeroAllocStrict - Strict zero-allocation verification
func BenchmarkFormatter_ZeroAllocStrict(b *testing.B) {
	f := NewTextFormatter()
	entry := &types.LogEntry{
		Level:   types.InfoLevel,
		Message: "zero alloc test",
		// Skip File and Line to test pure formatting performance without caller overhead
	}

	dst := make([]byte, 0, stackBufferSize)
	// Pre-calculate size for accurate benchmarking like SimpleMessage
	size := len(f.Format(entry, dst[:0]))
	b.SetBytes(int64(size))

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = f.Format(entry, dst[:0])
	}
}
