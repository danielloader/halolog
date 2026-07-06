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
	"sync"
)

// SensitiveFieldConfig represents a sensitive field configuration for O(1) lookup
type SensitiveFieldConfig struct {
	Name    string `yaml:"name" json:"name"`
	FieldID int    `yaml:"field_id" json:"field_id"` // Pre-computed FieldDict ID for O(1) lookup
	Mask    string `yaml:"mask" json:"mask"`
}

// SensitiveFieldRegistry provides O(1) lookup for sensitive field masking
// Designed for zero-allocation hot path operations
type SensitiveFieldRegistry struct {
	mu sync.RWMutex

	// O(1) lookup: fieldName -> maskValue
	sensitiveFields map[string]string

	// O(1) lookup: fieldID -> maskValue (NEW - for High-performance masking)
	sensitiveFieldsByID map[int]string

	// Pre-computed field IDs for integration with field dictionary
	fieldIDs map[string]int

	// Default mask for unknown sensitive fields (pre-allocated to avoid allocations)
	defaultUnknownMask string

	// Pre-computed common mask values to avoid string allocations
	precomputedMasks map[string]string
}

// NewSensitiveFieldRegistry creates a new registry with default sensitive fields
func NewSensitiveFieldRegistry() *SensitiveFieldRegistry {
	sfr := &SensitiveFieldRegistry{
		sensitiveFields:     make(map[string]string),
		sensitiveFieldsByID: make(map[int]string),
		fieldIDs:            make(map[string]int),
		defaultUnknownMask:  "***REDACTED***",
		precomputedMasks:    make(map[string]string),
	}

	// Pre-populate with common sensitive fields
	sfr.registerDefaultSensitiveFields()

	return sfr
}

// registerDefaultSensitiveFields registers common sensitive fields
func (sfr *SensitiveFieldRegistry) registerDefaultSensitiveFields() {
	// Security fields
	sfr.RegisterSensitiveField("password", "***PASSWORD***")
	sfr.RegisterSensitiveField("passwd", "***PASSWORD***")
	sfr.RegisterSensitiveField("pwd", "***PASSWORD***")
	sfr.RegisterSensitiveField("pass", "***PASSWORD***")
	sfr.RegisterSensitiveField("secret", "***SECRET***")
	sfr.RegisterSensitiveField("token", "***TOKEN***")
	sfr.RegisterSensitiveField("api_key", "***API_KEY***")
	sfr.RegisterSensitiveField("apikey", "***API_KEY***")
	sfr.RegisterSensitiveField("jwt", "***JWT***")
	sfr.RegisterSensitiveField("hash", "***HASH***")

	// Personal identification
	sfr.RegisterSensitiveField("email", "***EMAIL***")
	sfr.RegisterSensitiveField("ssn", "***SSN***")
	sfr.RegisterSensitiveField("social_security", "***SSN***")
	sfr.RegisterSensitiveField("phone", "***PHONE***")
	sfr.RegisterSensitiveField("mobile", "***PHONE***")
	sfr.RegisterSensitiveField("credit_card", "***CREDIT_CARD***")
	sfr.RegisterSensitiveField("card_number", "***CREDIT_CARD***")

	// Authentication
	sfr.RegisterSensitiveField("auth", "***AUTH***")
	sfr.RegisterSensitiveField("authorization", "***AUTH***")
	sfr.RegisterSensitiveField("cookie", "***COOKIE***")
	sfr.RegisterSensitiveField("session", "***SESSION***")
	sfr.RegisterSensitiveField("session_id", "***SESSION***")

	// Financial
	sfr.RegisterSensitiveField("bank_account", "***BANK_ACCOUNT***")
	sfr.RegisterSensitiveField("routing_number", "***ROUTING_NUMBER***")
	sfr.RegisterSensitiveField("pin", "***PIN***")

	// System
	sfr.RegisterSensitiveField("private_key", "***PRIVATE_KEY***")
	sfr.RegisterSensitiveField("public_key", "***PUBLIC_KEY***")
	sfr.RegisterSensitiveField("certificate", "***CERTIFICATE***")
}

// RegisterSensitiveField registers a new sensitive field with its mask value
func (sfr *SensitiveFieldRegistry) RegisterSensitiveField(fieldName, maskValue string) {
	sfr.mu.Lock()
	defer sfr.mu.Unlock()

	// Store the field-mask mapping
	sfr.sensitiveFields[fieldName] = maskValue

	// Store in precomputed masks to avoid future allocations
	sfr.precomputedMasks[fieldName] = maskValue
}

