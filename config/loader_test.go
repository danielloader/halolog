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
	"strings"
	"testing"
	"time"

	"github.com/go-gen-ecosystem/halolog/types"
)

func TestConfigLoader_LoadFromYAML(t *testing.T) {
	// Create temp YAML file
	yamlContent := `
loggers:
  - name: app
    level: DEBUG
    appenders:
      - type: console
      - type: file
        path: logs/app.log
        rotation:
          max_size: 10485760
          max_backups: 5
          max_age: 30
          compress: true
  - name: error
    level: ERROR
    appenders:
      - type: file
        path: logs/error.log

security:
  pii_patterns:
    - name: email
      pattern: "\\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\\.[A-Z|a-z]{2,}\\b"
      replacement: "[EMAIL]"
  mask_sensitive_data: true

performance:
  rate_limit: 1000
  buffer_size: 100
  flush_interval: 5s

environment:
  default_level: INFO
  production_level: WARN
  development_level: DEBUG
`

	loader := NewConfigLoader("HALOLOG_")
	config, err := loader.LoadFromYAML([]byte(yamlContent))
	if err != nil {
		t.Fatalf("LoadFromYAML failed: %v", err)
	}

	// Debug: Print the actual config structure
	t.Logf("Parsed config: %+v", config)
	t.Logf("Loggers: %+v", config.Loggers)
	t.Logf("First logger: %+v", config.Loggers[0])

	// Verify configuration structure
	if len(config.Loggers) != 2 {
		t.Errorf("Expected 2 loggers, got %d", len(config.Loggers))
	}

	// Verify first logger
	appLogger := config.Loggers[0]
	if appLogger.Name != "app" {
		t.Errorf("Expected logger name 'app', got '%s'", appLogger.Name)
	}
	if appLogger.Level != types.DebugLevel {
		t.Errorf("Expected level DEBUG, got %v", appLogger.Level)
	}
	if len(appLogger.Appenders) != 2 {
		t.Errorf("Expected 2 appenders for app logger, got %d", len(appLogger.Appenders))
	}

	// Verify console appender
	consoleAppender := appLogger.Appenders[0]
	if consoleAppender.Type != "console" {
		t.Errorf("Expected appender type 'console', got '%s'", consoleAppender.Type)
	}

	// Verify file appender with rotation
	fileAppender := appLogger.Appenders[1]
	if fileAppender.Type != "file" {
		t.Errorf("Expected appender type 'file', got '%s'", fileAppender.Type)
	}
	if fileAppender.Path != "logs/app.log" {
		t.Errorf("Expected file path 'logs/app.log', got '%s'", fileAppender.Path)
	}
	if fileAppender.Rotation.MaxSize != 10485760 {
		t.Errorf("Expected rotation max size 10485760, got %d", fileAppender.Rotation.MaxSize)
	}
	if !fileAppender.Rotation.Compress {
		t.Error("Expected rotation to enable compression")
	}

	// Verify security settings
	if !config.Security.MaskSensitiveData {
		t.Error("Expected security to mask sensitive data")
	}
	if len(config.Security.PIIPatterns) != 1 {
		t.Errorf("Expected 1 PII pattern, got %d", len(config.Security.PIIPatterns))
	}

	// Verify performance settings
	if config.Performance.RateLimit != 1000 {
		t.Errorf("Expected rate limit 1000, got %d", config.Performance.RateLimit)
	}
	if config.Performance.BufferSize != 100 {
		t.Errorf("Expected buffer size 100, got %d", config.Performance.BufferSize)
	}
	if config.Performance.FlushInterval != 5*time.Second {
		t.Errorf("Expected flush interval 5s, got %v", config.Performance.FlushInterval)
	}
}

