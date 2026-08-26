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
// Author: Admilson B. F. Cossa

package registry

import (
	"strings"
	"sync"
	"sync/atomic"

	"github.com/go-gen-ecosystem/halolog/fielddict"
)

// SensitiveFieldRegistry provides O(1) lookup for sensitive field identification
// O(1) lookup for sensitive field identification.
type SensitiveFieldRegistry struct {
	mu sync.RWMutex

	// O(1) lookup structures
	fieldIDs      map[int]bool    // Field ID -> sensitive flag
	fieldNames    map[string]bool // Field name -> sensitive flag
	fieldPatterns []string        // Pattern-based matching (slower path)

	// Pre-computed lookup for common sensitive fields
	commonSensitive map[string]bool

	// Default masks
	defaultMask        string // Default mask for sensitive fields
	defaultUnknownMask string // Default mask for unknown sensitive fields

	// Pre-computed interface{} values for zero-allocation masking
	defaultMaskInterface        interface{} // Pre-computed interface{} for default mask
	defaultUnknownMaskInterface interface{} // Pre-computed interface{} for unknown mask

	// Performance metrics. Atomic: they are bumped on the read (query) path,
	// which holds at most an RLock — concurrent readers would race on plain
	// ints (caught by the race detector on the registry's own concurrency test).
	hitCount   atomic.Int64 // O(1) hits
	missCount  atomic.Int64 // Misses requiring pattern matching
	checkCount atomic.Int64 // Total checks performed
}

// NewSensitiveFieldRegistry creates a new registry with pre-configured sensitive fields
func NewSensitiveFieldRegistry() *SensitiveFieldRegistry {
	defaultMask := "[REDACTED]"
	defaultUnknownMask := "[SENSITIVE]"

	registry := &SensitiveFieldRegistry{
		fieldIDs:                    make(map[int]bool, 100),
		fieldNames:                  make(map[string]bool, 100),
		fieldPatterns:               make([]string, 0, 20),
		commonSensitive:             make(map[string]bool),
		defaultMask:                 defaultMask,
		defaultUnknownMask:          defaultUnknownMask,
		defaultMaskInterface:        defaultMask,        // Pre-computed for zero-allocation
		defaultUnknownMaskInterface: defaultUnknownMask, // Pre-computed for zero-allocation
	}

	// Pre-configure common sensitive fields (Secure defaults)
	registry.initializeCommonSensitiveFields()

	return registry
}

// initializeCommonSensitiveFields pre-configures common sensitive field patterns
// Based on industry standard security requirements and GDPR compliance
func (r *SensitiveFieldRegistry) initializeCommonSensitiveFields() {
	commonFields := []string{
		// Authentication & Identity
		"password", "passwd", "pwd", "secret", "token", "api_key", "apikey",
		"access_token", "refresh_token", "auth_token", "bearer_token",

		// Personal Information (GDPR)
		"ssn", "social_security", "social_security_number",
		"credit_card", "cc_number", "card_number", "pan",
		"email", "phone", "mobile", "address", "zipcode", "zip_code",
		"date_of_birth", "dob", "birthdate", "birth_date",

		// Financial
		"bank_account", "account_number", "routing_number",
		"iban", "swift", "bic", "sort_code",
		"salary", "income", "compensation", "payment_info",

		// Health (HIPAA)
		"medical_record", "patient_id", "health_id", "mrn",
		"diagnosis", "treatment", "prescription", "medication",

		// Government
		"passport", "passport_number", "driver_license", "license_number",
		"tax_id", "taxpayer_id", "ein", "vat_number",

		// Session & Security
		"session_id", "session_token", "csrf_token", "xsrf_token",
		"cookie", "session_cookie", "auth_cookie",

		// Database
		"connection_string", "db_password", "database_password",
		"private_key", "privatekey", "priv_key", "client_secret",

		// Cloud & Infrastructure
		"aws_access_key", "aws_secret_key", "azure_key", "gcp_key",
		"ssh_key", "ssh_private_key", "ssl_key", "tls_key",
	}

	for _, field := range commonFields {
		r.commonSensitive[field] = true
		r.fieldNames[field] = true // Add to O(1) lookup table
	}
}

