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
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-gen-ecosystem/halolog/types"
	"github.com/go-gen-ecosystem/halolog/utils"
)

// Pre-allocated errors for test mocks
var (
	ErrMockWrite     = errors.New("mock write error")
	ErrMockWriteZero = errors.New("mock write zero error")
)

// MockAdapter for async testing
type asyncMockAdapter struct {
	name       string
	writeCalls int
	mu         sync.Mutex
}

func (m *asyncMockAdapter) Name() string {
	return m.name
}

func (m *asyncMockAdapter) Write(entry *types.LogEntry) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.writeCalls++
	return nil
}

// writes returns the write count under the mutex, so tests can read it without
// racing the background writer goroutine.
func (m *asyncMockAdapter) writes() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.writeCalls
}

func (m *asyncMockAdapter) Flush() error {
	return nil
}

func (m *asyncMockAdapter) SetFormatter(formatter types.Formatter) {
	// Mock implementation
}

func (m *asyncMockAdapter) WriteZero(entry *types.LogEntry) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.writeCalls++
	return nil
}

func (m *asyncMockAdapter) Close() error {
	return nil
}

func (m *asyncMockAdapter) Health() error {
	// Mock health check - always healthy
	return nil
}

func TestAsyncAdapter_BasicUsage(t *testing.T) {
	mock := &asyncMockAdapter{name: "test"}
	adapter := NewAsyncAdapter(mock, nil)

	// Test basic usage
	entry := &types.LogEntry{
		Level:   types.InfoLevel,
		Message: "Test message",
	}

	err := adapter.Write(entry)
	if err != nil {
		t.Errorf("Write should succeed: %v", err)
	}

	// Give async adapter time to process
	_ = adapter.Flush()

	if mock.writes() != 1 {
		t.Errorf("Expected 1 write call, got %d", mock.writes())
	}
}

func TestAsyncAdapter_Name(t *testing.T) {
	mock := &asyncMockAdapter{name: "test-adapter"}
	adapter := NewAsyncAdapter(mock, nil)

	name := adapter.Name()
	expected := "AsyncAdapter(test-adapter)"
	if name != expected {
		t.Errorf("Expected name %s, got %s", expected, name)
	}
}

func TestAsyncAdapter_Flush(t *testing.T) {
	mock := &asyncMockAdapter{name: "test"}
	adapter := NewAsyncAdapter(mock, nil)

	err := adapter.Flush()
	if err != nil {
		t.Errorf("Flush should succeed: %v", err)
	}
}

func TestAsyncAdapter_Close(t *testing.T) {
	mock := &asyncMockAdapter{name: "test"}
	adapter := NewAsyncAdapter(mock, nil)

	err := adapter.Close()
	if err != nil {
		t.Errorf("Close should succeed: %v", err)
	}
}

func TestAsyncAdapter_SetFormatter(t *testing.T) {
	mock := &asyncMockAdapter{name: "test"}
	adapter := NewAsyncAdapter(mock, nil)

	// Should not panic
	adapter.SetFormatter(nil)
	t.Log("Formatter set successfully")
}

func TestAsyncAdapter_Write(t *testing.T) {
	mock := &asyncMockAdapter{name: "test"}
	adapter := NewAsyncAdapter(mock, &AsyncAdapterOptions{
		BufferSize:    100,
		BatchSize:     10,
		FlushInterval: 10 * time.Millisecond,
	})
	defer func() { _ = adapter.Close() }()

	// Test writing a single entry
	entry := &types.LogEntry{
		Level:     types.InfoLevel,
		Message:   "Test async message",
		Timestamp: time.Now(),
		Fields:    []types.TypedFieldData{types.String("key", "value")},
	}

	err := adapter.Write(entry)
	if err != nil {
		t.Errorf("Write should not error: %v", err)
	}

	// Give async adapter time to process
	time.Sleep(50 * time.Millisecond)

	if mock.writes() != 1 {
		t.Errorf("Expected 1 write call, got %d", mock.writes())
	}
}

