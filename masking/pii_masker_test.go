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
// Package masking provides PII masking functionality
// Author: Admilson B. F. Cossa

package masking

import (
	"math/rand"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-gen-ecosystem/halolog/types"
	"github.com/go-gen-ecosystem/halolog/utils"
)

// --- Helpers -----------------------------------------------------------------

func assertEqual(t *testing.T, expected, actual string) {
	t.Helper()
	if expected != actual {
		t.Fatalf("\nEXPECTED: %q\nGOT:      %q\n", expected, actual)
	}
}

func assertMaskedField(t *testing.T, fields []types.TypedFieldData, key string, expected any) {
	t.Helper()
	for _, f := range fields {
		if f.Key == key {
			if f.Value != expected {
				t.Fatalf("Field %s expected %v, got %v", key, expected, f.Value)
			}
			return
		}
	}
	t.Fatalf("Field %s not found", key)
}

// --- MOCKS -------------------------------------------------------------------

type piiMockAdapter struct {
	name       string
	writeCalls int
	mu         sync.Mutex
}

func (m *piiMockAdapter) Name() string                   { return m.name }
func (m *piiMockAdapter) Health() error                  { return nil }
func (m *piiMockAdapter) Flush() error                   { return nil }
func (m *piiMockAdapter) Close() error                   { return nil }
func (m *piiMockAdapter) SetFormatter(f types.Formatter) {}

func (m *piiMockAdapter) Write(entry *types.LogEntry) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.writeCalls++
	return nil
}

func (m *piiMockAdapter) WriteZero(z *types.LogEntry) error {
	fields := make([]types.TypedFieldData, 0, len(z.Fields))
	for _, field := range z.Fields {
		fields = append(fields, field)
	}
	return m.Write(&types.LogEntry{
		Level:     types.LogLevel(z.Level),
		Message:   z.Message,
		Component: z.Component,
		Fields:    fields,
		Timestamp: z.Timestamp,
	})
}

// Capturing mock
type piiCapturingMockAdapter struct {
	name       string
	writeCalls int
	lastEntry  *types.LogEntry
	mu         sync.Mutex
}

func (m *piiCapturingMockAdapter) Name() string                   { return m.name }
func (m *piiCapturingMockAdapter) Health() error                  { return nil }
func (m *piiCapturingMockAdapter) Flush() error                   { return nil }
func (m *piiCapturingMockAdapter) Close() error                   { return nil }
func (m *piiCapturingMockAdapter) SetFormatter(f types.Formatter) {}

func (m *piiCapturingMockAdapter) Write(entry *types.LogEntry) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.writeCalls++
	m.lastEntry = entry
	return nil
}

func (m *piiCapturingMockAdapter) WriteZero(z *types.LogEntry) error {
	fields := make([]types.TypedFieldData, 0, len(z.Fields))
	for _, field := range z.Fields {
		fields = append(fields, field)
	}
	return m.Write(&types.LogEntry{
		Level:     types.LogLevel(z.Level),
		Message:   z.Message,
		Component: z.Component,
		Fields:    fields,
		Timestamp: z.Timestamp,
	})
}

// --- TESTS -------------------------------------------------------------------

func TestPIIMasker_LabelAware_Heuristic(t *testing.T) {
	m := NewPIIMasker()

	assertEqual(t,
		"Token: REDACTED",
		m.MaskString("Token: abc123def456ghi789"),
	)

	assertEqual(t,
		"Secret: REDACTED",
		m.MaskString("Secret: P@ssw0rd!123"),
	)
}

func TestPIIMasker_NoDoubleMasking(t *testing.T) {
	m := NewPIIMasker()

	assertEqual(t,
		"[EMAIL]",
		m.MaskString("[EMAIL]"),
	)

	assertEqual(t,
		"Token: REDACTED",
		m.MaskString("Token: REDACTED"),
	)
}

func TestPIIMasker_MultipleHeuristicTokens(t *testing.T) {
	m := NewPIIMasker()

	input := "Send Token: abc123def456 And Key: xyz999777aaa"
	expected := "Send Token: REDACTED And Key: REDACTED"

	assertEqual(t, expected, m.MaskString(input))
}

func TestPIIMasker_Stress_EntropyRandomStrings(t *testing.T) {
	m := NewPIIMasker()
	rand.Seed(time.Now().UnixNano())

	for i := 0; i < 500; i++ {
		s := randomHighEntropyString(32)
		masked := m.MaskString("X: " + s)
		if !strings.Contains(masked, "REDACTED") {
			t.Fatalf("High entropy string should be redacted: %s", masked)
		}
	}
}

func randomHighEntropyString(n int) string {
	const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789!@#$%^&*"
	b := make([]byte, n)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}