// RegisterSensitiveField registers a field as sensitive for O(1) lookup
// This is the fast path for exact field name matching
func (r *SensitiveFieldRegistry) RegisterSensitiveField(fieldName string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Normalize field name
	fieldName = strings.ToLower(strings.TrimSpace(fieldName))
	if fieldName == "" {
		return
	}

	// Add to fast lookup table
	r.fieldNames[fieldName] = true

	// Register field in FieldDict and get ID for O(1) lookup
	fieldID := fielddict.GlobalFieldDictionary.GetOrRegisterFieldID(fieldName)
	r.fieldIDs[fieldID] = true
}

// RegisterSensitivePattern registers a pattern for sensitive field matching
// This is slower than exact matching but supports wildcards and patterns
func (r *SensitiveFieldRegistry) RegisterSensitivePattern(pattern string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	pattern = strings.ToLower(strings.TrimSpace(pattern))
	if pattern == "" {
		return
	}

	r.fieldPatterns = append(r.fieldPatterns, pattern)
}

// IsSensitive checks if a field is sensitive using O(1) lookup
// Returns true if field should be masked, false otherwise
func (r *SensitiveFieldRegistry) IsSensitive(fieldName string) bool {
	r.checkCount.Add(1)

	// Normalize input
	fieldName = strings.ToLower(strings.TrimSpace(fieldName))
	if fieldName == "" {
		return false
	}

	// Fast path: O(1) exact name lookup
	r.mu.RLock()
	if sensitive, exists := r.fieldNames[fieldName]; exists {
		r.mu.RUnlock()
		r.hitCount.Add(1)
		return sensitive
	}
	r.mu.RUnlock()

	// Fast path: Check common sensitive fields
	if r.commonSensitive[fieldName] {
		r.hitCount.Add(1)
		return true
	}

	// Medium path: Pattern matching for registered patterns
	if r.patternMatch(fieldName) {
		r.hitCount.Add(1)
		return true
	}

	// Slow path: Heuristic detection for unregistered patterns
	if r.heuristicMatch(fieldName) {
		r.missCount.Add(1)
		return true
	}

	r.missCount.Add(1)
	return false
}

// IsSensitiveByID checks if a field is sensitive by its FieldDict ID
// This is the fastest possible lookup: O(1) integer key lookup
func (r *SensitiveFieldRegistry) IsSensitiveByID(fieldID int) bool {
	r.checkCount.Add(1)

	// Integer lookup path
	r.mu.RLock()
	if sensitive, exists := r.fieldIDs[fieldID]; exists {
		r.mu.RUnlock()
		r.hitCount.Add(1)
		return sensitive
	}
	r.mu.RUnlock()

	r.missCount.Add(1)
	return false
}

// patternMatch checks if field matches any registered patterns
func (r *SensitiveFieldRegistry) patternMatch(fieldName string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, pattern := range r.fieldPatterns {
		if r.matchPattern(fieldName, pattern) {
			return true
		}
	}

	return false
}

// matchPattern performs pattern matching (simplified implementation)
// Supports basic wildcards: * for any characters
func (r *SensitiveFieldRegistry) matchPattern(fieldName, pattern string) bool {
	// Simple wildcard matching - can be enhanced with regex if needed
	if pattern == "*" {
		return true
	}

	if strings.Contains(pattern, "*") {
		// Basic wildcard matching
		parts := strings.Split(pattern, "*")
		if len(parts) == 2 && parts[0] == "" && parts[1] == "" {
			return true // "*" pattern
		}

		// Prefix matching: "password_*"
		if len(parts) == 2 && parts[1] == "" {
			return strings.HasPrefix(fieldName, parts[0])
		}

		// Suffix matching: "*_token"
		if len(parts) == 2 && parts[0] == "" {
			return strings.HasSuffix(fieldName, parts[1])
		}

		// Contains matching: "*password*"
		if len(parts) == 3 && parts[0] == "" && parts[2] == "" {
			return strings.Contains(fieldName, parts[1])
		}
	}

	return fieldName == pattern
}

// heuristicMatch uses heuristics to detect sensitive patterns
// Used for unregistered but potentially sensitive fields
func (r *SensitiveFieldRegistry) heuristicMatch(fieldName string) bool {
	// Common patterns that indicate sensitive data
	sensitivePatterns := []string{
		"key", "secret", "password", "token", "auth", "credential",
		"private", "confidential", "sensitive", "personal", "pii",
	}

	for _, pattern := range sensitivePatterns {
		if strings.Contains(fieldName, pattern) {
			return true
		}
	}

	return false
}

