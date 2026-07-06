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
	"regexp"
	"testing"
)

// BenchmarkFieldDict_O1Lookup benchmarks the O(1) field lookup performance
func BenchmarkFieldDict_O1Lookup(b *testing.B) {
	dict := NewFieldDictionary()

	// Register common fields at startup (simulating production scenario)
	startupFields := []string{
		"user_id", "request_id", "session_id", "ip_address", "user_agent",
		"timestamp", "level", "message", "error", "duration",
		"method", "path", "status_code", "response_size", "content_type",
	}

	err := dict.RegisterFromConfig(startupFields, nil)
	if err != nil {
		b.Fatalf("Failed to register startup fields: %v", err)
	}

	// Reset timer to exclude setup time
	b.ResetTimer()

	// Benchmark O(1) lookup for existing fields
	b.Run("ExistingFields", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			fieldName := startupFields[i%len(startupFields)]
			_, exists := dict.GetFieldID(fieldName)
			if !exists {
				b.Errorf("Expected field %s to exist", fieldName)
			}
		}
	})

	// Benchmark lookup for non-existing fields (should still be fast)
	b.Run("NonExistingFields", func(b *testing.B) {
		nonExistingFields := []string{"unknown_field", "missing_field", "invalid_field"}
		for i := 0; i < b.N; i++ {
			fieldName := nonExistingFields[i%len(nonExistingFields)]
			_, exists := dict.GetFieldID(fieldName)
			if exists {
				b.Errorf("Expected field %s to not exist", fieldName)
			}
		}
	})
}

// BenchmarkRegexLookup benchmarks regex-based field matching (comparison baseline)
func BenchmarkRegexLookup(b *testing.B) {
	// Common regex patterns used for field matching in logging systems
	patterns := []string{
		`^user_`,
		`^request_`,
		`^session_`,
		`^ip_`,
		`^timestamp`,
		`^level$`,
		`^message$`,
		`^error$`,
		`^method$`,
		`^path$`,
		`^status_`,
		`^response_`,
		`^content_`,
	}

	// Compile regex patterns
	regexes := make([]*regexp.Regexp, len(patterns))
	for i, pattern := range patterns {
		regex, err := regexp.Compile(pattern)
		if err != nil {
			b.Fatalf("Failed to compile regex %s: %v", pattern, err)
		}
		regexes[i] = regex
	}

	// Test field names (mix of matching and non-matching)
	testFields := []string{
		"user_id", "request_id", "session_id", "ip_address", "user_agent",
		"timestamp", "level", "message", "error", "duration",
		"method", "path", "status_code", "response_size", "content_type",
		"unknown_field", "missing_field", "invalid_field",
	}

	b.ResetTimer()

	// Benchmark regex matching
	b.Run("RegexMatching", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			fieldName := testFields[i%len(testFields)]
			matched := false
			for _, regex := range regexes {
				if regex.MatchString(fieldName) {
					matched = true
					break
				}
			}
			// Use the result to prevent compiler optimization
			if matched && fieldName == "nonexistent" {
				b.Error("Unexpected match")
			}
		}
	})
}

// BenchmarkFieldDict_vs_RegexComparison provides a direct comparison
func BenchmarkFieldDict_vs_RegexComparison(b *testing.B) {
	// Setup FieldDict
	dict := NewFieldDictionary()
	startupFields := []string{
		"user_id", "request_id", "session_id", "ip_address", "user_agent",
		"timestamp", "level", "message", "error", "duration",
		"method", "path", "status_code", "response_size", "content_type",
	}
	err := dict.RegisterFromConfig(startupFields, nil)
	if err != nil {
		b.Fatalf("Failed to register startup fields: %v", err)
	}

	// Setup regex patterns
	patterns := []string{
		`^user_`, `^request_`, `^session_`, `^ip_`, `^timestamp`,
		`^level$`, `^message$`, `^error$`, `^method$`, `^path$`,
		`^status_`, `^response_`, `^content_`,
	}
	regexes := make([]*regexp.Regexp, len(patterns))
	for i, pattern := range patterns {
		regex, err := regexp.Compile(pattern)
		if err != nil {
			b.Fatalf("Failed to compile regex %s: %v", pattern, err)
		}
		regexes[i] = regex
	}

	testFields := []string{
		"user_id", "request_id", "session_id", "ip_address", "user_agent",
		"timestamp", "level", "message", "error", "duration",
		"method", "path", "status_code", "response_size", "content_type",
	}

	b.ResetTimer()

	// Benchmark FieldDict lookup
	b.Run("FieldDict_O1", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			fieldName := testFields[i%len(testFields)]
			_, exists := dict.GetFieldID(fieldName)
			if !exists {
				b.Errorf("Expected field %s to exist", fieldName)
			}
		}
	})

	// Benchmark regex matching
	b.Run("Regex_Matching", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			fieldName := testFields[i%len(testFields)]
			matched := false
			for _, regex := range regexes {
				if regex.MatchString(fieldName) {
					matched = true
					break
				}
			}
			// Use the result to prevent compiler optimization
			if !matched && fieldName == "user_id" {
				b.Error("Expected user_id to match")
			}
		}
	})
}

// BenchmarkFieldDict_MemoryAllocation benchmarks memory allocation
func BenchmarkFieldDict_MemoryAllocation(b *testing.B) {
	dict := NewFieldDictionary()
	startupFields := []string{"user_id", "request_id", "session_id", "ip_address", "user_agent"}
	err := dict.RegisterFromConfig(startupFields, nil)
	if err != nil {
		b.Fatalf("Failed to register startup fields: %v", err)
	}

	b.ResetTimer()

	// Benchmark memory allocation for lookups
	b.Run("Lookup_Allocations", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			fieldName := startupFields[i%len(startupFields)]
			_, exists := dict.GetFieldID(fieldName)
			if !exists {
				b.Errorf("Expected field %s to exist", fieldName)
			}
		}
	})
}