func TestAsyncAdapter_WriteNilEntry(t *testing.T) {
	mock := &asyncMockAdapter{name: "test"}
	adapter := NewAsyncAdapter(mock, nil)
	defer func() { _ = adapter.Close() }()

	err := adapter.Write(nil)
	if err == nil {
		t.Error("Expected error when writing nil entry")
	}
	if !strings.Contains(err.Error(), "cannot write nil entry") {
		t.Errorf("Expected specific error message, got: %v", err)
	}
}

func TestAsyncAdapter_MultipleWrites(t *testing.T) {
	mock := &asyncMockAdapter{name: "test"}
	adapter := NewAsyncAdapter(mock, &AsyncAdapterOptions{
		BufferSize:    100,
		BatchSize:     10,
		FlushInterval: 10 * time.Millisecond,
	})
	defer func() { _ = adapter.Close() }()

	// Write multiple entries
	for i := 0; i < 10; i++ {
		entry := &types.LogEntry{
			Level:     types.InfoLevel,
			Message:   utils.FormatIntWithPrefix("Message", i),
			Timestamp: time.Now(),
		}
		err := adapter.Write(entry)
		if err != nil {
			t.Errorf("Write %d should not error: %v", i, err)
		}
	}

	// Give async adapter time to process all entries
	time.Sleep(100 * time.Millisecond)

	if mock.writes() != 10 {
		t.Errorf("Expected 10 write calls, got %d", mock.writes())
	}
}

func TestAsyncAdapter_Flush_Comprehensive(t *testing.T) {
	mock := &asyncMockAdapter{name: "test"}
	adapter := NewAsyncAdapter(mock, &AsyncAdapterOptions{
		BufferSize:    100,
		BatchSize:     10,
		FlushInterval: 10 * time.Millisecond,
	})
	defer func() { _ = adapter.Close() }()

	// Write some entries
	for i := 0; i < 5; i++ {
		entry := &types.LogEntry{
			Level:     types.InfoLevel,
			Message:   utils.FormatIntWithPrefix("Flush test", i),
			Timestamp: time.Now(),
		}
		_ = adapter.Write(entry)
	}

	// Flush should wait for all entries to be processed
	err := adapter.Flush()
	if err != nil {
		t.Errorf("Flush should not error: %v", err)
	}

	if mock.writes() != 5 {
		t.Errorf("Expected 5 write calls after flush, got %d", mock.writes())
	}
}

func TestAsyncAdapter_FlushEmpty(t *testing.T) {
	mock := &asyncMockAdapter{name: "test"}
	adapter := NewAsyncAdapter(mock, nil)
	defer func() { _ = adapter.Close() }()

	// Flush with no entries should not error
	err := adapter.Flush()
	if err != nil {
		t.Errorf("Flush should not error when empty: %v", err)
	}

	if mock.writes() != 0 {
		t.Errorf("Expected 0 write calls, got %d", mock.writes())
	}
}

func TestAsyncAdapter_Close_Comprehensive(t *testing.T) {
	mock := &asyncMockAdapter{name: "test"}
	adapter := NewAsyncAdapter(mock, nil)

	// Write some entries
	for i := 0; i < 3; i++ {
		entry := &types.LogEntry{
			Level:     types.InfoLevel,
			Message:   utils.FormatIntWithPrefix("Close test", i),
			Timestamp: time.Now(),
		}
		_ = adapter.Write(entry)
	}

	// Close should process remaining entries and shut down cleanly
	err := adapter.Close()
	if err != nil {
		t.Errorf("Close should not error: %v", err)
	}

	// Should have processed all entries
	if mock.writes() != 3 {
		t.Errorf("Expected 3 write calls after close, got %d", mock.writes())
	}

	// Writing after close should error
	err = adapter.Write(&types.LogEntry{Level: types.InfoLevel, Message: "After close"})
	if err == nil {
		t.Error("Expected error when writing after close")
	}
}

