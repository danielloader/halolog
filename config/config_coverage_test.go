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
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-gen-ecosystem/halolog/types"
)

// TestConfigLoader_LoadFromYAML_ValidConfig tests loading valid YAML configuration
func TestConfigLoader_LoadFromYAML_ValidConfig(t *testing.T) {
	loader := NewConfigLoader("HALOLOG_")

	validYAML := `
loggers:
  - name: application-logger
    level: debug
    appenders:
      - type: console
        target: stdout
        layout:
          type: json
          show_timestamp: true
          show_level: true
          show_component: true
      - type: file
        target: /var/log/app.log
        layout:
          type: simple
          show_timestamp: true
          show_level: true
          show_file: true
  - name: error-logger
    level: error
    appenders:
      - type: file
        target: /var/log/errors.log
        layout:
          type: json
          show_timestamp: true
          show_level: true
          show_component: true
          show_file: true

security:
  mask_sensitive_data: true
  pii_patterns:
    - name: email
      pattern: "[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}"
    - name: ssn
      pattern: "\\d{3}-\\d{2}-\\d{4}"

performance:
  rate_limit: 1000
  buffer_size: 100
  flush_interval: 5s

environment:
  default_level: info
  production_level: warn
  development_level: debug
`

	config, err := loader.LoadFromYAML([]byte(validYAML))
	if err != nil {
		t.Fatalf("Failed to load valid YAML config: %v", err)
	}

	// Validate loggers
	if len(config.Loggers) != 2 {
		t.Errorf("Expected 2 loggers, got %d", len(config.Loggers))
	}

	// Validate first logger
	if config.Loggers[0].Name != "application-logger" {
		t.Errorf("Expected first logger name 'application-logger', got %s", config.Loggers[0].Name)
	}
	if config.Loggers[0].Level != types.DebugLevel {
		t.Errorf("Expected first logger level DEBUG, got %v", config.Loggers[0].Level)
	}
	if len(config.Loggers[0].Appenders) != 2 {
		t.Errorf("Expected first logger to have 2 appenders, got %d", len(config.Loggers[0].Appenders))
	}

	// Validate second logger
	if config.Loggers[1].Name != "error-logger" {
		t.Errorf("Expected second logger name 'error-logger', got %s", config.Loggers[1].Name)
	}
	if config.Loggers[1].Level != types.ErrorLevel {
		t.Errorf("Expected second logger level ERROR, got %v", config.Loggers[1].Level)
	}
	if len(config.Loggers[1].Appenders) != 1 {
		t.Errorf("Expected second logger to have 1 appender, got %d", len(config.Loggers[1].Appenders))
	}

	// Validate security config
	if !config.Security.MaskSensitiveData {
		t.Error("Expected Security.MaskSensitiveData to be true")
	}
	if len(config.Security.PIIPatterns) != 2 {
		t.Errorf("Expected 2 PII patterns, got %d", len(config.Security.PIIPatterns))
	}

	// Validate performance config
	if config.Performance.RateLimit != 1000 {
		t.Errorf("Expected Performance.RateLimit to be 1000, got %d", config.Performance.RateLimit)
	}
	if config.Performance.BufferSize != 100 {
		t.Errorf("Expected Performance.BufferSize to be 100, got %d", config.Performance.BufferSize)
	}
	if config.Performance.FlushInterval != 5*time.Second {
		t.Errorf("Expected Performance.FlushInterval to be 5s, got %v", config.Performance.FlushInterval)
	}

	// Validate environment config
	if config.EnvConfig.DefaultLevel != types.InfoLevel {
		t.Errorf("Expected Environment.DefaultLevel to be INFO, got %v", config.EnvConfig.DefaultLevel)
	}
	if config.EnvConfig.ProductionLevel != types.WarnLevel {
		t.Errorf("Expected Environment.ProductionLevel to be WARN, got %v", config.EnvConfig.ProductionLevel)
	}
	if config.EnvConfig.DevelopmentLevel != types.DebugLevel {
		t.Errorf("Expected Environment.DevelopmentLevel to be DEBUG, got %v", config.EnvConfig.DevelopmentLevel)
	}
}

