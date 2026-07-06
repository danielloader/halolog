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

func TestNewSensitiveFieldRegistry(t *testing.T) {
	registry := NewSensitiveFieldRegistry()

	if registry == nil {
		t.Fatal("NewSensitiveFieldRegistry should not return nil")
	}

	if registry.sensitiveFields == nil {
		t.Fatal("sensitiveFields map should be initialized")
	}

	if registry.sensitiveFieldsByID == nil {
		t.Fatal("sensitiveFieldsByID map should be initialized")
	}

	if registry.fieldIDs == nil {
		t.Fatal("fieldIDs map should be initialized")
	}

	if registry.precomputedMasks == nil {
		t.Fatal("precomputedMasks map should be initialized")
	}

	if registry.defaultUnknownMask != "***REDACTED***" {
		t.Errorf("Expected defaultUnknownMask to be '***REDACTED***', got '%s'", registry.defaultUnknownMask)
	}
}

func TestSensitiveFieldRegistry_RegisterSensitiveField(t *testing.T) {
	registry := NewSensitiveFieldRegistry()

	// Test registering a new sensitive field
	registry.RegisterSensitiveField("api_key", "***API_KEY***")

	mask, exists := registry.GetMask("api_key")
	if !exists {
		t.Error("Expected api_key to be registered as sensitive")
	}
	if mask != "***API_KEY***" {
		t.Errorf("Expected mask '***API_KEY***', got '%s'", mask)
	}

	// Test registering another field
	registry.RegisterSensitiveField("password", "***PASSWORD***")

	mask, exists = registry.GetMask("password")
	if !exists {
		t.Error("Expected password to be registered as sensitive")
	}
	if mask != "***PASSWORD***" {
		t.Errorf("Expected mask '***PASSWORD***', got '%s'", mask)
	}
}

func TestSensitiveFieldRegistry_GetMask(t *testing.T) {
	registry := NewSensitiveFieldRegistry()

	// Register test fields
	registry.RegisterSensitiveField("token", "***TOKEN***")
	registry.RegisterSensitiveField("email", "***EMAIL***")

	// Test getting existing masks
	tests := []struct {
		fieldName    string
		expectedMask string
		shouldExist  bool
	}{
		{"token", "***TOKEN***", true},
		{"email", "***EMAIL***", true},
		{"nonexistent", "", false},
		{"", "", false},
	}

	for _, test := range tests {
		mask, exists := registry.GetMask(test.fieldName)
		if exists != test.shouldExist {
			t.Errorf("Field '%s': expected exists=%v, got %v", test.fieldName, test.shouldExist, exists)
		}
		if exists && mask != test.expectedMask {
			t.Errorf("Field '%s': expected mask '%s', got '%s'", test.fieldName, test.expectedMask, mask)
		}
	}
}

func TestSensitiveFieldRegistry_IsSensitiveField(t *testing.T) {
	registry := NewSensitiveFieldRegistry()

	// Register test fields
	registry.RegisterSensitiveField("ssn", "***SSN***")
	registry.RegisterSensitiveField("credit_card", "***CREDIT_CARD***")

	// Test checking sensitivity
	tests := []struct {
		fieldName string
		expected  bool
	}{
		{"ssn", true},
		{"credit_card", true},
		{"non_sensitive", false},
		{"", false},
	}

	for _, test := range tests {
		result := registry.IsSensitiveField(test.fieldName)
		if result != test.expected {
			t.Errorf("Field '%s': expected IsSensitiveField=%v, got %v", test.fieldName, test.expected, result)
		}
	}
}

func TestSensitiveFieldRegistry_GetDefaultUnknownMask(t *testing.T) {
	registry := NewSensitiveFieldRegistry()

	defaultMask := registry.GetDefaultUnknownMask()
	if defaultMask != "***REDACTED***" {
		t.Errorf("Expected default unknown mask '***REDACTED***', got '%s'", defaultMask)
	}

	// Test setting custom default mask
	registry.SetDefaultUnknownMask("[HIDDEN]")

	defaultMask = registry.GetDefaultUnknownMask()
	if defaultMask != "[HIDDEN]" {
		t.Errorf("Expected custom default unknown mask '[HIDDEN]', got '%s'", defaultMask)
	}
}

