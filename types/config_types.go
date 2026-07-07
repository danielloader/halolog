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

package types

import (
	"time"
)

// LoggerConfig represents the complete logger configuration
type LoggerConfig struct {
	// Core settings
	Level       LogLevel
	Component   string
	Environment EnvironmentType

	// Performance settings
	WorkerCount     int
	BufferSize      int
	DebugBufferSize int

	// Feature flags
	MetricsEnabled  bool
	FastPathEnabled bool
	AutoMaskEnabled bool

	// Output configuration
	Outputs []OutputConfig

	// Field configuration
	MaxFieldsPerLog    int
	AutoRegisterFields bool
	UseGlobalDict      bool

	// Security settings
	MaskingRules []MaskingRule

	// Sampling configuration
	SamplingRate float64
	RateLimit    int

	// Timeouts and intervals
	HealthCheckInterval time.Duration
	FlushInterval       time.Duration
	RetryInterval       time.Duration
}

// GetLevel returns the configured minimum log level.
func (c *LoggerConfig) GetLevel() LogLevel { return c.Level }

// GetOutputs returns the configured outputs.
func (c *LoggerConfig) GetOutputs() []Output { return nil } // TODO: Convert OutputConfig to LogOutput

// GetHooks returns the configured log hooks.
func (c *LoggerConfig) GetHooks() []LogHook { return nil } // TODO: Add hooks support

// GetSamplingRate returns the configured sampling rate.
func (c *LoggerConfig) GetSamplingRate() float64 { return c.SamplingRate }

// GetBufferSize returns the configured buffer size.
func (c *LoggerConfig) GetBufferSize() int { return c.BufferSize }

// GetFlushInterval returns the configured flush interval in nanoseconds.
func (c *LoggerConfig) GetFlushInterval() int64 { return int64(c.FlushInterval) }

// GetBatchSize returns the configured batch size.
func (c *LoggerConfig) GetBatchSize() int { return 100 } // Default batch size

// GetBatchTimeout returns the configured batch timeout in milliseconds.
func (c *LoggerConfig) GetBatchTimeout() int64 { return 1000 } // Default 1 second

// IsFastPathEnabled reports whether the fast path is enabled.
func (c *LoggerConfig) IsFastPathEnabled() bool { return c.FastPathEnabled }

// GetFieldPoolSize returns the configured field pool size.
func (c *LoggerConfig) GetFieldPoolSize() int { return 1024 } // Default pool size

// GetStaticFieldCount returns the configured maximum static field count.
func (c *LoggerConfig) GetStaticFieldCount() int { return c.MaxFieldsPerLog }

// GetMaskingRules returns the configured masking rules.
func (c *LoggerConfig) GetMaskingRules() []MaskingRule { return c.MaskingRules }

// IsAutoMaskEnabled reports whether automatic masking is enabled.
func (c *LoggerConfig) IsAutoMaskEnabled() bool { return c.AutoMaskEnabled }

// IsMetricsEnabled reports whether metrics collection is enabled.
func (c *LoggerConfig) IsMetricsEnabled() bool { return c.MetricsEnabled }

// GetHealthCheckInterval returns the configured health check interval in nanoseconds.
func (c *LoggerConfig) GetHealthCheckInterval() int64 { return int64(c.HealthCheckInterval) }

// OutputConfig represents an output destination configuration
type OutputConfig struct {
	Type    OutputType
	Name    string
	Enabled bool
	Filter  FilterConfig
	Options map[string]interface{}
}

// OutputType represents different output types
type OutputType string

// Output type constants
const (
	OutputTypeConsole OutputType = "console"
	OutputTypeFile    OutputType = "file"
	OutputTypeNetwork OutputType = "network"
	OutputTypeCustom  OutputType = "custom"
)

// FilterConfig represents filter configuration
type FilterConfig struct {
	MinLevel        LogLevel
	MaxLevel        LogLevel
	Components      []string
	ExcludePatterns []string
	IncludePatterns []string
}

// ConsoleConfig represents console output configuration
type ConsoleConfig struct {
	Target OutputTarget
	Color  bool
	Filter FilterConfig
}

// FileConfig represents file output configuration
type FileConfig struct {
	Path       string
	Rotation   RotationConfig
	Async      bool
	BufferSize int
	Filter     FilterConfig
}

// NetworkConfig represents network output configuration
type NetworkConfig struct {
	Protocol       string
	Host           string
	Port           int
	Timeout        time.Duration
	RetryCount     int
	Filter         FilterConfig
	FallbackOutput string
}

// FormatterConfig represents formatter configuration
type FormatterConfig struct {
	Type    FormatterType
	Options map[string]interface{}
}

// FormatterType represents different formatter types
type FormatterType string

// Formatter type constants
const (
	FormatterTypeJSON    FormatterType = "json"
	FormatterTypeText    FormatterType = "text"
	FormatterTypeCompact FormatterType = "compact"
	FormatterTypePattern FormatterType = "pattern"
)

// SecurityConfig represents security-related configuration
type SecurityConfig struct {
	PIIMasking   PIIMaskingConfig
	FieldMasking FieldMaskingConfig
	SafeMode     bool
}

// PIIMaskingConfig represents PII masking configuration
type PIIMaskingConfig struct {
	Enabled        bool
	Rules          []MaskingRule
	AutoDetect     bool
	CustomPatterns []string
}

// FieldMaskingConfig represents field-level masking configuration
type FieldMaskingConfig struct {
	Enabled      bool
	MaskedFields []string
	MaskAll      bool
}

// PerformanceConfig represents performance-related configuration
type PerformanceConfig struct {
	WorkerPoolSize      int
	BufferSize          int
	BatchSize           int
	FlushInterval       time.Duration
	BackpressureEnabled bool
	MetricsEnabled      bool
}

// WriterConfig represents writer configuration
type WriterConfig struct {
	Provider      WriterProvider
	Async         bool
	BufferSize    int
	FlushInterval time.Duration
	RetryPolicy   RetryPolicy
}

// RetryPolicy represents retry configuration
type RetryPolicy struct {
	MaxRetries    int
	InitialDelay  time.Duration
	MaxDelay      time.Duration
	BackoffFactor float64
}

// Use interfaces from interfaces package to avoid duplication

// ----------------------------- Immutable Config ----------------------------

// WriteSyncFunc is the internal function signature for adapter writers.
// This eliminates the need for interface method calls in the hot path.
type WriteSyncFunc func(entry *LogEntry)
