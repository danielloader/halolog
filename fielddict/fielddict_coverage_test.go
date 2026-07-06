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
	"testing"
)

// TestFieldConstants tests all field constants
func TestFieldConstants(t *testing.T) {
	// Test all standard field constants
	constants := []struct {
		name     string
		constant string
		expected string
	}{
		{"Timestamp", FieldTimestamp, "timestamp"},
		{"Level", FieldLevel, "level"},
		{"Logger", FieldLogger, "logger"},
		{"Component", FieldComponent, "component"},
		{"Message", FieldMessage, "message"},
		{"Error", FieldError, "error"},
		{"File", FieldFile, "file"},
		{"Line", FieldLine, "line"},
		{"Function", FieldFunction, "function"},
		{"GoroutineID", FieldGoroutineID, "goroutine_id"},
		{"MemoryAlloc", FieldMemoryAlloc, "memory_alloc"},
		{"MemoryTotalAlloc", FieldMemoryTotalAlloc, "memory_total_alloc"},
		{"MemorySys", FieldMemorySys, "memory_sys"},
		{"NumCPU", FieldNumCPU, "num_cpu"},
		{"RequestID", FieldRequestID, "request_id"},
		{"UserID", FieldUserID, "user_id"},
		{"SessionID", FieldSessionID, "session_id"},
		{"IP", FieldIP, "ip"},
		{"UserAgent", FieldUserAgent, "user_agent"},
		{"Method", FieldMethod, "method"},
		{"URL", FieldURL, "url"},
		{"Status", FieldStatus, "status"},
		{"Duration", FieldDuration, "duration"},
		{"BytesIn", FieldBytesIn, "bytes_in"},
		{"BytesOut", FieldBytesOut, "bytes_out"},
		{"TraceID", FieldTraceID, "trace_id"},
		{"SpanID", FieldSpanID, "span_id"},
	}

	for _, test := range constants {
		if test.constant != test.expected {
			t.Errorf("%s constant = %s, want %s", test.name, test.constant, test.expected)
		}
	}
}

// TestGetStandardFields tests GetStandardFields function
func TestGetStandardFields(t *testing.T) {
	standardFields := GetStandardFields()

	// Check that we get a non-nil map
	if standardFields == nil {
		t.Fatal("GetStandardFields() should not return nil")
	}

	// Check that all standard fields are present
	expectedFields := []string{
		FieldTimestamp,
		FieldLevel,
		FieldLogger,
		FieldComponent,
		FieldMessage,
		FieldError,
		FieldFile,
		FieldLine,
		FieldFunction,
		FieldGoroutineID,
		FieldMemoryAlloc,
		FieldMemoryTotalAlloc,
		FieldMemorySys,
		FieldNumCPU,
	}

	for _, field := range expectedFields {
		if _, exists := standardFields[field]; !exists {
			t.Errorf("Standard field %s should be present", field)
		}
	}

	// Check that all values are properly formatted
	for field, value := range standardFields {
		if value == "" {
			t.Errorf("Field %s should not have empty value", field)
		}
	}
}

// TestGetRequestFields tests GetRequestFields function
func TestGetRequestFields(t *testing.T) {
	requestFields := GetRequestFields()

	// Check that we get a non-nil map
	if requestFields == nil {
		t.Fatal("GetRequestFields() should not return nil")
	}

	// Check that all request fields are present
	expectedFields := []string{
		FieldRequestID,
		FieldUserID,
		FieldSessionID,
		FieldIP,
		FieldUserAgent,
		FieldMethod,
		FieldURL,
		FieldStatus,
		FieldDuration,
		FieldBytesIn,
		FieldBytesOut,
	}

	for _, field := range expectedFields {
		if _, exists := requestFields[field]; !exists {
			t.Errorf("Request field %s should be present", field)
		}
	}

	// Check that all values are properly formatted
	for field, value := range requestFields {
		if value == "" {
			t.Errorf("Field %s should not have empty value", field)
		}
	}
}

// TestGetTracingFields tests GetTracingFields function
func TestGetTracingFields(t *testing.T) {
	tracingFields := GetTracingFields()

	// Check that we get a non-nil map
	if tracingFields == nil {
		t.Fatal("GetTracingFields() should not return nil")
	}

	// Check that all tracing fields are present
	expectedFields := []string{
		FieldTraceID,
		FieldSpanID,
	}

	for _, field := range expectedFields {
		if _, exists := tracingFields[field]; !exists {
			t.Errorf("Tracing field %s should be present", field)
		}
	}

	// Check that all values are properly formatted
	for field, value := range tracingFields {
		if value == "" {
			t.Errorf("Field %s should not have empty value", field)
		}
	}
}