// TestConfigLoader_LoadFromJSON_WithTempDir tests JSON loading with proper temp directory
func TestConfigLoader_LoadFromJSON_WithTempDir(t *testing.T) {
	loader := NewConfigLoader("HALOLOG_")

	// Create temp directory in user temp folder
	tempDir := filepath.Join(os.TempDir(), "halolog-test-json")
	os.MkdirAll(tempDir, 0755)
	defer os.RemoveAll(tempDir)

	jsonContent := `{
		"loggers": [
			{
				"name": "test-json-logger",
				"level": "debug",
				"appenders": [
					{
						"type": "file",
						"path": "/tmp/test.log"
					}
				]
			}
		],
		"environment": {
			"default_level": "INFO",
			"production_level": "WARN",
			"development_level": "DEBUG"
		},
		"performance": {
			"rate_limit": 1000,
			"buffer_size": 100,
			"flush_interval": "5s"
		}
	}`

	jsonFile := filepath.Join(tempDir, "test-config.json")
	err := os.WriteFile(jsonFile, []byte(jsonContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test JSON file: %v", err)
	}

	// Read the file content
	data, err := os.ReadFile(jsonFile)
	if err != nil {
		t.Fatalf("Failed to read JSON file: %v", err)
	}

	config, err := loader.LoadFromJSON(data)
	if err != nil {
		t.Fatalf("Failed to load JSON config: %v", err)
	}

	if len(config.Loggers) != 1 {
		t.Errorf("Expected 1 logger, got %d", len(config.Loggers))
	}

	if config.Loggers[0].Name != "test-json-logger" {
		t.Errorf("Expected logger name 'test-json-logger', got %s", config.Loggers[0].Name)
	}
}

// TestConfigLoader_LoadFromYAML_InvalidYAML tests invalid YAML handling
func TestConfigLoader_LoadFromYAML_InvalidYAML(t *testing.T) {
	loader := NewConfigLoader("HALOLOG_")

	invalidYAML := `
loggers:
  - name: test-logger
    level: invalid-level
    appenders:
      - type: console
        target: stdout
`

	_, err := loader.LoadFromYAML([]byte(invalidYAML))
	if err == nil {
		t.Error("Expected error for invalid YAML, got nil")
	}

	// Test with completely invalid YAML
	_, err = loader.LoadFromYAML([]byte("invalid yaml content"))
	if err == nil {
		t.Error("Expected error for completely invalid YAML, got nil")
	}
}

// TestConfigLoader_LoadFromJSON_InvalidJSON tests invalid JSON handling
func TestConfigLoader_LoadFromJSON_InvalidJSON(t *testing.T) {
	loader := NewConfigLoader("HALOLOG_")

	invalidJSON := `{
	"loggers": [{
		"name": "test-logger",
		"level": "invalid-level",
		"appenders": [{
			"type": "console",
			"target": "stdout"
		}]
	}]
}`

	_, err := loader.LoadFromJSON([]byte(invalidJSON))
	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}

	// Test with completely invalid JSON
	_, err = loader.LoadFromJSON([]byte("invalid json content"))
	if err == nil {
		t.Error("Expected error for completely invalid JSON, got nil")
	}
}

