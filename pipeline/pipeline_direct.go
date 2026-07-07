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
// Package pipeline provides logging pipeline functionality
// Author: Admilson B. F. Cossa

package pipeline

import (
	clock "github.com/go-gen-ecosystem/halolog/cache"
	pool "github.com/go-gen-ecosystem/halolog/pool"
	"github.com/go-gen-ecosystem/halolog/types"
)

// NewDirectPipeline creates a new DirectPipeline with adapter optimization
func NewDirectPipeline(adapters []types.Adapter) *DirectPipeline {
	if len(adapters) == 1 {
		return &DirectPipeline{
			SingleAdapter:    adapters[0],
			HasSingleAdapter: true,
		}
	}

	return &DirectPipeline{
		Adapters:         adapters,
		HasSingleAdapter: false,
	}
}

// DirectPipeline is optimized for 0-field configurations with NO features enabled.
// Used when: no fields, no masking, no sampling, no async, no colors
// Performance: optimized for speed
// Allocations: 0 (uses entry pool)
type DirectPipeline struct {
	// For single adapter configurations - direct reference eliminates virtual dispatch
	SingleAdapter types.Adapter
	// For multiple adapters - fallback to slice
	Adapters []types.Adapter
	// Optimization flag - true when we have exactly one simple adapter
	HasSingleAdapter bool
}

// Write implements the logging path for 0-field configurations.
//
// Thread-safety: Safe for concurrent use. Each call uses stack-allocated entry.
// Allocations: 0 (stack allocation, no pool overhead)
// Performance: Sub-10ns target
func (p *DirectPipeline) Write(clock *clock.CachedClock, level types.LogLevel, msg string, fields []types.TypedFieldData, fieldCount int) {
	// Fast-path: skip field processing for 0 fields
	if fieldCount > 0 {
		// This should not happen - DirectPipeline is for 0 fields only
		// Fallback to SimplePipeline behavior if called incorrectly
		p.handleFields(clock, level, msg, fields, fieldCount)
		return
	}

	// Stack-allocated entry - NO pool overhead!
	// Entry never escapes function, so Go keeps it on stack
	var entry types.LogEntry

	// Populate entry metadata only (no fields to process)
	entry.Timestamp = clock.Now() // Cached clock: ~1ns
	entry.Level = level
	entry.Message = msg
	entry.StaticFieldCount = 0

	// OPTIMIZED: Direct adapter call eliminates virtual dispatch
	if p.HasSingleAdapter {
		// Direct function call - no interface overhead.
		// Write error is non-actionable on the hot path; discard alloc-free.
		_ = p.SingleAdapter.WriteZero(&entry)
	} else {
		// Fallback for multiple adapters (rare)
		for _, adapter := range p.Adapters {
			_ = adapter.WriteZero(&entry)
		}
	}
	// No release needed - entry is stack-allocated and dies here
}

// handleFields provides fallback for when DirectPipeline is called with fields
// This should be rare - indicates a configuration error
func (p *DirectPipeline) handleFields(clock *clock.CachedClock, level types.LogLevel, msg string, fields []types.TypedFieldData, fieldCount int) {
	// Fallback to SimplePipeline-style handling
	entry := pool.GlobalPool.AcquireEntry()

	entry.Timestamp = clock.Now()
	entry.Level = level
	entry.Message = msg

	// Handle fields - use static array for small counts
	if fieldCount <= cap(entry.StaticFields) {
		// ✅ Avoid slice header allocation by modifying in place
		entry.StaticFields = entry.StaticFields[:fieldCount]
		copy(entry.StaticFields, fields[:fieldCount])
		entry.StaticFieldCount = fieldCount
	} else {
		entry.Fields = append(entry.Fields[:0], fields[:fieldCount]...)
		entry.StaticFieldCount = 0
	}

	// OPTIMIZED: Direct adapter call eliminates virtual dispatch
	if p.HasSingleAdapter {
		if err := p.SingleAdapter.WriteZero(entry); err != nil {
			_ = err // Handle error
		}
	} else {
		for _, adapter := range p.Adapters {
			if err := adapter.WriteZero(entry); err != nil {
				_ = err // Handle error
			}
		}
	}

	pool.GlobalPool.ReleaseEntry(entry)
}

// Reset clears the pipeline state for reuse.
// Called when logger is closed or pipeline is replaced.
func (p *DirectPipeline) Reset() {
	// Clear adapter references to allow GC
	for i := range p.Adapters {
		p.Adapters[i] = nil
	}
	p.Adapters = p.Adapters[:0]
}

// IsDirectPathEligible determines if a configuration is eligible for DirectPipeline.
// This is used during pipeline construction to choose the optimal strategy.
func IsDirectPathEligible(fieldCount int, hasMasking bool, hasSampling bool, hasAsync bool, hasColors bool) bool {
	// DirectPipeline eligibility criteria:
	// 1. Zero fields (fieldCount == 0)
	// 2. No masking enabled
	// 3. No sampling enabled
	// 4. No async enabled
	// 5. No colors enabled
	return fieldCount == 0 && !hasMasking && !hasSampling && !hasAsync && !hasColors
}
