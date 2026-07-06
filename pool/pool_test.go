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
	"time"

	"github.com/go-gen-ecosystem/halolog/types"
)

func TestEntryPool_ConcurrentAccess(t *testing.T) {
	// Pool is already initialized in init()
	const goroutines = 100
	const iterations = 1000

	var wg sync.WaitGroup
	wg.Add(goroutines)

	errCh := make(chan struct{}, 1)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				entry := AcquireEntry()
				if entry == nil {
					select {
					case errCh <- struct{}{}:
					default:
					}
					return
				}
				entry.Message = "test message"
				entry.Level = types.InfoLevel
				entry.Fields = append(entry.Fields, types.TypedFieldData{
					Key:   "field",
					Value: "value",
				})
				ReleaseEntry(entry)
			}
		}()
	}

	wg.Wait()
	select {
	case <-errCh:
		t.Fatal("AcquireEntry returned nil")
	default:
	}

	// Verify pool stats
	stats := GlobalPool.GetStats()
	if stats.AcquireCount == 0 {
		t.Error("Expected non-zero acquire count")
	}
	if stats.ReleaseCount == 0 {
		t.Error("Expected non-zero release count")
	}
}

func TestEntryPool_RaceCondition(t *testing.T) {
	// Pool is already initialized in init()
	const goroutines = 50
	const iterations = 100

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				entry := AcquireEntry()
				entry.Message = "race test"
				entry.Level = types.InfoLevel
				entry.Fields = append(entry.Fields, types.TypedFieldData{
					Key:   "goroutine",
					Value: id,
				})
				ReleaseEntry(entry)
			}
		}(i)
	}

	wg.Wait()
}

func TestEntryPool_MemorySafety(t *testing.T) {
	// Pool is already initialized in init()
	entry := AcquireEntry()
	entry.Message = "sensitive data"
	entry.Level = types.ErrorLevel
	entry.Fields = append(entry.Fields, types.TypedFieldData{
		Key:   "password",
		Value: "secret123",
	})

	ReleaseEntry(entry)

	// Acquire another entry to ensure memory was cleared
	newEntry := AcquireEntry()
	if newEntry.Message == "sensitive data" {
		t.Error("Memory not properly cleared after release")
	}
	for _, field := range newEntry.Fields {
		if field.Key == "password" {
			t.Error("Sensitive data not cleared from memory")
		}
	}
	ReleaseEntry(newEntry)
}

func TestEntryPool_Stats(t *testing.T) {
	// Pool is already initialized in init()
	stats := GlobalPool.GetStats()
	if !stats.Initialized {
		t.Error("Pool should be initialized")
	}
	if stats.Shutdown {
		t.Error("Pool should not be shutdown")
	}

	// Test basic operations
	entry := AcquireEntry()
	ReleaseEntry(entry)

	stats = GlobalPool.GetStats()
	if stats.AcquireCount == 0 {
		t.Error("Expected non-zero acquire count")
	}
	if stats.ReleaseCount == 0 {
		t.Error("Expected non-zero release count")
	}
}

func TestEntryPool_NilHandling(t *testing.T) {
	initialStats := GlobalPool.GetStats()
	// Test nil entry release
	ReleaseEntry(nil)

	// Should not panic
	newStats := GlobalPool.GetStats()
	if newStats.ReleaseCount != initialStats.ReleaseCount {
		t.Errorf("Release count changed from %d to %d for nil entry", initialStats.ReleaseCount, newStats.ReleaseCount)
	}
}

func TestEntryPool_ConcurrentStress(t *testing.T) {
	// Pool is already initialized in init()
	const duration = 100 * time.Millisecond
	const goroutines = 100

	var wg sync.WaitGroup
	wg.Add(goroutines)

	start := time.Now()
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for time.Since(start) < duration {
				entry := AcquireEntry()
				entry.Message = "stress test"
				entry.Level = types.InfoLevel
				entry.Fields = append(entry.Fields, types.TypedFieldData{
					Key:   "goroutine",
					Value: id,
				})
				time.Sleep(time.Microsecond) // Simulate some work
				ReleaseEntry(entry)
			}
		}(i)
	}

	wg.Wait()

	// Verify pool is still functional
	entry := AcquireEntry()
	if entry == nil {
		t.Fatal("Pool not functional after stress test")
	}
	ReleaseEntry(entry)
}