func TestPIIMasker_MessageMasking(t *testing.T) {
	m := NewPIIMasker()

	tests := []struct {
		name, msg, expected string
	}{
		{"Email", "User john@example.com", "User [EMAIL]"},
		{"CreditCard", "Use 4111111111111111", "Use [CREDIT_CARD]"},
		{"SSN", "SSN: 123-45-6789", "SSN: [SSN]"},
		{"Phone", "Call (555) 123-4567", "Call [PHONE]"},
		{"IP", "IP: 192.168.1.1", "IP: [IP_ADDRESS]"},
		{"APIKey", "API key: sk-abcdef123456", "API key: ***API_KEY***"},
		{"Password", "Password: mySecret123", "Password: ***PASSWORD***"},
		{
			"Multiple",
			"john@example.com SSN 123-45-6789",
			"[EMAIL] SSN [SSN]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertEqual(t, tt.expected, m.MaskString(tt.msg))
		})
	}
}

func TestPIIMasker_FieldsMasking(t *testing.T) {
	m := NewPIIMasker()

	fields := []types.TypedFieldData{
		{Key: "email", Value: "user@example.com", Type: types.TypedFieldString},
		{Key: "ssn", Value: "123-45-6789", Type: types.TypedFieldString},
		{Key: "ip", Value: "192.168.1.1", Type: types.TypedFieldString},
		{Key: "non_pii", Value: "safe value", Type: types.TypedFieldString},
		{Key: "number", Value: 12345, Type: types.TypedFieldInt64},
		{Key: "bool_field", Value: true, Type: types.TypedFieldBool},
	}

	masked := m.MaskFields(fields)

	assertMaskedField(t, masked, "email", "[EMAIL]")
	assertMaskedField(t, masked, "ssn", "[SSN]")
	assertMaskedField(t, masked, "ip", "[IP_ADDRESS]")

	assertMaskedField(t, masked, "non_pii", "safe value")
	assertMaskedField(t, masked, "number", 12345)
	assertMaskedField(t, masked, "bool_field", true)
}

func TestPIIMasker_PatternManagement(t *testing.T) {
	m := NewPIIMasker()

	// Add
	err := m.AddPattern("custom", `\bCUSTOM_[A-Z]+\b`, "[CUSTOM]")
	if err != nil {
		t.Fatalf("AddPattern failed: %v", err)
	}

	assertEqual(t,
		"Found [CUSTOM] OK",
		m.MaskString("Found CUSTOM_SECRET OK"),
	)

	// Remove
	m.RemovePattern("custom")
	if strings.Contains(strings.Join(m.GetPatterns(), ","), "custom") {
		t.Fatal("custom pattern should be removed")
	}
}

func TestPIIMasker_Empty(t *testing.T) {
	m := NewPIIMasker()

	assertEqual(t, "", m.MaskString(""))
	assertEqual(t, "Hi", m.MaskString("Hi"))
}

func TestPIIMasker_Concurrent(t *testing.T) {
	m := NewPIIMasker()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			m.AddPattern(utils.FormatIntWithPrefix("P", id), `\bX\d+\b`, "[X]")
			m.MaskString(utils.FormatWithPrefixAndSuffix("Value X%", id, " abc@example.com"))
			m.RemovePattern(utils.FormatIntWithPrefix("P", id))
		}(i)
	}
	wg.Wait()
}

func TestPIIMasker_REDACTEDFunctionality(t *testing.T) {
	m := NewPIIMasker()

	// Test basic REDACTED functionality
	assertEqual(t,
		"Token: REDACTED",
		m.MaskString("Token: abc123def456ghi789"),
	)

	assertEqual(t,
		"Secret: REDACTED",
		m.MaskString("Secret: P@ssw0rd!123"),
	)

	// Test multiple heuristic tokens
	input := "Send Token: abc123def456 And Key: xyz999777aaa"
	expected := "Send Token: REDACTED And Key: REDACTED"
	assertEqual(t, expected, m.MaskString(input))

	// Test that already masked values are not double-masked
	assertEqual(t,
		"Token: REDACTED",
		m.MaskString("Token: REDACTED"),
	)

	// Test that safe values are not masked
	assertEqual(t,
		"Message: Hello World",
		m.MaskString("Message: Hello World"),
	)

	// Test empty string
	assertEqual(t, "", m.MaskString(""))

	// Test high entropy strings
	highEntropy := "a1B2c3D4e5F6g7H8i9J0k1L2m3N4o5P6q7R8s9T0"
	result := m.MaskString("Token: " + highEntropy)
	if !strings.Contains(result, "REDACTED") {
		t.Fatalf("High entropy string should be redacted: %s", result)
	}
}