// GetMask returns the mask value for a field name (O(1) lookup, zero allocation)
func (sfr *SensitiveFieldRegistry) GetMask(fieldName string) (string, bool) {
	// Fast path: read lock only
	sfr.mu.RLock()
	mask, exists := sfr.sensitiveFields[fieldName]
	sfr.mu.RUnlock()

	return mask, exists
}

// GetDefaultUnknownMask returns the default mask for unknown sensitive fields
// This is pre-allocated to avoid allocations in hot path
func (sfr *SensitiveFieldRegistry) GetDefaultUnknownMask() string {
	return sfr.defaultUnknownMask
}

// SetDefaultUnknownMask sets the default mask for unknown sensitive fields
func (sfr *SensitiveFieldRegistry) SetDefaultUnknownMask(mask string) {
	sfr.mu.Lock()
	defer sfr.mu.Unlock()
	sfr.defaultUnknownMask = mask
}

// IsSensitiveField checks if a field is sensitive (O(1) lookup)
func (sfr *SensitiveFieldRegistry) IsSensitiveField(fieldName string) bool {
	sfr.mu.RLock()
	_, exists := sfr.sensitiveFields[fieldName]
	sfr.mu.RUnlock()

	return exists
}

// GetAllSensitiveFields returns all registered sensitive field names
func (sfr *SensitiveFieldRegistry) GetAllSensitiveFields() []string {
	sfr.mu.RLock()
	defer sfr.mu.RUnlock()

	fields := make([]string, 0, len(sfr.sensitiveFields))
	for field := range sfr.sensitiveFields {
		fields = append(fields, field)
	}
	return fields
}

// GetAllMasks returns all field-mask mappings
func (sfr *SensitiveFieldRegistry) GetAllMasks() map[string]string {
	sfr.mu.RLock()
	defer sfr.mu.RUnlock()

	result := make(map[string]string, len(sfr.sensitiveFields))
	for field, mask := range sfr.sensitiveFields {
		result[field] = mask
	}
	return result
}

// Size returns the number of registered sensitive fields
func (sfr *SensitiveFieldRegistry) Size() int {
	sfr.mu.RLock()
	defer sfr.mu.RUnlock()

	return len(sfr.sensitiveFields)
}

// RegisterFromConfig registers sensitive fields from startup configuration
// This enables O(1) field ID-based lookups for High-performance masking
func (sfr *SensitiveFieldRegistry) RegisterFromConfig(sensitiveFields []SensitiveFieldConfig) error {
	sfr.mu.Lock()
	defer sfr.mu.Unlock()

	for _, sf := range sensitiveFields {
		// Register by field name
		sfr.sensitiveFields[sf.Name] = sf.Mask
		sfr.precomputedMasks[sf.Name] = sf.Mask

		// Register by field ID for O(1) lookup
		if sf.FieldID > 0 {
			sfr.fieldIDs[sf.Name] = sf.FieldID
			sfr.sensitiveFieldsByID[sf.FieldID] = sf.Mask
		}
	}

	return nil
}

// RegisterWithFieldID registers a sensitive field with pre-computed field ID
// This enables O(1) field ID-based lookups
func (sfr *SensitiveFieldRegistry) RegisterWithFieldID(fieldName string, fieldID int, maskValue string) {
	sfr.mu.Lock()
	defer sfr.mu.Unlock()

	// Register by field name
	sfr.sensitiveFields[fieldName] = maskValue
	sfr.precomputedMasks[fieldName] = maskValue

	// Register by field ID for O(1) lookup
	sfr.fieldIDs[fieldName] = fieldID
	sfr.sensitiveFieldsByID[fieldID] = maskValue
}

// GetMaskByFieldID returns mask value using field ID (O(1), zero allocation)
// This is the High-performance path for pre-registered sensitive fields
func (sfr *SensitiveFieldRegistry) GetMaskByFieldID(fieldID int) (string, bool) {
	sfr.mu.RLock()
	mask, exists := sfr.sensitiveFieldsByID[fieldID]
	sfr.mu.RUnlock()

	return mask, exists
}

// IsSensitiveFieldByID checks if a field ID is sensitive (O(1))
// This is the High-performance path for pre-registered sensitive fields
func (sfr *SensitiveFieldRegistry) IsSensitiveFieldByID(fieldID int) bool {
	sfr.mu.RLock()
	_, exists := sfr.sensitiveFieldsByID[fieldID]
	sfr.mu.RUnlock()

	return exists
}
