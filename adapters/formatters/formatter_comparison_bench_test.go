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

package formatters

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/go-gen-ecosystem/halolog/adapters/formatters/json"
	"github.com/go-gen-ecosystem/halolog/adapters/formatters/text"
	"github.com/go-gen-ecosystem/halolog/types"
)

// BenchmarkFormatterComparison_SimpleMessage compares all formatters with simple message
func BenchmarkFormatterComparison_SimpleMessage(b *testing.B) {
	entry := &types.LogEntry{
		Timestamp: time.Now(),
		Level:     types.InfoLevel,
		Message:   "Simple log message",
	}

	b.Run("JSON", func(b *testing.B) {
		formatter := json.NewJsonFormatter()
		dst := make([]byte, 0, 512)
		sample := formatter.Format(entry, dst[:0])

		b.SetBytes(int64(len(sample)))
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			out := formatter.Format(entry, dst[:0])
			dst = out[:0]
		}
	})

	b.Run("Text", func(b *testing.B) {
		formatter := text.NewTextFormatter()
		dst := make([]byte, 0, 512)
		sample := formatter.Format(entry, dst[:0])

		b.SetBytes(int64(len(sample)))
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			out := formatter.Format(entry, dst[:0])
			dst = out[:0]
		}
	})
}

// BenchmarkFormatterComparison_WithFields compares formatters with 10 fields
func BenchmarkFormatterComparison_WithFields(b *testing.B) {
	entry := &types.LogEntry{
		Timestamp: time.Now(),
		Level:     types.InfoLevel,
		Message:   "Message with fields",
		Fields: []types.TypedFieldData{
			{Key: "user_id", Value: int64(12345), Type: types.TypedFieldInt64},
			{Key: "session_id", Value: "abc-123-def-456", Type: types.TypedFieldString},
			{Key: "duration_ms", Value: 42.5, Type: types.TypedFieldFloat64},
			{Key: "status", Value: "success", Type: types.TypedFieldString},
			{Key: "count", Value: int64(100), Type: types.TypedFieldInt64},
			{Key: "enabled", Value: true, Type: types.TypedFieldBool},
			{Key: "ratio", Value: 0.95, Type: types.TypedFieldFloat64},
			{Key: "name", Value: "test", Type: types.TypedFieldString},
			{Key: "index", Value: int64(42), Type: types.TypedFieldInt64},
			{Key: "active", Value: true, Type: types.TypedFieldBool},
		},
	}

	b.Run("JSON", func(b *testing.B) {
		formatter := json.NewJsonFormatter()
		dst := make([]byte, 0, 1024)
		sample := formatter.Format(entry, dst[:0])

		b.SetBytes(int64(len(sample)))
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			out := formatter.Format(entry, dst[:0])
			dst = out[:0]
		}
	})

	b.Run("Text", func(b *testing.B) {
		formatter := text.NewTextFormatter()
		dst := make([]byte, 0, 1024)
		sample := formatter.Format(entry, dst[:0])

		b.SetBytes(int64(len(sample)))
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			out := formatter.Format(entry, dst[:0])
			dst = out[:0]
		}
	})
}

// BenchmarkFormatterComparison_30Fields compares formatters with 30 fields (WORLD #1 proof)
func BenchmarkFormatterComparison_30Fields(b *testing.B) {
	entry := &types.LogEntry{
		Timestamp: time.Now(),
		Level:     types.InfoLevel,
		Message:   "Message with 30 fields",
		Fields:    make([]types.TypedFieldData, 30),
	}

	for i := 0; i < 30; i++ {
		entry.Fields[i] = types.TypedFieldData{
			Key:   fmt.Sprintf("field_%d", i),
			Value: i * 100,
			Type:  types.TypedFieldInt64,
		}
	}

	b.Run("JSON", func(b *testing.B) {
		formatter := json.NewJsonFormatter()
		dst := make([]byte, 0, 2048)
		sample := formatter.Format(entry, dst[:0])

		b.SetBytes(int64(len(sample)))
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			out := formatter.Format(entry, dst[:0])
			dst = out[:0]
		}
	})

	b.Run("Text", func(b *testing.B) {
		formatter := text.NewTextFormatter()
		dst := make([]byte, 0, 2048)
		sample := formatter.Format(entry, dst[:0])

		b.SetBytes(int64(len(sample)))
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			out := formatter.Format(entry, dst[:0])
			dst = out[:0]
		}
	})
}

