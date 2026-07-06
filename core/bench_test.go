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

package core

import (
	"testing"

	"github.com/go-gen-ecosystem/halolog/adapters/outputs/discard"
	"github.com/go-gen-ecosystem/halolog/types"
)

// benchmarkAdapter is a no-op adapter for benchmarks.
type benchmarkAdapter struct{}

func (b *benchmarkAdapter) Name() string                          { return "benchmark" }
func (b *benchmarkAdapter) Write(entry *types.LogEntry) error     { return nil }
func (b *benchmarkAdapter) WriteZero(entry *types.LogEntry) error { return nil }
func (b *benchmarkAdapter) Flush() error                          { return nil }
func (b *benchmarkAdapter) Close() error                          { return nil }
func (b *benchmarkAdapter) SetFormatter(f types.Formatter)        {}
func (b *benchmarkAdapter) Health() error                         { return nil }

var benchLogger *Logger

func init() {
	benchLogger = NewLogger(Config{
		Component: "bench",
		Level:     types.DebugLevel,
		Adapters:  []types.Adapter{&benchmarkAdapter{}},
	})
}

// BenchmarkLoggerInfoDiscard benchmarks Info with discard adapter.
// Target: < 2 ns/op (clean package achieves 1.68 ns/op)
// This is the primary benchmark for the hot path.
func BenchmarkLoggerInfoDiscard(b *testing.B) {
	logger := NewLogger(Config{
		Component: "bench",
		Level:     types.DebugLevel,
		Adapters:  []types.Adapter{discard.New()},
	})

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		logger.Info("test message")
	}
}

// BenchmarkLoggerInfo benchmarks simple Info logging with interface adapter.
// Target: < 10 ns/op
func BenchmarkLoggerInfo(b *testing.B) {
	logger := NewLogger(Config{
		Component: "bench",
		Level:     types.DebugLevel,
		Adapters:  []types.Adapter{&benchmarkAdapter{}},
	})

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		logger.Info("test message")
	}
}

// BenchmarkLoggerInfoDisabled benchmarks Info when level is disabled.
// Target: ~0 ns/op (no-op function)
func BenchmarkLoggerInfoDisabled(b *testing.B) {
	logger := NewLogger(Config{
		Component: "bench",
		Level:     types.WarnLevel, // Info is disabled
		Adapters:  []types.Adapter{&benchmarkAdapter{}},
	})

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		logger.Info("test message")
	}
}

// BenchmarkLoggerWithField1Field benchmarks logging with 1 field.
// Target: < 20 ns/op
func BenchmarkLoggerWithField1Field(b *testing.B) {
	logger := NewLogger(Config{
		Component: "bench",
		Level:     types.DebugLevel,
		Adapters:  []types.Adapter{&benchmarkAdapter{}},
	})

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		logger.WithField("key", "value").Info("test message")
	}
}

// BenchmarkLoggerWithField1FieldDiscard benchmarks logging with 1 field and discard adapter.
// Target: < 20 ns/op
func BenchmarkLoggerWithField1FieldDiscard(b *testing.B) {
	logger := NewLogger(Config{
		Component: "bench",
		Level:     types.DebugLevel,
		Adapters:  []types.Adapter{discard.New()},
	})

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		logger.WithField("key", "value").Info("test message")
	}
}

// BenchmarkLoggerTyped1FieldDiscard benchmarks TypedFieldBuilder with 1 field.
// Target: < 15 ns/op (no interface{} boxing)
func BenchmarkLoggerTyped1FieldDiscard(b *testing.B) {
	logger := NewLogger(Config{
		Component: "bench",
		Level:     types.DebugLevel,
		Adapters:  []types.Adapter{discard.New()},
	})

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		logger.Typed().WithString("key", "value").Info("test message")
	}
}

// BenchmarkLoggerWithField5Fields benchmarks logging with 5 fields.
// Target: < 35 ns/op
func BenchmarkLoggerWithField5Fields(b *testing.B) {
	logger := NewLogger(Config{
		Component: "bench",
		Level:     types.DebugLevel,
		Adapters:  []types.Adapter{&benchmarkAdapter{}},
	})

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		logger.WithField("key1", "value1").
			WithField("key2", "value2").
			WithField("key3", "value3").
			WithField("key4", "value4").
			WithField("key5", "value5").
			Info("test message")
	}
}

// BenchmarkLoggerWithField10Fields benchmarks logging with 10 fields.
// Target: < 45 ns/op
func BenchmarkLoggerWithField10Fields(b *testing.B) {
	logger := NewLogger(Config{
		Component: "bench",
		Level:     types.DebugLevel,
		Adapters:  []types.Adapter{&benchmarkAdapter{}},
	})

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		logger.WithField("key1", "value1").
			WithField("key2", "value2").
			WithField("key3", "value3").
			WithField("key4", "value4").
			WithField("key5", "value5").
			WithField("key6", "value6").
			WithField("key7", "value7").
			WithField("key8", "value8").
			WithField("key9", "value9").
			WithField("key10", "value10").
			Info("test message")
	}
}

// BenchmarkLoggerWithField20Fields benchmarks logging with 20 fields.
// Target: < 100 ns/op
func BenchmarkLoggerWithField20Fields(b *testing.B) {
	logger := NewLogger(Config{
		Component: "bench",
		Level:     types.DebugLevel,
		Adapters:  []types.Adapter{&benchmarkAdapter{}},
	})

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		fb := logger.WithField("key1", "value1")
		for j := 2; j <= 20; j++ {
			fb = fb.WithField("key", j)
		}
		fb.Info("test message")
	}
}

// BenchmarkLoggerWithFieldMetrics benchmarks logging with metrics enabled.
func BenchmarkLoggerWithFieldMetrics(b *testing.B) {
	logger := NewLogger(Config{
		Component:     "bench",
		Level:         types.DebugLevel,
		Adapters:      []types.Adapter{&benchmarkAdapter{}},
		EnableMetrics: true,
	})

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		logger.Info("test message")
	}
}

// BenchmarkEntry benchmarks Entry creation and field addition.
func BenchmarkEntry(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		var entry Entry
		entry.Level = types.InfoLevel
		entry.Message = "test message"
		entry.AddField("key", "value")
		_ = entry
	}
}

// BenchmarkEntryToLogEntry benchmarks Entry to LogEntry conversion.
func BenchmarkEntryToLogEntry(b *testing.B) {
	var entry Entry
	entry.Level = types.InfoLevel
	entry.Message = "test message"
	entry.AddField("key", "value")

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		var logEntry types.LogEntry
		entry.ToLogEntry(&logEntry)
		_ = logEntry
	}
}

// BenchmarkParallelLogging benchmarks concurrent logging.
func BenchmarkParallelLogging(b *testing.B) {
	logger := NewLogger(Config{
		Component: "bench",
		Level:     types.DebugLevel,
		Adapters:  []types.Adapter{&benchmarkAdapter{}},
	})

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Info("test message")
		}
	})
}
