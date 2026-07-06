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

package core

import (
	"testing"

	"github.com/go-gen-ecosystem/halolog/types"
)

// TestEntryAddField tests the Entry.AddField method.
func TestEntryAddField(t *testing.T) {
	var entry Entry

	// Test adding fields
	entry.AddField("key1", "value1")
	entry.AddField("key2", 42)
	entry.AddField("key3", true)

	if entry.FieldCount() != 3 {
		t.Errorf("Expected 3 fields, got %d", entry.FieldCount())
	}

	// Verify fields
	if entry.staticFields[0].Key != "key1" {
		t.Errorf("Expected key1, got %s", entry.staticFields[0].Key)
	}
	if entry.staticFields[0].Value != "value1" {
		t.Errorf("Expected value1, got %v", entry.staticFields[0].Value)
	}
}

// TestEntryStaticFieldsLimit tests that static fields work up to 16.
func TestEntryStaticFieldsLimit(t *testing.T) {
	var entry Entry

	// Add 16 fields (should all be in static storage)
	for i := 0; i < 16; i++ {
		entry.AddField("key", i)
	}

	if entry.fieldCount != 16 {
		t.Errorf("Expected 16 static fields, got %d", entry.fieldCount)
	}
	if len(entry.dynamicFields) != 0 {
		t.Errorf("Expected 0 dynamic fields, got %d", len(entry.dynamicFields))
	}

	// Add one more (should go to dynamic)
	entry.AddField("key17", 17)

	if entry.fieldCount != 16 {
		t.Errorf("Expected 16 static fields, got %d", entry.fieldCount)
	}
	if len(entry.dynamicFields) != 1 {
		t.Errorf("Expected 1 dynamic field, got %d", len(entry.dynamicFields))
	}
	if entry.FieldCount() != 17 {
		t.Errorf("Expected 17 total fields, got %d", entry.FieldCount())
	}
}

// TestEntryReset tests the Entry.Reset method.
func TestEntryReset(t *testing.T) {
	var entry Entry

	entry.Level = types.ErrorLevel
	entry.Message = "test message"
	entry.Component = "test-component"
	entry.AddField("key1", "value1")

	entry.Reset()

	if entry.Level != types.InfoLevel {
		t.Errorf("Expected InfoLevel after reset, got %v", entry.Level)
	}
	if entry.Message != "" {
		t.Errorf("Expected empty message after reset, got %s", entry.Message)
	}
	if entry.fieldCount != 0 {
		t.Errorf("Expected 0 fields after reset, got %d", entry.fieldCount)
	}
}

// TestEntryForEachField tests the Entry.ForEachField method.
func TestEntryForEachField(t *testing.T) {
	var entry Entry

	entry.AddField("key1", "value1")
	entry.AddField("key2", 42)
	entry.AddField("key3", true)

	count := 0
	entry.ForEachField(func(key string, value interface{}) bool {
		count++
		return true
	})

	if count != 3 {
		t.Errorf("Expected 3 iterations, got %d", count)
	}

	// Test early termination
	count = 0
	entry.ForEachField(func(key string, value interface{}) bool {
		count++
		return count < 2 // Stop after 2
	})

	if count != 2 {
		t.Errorf("Expected 2 iterations (early stop), got %d", count)
	}
}

// TestEntryToLogEntry tests conversion to types.LogEntry.
func TestEntryToLogEntry(t *testing.T) {
	var entry Entry

	entry.Level = types.WarnLevel
	entry.Message = "test message"
	entry.Component = "test-component"
	entry.Timestamp = 1234567890
	entry.AddField("key1", "value1")

	var logEntry types.LogEntry
	entry.ToLogEntry(&logEntry)

	if logEntry.Level != types.WarnLevel {
		t.Errorf("Expected WarnLevel, got %v", logEntry.Level)
	}
	if logEntry.Message != "test message" {
		t.Errorf("Expected 'test message', got %s", logEntry.Message)
	}
	if logEntry.Component != "test-component" {
		t.Errorf("Expected 'test-component', got %s", logEntry.Component)
	}
	if logEntry.TimestampUnix != 1234567890 {
		t.Errorf("Expected 1234567890, got %d", logEntry.TimestampUnix)
	}
	if logEntry.StaticFieldCount != 1 {
		t.Errorf("Expected 1 static field, got %d", logEntry.StaticFieldCount)
	}
}
