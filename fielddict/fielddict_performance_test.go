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
// Package fielddict provides field dictionary management
// Author: Admilson B. F. Cossa

package fielddict

import (
	"fmt"
	"sync"
	"testing"
)

// TestFieldDictionary_O1LookupPerformance tests O(1) lookup performance
func TestFieldDictionary_O1LookupPerformance(t *testing.T) {
	dict := NewFieldDictionary()

	// Register fields at startup
	startupFields := []string{"user_id", "request_id", "session_id", "ip_address", "user_agent"}
	err := dict.RegisterFromConfig(startupFields, nil)
	if err != nil {
		t.Fatalf("Failed to register startup fields: %v", err)
	}

	// Test O(1) lookup for existing fields
	for _, fieldName := range startupFields {
		fieldID, exists := dict.GetFieldID(fieldName)
		if !exists {
			t.Errorf("Expected field %s to exist", fieldName)
		}
		if fieldID < 0 {
			t.Errorf("Expected valid field ID for %s, got %d", fieldName, fieldID)
		}
	}

	// Verify fast path hits are recorded
	stats := dict.GetPerformanceStats()
	if stats.FastPathHits == 0 {
		t.Error("Expected fast path hits to be recorded")
	}
}

// TestFieldDictionary_GetOrRegisterFieldIDFast tests High-performance field registration
func TestFieldDictionary_GetOrRegisterFieldIDFast(t *testing.T) {
	dict := NewFieldDictionary()

	// Test new field registration
	fieldID, isNew, allocs := dict.GetOrRegisterFieldIDFast("test_field")
	if fieldID != 0 {
		t.Errorf("Expected field ID 0 for first field, got %d", fieldID)
	}
	if !isNew {
		t.Error("Expected isNew to be true for new field")
	}
	if allocs != 1 {
		t.Errorf("Expected 1 allocation for new field, got %d", allocs)
	}

	// Test existing field lookup (should be 0 allocations)
	fieldID2, isNew2, allocs2 := dict.GetOrRegisterFieldIDFast("test_field")
	if fieldID2 != fieldID {
		t.Errorf("Expected same field ID, got %d vs %d", fieldID2, fieldID)
	}
	if isNew2 {
		t.Error("Expected isNew to be false for existing field")
	}
	if allocs2 != 0 {
		t.Errorf("Expected 0 allocations for existing field, got %d", allocs2)
	}
}

// TestFieldDictionary_RegistrationStats tests registration statistics
func TestFieldDictionary_RegistrationStats(t *testing.T) {
	dict := NewFieldDictionary()

	// Register some fields at startup
	startupFields := []string{"field1", "field2", "field3"}
	err := dict.RegisterFromConfig(startupFields, nil)
	if err != nil {
		t.Fatalf("Failed to register startup fields: %v", err)
	}

	// Register some fields at runtime
	dict.GetOrRegisterFieldID("runtime_field1")
	dict.GetOrRegisterFieldID("runtime_field2")

	// Perform lookups to generate fast path hits and runtime misses
	dict.GetFieldID("field1")                      // Should be fast path hit
	dict.GetOrRegisterFieldID("field1")            // Should be fast path hit
	dict.GetOrRegisterFieldID("new_runtime_field") // Should be runtime miss

	// Get registration stats
	stats := dict.GetRegistrationStats()

	if stats.StartupRegistered != 3 {
		t.Errorf("Expected 3 startup registered fields, got %d", stats.StartupRegistered)
	}
	if stats.RuntimeRegistered != 3 { // 2 explicit + 1 from lookup
		t.Errorf("Expected 3 runtime registered fields, got %d", stats.RuntimeRegistered)
	}
	if stats.TotalFields != 6 { // 3 startup + 3 runtime
		t.Errorf("Expected 6 total fields, got %d", stats.TotalFields)
	}
	if stats.PrecomputedHits == 0 {
		t.Error("Expected some precomputed hits")
	}
	if stats.RuntimeMisses == 0 {
		t.Error("Expected some runtime misses")
	}
}