func TestConfigLoader_LoadFromJSON(t *testing.T) {
	// Create temp JSON file
	jsonContent := `{
  "loggers": [
    {
      "name": "web",
      "level": "INFO",
      "appenders": [
        {
          "type": "console",
          "layout": {
            "type": "simple",
            "show_timestamp": true,
            "show_level": true,
            "show_component": false,
            "show_file": false
          }
        }
      ]
    }
  ],
  "security": {
    "pii_patterns": [],
    "mask_sensitive_data": false
  },
  "performance": {
    "rate_limit": 500,
    "buffer_size": 50,
    "flush_interval": "10s"
  },
  "environment": {
    "default_level": "WARN",
    "production_level": "ERROR",
    "development_level": "DEBUG"
  }
}`

	loader := NewConfigLoader("HALOLOG_")
	config, err := loader.LoadFromJSON([]byte(jsonContent))
	if err != nil {
		t.Fatalf("LoadFromJSON failed: %v", err)
	}

	// Verify configuration
	if len(config.Loggers) != 1 {
		t.Errorf("Expected 1 logger, got %d", len(config.Loggers))
	}

	webLogger := config.Loggers[0]
	if webLogger.Name != "web" {
		t.Errorf("Expected logger name 'web', got '%s'", webLogger.Name)
	}
	if webLogger.Level != types.InfoLevel {
		t.Errorf("Expected level INFO, got %v", webLogger.Level)
	}

	if config.Performance.RateLimit != 500 {
		t.Errorf("Expected rate limit 500, got %d", config.Performance.RateLimit)
	}
}

func TestConfigLoader_LoadFromEnv(t *testing.T) {
	// Set environment variables
	os.Setenv("HALOLOG_LOGGER_NAME", "env-logger")
	os.Setenv("HALOLOG_LOGGER_LEVEL", "WARN")
	os.Setenv("HALOLOG_APPENDER_TYPE", "file")
	os.Setenv("HALOLOG_FILE_PATH", "logs/env.log")
	os.Setenv("HALOLOG_RATE_LIMIT", "2000")
	os.Setenv("HALOLOG_BUFFER_SIZE", "200")

	defer func() {
		os.Unsetenv("HALOLOG_LOGGER_NAME")
		os.Unsetenv("HALOLOG_LOGGER_LEVEL")
		os.Unsetenv("HALOLOG_APPENDER_TYPE")
		os.Unsetenv("HALOLOG_FILE_PATH")
		os.Unsetenv("HALOLOG_RATE_LIMIT")
		os.Unsetenv("HALOLOG_BUFFER_SIZE")
	}()

	loader := NewConfigLoader("HALOLOG_")
	config, err := loader.LoadFromEnvironment()
	if err != nil {
		t.Fatalf("LoadFromEnvironment failed: %v", err)
	}

	// Verify environment-based configuration
	if len(config.Loggers) != 1 {
		t.Errorf("Expected 1 logger from env, got %d", len(config.Loggers))
	}

	envLogger := config.Loggers[0]
	if envLogger.Name != "env-logger" {
		t.Errorf("Expected logger name 'env-logger', got '%s'", envLogger.Name)
	}
	if envLogger.Level != types.WarnLevel {
		t.Errorf("Expected level WARN, got %v", envLogger.Level)
	}

	if config.Performance.RateLimit != 2000 {
		t.Errorf("Expected rate limit 2000, got %d", config.Performance.RateLimit)
	}
	if config.Performance.BufferSize != 200 {
		t.Errorf("Expected buffer size 200, got %d", config.Performance.BufferSize)
	}
}

