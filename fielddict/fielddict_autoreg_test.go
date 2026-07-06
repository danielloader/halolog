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

func TestGlobalFieldDictionary_AutoRegistration(t *testing.T) {
	// Reset global dictionary for clean test
	GlobalFieldDictionary = NewFieldDictionary()

	// Test auto-registration with NewField
	field1 := NewField("user_id", 12345)
	if field1.KeyID != 0 {
		t.Errorf("Expected KeyID 0 for first field, got %d", field1.KeyID)
	}

	field2 := NewField("request_id", "abc-123")
	if field2.KeyID != 1 {
		t.Errorf("Expected KeyID 1 for second field, got %d", field2.KeyID)
	}

	// Test duplicate key returns same ID
	field3 := NewField("user_id", 67890)
	if field3.KeyID != field1.KeyID {
		t.Errorf("Expected same KeyID for duplicate key, got %d vs %d", field3.KeyID, field1.KeyID)
	}

	// Verify dictionary size
	if GlobalFieldDictionary.Size() != 2 {
		t.Errorf("Expected dictionary size 2, got %d", GlobalFieldDictionary.Size())
	}
}

func TestNewFieldWithDict_AutoRegistration(t *testing.T) {
	customDict := NewFieldDictionary()

	// Test with custom dictionary
	field1 := NewFieldWithDict(customDict, "custom_field", "value1")
	if field1.KeyID != 0 {
		t.Errorf("Expected KeyID 0 for first field in custom dict, got %d", field1.KeyID)
	}

	field2 := NewFieldWithDict(customDict, "another_field", "value2")
	if field2.KeyID != 1 {
		t.Errorf("Expected KeyID 1 for second field in custom dict, got %d", field2.KeyID)
	}

	// Test nil dictionary falls back to global
	// Reset global dictionary for predictable results
	originalGlobal := GlobalFieldDictionary
	GlobalFieldDictionary = NewFieldDictionary()
	defer func() { GlobalFieldDictionary = originalGlobal }()

	field3 := NewFieldWithDict(nil, "global_field", "value3")
	if field3.KeyID != 0 {
		t.Errorf("Expected KeyID 0 for first field in global dict, got %d", field3.KeyID)
	}
}

func TestField_StringRepresentation(t *testing.T) {
	// Reset global dictionary for clean test
	GlobalFieldDictionary = NewFieldDictionary()

	field := NewField("test_key", "test_value")
	str := field.String()

	if str != "test_key=test_value" {
		t.Errorf("Expected string 'test_key=test_value', got '%s'", str)
	}

	// Test with custom dictionary
	customDict := NewFieldDictionary()
	field2 := NewFieldWithDict(customDict, "custom_key", 42)
	str2 := field2.StringWithDict(customDict)

	if str2 != "custom_key=42" {
		t.Errorf("Expected string 'custom_key=42', got '%s'", str2)
	}
}

func TestField_StringWithMissingKey(t *testing.T) {
	// Create field with invalid KeyID
	field := Field{KeyID: 999, Value: "test_value"}
	str := field.String()

	expected := "field_999=test_value"
	if str != expected {
		t.Errorf("Expected string '%s', got '%s'", expected, str)
	}
}

func TestCreateStandardFieldDictionary(t *testing.T) {
	stdDict := CreateStandardFieldDictionary()

	// Verify it has standard fields registered
	if stdDict.Size() == 0 {
		t.Error("Standard field dictionary should have pre-registered fields")
	}

	// Test that we can retrieve standard fields
	standardFields := GetStandardFields()
	for fieldName := range standardFields {
		id, exists := stdDict.GetID(fieldName)
		if !exists {
			t.Errorf("Standard field '%s' should be registered", fieldName)
		}
		if id < 0 {
			t.Errorf("Field ID for '%s' should be non-negative, got %d", fieldName, id)
		}
	}

	// Test GetAllFields method
	allFields := stdDict.GetAllFields()
	if len(allFields) != stdDict.Size() {
		t.Errorf("GetAllFields() returned %d fields, expected %d", len(allFields), stdDict.Size())
	}

	// Test GetAllFieldIDs method
	allIDs := stdDict.GetAllFieldIDs()
	if len(allIDs) != stdDict.Size() {
		t.Errorf("GetAllFieldIDs() returned %d entries, expected %d", len(allIDs), stdDict.Size())
	}
}

func TestGlobalFieldDictionary_ConcurrentAutoRegistration(t *testing.T) {
	// Reset global dictionary for clean test
	GlobalFieldDictionary = NewFieldDictionary()

	var wg sync.WaitGroup
	errors := make(chan error, 100)

	// Concurrent field creation with auto-registration
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// Create multiple fields concurrently
			for j := 0; j < 10; j++ {
				fieldName := "concurrent_field_" + string(rune('a'+id))
				field := NewField(fieldName, j)

				// Verify field was created
				if field.KeyID < 0 {
					errors <- fmt.Errorf("invalid KeyID for field %s: %d", fieldName, field.KeyID)
					return
				}

				// Verify string representation works
				str := field.String()
				if str == "" {
					errors <- fmt.Errorf("empty string representation for field %s", fieldName)
					return
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		t.Error(err)
	}

	// Verify final state
	finalSize := GlobalFieldDictionary.Size()
	if finalSize == 0 {
		t.Error("Global dictionary should have registered fields after concurrent access")
	}
	t.Logf("Final dictionary size after concurrent access: %d", finalSize)
}

func TestFieldDictionary_IntegrationWithOptimizedStore(t *testing.T) {
	// This test simulates how the field dictionary integrates with optimized storage
	dict := NewFieldDictionary()

	// Simulate field registration for optimized store
	fieldNames := []string{"user_id", "request_id", "session_id", "trace_id", "span_id"}
	expectedIDs := make(map[string]int)

	for _, fieldName := range fieldNames {
		id := dict.GetOrRegisterFieldID(fieldName)
		expectedIDs[fieldName] = id
	}

	// Verify all fields are registered
	if dict.Size() != len(fieldNames) {
		t.Errorf("Expected %d registered fields, got %d", len(fieldNames), dict.Size())
	}

	// Verify we can retrieve all field IDs
	allIDs := dict.GetAllFieldIDs()
	if len(allIDs) != len(fieldNames) {
		t.Errorf("Expected %d field IDs, got %d", len(fieldNames), len(allIDs))
	}

	// Verify each field ID matches expected
	for fieldName, expectedID := range expectedIDs {
		actualID, exists := allIDs[fieldName]
		if !exists {
			t.Errorf("Field '%s' not found in GetAllFieldIDs()", fieldName)
			continue
		}
		if actualID != expectedID {
			t.Errorf("Field '%s' ID mismatch: expected %d, got %d", fieldName, expectedID, actualID)
		}
	}

	// Test bitmask creation for optimized store optimization
	bitmask := dict.CreateFieldBitmask(fieldNames...)
	if bitmask == 0 {
		t.Error("Bitmask should not be zero for valid field names")
	}

	// Test field presence checking
	for _, fieldName := range fieldNames {
		fieldID := expectedIDs[fieldName]
		if !dict.IsFieldInBitmask(fieldID, bitmask) {
			t.Errorf("Field '%s' (ID: %d) should be in bitmask", fieldName, fieldID)
		}
	}
}