// TestFieldDictionary_RegisterCustomFieldsWithIDs tests custom field registration with pre-assigned IDs
func TestFieldDictionary_RegisterCustomFieldsWithIDs(t *testing.T) {
	dict := NewFieldDictionary()

	// Register fields with specific IDs
	customFields := map[string]int{
		"custom_field1": 10,
		"custom_field2": 20,
		"custom_field3": 5,
	}

	err := dict.RegisterCustomFieldsWithIDs(customFields)
	if err != nil {
		t.Fatalf("Failed to register custom fields with IDs: %v", err)
	}

	// Verify fields are registered with correct IDs
	for fieldName, expectedID := range customFields {
		actualID, exists := dict.GetID(fieldName)
		if !exists {
			t.Errorf("Expected field %s to exist", fieldName)
		}
		if actualID != expectedID {
			t.Errorf("Expected field %s to have ID %d, got %d", fieldName, expectedID, actualID)
		}
	}

	// Verify GetFieldByID works correctly
	if fieldName := dict.GetFieldByID(10); fieldName != "custom_field1" {
		t.Errorf("Expected field ID 10 to be 'custom_field1', got '%s'", fieldName)
	}
	if fieldName := dict.GetFieldByID(20); fieldName != "custom_field2" {
		t.Errorf("Expected field ID 20 to be 'custom_field2', got '%s'", fieldName)
	}
	if fieldName := dict.GetFieldByID(5); fieldName != "custom_field3" {
		t.Errorf("Expected field ID 5 to be 'custom_field3', got '%s'", fieldName)
	}
}

// TestFieldDictionary_BitmaskOperations tests bitmask functionality
func TestFieldDictionary_BitmaskOperations(t *testing.T) {
	dict := NewFieldDictionary()

	// Register fields
	fields := []string{"field1", "field2", "field3", "field4"}
	for _, field := range fields {
		dict.Register(field)
	}

	// Create bitmask for multiple fields
	bitmask := dict.CreateFieldBitmask("field1", "field3")

	// Test field presence in bitmask
	field1ID, _ := dict.GetID("field1")
	field2ID, _ := dict.GetID("field2")
	field3ID, _ := dict.GetID("field3")

	if !dict.IsFieldInBitmask(field1ID, bitmask) {
		t.Error("Expected field1 to be in bitmask")
	}
	if dict.IsFieldInBitmask(field2ID, bitmask) {
		t.Error("Expected field2 to NOT be in bitmask")
	}
	if !dict.IsFieldInBitmask(field3ID, bitmask) {
		t.Error("Expected field3 to be in bitmask")
	}
}

// TestFieldDictionary_ConcurrentAccessPerformance tests concurrent access patterns with performance focus
func TestFieldDictionary_ConcurrentAccessPerformance(t *testing.T) {
	dict := NewFieldDictionary()

	// Register some initial fields
	dict.RegisterFromConfig([]string{"initial_field1", "initial_field2"}, nil)

	var wg sync.WaitGroup
	numGoroutines := 10
	numOperations := 100

	// Concurrent reads and writes
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				// Mix of operations
				if j%2 == 0 {
					// Read operation
					dict.GetFieldID("initial_field1")
					dict.GetFieldID("initial_field2")
				} else {
					// Write operation
					fieldName := fmt.Sprintf("concurrent_field_%d_%d", goroutineID, j)
					dict.GetOrRegisterFieldID(fieldName)
				}
			}
		}(i)
	}

	wg.Wait()

	// Verify stats after concurrent operations
	stats := dict.GetRegistrationStats()
	if stats.RuntimeRegistered == 0 {
		t.Error("Expected some runtime registered fields from concurrent operations")
	}
}