func TestConfigLoader_MergeConfigs(t *testing.T) {
	// Create base config
	baseConfig := &HaloLogConfig{
		Loggers: []LoggerConfig{
			{
				Name:  "base-logger",
				Level: types.InfoLevel,
				Appenders: []AppenderConfig{
					{
						Type: "console",
					},
				},
			},
		},
		Performance: PerformanceConfig{
			RateLimit:     1000,
			BufferSize:    100,
			FlushInterval: 5 * time.Second,
		},
	}

	// Create override config
	overrideConfig := &HaloLogConfig{
		Loggers: []LoggerConfig{
			{
				Name:  "override-logger",
				Level: types.DebugLevel,
				Appenders: []AppenderConfig{
					{
						Type: "file",
						Path: "override.log",
					},
				},
			},
		},
		Performance: PerformanceConfig{
			RateLimit:     2000,             // Override rate limit
			BufferSize:    200,              // Override buffer size
			FlushInterval: 10 * time.Second, // Override flush interval
		},
		Security: SecurityConfig{
			MaskSensitiveData: true,
			PIIPatterns: []PIIPattern{
				{
					Name:        "email",
					Pattern:     `\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`,
					Replacement: "[EMAIL]",
				},
			},
		},
	}

	loader := NewConfigLoader("HALOLOG_")
	mergedConfig, err := loader.MergeConfigurations(baseConfig, overrideConfig)
	if err != nil {
		t.Fatalf("MergeConfigurations failed: %v", err)
	}

	// Verify merged configuration
	if len(mergedConfig.Loggers) != 1 {
		t.Errorf("Expected 1 logger after merge, got %d", len(mergedConfig.Loggers))
	}

	// Verify performance settings were overridden
	if mergedConfig.Performance.RateLimit != 2000 {
		t.Errorf("Expected rate limit 2000 after merge, got %d", mergedConfig.Performance.RateLimit)
	}
	if mergedConfig.Performance.BufferSize != 200 {
		t.Errorf("Expected buffer size 200 after merge, got %d", mergedConfig.Performance.BufferSize)
	}

	// Verify security settings were added
	if !mergedConfig.Security.MaskSensitiveData {
		t.Error("Expected security settings to be added from override config")
	}
	if len(mergedConfig.Security.PIIPatterns) != 1 {
		t.Errorf("Expected 1 PII pattern after merge, got %d", len(mergedConfig.Security.PIIPatterns))
	}
}

func TestConfigLoader_MergeConfigs_EmptyConfigs(t *testing.T) {
	loader := NewConfigLoader("HALOLOG_")
	_, err := loader.MergeConfigurations()
	if err == nil {
		t.Error("Expected error when no configurations provided")
	}
	if !strings.Contains(err.Error(), "no configurations provided") {
		t.Errorf("Expected error message to contain 'no configurations provided', got: %v", err)
	}
}

func TestConfigLoader_MergeConfigs_NilOverride(t *testing.T) {
	baseConfig := &HaloLogConfig{
		Loggers: []LoggerConfig{
			{
				Name:  "base-logger",
				Level: types.InfoLevel,
			},
		},
		Performance: PerformanceConfig{
			RateLimit:     1000,
			BufferSize:    100,
			FlushInterval: 5 * time.Second,
		},
	}

	loader := NewConfigLoader("HALOLOG_")
	mergedConfig, err := loader.MergeConfigurations(baseConfig, nil)
	if err != nil {
		t.Fatalf("MergeConfigurations failed: %v", err)
	}

	// Should return base config unchanged when override is nil
	if len(mergedConfig.Loggers) != 1 {
		t.Errorf("Expected 1 logger after merge with nil, got %d", len(mergedConfig.Loggers))
	}
	if mergedConfig.Loggers[0].Name != "base-logger" {
		t.Errorf("Expected base logger name to be preserved, got '%s'", mergedConfig.Loggers[0].Name)
	}
	if mergedConfig.Performance.RateLimit != 1000 {
		t.Errorf("Expected base rate limit to be preserved, got %d", mergedConfig.Performance.RateLimit)
	}
}

func TestConfigLoader_InvalidYAML(t *testing.T) {
	// Create invalid YAML file
	invalidYAML := `
loggers:
  - name: invalid
    level: INVALID_LEVEL
    invalid_field: should_fail
`

	loader := NewConfigLoader("HALOLOG_")
	_, err := loader.LoadFromYAML([]byte(invalidYAML))
	if err == nil {
		t.Error("Expected error for invalid YAML, got nil")
	}
}

func TestConfigLoader_InvalidJSON(t *testing.T) {
	// Create invalid JSON file
	invalidJSON := `{
  "loggers": [
    {
      "name": "invalid",
      "level": "INVALID_LEVEL",
      "invalid_field": "should_fail"
    }
  ]
}`

	loader := NewConfigLoader("HALOLOG_")
	_, err := loader.LoadFromJSON([]byte(invalidJSON))
	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}
}

func TestConfigLoader_NonExistentFile(t *testing.T) {
	loader := NewConfigLoader("HALOLOG_")

	// Test non-existent YAML file - now LoadFromYAML expects []byte, so this test is invalid
	// We'll test with invalid YAML content instead
	_, err := loader.LoadFromYAML([]byte("invalid: yaml: content: ["))
	if err == nil {
		t.Error("Expected error for invalid YAML content, got nil")
	}

	// Test non-existent JSON file - now LoadFromJSON expects []byte, so this test is invalid
	// We'll test with invalid JSON content instead
	_, err = loader.LoadFromJSON([]byte("{invalid json content"))
	if err == nil {
		t.Error("Expected error for invalid JSON content, got nil")
	}
}