// GetPerformanceStats returns registry performance statistics
func (r *SensitiveFieldRegistry) GetPerformanceStats() SensitiveFieldStats {
	r.mu.RLock()
	defer r.mu.RUnlock()

	hits := r.hitCount.Load()
	checks := r.checkCount.Load()
	hitRate := 0.0
	if checks > 0 {
		hitRate = float64(hits) / float64(checks) * 100
	}
	return SensitiveFieldStats{
		TotalFields:   len(r.fieldNames),
		TotalPatterns: len(r.fieldPatterns),
		HitCount:      hits,
		MissCount:     r.missCount.Load(),
		CheckCount:    checks,
		HitRate:       hitRate,
	}
}

// Reset clears all registered sensitive fields and patterns
func (r *SensitiveFieldRegistry) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.fieldIDs = make(map[int]bool, 100)
	r.fieldNames = make(map[string]bool, 100)
	r.fieldPatterns = make([]string, 0, 20)

	// Reset metrics
	r.hitCount.Store(0)
	r.missCount.Store(0)
	r.checkCount.Store(0)
}

// GetMask returns the mask for a sensitive field
func (r *SensitiveFieldRegistry) GetMask(fieldName string) (string, bool) {
	// Fast path: Check if field is already normalized (common case)
	// This avoids string allocation from ToLower/TrimSpace for pre-normalized fields
	r.mu.RLock()
	if sensitive, exists := r.fieldNames[fieldName]; exists && sensitive {
		r.mu.RUnlock()
		return r.defaultMask, true
	}
	r.mu.RUnlock()

	// Fallback to full IsSensitive check with normalization
	if r.IsSensitive(fieldName) {
		return r.defaultMask, true
	}
	return "", false
}

// GetDefaultUnknownMask returns the default mask for unknown sensitive fields
func (r *SensitiveFieldRegistry) GetDefaultUnknownMask() string {
	return r.defaultUnknownMask
}

// GetMaskInterface returns the mask for a sensitive field as pre-computed interface{}
// This eliminates the 16 B allocation when assigning to interface{} fields
func (r *SensitiveFieldRegistry) GetMaskInterface(fieldName string) (interface{}, bool) {
	// Fast path: Check if field is already normalized (common case)
	// This avoids string allocation from ToLower/TrimSpace for pre-normalized fields
	r.mu.RLock()
	if sensitive, exists := r.fieldNames[fieldName]; exists && sensitive {
		r.mu.RUnlock()
		return r.defaultMaskInterface, true
	}
	r.mu.RUnlock()

	// Fallback to full IsSensitive check with normalization
	if r.IsSensitive(fieldName) {
		return r.defaultMaskInterface, true
	}
	return nil, false
}

// GetDefaultUnknownMaskInterface returns the pre-computed interface{} for unknown sensitive fields
// This eliminates the 16 B allocation when assigning to interface{} fields
func (r *SensitiveFieldRegistry) GetDefaultUnknownMaskInterface() interface{} {
	return r.defaultUnknownMaskInterface
}

// SensitiveFieldStats contains performance statistics for the registry
type SensitiveFieldStats struct {
	TotalFields   int
	TotalPatterns int
	HitCount      int64
	MissCount     int64
	CheckCount    int64
	HitRate       float64
}

// GlobalSensitiveFieldRegistry provides a singleton instance for system-wide use
var GlobalSensitiveFieldRegistry = NewSensitiveFieldRegistry()

// RegisterSensitiveFieldsFromConfig registers sensitive fields from configuration
// This is called at startup to pre-populate the registry
func RegisterSensitiveFieldsFromConfig(config map[string]interface{}) {
	if config == nil {
		return
	}

	// Register explicit field names
	if fields, ok := config["sensitive_fields"].([]string); ok {
		for _, field := range fields {
			GlobalSensitiveFieldRegistry.RegisterSensitiveField(field)
		}
	}

	// Register patterns
	if patterns, ok := config["sensitive_patterns"].([]string); ok {
		for _, pattern := range patterns {
			GlobalSensitiveFieldRegistry.RegisterSensitivePattern(pattern)
		}
	}
}
