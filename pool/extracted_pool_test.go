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
	"testing"

	"github.com/go-gen-ecosystem/halolog/types"
)

// TestExtractedPoolBasic tests basic pool operations
func TestExtractedPoolBasic(t *testing.T) {
	pool := NewExtractedPool()

	// Test get/put cycle
	state := pool.GetPerPState()
	if state == nil {
		t.Fatal("Expected non-nil state")
	}

	// Verify state is zero-initialized
	if state.fieldCount != 0 {
		t.Errorf("Expected fieldCount=0, got %d", state.fieldCount)
	}
	if state.timestamp != 0 {
		t.Errorf("Expected timestamp=0, got %d", state.timestamp)
	}
	if state.level != 0 {
		t.Errorf("Expected level=0, got %d", state.level)
	}

	pool.PutPerPState(state)
}

// TestExtractedPoolConcurrent tests concurrent access
func TestExtractedPoolConcurrent(t *testing.T) {
	pool := NewExtractedPool()
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			state := pool.GetPerPState()
			if state == nil {
				t.Error("Expected non-nil state")
				return
			}

			// Simulate some work
			state.fieldCount = uint8(id % 40)
			state.timestamp = int64(id)
			state.level = types.InfoLevel

			pool.PutPerPState(state)
		}(i)
	}

	wg.Wait()
}

// TestExtractedEntryPool tests the entry pool
func TestExtractedEntryPool(t *testing.T) {
	pool := NewExtractedEntryPool()

	entry := pool.AcquireEntry()
	if entry == nil {
		t.Fatal("Expected non-nil entry")
	}

	// Verify entry structure
	if entry.StaticFields == nil {
		t.Error("Expected non-nil StaticFields")
	}
	if cap(entry.StaticFields) < 16 {
		t.Error("Expected StaticFields capacity >= 16")
	}
	if entry.StaticContext == nil {
		t.Error("Expected non-nil StaticContext")
	}
	if cap(entry.StaticContext) < 16 {
		t.Error("Expected StaticContext capacity >= 16")
	}
}

// TestExtractedFieldBuilder tests field builder functionality
func TestExtractedFieldBuilder(t *testing.T) {
	pool := NewExtractedPool()
	builder := NewExtractedFieldBuilder(pool)

	// Build entry with fields
	builder.WithField("key1", "value1").
		WithField("key2", 42).
		WithField("key3", true)

	entry := builder.GetEntry()
	if entry == nil {
		t.Fatal("Expected non-nil entry")
	}

	if entry.StaticFieldCount != 3 {
		t.Errorf("Expected 3 fields, got %d", entry.StaticFieldCount)
	}

	// Verify field values
	if entry.StaticFields[0].Key != "key1" || entry.StaticFields[0].Value != "value1" {
		t.Error("Field 0 mismatch")
	}
	if entry.StaticFields[1].Key != "key2" || entry.StaticFields[1].Value != 42 {
		t.Error("Field 1 mismatch")
	}
	if entry.StaticFields[2].Key != "key3" || entry.StaticFields[2].Value != true {
		t.Error("Field 2 mismatch")
	}

	builder.Reset()
}

// TestExtractedFieldBuilderNoFields tests builder with no fields
func TestExtractedFieldBuilderNoFields(t *testing.T) {
	pool := NewExtractedPool()
	builder := NewExtractedFieldBuilder(pool)

	// Don't add any fields, just get entry
	entry := builder.GetEntry()
	if entry != nil {
		t.Error("Expected nil entry when no fields added")
	}

	builder.Reset()
}

// TestGlobalExtractedPools tests global pool instances
func TestGlobalExtractedPools(t *testing.T) {
	if GlobalExtractedPool == nil {
		t.Fatal("Expected GlobalExtractedPool to be initialized")
	}
	if GlobalExtractedEntryPool == nil {
		t.Fatal("Expected GlobalExtractedEntryPool to be initialized")
	}

	// Test global pools work
	state := GlobalExtractedPool.GetPerPState()
	if state == nil {
		t.Fatal("Expected non-nil state from global pool")
	}
	GlobalExtractedPool.PutPerPState(state)

	entry := GlobalExtractedEntryPool.AcquireEntry()
	if entry == nil {
		t.Fatal("Expected non-nil entry from global pool")
	}
}

// BenchmarkExtractedPool benchmarks the extracted pool performance
func BenchmarkExtractedPool(b *testing.B) {
	pool := NewExtractedPool()

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			state := pool.GetPerPState()
			state.fieldCount = 0
			state.timestamp = 0
			state.level = types.InfoLevel
			pool.PutPerPState(state)
		}
	})
}

// BenchmarkExtractedEntryPool benchmarks the extracted entry pool
func BenchmarkExtractedEntryPool(b *testing.B) {
	pool := NewExtractedEntryPool()

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			entry := pool.AcquireEntry()
			_ = entry
		}
	})
}

// BenchmarkExtractedFieldBuilder benchmarks the extracted field builder
func BenchmarkExtractedFieldBuilder(b *testing.B) {
	pool := NewExtractedPool()

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			builder := NewExtractedFieldBuilder(pool)
			builder.WithField("key1", "value1").
				WithField("key2", 42).
				WithField("key3", true)
			entry := builder.GetEntry()
			_ = entry
			builder.Reset()
		}
	})
}
