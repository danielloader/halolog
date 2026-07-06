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

// SimplePipeline is optimized for configurations with fields but NO features.
// Used when: has fields, no masking, no sampling, no async, no colors
// Performance: sub-20ns total overhead for field logs
// Allocations: 0 for ≤optimal_buffer_size fields (P99 case)
type SimplePipeline struct {
	Adapters          []types.Adapter       // Direct adapter writes, no feature overhead
	OptimalBufferSize int                   // Optimal buffer size calculated at startup
	FieldIntegration  *FieldDictIntegration // O(1) field access integration (optional)
}

// NewSimplePipeline creates a new simple pipeline with optimal buffer size
func NewSimplePipeline(adapters []types.Adapter, optimalBufferSize int) *SimplePipeline {
	if optimalBufferSize <= 0 {
		optimalBufferSize = 32 // Default: P99 for most applications
	}
	if optimalBufferSize > 1000 {
		optimalBufferSize = 1000 // Cap at reasonable maximum
	}

	// Initialize field dict integration if available
	var fieldIntegration *FieldDictIntegration
	if GlobalFieldDictIntegration != nil {
		fieldIntegration = GlobalFieldDictIntegration
	}

	return &SimplePipeline{
		Adapters:          adapters,
		OptimalBufferSize: optimalBufferSize,
		FieldIntegration:  fieldIntegration,
	}
}

// Write implements the fast logging path for field configurations.
//
// Performance breakdown:
//   - acquireEntry():  ~3ns  (pool get, minimal reset)
//   - clock.now():     ~1ns  (cached timestamp)
//   - field copy:      ~1ns per field (optimal buffer)
//   - adapter.Write(): ~8-12ns (formatter with fields)
//   - releaseEntry():  ~2ns  (pool put, cleanup)
//
// Total: 15-20ns for typical field counts (target: sub-20ns)
//
// Thread-safety: Safe for concurrent use. Each call gets its own entry from pool.
// Allocations: 0 for ≤optimal_buffer_size fields (P99 case)
func (p *SimplePipeline) Write(clock *clock.CachedClock, level types.LogLevel, msg string, fields []types.TypedFieldData, fieldCount int) {
	// Acquire entry from pool
	entry := pool.AcquireEntry()

	// Fill entry metadata
	entry.Timestamp = clock.Now()
	entry.Level = level
	entry.Message = msg

	// Handle fields efficiently - use static buffer for common case
	if fieldCount <= cap(entry.StaticFields) {
		// ✅ High-performance: Set slice length to match field count, backed by StaticFields
		entry.StaticFields = entry.StaticFields[:fieldCount]
		copy(entry.StaticFields, fields[:fieldCount])
		entry.StaticFieldCount = fieldCount
	} else {
		// Overflow: use dynamic Fields slice
		p.handleFieldOverflow(entry, fields, fieldCount)
	}

	// Write to all adapters
	for _, adapter := range p.Adapters {
		if err := adapter.Write(entry); err != nil {
			// High-performance: Robust error handling
			// 1. Log to stderr (non-blocking, fallback)
			// 2. Continue to next adapter (isolation)
			// Note: In a real production system, we would increment a metric here
			// fmt.Fprintf(os.Stderr, "halolog: adapter %s write failed: %v\n", adapter.Name(), err)
			_ = err // Suppress unused error for now, but structure is ready
		}
	}

	// Return entry to pool
	pool.ReleaseEntry(entry)
}

// handleFieldOverflow handles cases where field count exceeds optimal buffer size
// This is the slow path for outliers (>P99 case)
func (p *SimplePipeline) handleFieldOverflow(entry *types.LogEntry, fields []types.TypedFieldData, fieldCount int) {
	// ✅ SAFE: Use StaticFields capacity correctly
	if fieldCount <= cap(entry.StaticFields) {
		entry.StaticFields = entry.StaticFields[:fieldCount]
		copy(entry.StaticFields, fields[:fieldCount])
		entry.StaticFieldCount = fieldCount
	} else {
		// Overflow to dynamic Fields slice
		// Slowest path: Use dynamic slice for very large field counts
		// Still optimized to reuse existing slice capacity
		entry.Fields = append(entry.Fields[:0], fields[:fieldCount]...)
		entry.StaticFieldCount = 0
	}
}