func TestConfigLoader_getEnvBool(t *testing.T) {
	loader := NewConfigLoader("TEST_")

	// Test default value when env var is not set
	result := loader.getEnvBool("NONEXISTENT_BOOL", true)
	if result != true {
		t.Errorf("Expected default true, got %v", result)
	}

	result = loader.getEnvBool("NONEXISTENT_BOOL", false)
	if result != false {
		t.Errorf("Expected default false, got %v", result)
	}

	// Test true values
	os.Setenv("TEST_TRUE_BOOL", "true")
	os.Setenv("TEST_TRUE_BOOL_1", "1")
	os.Setenv("TEST_TRUE_BOOL_YES", "yes")
	os.Setenv("TEST_TRUE_BOOL_ON", "on")
	defer func() {
		os.Unsetenv("TEST_TRUE_BOOL")
		os.Unsetenv("TEST_TRUE_BOOL_1")
		os.Unsetenv("TEST_TRUE_BOOL_YES")
		os.Unsetenv("TEST_TRUE_BOOL_ON")
	}()

	trueCases := []string{"true", "1", "yes", "on", "TRUE", "True", "YES", "Yes"}
	for _, tc := range trueCases {
		os.Setenv("TEST_BOOL", tc)
		result := loader.getEnvBool("BOOL", false)
		if result != true {
			t.Errorf("Expected true for '%s', got %v", tc, result)
		}
	}

	// Test false values
	falseCases := []string{"false", "0", "no", "off", "FALSE", "False", "NO", "No"}
	for _, tc := range falseCases {
		os.Setenv("TEST_BOOL", tc)
		result := loader.getEnvBool("BOOL", true)
		if result != false {
			t.Errorf("Expected false for '%s', got %v", tc, result)
		}
	}

	// Test invalid values (should return default)
	invalidCases := []string{"invalid", "maybe", "", "2", "-1"}
	for _, tc := range invalidCases {
		os.Setenv("TEST_BOOL", tc)
		result := loader.getEnvBool("BOOL", true)
		if result != true {
			t.Errorf("Expected default true for invalid '%s', got %v", tc, result)
		}
	}
}

func TestConfigLoader_getEnvInt(t *testing.T) {
	loader := NewConfigLoader("TEST_")

	// Test default value when env var is not set
	result := loader.getEnvInt("NONEXISTENT_INT", 42)
	if result != 42 {
		t.Errorf("Expected default 42, got %d", result)
	}

	// Test valid integer values
	os.Setenv("TEST_VALID_INT", "123")
	os.Setenv("TEST_ZERO_INT", "0")
	os.Setenv("TEST_NEGATIVE_INT", "-456")
	defer func() {
		os.Unsetenv("TEST_VALID_INT")
		os.Unsetenv("TEST_ZERO_INT")
		os.Unsetenv("TEST_NEGATIVE_INT")
	}()

	result = loader.getEnvInt("VALID_INT", 0)
	if result != 123 {
		t.Errorf("Expected 123, got %d", result)
	}

	result = loader.getEnvInt("ZERO_INT", 42)
	if result != 0 {
		t.Errorf("Expected 0, got %d", result)
	}

	result = loader.getEnvInt("NEGATIVE_INT", 0)
	if result != -456 {
		t.Errorf("Expected -456, got %d", result)
	}

	// Test invalid values (should return default)
	os.Setenv("TEST_INVALID_INT", "not_a_number")
	os.Setenv("TEST_EMPTY_INT", "")
	defer func() {
		os.Unsetenv("TEST_INVALID_INT")
		os.Unsetenv("TEST_EMPTY_INT")
	}()

	result = loader.getEnvInt("INVALID_INT", 999)
	if result != 999 {
		t.Errorf("Expected default 999 for invalid int, got %d", result)
	}

	result = loader.getEnvInt("EMPTY_INT", 888)
	if result != 888 {
		t.Errorf("Expected default 888 for empty int, got %d", result)
	}
}