func TestAsyncAdapter_BackgroundWriter(t *testing.T) {
	mock := &asyncMockAdapter{name: "test"}

	// Create adapter with small batch size to test batching
	adapter := NewAsyncAdapter(mock, &AsyncAdapterOptions{
		BufferSize:    100,
		BatchSize:     3,
		FlushInterval: 10 * time.Millisecond,
	})
	defer func() { _ = adapter.Close() }()

	// Write exactly batch size entries
	for i := 0; i < 3; i++ {
		entry := &types.LogEntry{
			Level:     types.InfoLevel,
			Message:   utils.FormatIntWithPrefix("Batch test", i),
			Timestamp: time.Now(),
		}
		_ = adapter.Write(entry)
	}

	// Give background writer time to process
	time.Sleep(50 * time.Millisecond)

	// Should have processed all entries
	if mock.writes() != 3 {
		t.Errorf("Expected 3 write calls, got %d", mock.writes())
	}
}

func TestAsyncAdapter_FlushBatch(t *testing.T) {
	mock := &asyncMockAdapter{name: "test"}

	// Create adapter with batch size
	adapter := NewAsyncAdapter(mock, &AsyncAdapterOptions{
		BatchSize: 5,
	})
	defer func() { _ = adapter.Close() }()

	// Write less than batch size entries
	for i := 0; i < 3; i++ {
		entry := &types.LogEntry{
			Level:     types.InfoLevel,
			Message:   utils.FormatIntWithPrefix("Partial batch", i),
			Timestamp: time.Now(),
		}
		_ = adapter.Write(entry)
	}

	// Give some time for entries to be processed
	time.Sleep(10 * time.Millisecond)

	// Flush should force processing of partial batch
	err := adapter.Flush()
	if err != nil {
		t.Errorf("Flush should not error: %v", err)
	}

	if mock.writes() != 3 {
		t.Errorf("Expected 3 write calls after flush, got %d", mock.writes())
	}
}

func TestAsyncAdapter_WriteError(t *testing.T) {
	// Create a mock adapter that returns an error
	mock := &asyncErrorAdapter{name: "error-test"}
	adapter := NewAsyncAdapter(mock, &AsyncAdapterOptions{
		BatchSize:     1, // Force immediate batch processing
		FlushInterval: 10 * time.Millisecond,
	})
	defer func() { _ = adapter.Close() }()

	entry := &types.LogEntry{
		Level:     types.InfoLevel,
		Message:   "Error test message",
		Timestamp: time.Now(),
	}

	err := adapter.Write(entry)
	if err != nil {
		t.Errorf("Write should not error: %v", err)
	}

	// Wait for background writer to process the entry
	time.Sleep(50 * time.Millisecond)

	// Now test direct write through Flush
	err = adapter.Flush()
	if err == nil {
		t.Error("Expected error from underlying adapter")
	}
	if err != ErrAsyncWriteEntry {
		t.Errorf("Expected ErrAsyncWriteEntry, got: %v", err)
	}
}

func TestAsyncAdapter_ConcurrentWrites(t *testing.T) {
	mock := &asyncMockAdapter{name: "concurrent-test"}
	adapter := NewAsyncAdapter(mock, &AsyncAdapterOptions{
		BufferSize:    1000,
		BatchSize:     10,
		FlushInterval: 10 * time.Millisecond,
	})
	defer func() { _ = adapter.Close() }()

	var wg sync.WaitGroup
	numGoroutines := 10
	writesPerGoroutine := 10

	// Launch concurrent writers
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < writesPerGoroutine; j++ {
				entry := &types.LogEntry{
					Level:     types.InfoLevel,
					Message:   utils.FormatTwoIntsSimple("Concurrent", id, '-', j, ""),
					Timestamp: time.Now(),
				}
				_ = adapter.Write(entry)
			}
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()

	// Give background writer time to process
	time.Sleep(50 * time.Millisecond)

	// Flush to ensure all entries are processed
	_ = adapter.Flush()

	expectedWrites := numGoroutines * writesPerGoroutine
	if mock.writes() != expectedWrites {
		t.Errorf("Expected %d write calls, got %d", expectedWrites, mock.writes())
	}
}

