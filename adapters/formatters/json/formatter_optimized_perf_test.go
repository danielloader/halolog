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

package json

import (
	"testing"
	"time"

	"github.com/go-gen-ecosystem/halolog/types"
)

// BenchmarkHighPerformanceJSONFormatter_PurePerformance benchmarks pure formatting performance
// without JSON validation overhead for accurate performance measurement
func BenchmarkOptimizedJSONFormatter_PurePerformance(b *testing.B) {
	formatter := NewJsonFormatter()

	// Use fixed Unix timestamp to avoid time.Now() system calls in benchmark loop
	fixedTime := time.Unix(1700000000, 0)
	entry := &types.LogEntry{
		Timestamp:     fixedTime,
		TimestampUnix: fixedTime.Unix(),
		Level:         types.InfoLevel,
		Message:       "hello world",
	}

	dst := make([]byte, 0, 256)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		result := formatter.Format(entry, dst)
		dst = result[:0]
	}
}

// BenchmarkHighPerformanceJSONFormatter_StringEscaping benchmarks string escaping performance
func BenchmarkOptimizedJSONFormatter_StringEscaping(b *testing.B) {
	formatter := NewJsonFormatter()

	// Test cases with different escaping requirements
	testCases := []struct {
		name    string
		message string
	}{
		{"NoEscape", "hello world"},
		{"Quote", `hello "world"`},
		{"Backslash", `hello\world`},
		{"Newline", "hello\nworld"},
		{"Tab", "hello\tworld"},
		{"Mixed", `"Hello"\nWorld\t!`},
		{"Complex", `path\to\file"with"quotes\nand\newlines`},
	}

	fixedTime := time.Unix(1700000000, 0)

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			entry := &types.LogEntry{
				Timestamp:     fixedTime,
				TimestampUnix: fixedTime.Unix(),
				Level:         types.InfoLevel,
				Message:       tc.message,
			}

			dst := make([]byte, 0, 512)

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				result := formatter.Format(entry, dst)
				dst = result[:0]
			}
		})
	}
}

// BenchmarkHighPerformanceJSONFormatter_VariousMessageSizes benchmarks different message sizes
func BenchmarkOptimizedJSONFormatter_VariousMessageSizes(b *testing.B) {
	formatter := NewJsonFormatter()

	testMessages := []struct {
		name    string
		message string
	}{
		{"Tiny", "hi"},
		{"Small", "hello world"},
		{"Medium", "this is a medium length log message with some details"},
		{"Large", "this is a much longer log message that contains more detailed information about the system state and various operational parameters"},
		{"VeryLarge", "this is an extremely long log message that simulates detailed debug output with extensive contextual information, system metrics, performance data, and various operational details that would typically be found in comprehensive logging scenarios"},
	}

	fixedTime := time.Unix(1700000000, 0)

	for _, tm := range testMessages {
		b.Run(tm.name, func(b *testing.B) {
			entry := &types.LogEntry{
				Timestamp:     fixedTime,
				TimestampUnix: fixedTime.Unix(),
				Level:         types.InfoLevel,
				Message:       tm.message,
			}

			// Pre-allocate appropriate buffer size based on message length
			bufferSize := len(tm.message) + 128 // Base overhead for JSON structure
			dst := make([]byte, 0, bufferSize)

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				result := formatter.Format(entry, dst)
				dst = result[:0]
			}
		})
	}
}

// BenchmarkHighPerformanceJSONFormatter_BufferReuse benchmarks buffer reuse efficiency
func BenchmarkOptimizedJSONFormatter_BufferReuse(b *testing.B) {
	formatter := NewJsonFormatter()

	fixedTime := time.Unix(1700000000, 0)
	entry := &types.LogEntry{
		Timestamp:     fixedTime,
		TimestampUnix: fixedTime.Unix(),
		Level:         types.InfoLevel,
		Message:       "buffer reuse test message",
	}

	b.ResetTimer()
	b.ReportAllocs()

	// Test different buffer reuse strategies
	b.Run("SmallBuffer", func(b *testing.B) {
		dst := make([]byte, 0, 64)
		for i := 0; i < b.N; i++ {
			result := formatter.Format(entry, dst)
			dst = result[:0]
		}
	})

	b.Run("OptimalBuffer", func(b *testing.B) {
		dst := make([]byte, 0, 256)
		for i := 0; i < b.N; i++ {
			result := formatter.Format(entry, dst)
			dst = result[:0]
		}
	})

	b.Run("LargeBuffer", func(b *testing.B) {
		dst := make([]byte, 0, 1024)
		for i := 0; i < b.N; i++ {
			result := formatter.Format(entry, dst)
			dst = result[:0]
		}
	})
}

// BenchmarkHighPerformanceJSONFormatter_CompleteEntry benchmarks full-featured log entries
func BenchmarkOptimizedJSONFormatter_CompleteEntry(b *testing.B) {
	formatter := NewJsonFormatter()

	fixedTime := time.Unix(1700000000, 0)
	entry := &types.LogEntry{
		Timestamp:     fixedTime,
		TimestampUnix: fixedTime.Unix(),
		Level:         types.InfoLevel,
		Message:       "user login successful",
		File:          "auth/service.go",
		Line:          142,
		Fields: []types.TypedFieldData{
			{Key: "user_id", Value: "12345"},
			{Key: "username", Value: "john_doe"},
			{Key: "ip_address", Value: "192.168.1.100"},
			{Key: "user_agent", Value: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"},
		},
	}

	dst := make([]byte, 0, 1024)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		result := formatter.Format(entry, dst)
		dst = result[:0]
	}
}
