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
	"runtime"
	"sync"
	"testing"

	"github.com/go-gen-ecosystem/halolog/types"
)

// BenchmarkEntryPoolConsolidated measures the performance of the consolidated entry pool
func BenchmarkEntryPoolConsolidated(b *testing.B) {
	// Initialize pool (already done in init)
	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			entry := GlobalPool.AcquireEntry()
			entry.Message = "test message"
			entry.Level = types.InfoLevel
			GlobalPool.ReleaseEntry(entry)
		}
	})
}

// BenchmarkEntryPoolSyncPool measures the performance of standard sync.Pool approach
func BenchmarkEntryPoolSyncPool(b *testing.B) {
	pool := &sync.Pool{
		New: func() interface{} {
			return &types.LogEntry{
				Fields: make([]types.TypedFieldData, 0, 32),
			}
		},
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			entry := pool.Get().(*types.LogEntry)
			entry.Message = "test message"
			entry.Level = types.InfoLevel
			pool.Put(entry)
		}
	})
}

// BenchmarkEntryPoolComparison compares consolidated vs sync.Pool approaches
func BenchmarkEntryPoolComparison(b *testing.B) {
	b.Run("Consolidated_Pool", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				entry := GlobalPool.AcquireEntry()
				entry.Message = "test message"
				entry.Level = types.InfoLevel
				GlobalPool.ReleaseEntry(entry)
			}
		})
	})

	b.Run("Sync_Pool", func(b *testing.B) {
		pool := &sync.Pool{
			New: func() interface{} {
				return &types.LogEntry{
					Fields: make([]types.TypedFieldData, 0, 32),
				}
			},
		}

		b.ResetTimer()
		b.ReportAllocs()

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				entry := pool.Get().(*types.LogEntry)
				entry.Message = "test message"
				entry.Level = types.InfoLevel
				pool.Put(entry)
			}
		})
	})
}

// BenchmarkEntryPoolConcurrent tests concurrent access patterns
func BenchmarkEntryPoolConcurrent(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		goroutine := runtime.GOMAXPROCS(0)

		for pb.Next() {
			entry := GlobalPool.AcquireEntry()
			entry.Message = "concurrent test"
			entry.Level = types.InfoLevel
			entry.Fields = append(entry.Fields, types.TypedFieldData{
				Key:   "goroutine",
				Value: goroutine,
			})
			GlobalPool.ReleaseEntry(entry)
		}
	})
}

// BenchmarkEntryPoolRace tests for race conditions
func BenchmarkEntryPoolRace(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// Multiple goroutines acquiring/releasing concurrently
			entry := GlobalPool.AcquireEntry()
			entry.Message = "race test"
			entry.Level = types.InfoLevel
			GlobalPool.ReleaseEntry(entry)
		}
	})
}