func TestAsyncAdapter_BufferFull(t *testing.T) {
	mock := &asyncMockAdapter{name: "buffer-test"}

	// Create adapter with reasonable buffer and small batch size
	adapter := NewAsyncAdapter(mock, &AsyncAdapterOptions{
		BufferSize:    100,
		BatchSize:     5,
		FlushInterval: 10 * time.Millisecond,
	})
	defer func() { _ = adapter.Close() }()

	// Write many entries quickly to test buffer handling
	for i := 0; i < 20; i++ {
		entry := &types.LogEntry{
			Level:     types.InfoLevel,
			Message:   utils.FormatIntWithPrefix("Buffer test", i),
			Timestamp: time.Now(),
		}
		_ = adapter.Write(entry)
	}

	// Give background writer time to process
	time.Sleep(50 * time.Millisecond)

	// Flush to ensure all entries are processed
	_ = adapter.Flush()

	if mock.writes() != 20 {
		t.Errorf("Expected 20 write calls, got %d", mock.writes())
	}
}

// Mock adapter that returns errors
type asyncErrorAdapter struct {
	name string
}

func (m *asyncErrorAdapter) Name() string {
	return m.name
}

func (m *asyncErrorAdapter) Write(entry *types.LogEntry) error {
	return ErrMockWrite
}

func (m *asyncErrorAdapter) Flush() error {
	return nil
}

func (m *asyncErrorAdapter) SetFormatter(formatter types.Formatter) {
	// Mock implementation
}

func (m *asyncErrorAdapter) WriteZero(entry *types.LogEntry) error {
	return ErrMockWriteZero
}

func (m *asyncErrorAdapter) Close() error {
	return nil
}

func (m *asyncErrorAdapter) Health() error {
	// Mock health check - always healthy
	return nil
}

// BenchmarkAsyncAdapter_Info_HelloWorld benchmarks simple info message
func BenchmarkAsyncAdapter_Info_HelloWorld(b *testing.B) {
	mock := &asyncMockAdapter{name: "benchmark"}
	adapter := NewAsyncAdapter(mock, &AsyncAdapterOptions{
		BufferSize:    1000,
		BatchSize:     100,
		FlushInterval: 10 * time.Millisecond,
	})
	defer func() { _ = adapter.Close() }()

	entry := &types.LogEntry{
		Timestamp: time.Now(),
		Level:     types.InfoLevel,
		Message:   "hello world",
		File:      "benchmark.go",
		Line:      42,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		entry.Timestamp = time.Now()
		_ = adapter.Write(entry)
	}

	// Ensure all entries are processed
	_ = adapter.Flush()
}

// BenchmarkAsyncAdapter_WithField_Info_HelloWorld benchmarks info message with field
func BenchmarkAsyncAdapter_WithField_Info_HelloWorld(b *testing.B) {
	mock := &asyncMockAdapter{name: "benchmark"}
	adapter := NewAsyncAdapter(mock, &AsyncAdapterOptions{
		BufferSize:    1000,
		BatchSize:     100,
		FlushInterval: 10 * time.Millisecond,
	})
	defer func() { _ = adapter.Close() }()

	entry := &types.LogEntry{
		Timestamp: time.Now(),
		Level:     types.InfoLevel,
		Message:   "hello world",
		File:      "benchmark.go",
		Line:      42,
		Fields: []types.TypedFieldData{
			{Key: "user_id", Value: "12345"},
		},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		entry.Timestamp = time.Now()
		_ = adapter.Write(entry)
	}

	// Ensure all entries are processed
	_ = adapter.Flush()
}
