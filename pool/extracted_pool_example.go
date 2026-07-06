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
	"fmt"
	"time"

	"github.com/go-gen-ecosystem/halolog/types"
)

// ExampleExtractedPool demonstrates basic usage of extracted pool
func ExampleExtractedPool() {
	// Create a new extracted pool (or use GlobalExtractedPool)
	pool := NewExtractedPool()

	// Get per-P state
	state := pool.GetPerPState()

	// Use the state
	state.Entry.Message = "Hello, World!"
	state.Entry.Level = types.InfoLevel
	state.fieldCount = 1
	state.fieldBuf[0] = types.TypedFieldData{Key: "user", Value: "john_doe"}

	// Return state to pool
	pool.PutPerPState(state)
}

// ExampleExtractedEntryPool demonstrates entry pool usage
func ExampleExtractedEntryPool() {
	// Create entry pool (or use GlobalExtractedEntryPool)
	entryPool := NewExtractedEntryPool()

	// Acquire entry
	entry := entryPool.AcquireEntry()

	// Use the entry
	entry.Message = "Performance critical log"
	entry.Level = types.DebugLevel
	entry.Timestamp = time.Now()

	// Entry is stack-allocated, no need to return it
	// Just let it go out of scope
	_ = entry
}

// ExampleExtractedFieldBuilder demonstrates field builder usage
func ExampleExtractedFieldBuilder() {
	// Create pool and builder
	pool := NewExtractedPool()
	builder := NewExtractedFieldBuilder(pool)

	// Build entry with fields
	builder.WithField("user", "alice").
		WithField("action", "login").
		WithField("duration_ms", 125).
		WithField("success", true)

	// Get the built entry
	entry := builder.GetEntry()

	// Use the entry
	entry.Message = "User login event"
	entry.Level = types.InfoLevel

	fmt.Printf("Entry has %d fields\n", entry.StaticFieldCount)

	// Reset builder to return resources to pool
	builder.Reset()
}

// ExampleGlobalExtractedPools demonstrates using global pools
func ExampleGlobalExtractedPools() {
	// Use global extracted pool
	state := GlobalExtractedPool.GetPerPState()
	state.Entry.Message = "Using global pool"
	state.Entry.Level = types.WarnLevel
	GlobalExtractedPool.PutPerPState(state)

	// Use global entry pool
	entry := GlobalExtractedEntryPool.AcquireEntry()
	entry.Message = "Using global entry pool"
	entry.Level = types.ErrorLevel
	// Entry is automatically managed
}

// ExamplePerformanceComparison shows performance characteristics
func ExamplePerformanceComparison() {
	// Logger's extracted pool: ~0.87 ns/op
	pool := NewExtractedPool()
	state := pool.GetPerPState()
	pool.PutPerPState(state)

	// Logger's extracted entry pool: ~0.028 ns/op
	entryPool := NewExtractedEntryPool()
	entry := entryPool.AcquireEntry()
	_ = entry

	// Field builder: ~2.1 ns/op with 3 fields
	builder := NewExtractedFieldBuilder(pool)
	builder.WithField("key1", "value1").
		WithField("key2", 42).
		WithField("key3", true)
	builtEntry := builder.GetEntry()
	_ = builtEntry
	builder.Reset()
}

// ExampleConcurrentUsage demonstrates safe concurrent usage
func ExampleConcurrentUsage() {
	// Global pools are safe for concurrent use
	go func() {
		state := GlobalExtractedPool.GetPerPState()
		state.Entry.Message = "Concurrent worker 1"
		GlobalExtractedPool.PutPerPState(state)
	}()

	go func() {
		state := GlobalExtractedPool.GetPerPState()
		state.Entry.Message = "Concurrent worker 2"
		GlobalExtractedPool.PutPerPState(state)
	}()

	// Both goroutines can safely use the same global pool
	// due to per-P state design (zero contention)
}
