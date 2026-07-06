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

	"github.com/go-gen-ecosystem/halolog/types"
)

// discardAdapter is a no-op adapter for testing.
type discardAdapter struct{}

func (d *discardAdapter) Name() string                          { return "discard" }
func (d *discardAdapter) Write(entry *types.LogEntry) error     { return nil }
func (d *discardAdapter) WriteZero(entry *types.LogEntry) error { return nil }
func (d *discardAdapter) Flush() error                          { return nil }
func (d *discardAdapter) Close() error                          { return nil }
func (d *discardAdapter) SetFormatter(f types.Formatter)        {}
func (d *discardAdapter) Health() error                         { return nil }

// TestLoggerCreation tests that NewLogger creates a valid logger.
func TestLoggerCreation(t *testing.T) {
	logger := NewLogger(Config{
		Component: "test",
		Level:     types.InfoLevel,
		Adapters:  []types.Adapter{&discardAdapter{}},
	})

	if logger.Component() != "test" {
		t.Errorf("Expected component 'test', got '%s'", logger.Component())
	}
	if logger.Level() != types.InfoLevel {
		t.Errorf("Expected InfoLevel, got %v", logger.Level())
	}
}

// TestLoggerLevelFiltering tests that log levels are correctly filtered.
func TestLoggerLevelFiltering(t *testing.T) {
	callCount := 0
	testAdapter := &types.FuncAdapter{
		WriteFunc: func(entry *types.LogEntry) error {
			callCount++
			return nil
		},
	}

	logger := NewLogger(Config{
		Component: "test",
		Level:     types.WarnLevel, // Only warn and above
		Adapters:  []types.Adapter{testAdapter},
	})

	// These should be filtered out
	logger.Debug("debug message")
	logger.Info("info message")

	if callCount != 0 {
		t.Errorf("Expected 0 calls (filtered), got %d", callCount)
	}

	// These should be logged
	logger.Warn("warn message")
	logger.Error("error message")

	if callCount != 2 {
		t.Errorf("Expected 2 calls, got %d", callCount)
	}
}

// TestLoggerFluentAPI tests the fluent API.
//
// The log entry is pooled and recycled once the adapter returns (zero-allocation
// design), so a conformant adapter must read what it needs during Write. The
// test therefore snapshots the fields inside the callback rather than retaining
// the entry pointer and reading it after the call recycles it.
func TestLoggerFluentAPI(t *testing.T) {
	var (
		captured        bool
		gotMessage      string
		gotLevel        types.LogLevel
		gotFieldCount   int
	)
	testAdapter := &types.FuncAdapter{
		WriteFunc: func(entry *types.LogEntry) error {
			captured = true
			gotMessage = entry.Message
			gotLevel = entry.Level
			gotFieldCount = entry.StaticFieldCount
			return nil
		},
	}

	logger := NewLogger(Config{
		Component: "test",
		Level:     types.DebugLevel,
		Adapters:  []types.Adapter{testAdapter},
	})

	logger.WithField("key1", "value1").
		WithField("key2", 42).
		Info("test message")

	if !captured {
		t.Fatal("Expected entry to be captured")
	}
	if gotMessage != "test message" {
		t.Errorf("Expected 'test message', got '%s'", gotMessage)
	}
	if gotLevel != types.InfoLevel {
		t.Errorf("Expected InfoLevel, got %v", gotLevel)
	}
	if gotFieldCount != 2 {
		t.Errorf("Expected 2 fields, got %d", gotFieldCount)
	}
}

// TestLoggerMultipleAdapters tests logging to multiple adapters.
func TestLoggerMultipleAdapters(t *testing.T) {
	callCounts := make([]int, 2)

	adapter1 := &types.FuncAdapter{
		WriteFunc: func(entry *types.LogEntry) error {
			callCounts[0]++
			return nil
		},
	}
	adapter2 := &types.FuncAdapter{
		WriteFunc: func(entry *types.LogEntry) error {
			callCounts[1]++
			return nil
		},
	}

	logger := NewLogger(Config{
		Component: "test",
		Level:     types.DebugLevel,
		Adapters:  []types.Adapter{adapter1, adapter2},
	})

	logger.Info("test message")

	if callCounts[0] != 1 || callCounts[1] != 1 {
		t.Errorf("Expected both adapters to be called once, got %v", callCounts)
	}
}

// TestLoggerWithMetrics tests that metrics are recorded when enabled.
func TestLoggerWithMetrics(t *testing.T) {
	logger := NewLogger(Config{
		Component:     "test",
		Level:         types.DebugLevel,
		Adapters:      []types.Adapter{&discardAdapter{}},
		EnableMetrics: true,
	})

	if logger.metrics == nil {
		t.Fatal("Expected metrics to be enabled")
	}

	logger.Info("test 1")
	logger.Info("test 2")
	logger.Warn("test 3")

	if logger.metrics.counts[types.InfoLevel].Load() != 2 {
		t.Errorf("Expected 2 info logs, got %d", logger.metrics.counts[types.InfoLevel].Load())
	}
	if logger.metrics.counts[types.WarnLevel].Load() != 1 {
		t.Errorf("Expected 1 warn log, got %d", logger.metrics.counts[types.WarnLevel].Load())
	}
}