// TestHaloLogConfig_Basic tests basic HaloLogConfig functionality
func TestHaloLogConfig_Basic(t *testing.T) {
	// Test creating a basic config
	config := &HaloLogConfig{
		Loggers: []LoggerConfig{
			{
				Name:  "test-logger",
				Level: types.InfoLevel,
				Appenders: []AppenderConfig{
					{
						Type: "console",
					},
				},
			},
		},
		Security: SecurityConfig{
			MaskSensitiveData: true,
			PIIPatterns:       []PIIPattern{},
		},
		Performance: PerformanceConfig{
			RateLimit:     1000,
			BufferSize:    100,
			FlushInterval: 5 * time.Second,
		},
		EnvConfig: EnvironmentConfig{
			DefaultLevel:     types.InfoLevel,
			ProductionLevel:  types.WarnLevel,
			DevelopmentLevel: types.DebugLevel,
		},
	}

	// Basic validation
	if len(config.Loggers) != 1 {
		t.Errorf("Expected 1 logger, got %d", len(config.Loggers))
	}
	if config.Loggers[0].Name != "test-logger" {
		t.Errorf("Expected logger name 'test-logger', got %s", config.Loggers[0].Name)
	}
	if config.Loggers[0].Level != types.InfoLevel {
		t.Errorf("Expected logger level INFO, got %v", config.Loggers[0].Level)
	}
	if len(config.Loggers[0].Appenders) != 1 {
		t.Errorf("Expected 1 appender, got %d", len(config.Loggers[0].Appenders))
	}
	if config.Loggers[0].Appenders[0].Type != "console" {
		t.Errorf("Expected appender type 'console', got %s", config.Loggers[0].Appenders[0].Type)
	}
}

// TestConfigLoader_LoadFromFile tests loading from file with auto-detection
func TestConfigLoader_LoadFromFile(t *testing.T) {
	loader := NewConfigLoader("HALOLOG_")

	// Create temp directory in user temp folder
	tempDir := filepath.Join(os.TempDir(), "halolog-test-auto")
	os.MkdirAll(tempDir, 0755)
	defer os.RemoveAll(tempDir)

	// Test YAML file
	yamlContent := `
loggers:
  - name: auto-yaml-logger
    level: info
    appenders:
      - type: console
        target: stdout
environment:
  default_level: INFO
  production_level: WARN
  development_level: DEBUG
performance:
  rate_limit: 1000
  buffer_size: 100
  flush_interval: 5s
`
	yamlFile := filepath.Join(tempDir, "test-config.yaml")
	err := os.WriteFile(yamlFile, []byte(yamlContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create YAML file: %v", err)
	}

	config, err := loader.LoadFromFile(yamlFile)
	if err != nil {
		t.Fatalf("Failed to load YAML file: %v", err)
	}

	if len(config.Loggers) != 1 {
		t.Errorf("Expected 1 logger from YAML, got %d", len(config.Loggers))
	}

	// Test JSON file
	jsonContent := `{
		"loggers": [
			{
				"name": "auto-json-logger",
				"level": "debug",
				"appenders": [
					{
						"type": "file",
						"path": "/tmp/test.log"
					}
				]
			}
		],
		"environment": {
			"default_level": "INFO",
			"production_level": "WARN",
			"development_level": "DEBUG"
		},
		"performance": {
			"rate_limit": 1000,
			"buffer_size": 100,
			"flush_interval": "5s"
		}
	}`

	jsonFile := filepath.Join(tempDir, "test-config.json")
	err = os.WriteFile(jsonFile, []byte(jsonContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create JSON file: %v", err)
	}

	config, err = loader.LoadFromFile(jsonFile)
	if err != nil {
		t.Fatalf("Failed to load JSON file: %v", err)
	}

	if len(config.Loggers) != 1 {
		t.Errorf("Expected 1 logger from JSON, got %d", len(config.Loggers))
	}

	// Test unsupported file type
	unsupportedFile := filepath.Join(tempDir, "test-config.txt")
	err = os.WriteFile(unsupportedFile, []byte("unsupported"), 0644)
	if err != nil {
		t.Fatalf("Failed to create unsupported file: %v", err)
	}

	_, err = loader.LoadFromFile(unsupportedFile)
	if err == nil {
		t.Error("Expected error for unsupported file type")
	}
}