func TestSensitiveFieldRegistry_SetDefaultUnknownMask(t *testing.T) {
	registry := NewSensitiveFieldRegistry()

	// Test setting custom default mask
	registry.SetDefaultUnknownMask("[CONFIDENTIAL]")

	defaultMask := registry.GetDefaultUnknownMask()
	if defaultMask != "[CONFIDENTIAL]" {
		t.Errorf("Expected custom default unknown mask '[CONFIDENTIAL]', got '%s'", defaultMask)
	}

	// Test setting empty mask
	registry.SetDefaultUnknownMask("")

	defaultMask = registry.GetDefaultUnknownMask()
	if defaultMask != "" {
		t.Errorf("Expected empty default unknown mask, got '%s'", defaultMask)
	}
}

func TestSensitiveFieldRegistry_GetAllSensitiveFields(t *testing.T) {
	registry := NewSensitiveFieldRegistry()

	// Get initial count (default fields)
	initialFields := registry.GetAllSensitiveFields()
	initialCount := len(initialFields)

	// Register test fields
	registry.RegisterSensitiveField("field1", "***FIELD1***")
	registry.RegisterSensitiveField("field2", "***FIELD2***")
	registry.RegisterSensitiveField("field3", "***FIELD3***")

	fields := registry.GetAllSensitiveFields()

	expectedCount := initialCount + 3
	if len(fields) != expectedCount {
		t.Errorf("Expected %d sensitive fields, got %d", expectedCount, len(fields))
	}

	// Check that all registered fields are in the result
	fieldMap := make(map[string]bool)
	for _, field := range fields {
		fieldMap[field] = true
	}

	expectedFields := []string{"field1", "field2", "field3"}
	for _, expected := range expectedFields {
		if !fieldMap[expected] {
			t.Errorf("Expected field '%s' not found in result", expected)
		}
	}
}

func TestSensitiveFieldRegistry_GetAllMasks(t *testing.T) {
	registry := NewSensitiveFieldRegistry()

	// Get initial count (default fields)
	initialMasks := registry.GetAllMasks()
	initialCount := len(initialMasks)

	// Register test fields
	registry.RegisterSensitiveField("api_token", "***API_TOKEN***")
	registry.RegisterSensitiveField("user_email", "***USER_EMAIL***")

	masks := registry.GetAllMasks()

	expectedCount := initialCount + 2
	if len(masks) != expectedCount {
		t.Errorf("Expected %d mask mappings, got %d", expectedCount, len(masks))
	}

	expectedMasks := map[string]string{
		"api_token":  "***API_TOKEN***",
		"user_email": "***USER_EMAIL***",
	}

	for field, expectedMask := range expectedMasks {
		if mask, exists := masks[field]; !exists {
			t.Errorf("Expected field '%s' not found in masks", field)
		} else if mask != expectedMask {
			t.Errorf("Field '%s': expected mask '%s', got '%s'", field, expectedMask, mask)
		}
	}
}

func TestSensitiveFieldRegistry_Size(t *testing.T) {
	registry := NewSensitiveFieldRegistry()

	// Get initial size (default fields)
	initialSize := registry.Size()
	if initialSize == 0 {
		t.Error("Expected initial size to be greater than 0 (default fields)")
	}

	registry.RegisterSensitiveField("field1", "***FIELD1***")
	if registry.Size() != initialSize+1 {
		t.Errorf("Expected size %d, got %d", initialSize+1, registry.Size())
	}

	registry.RegisterSensitiveField("field2", "***FIELD2***")
	registry.RegisterSensitiveField("field3", "***FIELD3***")
	if registry.Size() != initialSize+3 {
		t.Errorf("Expected size %d, got %d", initialSize+3, registry.Size())
	}
}