// TestFieldDictionary_PerformanceMetrics tests performance metrics collection
func TestFieldDictionary_PerformanceMetrics(t *testing.T) {
	dict := NewFieldDictionary()

	// Register field using GetOrRegisterFieldID to ensure counters are updated
	dict.GetOrRegisterFieldID("test_field")

	// Perform multiple lookups
	for i := 0; i < 10; i++ {
		dict.GetFieldID("test_field")
		dict.GetOrRegisterFieldID("test_field")
	}

	// Check performance stats
	stats := dict.GetPerformanceStats()
	if stats.FastPathHits == 0 {
		t.Error("Expected fast path hits to be recorded")
	}
	if stats.TotalFields != 1 {
		t.Errorf("Expected 1 total field, got %d", stats.TotalFields)
	}
}

// TestFieldDictionary_StartupVsRuntimeRegistration tests startup vs runtime registration tracking
func TestFieldDictionary_StartupVsRuntimeRegistration(t *testing.T) {
	dict := NewFieldDictionary()

	// Register fields at startup
	startupFields := []string{"startup_field1", "startup_field2"}
	err := dict.RegisterFromConfig(startupFields, []string{"sensitive_field1", "sensitive_field2"})
	if err != nil {
		t.Fatalf("Failed to register startup fields: %v", err)
	}

	// Register fields at runtime
	runtimeFields := []string{"runtime_field1", "runtime_field2", "runtime_field3"}
	for _, field := range runtimeFields {
		dict.GetOrRegisterFieldID(field)
	}

	stats := dict.GetRegistrationStats()
	if stats.StartupRegistered != 4 { // 2 custom + 2 sensitive
		t.Errorf("Expected 4 startup registered fields, got %d", stats.StartupRegistered)
	}
	if stats.RuntimeRegistered != 3 {
		t.Errorf("Expected 3 runtime registered fields, got %d", stats.RuntimeRegistered)
	}
	if stats.TotalFields != 7 {
		t.Errorf("Expected 7 total fields, got %d", stats.TotalFields)
	}
}

// TestFieldDictionary_BulkRegistration tests bulk field registration
func TestFieldDictionary_BulkRegistration(t *testing.T) {
	dict := NewFieldDictionary()

	// Register multiple fields in bulk
	fields := []string{"bulk_field1", "bulk_field2", "bulk_field3", "bulk_field4", "bulk_field5"}
	ids := dict.RegisterBulk(fields)

	if len(ids) != len(fields) {
		t.Errorf("Expected %d IDs, got %d", len(fields), len(ids))
	}

	// Verify all fields are registered
	for i, field := range fields {
		fieldID, exists := dict.GetID(field)
		if !exists {
			t.Errorf("Expected field %s to exist", field)
		}
		if fieldID != ids[i] {
			t.Errorf("Expected field %s to have ID %d, got %d", field, ids[i], fieldID)
		}
	}
}

// TestFieldDictionary_PerformanceEdgeCases tests edge cases and error conditions for performance features
func TestFieldDictionary_PerformanceEdgeCases(t *testing.T) {
	dict := NewFieldDictionary()

	// Test empty field name
	fieldID := dict.Register("")
	if fieldID != 0 {
		t.Errorf("Expected field ID 0 for empty field, got %d", fieldID)
	}

	// Test GetFieldByID with invalid ID
	fieldName := dict.GetFieldByID(-1)
	if fieldName != "" {
		t.Errorf("Expected empty string for invalid field ID, got '%s'", fieldName)
	}

	fieldName = dict.GetFieldByID(9999)
	if fieldName != "" {
		t.Errorf("Expected empty string for out-of-bounds field ID, got '%s'", fieldName)
	}

	// Test IsFieldInBitmask with field ID >= 64
	result := dict.IsFieldInBitmask(64, 0xFF)
	if result {
		t.Error("Expected false for field ID >= 64")
	}

	// Test RegisterFromConfig with nil slices
	err := dict.RegisterFromConfig(nil, nil)
	if err != nil {
		t.Errorf("Expected no error for nil config, got %v", err)
	}

	// Test RegisterCustomFieldsWithIDs with nil map
	err = dict.RegisterCustomFieldsWithIDs(nil)
	if err != nil {
		t.Errorf("Expected no error for nil custom fields, got %v", err)
	}
}