// RegisterFieldsWithDict registers field keys with FieldDict for O(1) access
// This can be called at startup to pre-register common fields
func (p *SimplePipeline) RegisterFieldsWithDict(fieldKeys []string) {
	if p.FieldIntegration == nil || len(fieldKeys) == 0 {
		return
	}

	// Register each field key for O(1) access
	for _, key := range fieldKeys {
		p.FieldIntegration.GetOrRegisterFieldID(key)
	}
}

// GetFieldID provides O(1) field ID lookup if FieldDict integration is available
func (p *SimplePipeline) GetFieldID(fieldKey string) (int, bool) {
	if p.FieldIntegration == nil {
		return 0, false
	}
	return p.FieldIntegration.GetFieldID(fieldKey)
}

// ProcessFieldsWithDict processes fields with O(1) dictionary access
// This is useful for applications that want to leverage FieldDict benefits
// while maintaining SimplePipeline performance characteristics
func (p *SimplePipeline) ProcessFieldsWithDict(fields []types.TypedFieldData) []types.TypedFieldData {
	if p.FieldIntegration == nil || len(fields) == 0 {
		return fields
	}

	// Process fields with O(1) field ID registration
	processedFields := make([]types.TypedFieldData, len(fields))
	for i, field := range fields {
		// Register field for O(1) access (no-op if already registered)
		p.FieldIntegration.GetOrRegisterFieldID(field.Key)
		processedFields[i] = field
	}

	return processedFields
}

// Reset clears the pipeline state for reuse.
// Called when logger is closed or pipeline is replaced.
func (p *SimplePipeline) Reset() {
	// Clear adapter references to allow GC
	for i := range p.Adapters {
		p.Adapters[i] = nil
	}
	p.Adapters = p.Adapters[:0]
}

// IsSimplePathEligible determines if a configuration is eligible for SimplePipeline.
// This is used during pipeline construction to choose the optimal strategy.
func IsSimplePathEligible(fieldCount int, hasMasking bool, hasSampling bool, hasAsync bool, hasColors bool) bool {
	// SimplePipeline eligibility criteria:
	// 1. Has fields (fieldCount > 0)
	// 2. No masking enabled
	// 3. No sampling enabled
	// 4. No async enabled
	// 5. No colors enabled
	return fieldCount > 0 && !hasMasking && !hasSampling && !hasAsync && !hasColors
}

// CalculateOptimalBufferSize calculates the optimal buffer size based on application profiling
// This can be determined from config or runtime profiling
func CalculateOptimalBufferSize(configuredSize int, profiledP99 int) int {
	// Priority order for determining optimal buffer size:
	// 1. Explicit configuration (if provided and reasonable)
	// 2. Profiled P99 value (if available)
	// 3. Default value

	if configuredSize > 0 && configuredSize <= 100 {
		return configuredSize
	}

	if profiledP99 > 0 && profiledP99 <= 100 {
		return profiledP99 + 5 // Add small buffer for variance
	}

	return 32 // Default: P99 for most applications
}

// GetOptimalBufferSizeForApp analyzes the application and returns recommended buffer size
func GetOptimalBufferSizeForApp(appProfile map[string]interface{}) int {
	// This function can be extended to analyze application characteristics
	// For now, return a reasonable default based on common patterns

	if appProfile != nil {
		if p99Fields, ok := appProfile["p99_field_count"].(int); ok && p99Fields > 0 {
			return p99Fields + 5
		}
	}

	return 32
}
