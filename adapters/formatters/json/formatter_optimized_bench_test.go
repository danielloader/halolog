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

// BenchmarkHighPerformanceJSONFormatter_SimpleInfo benchmarks High-performance simple info
func BenchmarkOptimizedJSONFormatter_SimpleInfo(b *testing.B) {
	formatter := NewJsonFormatter()

	// Use fixed Unix timestamp to avoid time.Now() system calls in benchmark loop
	fixedTime := time.Unix(1700000000, 0) // Fixed timestamp for consistent benchmarking
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
		// Keep timestamp fixed to avoid system call overhead
		result := formatter.Format(entry, dst)
		dst = result[:0]
	}
}

// BenchmarkHighPerformanceJSONFormatter_SimpleInfoWithFile benchmarks High-performance with file info
func BenchmarkOptimizedJSONFormatter_SimpleInfoWithFile(b *testing.B) {
	formatter := NewJsonFormatter()

	// Use fixed Unix timestamp to avoid time.Now() system calls in benchmark loop
	fixedTime := time.Unix(1700000000, 0) // Fixed timestamp for consistent benchmarking
	entry := &types.LogEntry{
		Timestamp:     fixedTime,
		TimestampUnix: fixedTime.Unix(),
		Level:         types.InfoLevel,
		Message:       "hello world",
		File:          "benchmark.go",
		Line:          42,
	}

	dst := make([]byte, 0, 256)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Keep timestamp fixed to avoid system call overhead
		result := formatter.Format(entry, dst)
		dst = result[:0]
	}
}

// BenchmarkHighPerformanceJSONFormatter_WithField benchmarks High-performance with single field
func BenchmarkOptimizedJSONFormatter_WithField(b *testing.B) {
	formatter := NewJsonFormatter()

	// Use fixed Unix timestamp to avoid time.Now() system calls in benchmark loop
	fixedTime := time.Unix(1700000000, 0) // Fixed timestamp for consistent benchmarking
	entry := &types.LogEntry{
		Timestamp:     fixedTime,
		TimestampUnix: fixedTime.Unix(),
		Level:         types.InfoLevel,
		Message:       "hello world",
		File:          "benchmark.go",
		Line:          42,
		Fields: []types.TypedFieldData{
			{Key: "user_id", Value: "12345"},
		},
	}

	dst := make([]byte, 0, 256)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Keep timestamp fixed to avoid system call overhead
		result := formatter.Format(entry, dst)
		dst = result[:0]
	}
}

// BenchmarkHighPerformanceJSONFormatter_Parallel benchmarks concurrent formatting
func BenchmarkOptimizedJSONFormatter_Parallel(b *testing.B) {
	formatter := NewJsonFormatter()

	// Use fixed Unix timestamp to avoid time.Now() system calls in benchmark loop
	fixedTime := time.Unix(1700000000, 0) // Fixed timestamp for consistent benchmarking

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		entry := &types.LogEntry{
			Timestamp:     fixedTime,
			TimestampUnix: fixedTime.Unix(),
			Level:         types.InfoLevel,
			Message:       "hello world",
		}
		dst := make([]byte, 0, 256)

		for pb.Next() {
			// Keep timestamp fixed to avoid system call overhead
			result := formatter.Format(entry, dst)
			dst = result[:0]
		}
	})
}

// BenchmarkHighPerformanceJSONFormatter_Comparison compares with original formatter
func BenchmarkOptimizedJSONFormatter_Comparison(b *testing.B) {
	b.Run("Original", func(b *testing.B) {
		formatter := NewJsonFormatter()
		entry := &types.LogEntry{
			Timestamp: time.Now(),
			Level:     types.InfoLevel,
			Message:   "hello world",
			File:      "benchmark.go",
			Line:      42,
		}
		dst := make([]byte, 0, 512)

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			entry.Timestamp = time.Now()
			result := formatter.Format(entry, dst)
			dst = result[:0]
		}
	})

	b.Run("Optimized", func(b *testing.B) {
		formatter := NewJsonFormatter()
		// Use fixed Unix timestamp to avoid time.Now() system calls in benchmark loop
		fixedTime := time.Unix(1700000000, 0)
		entry := &types.LogEntry{
			Timestamp:     fixedTime,
			TimestampUnix: fixedTime.Unix(),
			Level:         types.InfoLevel,
			Message:       "hello world",
			File:          "benchmark.go",
			Line:          42,
		}
		dst := make([]byte, 0, 256)

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			result := formatter.Format(entry, dst)
			dst = result[:0]
		}
	})
}