func TestConfigLoader_getEnvStringSlice(t *testing.T) {
	loader := NewConfigLoader("TEST_")

	// Test default value when env var is not set
	defaultValue := []string{"default1", "default2"}
	result := loader.getEnvStringSlice("NONEXISTENT_SLICE", defaultValue)
	if len(result) != 2 || result[0] != "default1" || result[1] != "default2" {
		t.Errorf("Expected default slice, got %v", result)
	}

	// Test empty env var (should return default value)
	os.Setenv("TEST_EMPTY_SLICE", "")
	defer os.Unsetenv("TEST_EMPTY_SLICE")

	result = loader.getEnvStringSlice("EMPTY_SLICE", defaultValue)
	if len(result) != 2 || result[0] != "default1" || result[1] != "default2" {
		t.Errorf("Expected default slice when env var is empty, got %v", result)
	}

	// Test single value
	os.Setenv("TEST_SINGLE_SLICE", "single")
	defer os.Unsetenv("TEST_SINGLE_SLICE")

	result = loader.getEnvStringSlice("SINGLE_SLICE", defaultValue)
	if len(result) != 1 || result[0] != "single" {
		t.Errorf("Expected single value slice, got %v", result)
	}

	// Test multiple values
	os.Setenv("TEST_MULTI_SLICE", "value1,value2,value3")
	defer os.Unsetenv("TEST_MULTI_SLICE")

	result = loader.getEnvStringSlice("MULTI_SLICE", defaultValue)
	if len(result) != 3 || result[0] != "value1" || result[1] != "value2" || result[2] != "value3" {
		t.Errorf("Expected multi value slice, got %v", result)
	}

	// Test values with spaces
	os.Setenv("TEST_SPACED_SLICE", "item1, item2,item3")
	defer os.Unsetenv("TEST_SPACED_SLICE")

	result = loader.getEnvStringSlice("SPACED_SLICE", defaultValue)
	if len(result) != 3 || result[0] != "item1" || result[1] != " item2" || result[2] != "item3" {
		t.Errorf("Expected spaced slice, got %v", result)
	}
}

func TestConfigLoader_ParseLogLevel(t *testing.T) {
	// Test valid log levels
	testCases := []struct {
		input    string
		expected types.LogLevel
	}{
		{"DEBUG", types.DebugLevel},
		{"INFO", types.InfoLevel},
		{"WARN", types.WarnLevel},
		{"WARNING", types.WarnLevel},
		{"ERROR", types.ErrorLevel},
		{"FATAL", types.FatalLevel},
		{"debug", types.DebugLevel},
		{"info", types.InfoLevel},
		{"warn", types.WarnLevel},
		{"error", types.ErrorLevel},
		{"fatal", types.FatalLevel},
	}

	for _, tc := range testCases {
		result := ParseLogLevel(tc.input)
		if result != tc.expected {
			t.Errorf("ParseLogLevel(%s) = %v, expected %v", tc.input, result, tc.expected)
		}
	}

	// Test invalid log levels (should return types.InfoLevel as default)
	invalidCases := []string{"", "INVALID", "CRITICAL", "123"}
	for _, invalid := range invalidCases {
		result := ParseLogLevel(invalid)
		if result != types.InfoLevel {
			t.Errorf("ParseLogLevel(%s) = %v, expected types.InfoLevel for invalid input", invalid, result)
		}
	}

	// Test TRACE level separately as it's now valid
	traceResult := ParseLogLevel("TRACE")
	if traceResult != types.TraceLevel {
		t.Errorf("ParseLogLevel(TRACE) = %v, expected TraceLevel", traceResult)
	}
}

// Helper functions

func createTempYAMLFile(content string) (string, error) {
	tmpFile, err := os.CreateTemp("", "halolog-config-*.yaml")
	if err != nil {
		return "", err
	}

	if _, err := tmpFile.WriteString(content); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return "", err
	}

	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpFile.Name())
		return "", err
	}

	return tmpFile.Name(), nil
}

func createTempJSONFile(content string) (string, error) {
	tmpFile, err := os.CreateTemp("", "halolog-config-*.json")
	if err != nil {
		return "", err
	}

	if _, err := tmpFile.WriteString(content); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return "", err
	}

	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpFile.Name())
		return "", err
	}

	return tmpFile.Name(), nil
}
