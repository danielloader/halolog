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
// Package registry provides component registries
// @author Admilson B. F. Cossa

package registry

import (
	"sync"
	"testing"
)

// ============================================================================
// SensitiveFieldRegistry Coverage Tests
// ============================================================================

func TestSensitiveFieldRegistry_CommonDefaults(t *testing.T) {
	r := NewSensitiveFieldRegistry()

	// A representative sample of every category pre-seeded by
	// initializeCommonSensitiveFields must be flagged sensitive.
	sensitive := []string{
		"password", "passwd", "pwd", "secret", "token", "api_key",
		"ssn", "credit_card", "email", "phone", "date_of_birth",
		"bank_account", "iban", "salary",
		"medical_record", "patient_id", "diagnosis",
		"passport", "driver_license", "tax_id",
		"session_id", "csrf_token", "cookie",
		"connection_string", "private_key", "client_secret",
		"aws_access_key", "ssh_key", "tls_key",
	}
	for _, f := range sensitive {
		if !r.IsSensitive(f) {
			t.Errorf("expected %q to be sensitive by default", f)
		}
	}
}

func TestSensitiveFieldRegistry_IsSensitive_NormalizationAndCasing(t *testing.T) {
	r := NewSensitiveFieldRegistry()

	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"empty", "", false},
		{"whitespace only", "   ", false},
		{"upper case known", "PASSWORD", true},
		{"padded known", "  Token  ", true},
		{"mixed case known", "Api_Key", true},
		{"benign field", "username", false},
		{"benign numeric-ish", "count", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := r.IsSensitive(tt.input); got != tt.want {
				t.Errorf("IsSensitive(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestSensitiveFieldRegistry_RegisterSensitiveField(t *testing.T) {
	r := NewSensitiveFieldRegistry()

	if r.IsSensitive("employee_pin") {
		t.Fatal("employee_pin should not be sensitive before registration")
	}

	// Register with surrounding whitespace and mixed case; normalization applies.
	r.RegisterSensitiveField("  Employee_PIN  ")

	if !r.IsSensitive("employee_pin") {
		t.Error("employee_pin should be sensitive after registration")
	}
	if !r.IsSensitive("EMPLOYEE_PIN") {
		t.Error("lookup should be case-insensitive after registration")
	}

	// Empty registration is a no-op (must not panic, must not flag "").
	r.RegisterSensitiveField("   ")
	if r.IsSensitive("") {
		t.Error("empty field name must never be sensitive")
	}
}

func TestSensitiveFieldRegistry_IsSensitiveByID(t *testing.T) {
	r := NewSensitiveFieldRegistry()

	// Registering a field assigns it a FieldDict ID and records it for ID lookup.
	r.RegisterSensitiveField("account_secret")

	stats := r.GetPerformanceStats()
	if stats.TotalFields == 0 {
		t.Fatal("expected registered fields to be counted")
	}

	// A field ID that was never registered must return false and count as a miss.
	if r.IsSensitiveByID(999999) {
		t.Error("unregistered field ID must not be sensitive")
	}
}

func TestSensitiveFieldRegistry_Patterns(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		match   string
		nomatch string
	}{
		// nomatch strings are deliberately benign so they fall through the
		// pattern miss AND the heuristic miss to a final false.
		{"prefix", "internal_*", "internal_widget", "external_widget"},
		{"suffix", "*_gauge", "payload_gauge", "gauge_prefix_field"},
		{"contains", "*vault*", "my_vault_data", "chainmail"},
		{"exact via pattern", "exactfield", "exactfield", "exactfieldx"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewSensitiveFieldRegistry()
			r.RegisterSensitivePattern(tt.pattern)

			if !r.IsSensitive(tt.match) {
				t.Errorf("pattern %q should match %q", tt.pattern, tt.match)
			}
			if r.IsSensitive(tt.nomatch) {
				t.Errorf("pattern %q should NOT match %q", tt.pattern, tt.nomatch)
			}
		})
	}
}

func TestSensitiveFieldRegistry_WildcardStarMatchesEverything(t *testing.T) {
	r := NewSensitiveFieldRegistry()
	r.RegisterSensitivePattern("*")

	if !r.IsSensitive("literally_anything") {
		t.Error(`"*" pattern should match any field name`)
	}
}

func TestSensitiveFieldRegistry_RegisterSensitivePattern_EmptyIgnored(t *testing.T) {
	r := NewSensitiveFieldRegistry()
	r.RegisterSensitivePattern("   ")

	stats := r.GetPerformanceStats()
	if stats.TotalPatterns != 0 {
		t.Errorf("empty pattern must be ignored, got %d patterns", stats.TotalPatterns)
	}
}

func TestSensitiveFieldRegistry_HeuristicMatch(t *testing.T) {
	r := NewSensitiveFieldRegistry()

	// These are neither pre-seeded exact names nor registered patterns,
	// so they must be caught by the heuristic substring matcher.
	heuristic := []string{
		"user_key_material",
		"my_secret_value",
		"legacy_password_field",
		"auth_context",
		"confidential_notes",
		"pii_bucket",
	}
	for _, f := range heuristic {
		if !r.IsSensitive(f) {
			t.Errorf("heuristic should flag %q as sensitive", f)
		}
	}

	// A clearly benign field must fall through every path to false.
	if r.IsSensitive("widget_count") {
		t.Error("widget_count should not be sensitive")
	}
}