// BenchmarkFormatterComparison_WithError compares error formatting
func BenchmarkFormatterComparison_WithError(b *testing.B) {
	entry := &types.LogEntry{
		Timestamp: time.Now(),
		Level:     types.ErrorLevel,
		Message:   "Error occurred",
		Error:     errors.New("database connection failed"),
	}

	b.Run("JSON", func(b *testing.B) {
		formatter := json.NewJsonFormatter()
		dst := make([]byte, 0, 512)
		sample := formatter.Format(entry, dst[:0])

		b.SetBytes(int64(len(sample)))
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			out := formatter.Format(entry, dst[:0])
			dst = out[:0]
		}
	})

	b.Run("Text", func(b *testing.B) {
		formatter := text.NewTextFormatter()
		dst := make([]byte, 0, 512)
		sample := formatter.Format(entry, dst[:0])

		b.SetBytes(int64(len(sample)))
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			out := formatter.Format(entry, dst[:0])
			dst = out[:0]
		}
	})
}

// BenchmarkFormatterComparison_Concurrent compares concurrent performance
func BenchmarkFormatterComparison_Concurrent(b *testing.B) {
	entry := &types.LogEntry{
		Timestamp: time.Now(),
		Level:     types.InfoLevel,
		Message:   "Concurrent test",
	}

	b.Run("JSON", func(b *testing.B) {
		formatter := json.NewJsonFormatter()
		sample := formatter.Format(entry, nil)
		b.SetBytes(int64(len(sample)))
		b.ReportAllocs()

		b.RunParallel(func(pb *testing.PB) {
			dst := make([]byte, 0, 512)
			for pb.Next() {
				out := formatter.Format(entry, dst[:0])
				dst = out[:0]
			}
		})
	})

	b.Run("Text", func(b *testing.B) {
		formatter := text.NewTextFormatter()
		sample := formatter.Format(entry, nil)
		b.SetBytes(int64(len(sample)))
		b.ReportAllocs()

		b.RunParallel(func(pb *testing.PB) {
			dst := make([]byte, 0, 512)
			for pb.Next() {
				out := formatter.Format(entry, dst[:0])
				dst = out[:0]
			}
		})
	})
}

// BenchmarkFormatterComparison_RealWorld simulates real-world web request logging
func BenchmarkFormatterComparison_RealWorld(b *testing.B) {
	entry := &types.LogEntry{
		Timestamp: time.Now(),
		Level:     types.InfoLevel,
		Message:   "HTTP request completed",
		Component: "api.handler",
		File:      "handlers/user.go",
		Line:      142,
		Fields: []types.TypedFieldData{
			{Key: "method", Value: "POST", Type: types.TypedFieldString},
			{Key: "path", Value: "/api/v1/users", Type: types.TypedFieldString},
			{Key: "status_code", Value: int64(201), Type: types.TypedFieldInt64},
			{Key: "duration_ms", Value: 45.3, Type: types.TypedFieldFloat64},
			{Key: "bytes_sent", Value: int64(1024), Type: types.TypedFieldInt64},
		},
		Context: []types.TypedFieldData{
			{Key: "request_id", Value: "req-abc-123", Type: types.TypedFieldString},
			{Key: "user_id", Value: int64(12345), Type: types.TypedFieldInt64},
		},
	}

	b.Run("JSON", func(b *testing.B) {
		formatter := json.NewJsonFormatter()
		dst := make([]byte, 0, 2048)
		sample := formatter.Format(entry, dst[:0])

		b.SetBytes(int64(len(sample)))
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			out := formatter.Format(entry, dst[:0])
			dst = out[:0]
		}
	})

	b.Run("Text", func(b *testing.B) {
		formatter := text.NewTextFormatter()
		dst := make([]byte, 0, 2048)
		sample := formatter.Format(entry, dst[:0])

		b.SetBytes(int64(len(sample)))
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			out := formatter.Format(entry, dst[:0])
			dst = out[:0]
		}
	})
}

// BenchmarkFormatterComparison_Throughput measures ops/sec
func BenchmarkFormatterComparison_Throughput(b *testing.B) {
	entry := &types.LogEntry{
		Timestamp: time.Now(),
		Level:     types.InfoLevel,
		Message:   "Throughput test",
	}

	b.Run("JSON", func(b *testing.B) {
		formatter := json.NewJsonFormatter()
		dst := make([]byte, 0, 256)
		b.ReportAllocs()
		b.ResetTimer()

		start := time.Now()
		for i := 0; i < b.N; i++ {
			out := formatter.Format(entry, dst[:0])
			dst = out[:0]
		}
		elapsed := time.Since(start)

		opsPerSec := float64(b.N) / elapsed.Seconds()
		b.ReportMetric(opsPerSec, "ops/sec")
	})

	b.Run("Text", func(b *testing.B) {
		formatter := text.NewTextFormatter()
		dst := make([]byte, 0, 256)
		b.ReportAllocs()
		b.ResetTimer()

		start := time.Now()
		for i := 0; i < b.N; i++ {
			out := formatter.Format(entry, dst[:0])
			dst = out[:0]
		}
		elapsed := time.Since(start)

		opsPerSec := float64(b.N) / elapsed.Seconds()
		b.ReportMetric(opsPerSec, "ops/sec")
	})
}