func TestSensitiveFieldRegistry_RegisterWithFieldID(t *testing.T) {
	registry := NewSensitiveFieldRegistry()

	// Register field with field ID
	registry.RegisterWithFieldID("api_key", 100, "***API_KEY***")

	// Test lookup by field name
	mask, exists := registry.GetMask("api_key")
	if !exists {
		t.Error("Expected api_key to be registered by name")
	}
	if mask != "***API_KEY***" {
		t.Errorf("Expected mask '***API_KEY***', got '%s'", mask)
	}

	// Test lookup by field ID (O(1) High-performance path)
	mask, exists = registry.GetMaskByFieldID(100)
	if !exists {
		t.Error("Expected field ID 100 to be registered")
	}
	if mask != "***API_KEY***" {
		t.Errorf("Expected mask '***API_KEY***' for field ID 100, got '%s'", mask)
	}

	// Test sensitivity check by field ID
	if !registry.IsSensitiveFieldByID(100) {
		t.Error("Expected field ID 100 to be sensitive")
	}

	if registry.IsSensitiveFieldByID(999) {
		t.Error("Expected field ID 999 to not be sensitive")
	}
}

func TestSensitiveFieldRegistry_GetMaskByFieldID(t *testing.T) {
	registry := NewSensitiveFieldRegistry()

	// Register fields with IDs
	registry.RegisterWithFieldID("field1", 1, "***FIELD1***")
	registry.RegisterWithFieldID("field2", 2, "***FIELD2***")

	tests := []struct {
		fieldID      int
		expectedMask string
		shouldExist  bool
	}{
		{1, "***FIELD1***", true},
		{2, "***FIELD2***", true},
		{999, "", false},
		{0, "", false},
	}

	for _, test := range tests {
		mask, exists := registry.GetMaskByFieldID(test.fieldID)
		if exists != test.shouldExist {
			t.Errorf("Field ID %d: expected exists=%v, got %v", test.fieldID, test.shouldExist, exists)
		}
		if exists && mask != test.expectedMask {
			t.Errorf("Field ID %d: expected mask '%s', got '%s'", test.fieldID, test.expectedMask, mask)
		}
	}
}

func TestSensitiveFieldRegistry_IsSensitiveFieldByID(t *testing.T) {
	registry := NewSensitiveFieldRegistry()

	// Register fields with IDs
	registry.RegisterWithFieldID("sensitive1", 100, "***SENSITIVE1***")
	registry.RegisterWithFieldID("sensitive2", 200, "***SENSITIVE2***")

	tests := []struct {
		fieldID  int
		expected bool
	}{
		{100, true},
		{200, true},
		{999, false},
		{0, false},
		{-1, false},
	}

	for _, test := range tests {
		result := registry.IsSensitiveFieldByID(test.fieldID)
		if result != test.expected {
			t.Errorf("Field ID %d: expected IsSensitiveFieldByID=%v, got %v", test.fieldID, test.expected, result)
		}
	}
}

func TestSensitiveFieldRegistry_RegisterFromConfig(t *testing.T) {
	registry := NewSensitiveFieldRegistry()

	// Create test configuration
	config := []SensitiveFieldConfig{
		{Name: "api_key", FieldID: 100, Mask: "***API_KEY***"},
		{Name: "token", FieldID: 101, Mask: "***TOKEN***"},
		{Name: "email", FieldID: 0, Mask: "***EMAIL***"}, // FieldID 0 means no ID-based lookup
	}

	err := registry.RegisterFromConfig(config)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Test lookup by name
	tests := []struct {
		fieldName    string
		expectedMask string
		shouldExist  bool
	}{
		{"api_key", "***API_KEY***", true},
		{"token", "***TOKEN***", true},
		{"email", "***EMAIL***", true},
		{"nonexistent", "", false},
	}

	for _, test := range tests {
		mask, exists := registry.GetMask(test.fieldName)
		if exists != test.shouldExist {
			t.Errorf("Field '%s': expected exists=%v, got %v", test.fieldName, test.shouldExist, exists)
		}
		if exists && mask != test.expectedMask {
			t.Errorf("Field '%s': expected mask '%s', got '%s'", test.fieldName, test.expectedMask, mask)
		}
	}

	// Test lookup by field ID (only for fields with FieldID > 0)
	mask, exists := registry.GetMaskByFieldID(100)
	if !exists {
		t.Error("Expected field ID 100 to be registered")
	}
	if mask != "***API_KEY***" {
		t.Errorf("Expected mask '***API_KEY***' for field ID 100, got '%s'", mask)
	}

	mask, exists = registry.GetMaskByFieldID(101)
	if !exists {
		t.Error("Expected field ID 101 to be registered")
	}
	if mask != "***TOKEN***" {
		t.Errorf("Expected mask '***TOKEN***' for field ID 101, got '%s'", mask)
	}

	// Field ID 0 should not be registered for ID-based lookup
	_, exists = registry.GetMaskByFieldID(0)
	if exists {
		t.Error("Expected field ID 0 to not be registered for ID-based lookup")
	}
}

