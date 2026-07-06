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
// Package config provides configuration management
// Author: Admilson B. F. Cossa

package config

import (
	"testing"

	"github.com/go-gen-ecosystem/halolog/types"
)

func TestConfigBuilder_WithStartupRegistration(t *testing.T) {
	startupConfig := &StartupConfig{
		CustomFields: []string{"order_id", "customer_tier", "payment_method"},
		SensitiveFields: []SensitiveFieldConfig{
			{Name: "password", FieldID: 42, Mask: "***PASSWORD***"},
			{Name: "credit_card", FieldID: 156, Mask: "**** **** **** XXXX"},
		},
		FieldColors: map[string]types.Color{
			"error":         types.ColorRed,
			"order_id":      types.ColorCyan,
			"customer_tier": types.ColorYellow,
		},
		OptimalBufferSize: 32,
	}

	config := NewConfig().
		WithStartupRegistration(startupConfig).
		Build()

	if config.StartupRegistration == nil {
		t.Fatal("Expected StartupRegistration to be set")
	}

	if len(config.StartupRegistration.CustomFields) != 3 {
		t.Errorf("Expected 3 custom fields, got %d", len(config.StartupRegistration.CustomFields))
	}

	if len(config.StartupRegistration.SensitiveFields) != 2 {
		t.Errorf("Expected 2 sensitive fields, got %d", len(config.StartupRegistration.SensitiveFields))
	}

	if config.StartupRegistration.OptimalBufferSize != 32 {
		t.Errorf("Expected OptimalBufferSize 32, got %d", config.StartupRegistration.OptimalBufferSize)
	}
}

func TestConfigBuilder_WithCustomFields(t *testing.T) {
	customFields := []string{"order_id", "customer_tier", "payment_method", "session_id"}

	config := NewConfig().
		WithCustomFields(customFields).
		Build()

	if len(config.RegisteredFieldIDs) != 4 {
		t.Errorf("Expected 4 registered field IDs, got %d", len(config.RegisteredFieldIDs))
	}

	// Check that fields are properly mapped
	expectedFields := map[string]int{
		"order_id":       0,
		"customer_tier":  1,
		"payment_method": 2,
		"session_id":     3,
	}

	for fieldName, expectedID := range expectedFields {
		if id, exists := config.RegisteredFieldIDs[fieldName]; !exists || id != expectedID {
			t.Errorf("Expected field %s to have ID %d, got %d (exists: %v)", fieldName, expectedID, id, exists)
		}
	}
}

func TestConfigBuilder_WithFieldColors(t *testing.T) {
	fieldColors := map[string]types.Color{
		"error":   types.ColorRed,
		"warning": types.ColorYellow,
		"info":    types.ColorGreen,
		"debug":   types.ColorBlue,
	}

	config := NewConfig().
		WithFieldColors(fieldColors).
		Build()

	if len(config.FieldColorMap) != 4 {
		t.Errorf("Expected 4 field color mappings, got %d", len(config.FieldColorMap))
	}

	// Verify that each field name has a corresponding field ID and color
	for fieldName, expectedColor := range fieldColors {
		fieldID, exists := config.RegisteredFieldIDs[fieldName]
		if !exists {
			t.Errorf("Expected field %s to have a registered field ID", fieldName)
			continue
		}

		color, colorExists := config.FieldColorMap[fieldID]
		if !colorExists || color != expectedColor {
			t.Errorf("Expected field %s (ID %d) to have color %v, got %v (exists: %v)", fieldName, fieldID, expectedColor, color, colorExists)
		}
	}
}

func TestConfigBuilder_WithOptimalBufferSize(t *testing.T) {
	tests := []struct {
		name         string
		bufferSize   int
		expectedSize int
	}{
		{"Small buffer", 16, 16},
		{"Medium buffer", 32, 32},
		{"Large buffer", 64, 64},
		{"Default buffer", 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := NewConfig().
				WithOptimalBufferSize(tt.bufferSize).
				Build()

			if config.OptimalBufferSize != tt.expectedSize {
				t.Errorf("Expected buffer size %d, got %d", tt.expectedSize, config.OptimalBufferSize)
			}
		})
	}
}

func TestConfigBuilder_BuildWithStartupConfig(t *testing.T) {
	// Test building config with all startup registration features
	startupConfig := &StartupConfig{
		CustomFields: []string{"user_id", "request_id", "response_time"},
		SensitiveFields: []SensitiveFieldConfig{
			{Name: "api_key", FieldID: 100, Mask: "***API_KEY***"},
			{Name: "token", FieldID: 101, Mask: "***TOKEN***"},
		},
		FieldColors: map[string]types.Color{
			"error":   types.ColorRed,
			"success": types.ColorGreen,
		},
		OptimalBufferSize: 48,
	}

	builder := NewConfig().
		WithStartupRegistration(startupConfig).
		WithLevel(types.InfoLevel).
		WithColorized(true)

	config := builder.Build()

	// Verify all startup registration data is preserved
	if config.StartupRegistration == nil {
		t.Fatal("StartupRegistration should not be nil")
	}

	if len(config.RegisteredFieldIDs) != 5 {
		t.Errorf("Expected 5 registered field IDs (3 custom + 2 sensitive), got %d", len(config.RegisteredFieldIDs))
	}

	if len(config.SensitiveFieldMap) != 2 {
		t.Errorf("Expected 2 sensitive field mappings, got %d", len(config.SensitiveFieldMap))
	}

	if len(config.FieldColorMap) != 0 {
		t.Errorf("Expected 0 field color mappings (since 'error' and 'success' are not registered fields), got %d", len(config.FieldColorMap))
	}

	if config.OptimalBufferSize != 48 {
		t.Errorf("Expected OptimalBufferSize 48, got %d", config.OptimalBufferSize)
	}

	// Verify sensitive field mappings
	if mask, exists := config.SensitiveFieldMap[100]; !exists || mask != "***API_KEY***" {
		t.Errorf("Expected api_key (ID 100) to have mask '***API_KEY***', got '%s' (exists: %v)", mask, exists)
	}

	if mask, exists := config.SensitiveFieldMap[101]; !exists || mask != "***TOKEN***" {
		t.Errorf("Expected token (ID 101) to have mask '***TOKEN***', got '%s' (exists: %v)", mask, exists)
	}
}

func TestSensitiveFieldConfig_Validation(t *testing.T) {
	tests := []struct {
		name        string
		config      SensitiveFieldConfig
		expectValid bool
	}{
		{
			name: "Valid sensitive field",
			config: SensitiveFieldConfig{
				Name:    "password",
				FieldID: 42,
				Mask:    "***PASSWORD***",
			},
			expectValid: true,
		},
		{
			name: "Missing field ID",
			config: SensitiveFieldConfig{
				Name: "password",
				Mask: "***PASSWORD***",
			},
			expectValid: true, // Field ID is optional (will be auto-assigned)
		},
		{
			name: "Empty mask",
			config: SensitiveFieldConfig{
				Name:    "password",
				FieldID: 42,
				Mask:    "",
			},
			expectValid: true, // Empty mask is valid (field will be hidden)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that the config can be used without panicking
			startupConfig := &StartupConfig{
				SensitiveFields: []SensitiveFieldConfig{tt.config},
			}

			builder := NewConfig().WithStartupRegistration(startupConfig)
			config := builder.Build()

			if config.StartupRegistration == nil {
				t.Fatal("StartupRegistration should not be nil")
			}
		})
	}
}