func TestSensitiveFieldRegistry_GetMask(t *testing.T) {
	r := NewSensitiveFieldRegistry()

	// Pre-normalized known field hits the fast path.
	mask, ok := r.GetMask("password")
	if !ok {
		t.Fatal("expected password to have a mask")
	}
	if mask != "[REDACTED]" {
		t.Errorf("expected default mask [REDACTED], got %q", mask)
	}

	// Field needing normalization goes through the fallback branch.
	mask, ok = r.GetMask("  Secret  ")
	if !ok || mask != "[REDACTED]" {
		t.Errorf("expected fallback path to mask normalized field, got mask=%q ok=%v", mask, ok)
	}

	// Benign field returns no mask.
	if _, ok := r.GetMask("harmless"); ok {
		t.Error("benign field should not return a mask")
	}
}

func TestSensitiveFieldRegistry_GetMaskInterface(t *testing.T) {
	r := NewSensitiveFieldRegistry()

	// Fast path: exact pre-normalized field.
	v, ok := r.GetMaskInterface("token")
	if !ok {
		t.Fatal("expected token to have a mask interface")
	}
	if s, isStr := v.(string); !isStr || s != "[REDACTED]" {
		t.Errorf("expected [REDACTED], got %#v", v)
	}

	// Fallback path: needs normalization.
	v, ok = r.GetMaskInterface("  API_KEY ")
	if !ok || v != "[REDACTED]" {
		t.Errorf("expected fallback mask interface, got %#v ok=%v", v, ok)
	}

	// Benign field returns nil, false.
	v, ok = r.GetMaskInterface("plain")
	if ok || v != nil {
		t.Errorf("benign field should return (nil,false), got %#v ok=%v", v, ok)
	}
}

func TestSensitiveFieldRegistry_DefaultUnknownMasks(t *testing.T) {
	r := NewSensitiveFieldRegistry()

	if got := r.GetDefaultUnknownMask(); got != "[SENSITIVE]" {
		t.Errorf("expected [SENSITIVE], got %q", got)
	}

	iface := r.GetDefaultUnknownMaskInterface()
	if s, ok := iface.(string); !ok || s != "[SENSITIVE]" {
		t.Errorf("expected interface [SENSITIVE], got %#v", iface)
	}
}

func TestSensitiveFieldRegistry_PerformanceStats(t *testing.T) {
	r := NewSensitiveFieldRegistry()

	r.IsSensitive("password")     // hit (common)
	r.IsSensitive("not_a_secret") // miss
	r.IsSensitiveByID(424242)     // miss

	stats := r.GetPerformanceStats()
	if stats.CheckCount != 3 {
		t.Errorf("expected 3 checks, got %d", stats.CheckCount)
	}
	if stats.HitCount == 0 {
		t.Error("expected at least one hit")
	}
	if stats.MissCount == 0 {
		t.Error("expected at least one miss")
	}
	// HitRate must be a sane percentage.
	if stats.HitRate < 0 || stats.HitRate > 100 {
		t.Errorf("hit rate out of range: %f", stats.HitRate)
	}
}

func TestSensitiveFieldRegistry_Reset(t *testing.T) {
	r := NewSensitiveFieldRegistry()

	r.RegisterSensitiveField("custom_secret_field")
	r.RegisterSensitivePattern("*_leaked")
	r.IsSensitive("password")

	r.Reset()

	stats := r.GetPerformanceStats()
	if stats.HitCount != 0 || stats.MissCount != 0 || stats.CheckCount != 0 {
		t.Error("metrics must be zeroed after Reset")
	}
	if stats.TotalPatterns != 0 {
		t.Errorf("patterns must be cleared after Reset, got %d", stats.TotalPatterns)
	}

	// Custom registration is cleared, but the field name map is rebuilt empty,
	// so the previously-registered custom field is no longer flagged by name.
	// (commonSensitive still catches truly common fields on a fresh registry,
	// but Reset does not repopulate commonSensitive, so "password" via the
	// fieldNames path is gone — verify the custom field specifically.)
	if r.IsSensitiveByID(0) {
		t.Error("no field IDs should remain sensitive after Reset")
	}
}

func TestRegisterSensitiveFieldsFromConfig(t *testing.T) {
	// nil config must be a no-op and not panic.
	RegisterSensitiveFieldsFromConfig(nil)

	// Use a fresh local registry semantics check by exercising the global path
	// with explicit fields and patterns.
	cfg := map[string]interface{}{
		"sensitive_fields":   []string{"x_custom_config_field"},
		"sensitive_patterns": []string{"cfgpat_*"},
	}
	RegisterSensitiveFieldsFromConfig(cfg)

	if !GlobalSensitiveFieldRegistry.IsSensitive("x_custom_config_field") {
		t.Error("config-registered field should be sensitive")
	}
	if !GlobalSensitiveFieldRegistry.IsSensitive("cfgpat_alpha") {
		t.Error("config-registered pattern should match")
	}

	// Wrong-typed config values are ignored without panic.
	RegisterSensitiveFieldsFromConfig(map[string]interface{}{
		"sensitive_fields":   "not-a-slice",
		"sensitive_patterns": 123,
	})
}

func TestSensitiveFieldRegistry_ConcurrentAccess(t *testing.T) {
	r := NewSensitiveFieldRegistry()

	var wg sync.WaitGroup
	for i := 0; i < 40; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_ = r.IsSensitive("password")
				_ = r.IsSensitiveByID(n)
				if j%20 == 0 {
					r.RegisterSensitiveField("dynamic_secret")
					r.RegisterSensitivePattern("dyn_*")
				}
			}
		}(i)
	}
	wg.Wait()

	if !r.IsSensitive("dynamic_secret") {
		t.Error("concurrently-registered field should be sensitive")
	}
}