func TestSensitiveFieldRegistry_ConcurrentAccess(t *testing.T) {
	registry := NewSensitiveFieldRegistry()

	// Test concurrent reads and writes
	var wg sync.WaitGroup
	numGoroutines := 100

	// Concurrent writes
	for i := 0; i < numGoroutines/2; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			fieldName := fmt.Sprintf("field_%d", id)
			maskValue := fmt.Sprintf("***MASK_%d***", id)
			registry.RegisterSensitiveField(fieldName, maskValue)
		}(i)
	}

	// Concurrent reads
	for i := 0; i < numGoroutines/2; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			fieldName := fmt.Sprintf("field_%d", id)
			registry.GetMask(fieldName)
			registry.IsSensitiveField(fieldName)
		}(i)
	}

	wg.Wait()

	// Verify some fields were registered
	size := registry.Size()
	if size == 0 {
		t.Error("Expected some fields to be registered after concurrent access")
	}
}

func TestSensitiveFieldRegistry_DefaultSensitiveFields(t *testing.T) {
	registry := NewSensitiveFieldRegistry()

	// Test that default sensitive fields are registered
	defaultFields := []string{
		"password", "passwd", "pwd", "pass",
		"secret", "token", "api_key", "apikey",
		"jwt", "hash", "email", "ssn",
		"social_security", "phone", "mobile",
		"credit_card", "card_number",
	}

	for _, field := range defaultFields {
		if !registry.IsSensitiveField(field) {
			t.Errorf("Expected default sensitive field '%s' to be registered", field)
		}

		mask, exists := registry.GetMask(field)
		if !exists {
			t.Errorf("Expected default sensitive field '%s' to have a mask", field)
		}
		if mask == "" {
			t.Errorf("Expected default sensitive field '%s' to have a non-empty mask", field)
		}
	}
}

func TestSensitiveFieldRegistry_Performance_O1Lookup(t *testing.T) {
	registry := NewSensitiveFieldRegistry()

	// Register many fields to test O(1) performance
	numFields := 1000
	for i := 0; i < numFields; i++ {
		fieldName := fmt.Sprintf("field_%d", i)
		maskValue := fmt.Sprintf("***MASK_%d***", i)
		registry.RegisterSensitiveField(fieldName, maskValue)
	}

	// Test lookup performance (should be O(1))
	testField := "field_500"
	expectedMask := "***MASK_500***"

	mask, exists := registry.GetMask(testField)
	if !exists {
		t.Error("Expected field to be found")
	}
	if mask != expectedMask {
		t.Errorf("Expected mask '%s', got '%s'", expectedMask, mask)
	}

	// Test that lookup time doesn't significantly increase with more fields
	// This is more of a functional test - true performance testing would require benchmarks
	if !registry.IsSensitiveField(testField) {
		t.Error("Expected field to be marked as sensitive")
	}
}

func TestSensitiveFieldRegistry_IntegrationWithFieldID(t *testing.T) {
	registry := NewSensitiveFieldRegistry()

	// Test the complete integration: register with ID, lookup by ID, check sensitivity by ID
	registry.RegisterWithFieldID("user_token", 42, "***USER_TOKEN***")

	// Verify all lookup methods work
	if !registry.IsSensitiveField("user_token") {
		t.Error("Expected field to be sensitive by name")
	}

	if !registry.IsSensitiveFieldByID(42) {
		t.Error("Expected field ID 42 to be sensitive")
	}

	mask, exists := registry.GetMask("user_token")
	if !exists || mask != "***USER_TOKEN***" {
		t.Error("Expected correct mask lookup by name")
	}

	mask, exists = registry.GetMaskByFieldID(42)
	if !exists || mask != "***USER_TOKEN***" {
		t.Error("Expected correct mask lookup by field ID")
	}
}
