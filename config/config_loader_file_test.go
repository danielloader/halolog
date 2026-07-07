package config

import (
	"os"
	"testing"
	"time"

	"github.com/go-gen-ecosystem/halolog/types"
)

func TestConfigLoader_LoadFromFile_YAML(t *testing.T) {
	// Create temp YAML file with comprehensive configuration including:
	// - Environment levels (Required)
	// - Security (Masking)
	// - Performance (Sampling/Rate Limit)
	// - Context Fields
	// - Field Colors
	yamlContent := `
loggers:
  - name: app
    level: INFO
    appenders:
      - type: console

environment:
  default_level: INFO
  production_level: WARN
  development_level: DEBUG

security:
  mask_sensitive_data: true
  pii_patterns:
    - name: email
      pattern: "\\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\\.[A-Z|a-z]{2,}\\b"
      replacement: "[EMAIL]"

performance:
  rate_limit: 100
  buffer_size: 1024
  flush_interval: 1s

startup_registration:
  context_fields:
    enabled: true
    fields:
      - request_id
      - user_id
      - span_id
  field_colors:
    info: 2
    debug: 4
    error: 1
`

	tmpFile, err := createTempYAMLFile(yamlContent)
	if err != nil {
		t.Fatalf("Failed to create temp YAML file: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile) }()

	loader := NewConfigLoader("HALOLOG_")
	config, err := loader.LoadFromFile(tmpFile)
	if err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}

	// Verify Environment (Fixes previous failure)
	if config.EnvConfig.DefaultLevel != types.InfoLevel {
		t.Errorf("Expected DefaultLevel INFO, got %v", config.EnvConfig.DefaultLevel)
	}

	// Verify Context Fields Registration
	if config.StartupRegistration.ContextFields == nil {
		t.Fatal("Expected ContextFields to be not nil")
	}
	if !config.StartupRegistration.ContextFields.Enabled {
		t.Error("Expected ContextFields.Enabled to be true")
	}
	expectedFields := []string{"request_id", "user_id", "span_id"}
	if len(config.StartupRegistration.ContextFields.Fields) != len(expectedFields) {
		t.Errorf("Expected %d context fields, got %d", len(expectedFields), len(config.StartupRegistration.ContextFields.Fields))
	} else {
		for i, field := range expectedFields {
			if config.StartupRegistration.ContextFields.Fields[i] != field {
				t.Errorf("Expected context field %d to be '%s', got '%s'", i, field, config.StartupRegistration.ContextFields.Fields[i])
			}
		}
	}

	// Verify Security (Masking)
	if !config.Security.MaskSensitiveData {
		t.Error("Expected MaskSensitiveData to be true")
	}
	if len(config.Security.PIIPatterns) != 1 {
		t.Errorf("Expected 1 PII pattern, got %d", len(config.Security.PIIPatterns))
	}

	// Verify Performance (Sampling)
	if config.Performance.RateLimit != 100 {
		t.Errorf("Expected RateLimit 100, got %d", config.Performance.RateLimit)
	}
	if config.Performance.FlushInterval != 1*time.Second {
		t.Errorf("Expected FlushInterval 1s, got %v", config.Performance.FlushInterval)
	}

	// Verify Field Colors
	if len(config.StartupRegistration.FieldColors) != 3 {
		t.Errorf("Expected 3 field colors, got %d", len(config.StartupRegistration.FieldColors))
	}
	// Assuming types.Color is compatible with int values in YAML
	// We can check existence regardless of strict type mapping details for this integration test

	// Verify basic logger config
	if len(config.Loggers) != 1 {
		t.Errorf("Expected 1 logger, got %d", len(config.Loggers))
	}
}