// TestGetAllFields tests GetAllFields function
func TestGetAllFields(t *testing.T) {
	allFields := GetAllFields()

	// Check that we get a non-nil map
	if allFields == nil {
		t.Fatal("GetAllFields() should not return nil")
	}

	// Check that all field categories are included
	standardFields := GetStandardFields()
	requestFields := GetRequestFields()
	tracingFields := GetTracingFields()

	// Verify standard fields are included
	for field, description := range standardFields {
		if allFields[field] != description {
			t.Errorf("Standard field %s should be in all fields", field)
		}
	}

	// Verify request fields are included
	for field, description := range requestFields {
		if allFields[field] != description {
			t.Errorf("Request field %s should be in all fields", field)
		}
	}

	// Verify tracing fields are included
	for field, description := range tracingFields {
		if allFields[field] != description {
			t.Errorf("Tracing field %s should be in all fields", field)
		}
	}

	// Check that we have a reasonable number of fields
	if len(allFields) < 20 {
		t.Errorf("Expected at least 20 fields in GetAllFields(), got %d", len(allFields))
	}
}

// TestFieldDescriptions tests field descriptions
func TestFieldDescriptions(t *testing.T) {
	// Test standard field descriptions
	standardFields := GetStandardFields()

	expectedDescriptions := map[string]string{
		FieldTimestamp:        "Unix timestamp when the log was created",
		FieldLevel:            "Log level (DEBUG, INFO, WARN, ERROR, FATAL)",
		FieldLogger:           "Name of the logger instance",
		FieldComponent:        "Component or module name",
		FieldMessage:          "Log message content",
		FieldError:            "Error message or stack trace",
		FieldFile:             "Source file where log was generated",
		FieldLine:             "Line number in source file",
		FieldFunction:         "Function name where log was generated",
		FieldGoroutineID:      "Go routine ID for debugging",
		FieldMemoryAlloc:      "Current memory allocation in bytes",
		FieldMemoryTotalAlloc: "Total memory allocated in bytes",
		FieldMemorySys:        "Memory obtained from system in bytes",
		FieldNumCPU:           "Number of CPU cores",
	}

	for field, expectedDesc := range expectedDescriptions {
		if standardFields[field] != expectedDesc {
			t.Errorf("Field %s description = %s, want %s", field, standardFields[field], expectedDesc)
		}
	}

	// Test request field descriptions
	requestFields := GetRequestFields()

	expectedRequestDescriptions := map[string]string{
		FieldRequestID: "Unique request identifier",
		FieldUserID:    "User identifier",
		FieldSessionID: "Session identifier",
		FieldIP:        "Client IP address",
		FieldUserAgent: "Client user agent string",
		FieldMethod:    "HTTP request method",
		FieldURL:       "Request URL",
		FieldStatus:    "HTTP response status code",
		FieldDuration:  "Request duration in milliseconds",
		FieldBytesIn:   "Bytes received",
		FieldBytesOut:  "Bytes sent",
	}

	for field, expectedDesc := range expectedRequestDescriptions {
		if requestFields[field] != expectedDesc {
			t.Errorf("Request field %s description = %s, want %s", field, requestFields[field], expectedDesc)
		}
	}

	// Test tracing field descriptions
	tracingFields := GetTracingFields()

	expectedTracingDescriptions := map[string]string{
		FieldTraceID: "Distributed trace identifier",
		FieldSpanID:  "Span identifier within trace",
	}

	for field, expectedDesc := range expectedTracingDescriptions {
		if tracingFields[field] != expectedDesc {
			t.Errorf("Tracing field %s description = %s, want %s", field, tracingFields[field], expectedDesc)
		}
	}
}

// TestFieldConsistency tests that all field constants are consistent
func TestFieldConsistency(t *testing.T) {
	allFields := GetAllFields()

	// Verify that all field constants are used
	fieldConstants := []string{
		FieldTimestamp,
		FieldLevel,
		FieldLogger,
		FieldComponent,
		FieldMessage,
		FieldError,
		FieldFile,
		FieldLine,
		FieldFunction,
		FieldGoroutineID,
		FieldMemoryAlloc,
		FieldMemoryTotalAlloc,
		FieldMemorySys,
		FieldNumCPU,
		FieldRequestID,
		FieldUserID,
		FieldSessionID,
		FieldIP,
		FieldUserAgent,
		FieldMethod,
		FieldURL,
		FieldStatus,
		FieldDuration,
		FieldBytesIn,
		FieldBytesOut,
		FieldTraceID,
		FieldSpanID,
	}

	for _, field := range fieldConstants {
		if _, exists := allFields[field]; !exists {
			t.Errorf("Field constant %s should be in GetAllFields()", field)
		}
	}

	// Verify that all fields in GetAllFields() have proper descriptions
	for field, description := range allFields {
		if description == "" {
			t.Errorf("Field %s should have a non-empty description", field)
		}

		if len(description) < 5 {
			t.Errorf("Field %s description should be at least 5 characters, got %d", field, len(description))
		}
	}
}
