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

// Package config provides configuration management for HaloLogger
// Author: Admilson B. F. Cossa

package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/go-gen-ecosystem/halolog/masking"

	"github.com/go-gen-ecosystem/halolog/interfaces"
	"github.com/go-gen-ecosystem/halolog/registry"
	"github.com/go-gen-ecosystem/halolog/types"
)

// ImmutableConfig is the runtime-config object. It's immutable: any change requires copying
// and atomically swapping the pointer. Keep it small and pre-resolve complex objects.
type ImmutableConfig struct {
	Level    types.LogLevel
	Adapters []types.Adapter // Concrete adapters resolved at build time

	// Features
	EnableMasking     bool
	EnableAsync       bool
	EnableJSON        bool
	EnableColorized   bool
	EnablePrettyPrint bool
	EnableMetrics     bool

	// Optional components
	Sampler types.Sampler
	Masker  types.PIIMasker

	// Worker pool size for async mode
	WorkerCount int

	// Policy rules for filtering
	PolicyRules []interfaces.PolicyRule

	// Panic function for panic logs
	PanicFunc func(string)

	// Line info settings
	LineInDebug bool
	LineInError bool

	// Zero-allocation field-based PII masking
	SensitiveFieldRegistry  *registry.SensitiveFieldRegistry
	EnableFieldBasedMasking bool

	// Startup registration configuration for O(1) field access
	StartupRegistration *StartupConfig
	RegisteredFieldIDs  map[string]int      // fieldName -> fieldID for O(1) lookup
	SensitiveFieldMap   map[int]string      // fieldID -> maskValue for O(1) masking
	FieldColorMap       map[int]types.Color // fieldID -> color for O(1) color lookup
	OptimalBufferSize   int
	ContextFieldsConfig *ContextFieldsConfig // Context field auto-loading
}

// ConfigBuilder provides a fluent interface for building configurations
type ConfigBuilder struct {
	env             string
	level           types.LogLevel
	colorized       bool
	prettyPrint     bool
	adapters        []types.Adapter
	policyRules     []PolicyRule
	maskingRules    []types.MaskingRule // Custom masking rules
	debugBufferSize int
	maxDebugHistory int
	lineInDebug     bool
	lineInError     bool
	verbose         bool
	errorHandler    interfaces.ErrorHandler
	exitFunc        func(int)    // Allows custom exit logic for testing
	panicFunc       func(string) // Allows custom panic logic for testing

	// integration options
	enableMetrics    bool // Enable automatic metrics collection
	enablePooling    bool // Enable buffer/field pooling
	enableAutoFields bool // Enable automatic standard field population
	enableAsync      bool // Enable async buffering for performance

	// Logging-focused configuration (new)
	formatter     string                   // "json" or "text"
	fileOutput    *types.FileOutputConfig  // File output configuration
	consoleOutput string                   // "stdout", "stderr", or ""
	asyncBuffer   *types.AsyncBufferConfig // Async buffering configuration

	// Field-based PII masking configuration
	sensitiveFields map[string]string // fieldName -> maskValue mapping

	// Startup registration configuration
	startupRegistration *StartupConfig
	customFields        []string
	fieldColors         map[string]types.Color
	optimalBufferSize   int
	contextFieldsConfig *ContextFieldsConfig // Context field auto-loading

	// Tracking for environment defaults
	envSet            bool
	levelSet          bool
	colorizedSet      bool
	prettyPrintSet    bool
	lineInDebugSet    bool
	lineInErrorSet    bool
	levelFromEnv      bool // Track if level came from environment variables
	levelFromDefaults bool // Track if level came from defaults functions
}

// WithMaskingRule adds a custom PII masking rule
func (c *ConfigBuilder) WithMaskingRule(rule types.MaskingRule) *ConfigBuilder {
	c.maskingRules = append(c.maskingRules, rule)
	return c
}

// Build creates the final immutable configuration
func (c *ConfigBuilder) Build() *ImmutableConfig {
	// Apply environment-specific settings
	c.applyEnvironmentSettings()

	// Convert adapters (direct copy since we use types.Adapter now)
	adapters := make([]types.Adapter, len(c.adapters))
	copy(adapters, c.adapters)

	// Create file adapter if file output is configured
	if c.fileOutput != nil {
		// For now, create a simple console adapter as placeholder
		// In production, this would create the actual file adapter
		// with rotation support from the file package
		// adapters = append(adapters, file.NewFileAdapter(c.fileOutput.Path, &file.RotationConfig{
		//     MaxSize:    c.fileOutput.MaxSize,
		//     MaxBackups: c.fileOutput.MaxBackups,
		//     MaxAge:     time.Duration(c.fileOutput.MaxAge) * 24 * time.Hour,
		// }))
	}

	// Convert masking rules
	var maskingRules []types.MaskingRule
	maskingRules = append(maskingRules, getDefaultMaskingRules()...)
	maskingRules = append(maskingRules, c.maskingRules...) // Add custom rules

	// Convert policy rules
	var policyRules []interfaces.PolicyRule
	for _, rule := range c.policyRules {
		policyRules = append(policyRules, interfaces.PolicyRule{
			Name:    rule.Name,
			Pattern: rule.Pattern,
			Action:  rule.Action,
		})
	}

	// Create field-based masking registry if enabled
	var sensitiveFieldRegistry *registry.SensitiveFieldRegistry
	enableFieldBasedMasking := len(c.sensitiveFields) > 0

	if enableFieldBasedMasking {
		sensitiveFieldRegistry = registry.NewSensitiveFieldRegistry()
		// Register all configured sensitive fields
		for fieldName := range c.sensitiveFields {
			sensitiveFieldRegistry.RegisterSensitiveField(fieldName)
		}
	}

	// Initialize PII Masker
	var masker types.PIIMasker
	if len(maskingRules) > 0 || enableFieldBasedMasking {
		if enableFieldBasedMasking {
			// Use registry-based masker for O(1) field lookup
			masker = masking.NewPIIMaskerWithRegistry(sensitiveFieldRegistry)
		} else {
			// Use standard masker
			masker = masking.NewPIIMasker()
		}

		// Apply all masking rules
		for _, rule := range maskingRules {
			masker.AddRule(rule.Pattern, rule.Replace, rule.Type)
		}
	}

	// Build startup registration configuration
	var startupConfig *StartupConfig
	registeredFieldIDs := make(map[string]int)
	sensitiveFieldMap := make(map[int]string)
	fieldColorMap := make(map[int]types.Color)
	optimalBufferSize := c.optimalBufferSize

	if c.startupRegistration != nil {
		startupConfig = c.startupRegistration
		// Populate field ID maps from startup config

		// Process custom fields - assign sequential IDs starting from 0
		for i, fieldName := range c.startupRegistration.CustomFields {
			registeredFieldIDs[fieldName] = i
		}

		// Process sensitive fields
		for _, sf := range c.startupRegistration.SensitiveFields {
			registeredFieldIDs[sf.Name] = sf.FieldID
			sensitiveFieldMap[sf.FieldID] = sf.Mask
		}

		// Process field colors - map to field IDs (only for existing fields)
		for fieldName, color := range c.startupRegistration.FieldColors {
			if fieldID, exists := registeredFieldIDs[fieldName]; exists {
				fieldColorMap[fieldID] = color
			}
			// Skip fields that don't exist - colors should only apply to registered fields
		}

		if c.startupRegistration.OptimalBufferSize > 0 {
			optimalBufferSize = c.startupRegistration.OptimalBufferSize
		}
	} else if len(c.customFields) > 0 || len(c.fieldColors) > 0 || c.optimalBufferSize > 0 || len(c.sensitiveFields) > 0 {
		// Build from individual fields - assign sequential field IDs starting from 0
		startupConfig = &StartupConfig{
			CustomFields:      c.customFields,
			SensitiveFields:   []SensitiveFieldConfig{},
			FieldColors:       c.fieldColors,
			OptimalBufferSize: c.optimalBufferSize,
		}

		// Assign field IDs to custom fields
		for i, fieldName := range c.customFields {
			registeredFieldIDs[fieldName] = i
		}

		// Convert sensitive fields to SensitiveFieldConfig and assign field IDs
		for fieldName, maskValue := range c.sensitiveFields {
			fieldID := len(registeredFieldIDs) // Use next available ID
			registeredFieldIDs[fieldName] = fieldID
			sensitiveFieldMap[fieldID] = maskValue

			startupConfig.SensitiveFields = append(startupConfig.SensitiveFields, SensitiveFieldConfig{
				Name:    fieldName,
				FieldID: fieldID,
				Mask:    maskValue,
			})
		}

		// Map field colors to field IDs
		for fieldName, color := range c.fieldColors {
			if fieldID, exists := registeredFieldIDs[fieldName]; exists {
				fieldColorMap[fieldID] = color
			} else {
				// If field doesn't have an ID yet, assign one
				fieldID := len(registeredFieldIDs)
				registeredFieldIDs[fieldName] = fieldID
				fieldColorMap[fieldID] = color
			}
		}
	}

	// Calculate worker count - handle nil asyncBuffer
	workerCount := 4 // Default worker count
	if c.asyncBuffer != nil {
		workerCount = c.asyncBuffer.BufferSize / 8
		if workerCount == 0 {
			workerCount = 4 // Minimum worker count
		}
	}

	// Build the core ImmutableConfig directly
	return &ImmutableConfig{
		Level:                   c.level,
		Adapters:                adapters,
		EnableMasking:           len(maskingRules) > 0,
		EnableAsync:             c.enableAsync,
		EnableJSON:              c.formatter == "json",
		EnableColorized:         c.colorized,
		EnablePrettyPrint:       c.prettyPrint,
		EnableMetrics:           c.enableMetrics,
		Masker:                  masker,
		WorkerCount:             workerCount,
		PolicyRules:             policyRules,
		PanicFunc:               c.panicFunc,
		LineInDebug:             c.lineInDebug,
		LineInError:             c.lineInError,
		SensitiveFieldRegistry:  sensitiveFieldRegistry,
		EnableFieldBasedMasking: enableFieldBasedMasking,
		StartupRegistration:     startupConfig,
		RegisteredFieldIDs:      registeredFieldIDs,
		SensitiveFieldMap:       sensitiveFieldMap,
		FieldColorMap:           fieldColorMap,
		OptimalBufferSize:       optimalBufferSize,
		ContextFieldsConfig:     c.contextFieldsConfig,
	}
}

// PolicyRule defines a logging policy rule
type PolicyRule struct {
	Name    string
	Pattern string
	Action  string // "allow", "deny", "mask"
}

// FileOutputConfig configures file output with rotation
type FileOutputConfig struct {
	Path       string
	MaxSize    int64
	MaxBackups int
	MaxAge     int
}

// AsyncBufferConfig configures async buffering for performance
type AsyncBufferConfig struct {
	BufferSize    int
	FlushInterval time.Duration
}

// ErrorHandler handles logging errors
type ErrorHandler interface {
	HandleError(err error)
}

// getDefaultMaskingRules returns default sensitive field masking rules
func getDefaultMaskingRules() []types.MaskingRule {
	return []types.MaskingRule{
		{
			Pattern:  `(?i)password`,
			Replace:  "[REDACTED]",
			Type:     "regex",
			Compiled: nil,
		},
		{
			Pattern:  `(?i)token`,
			Replace:  "[REDACTED]",
			Type:     "regex",
			Compiled: nil,
		},
		{
			Pattern:  `(?i)api[_-]?key`,
			Replace:  "[REDACTED]",
			Type:     "regex",
			Compiled: nil,
		},
		{
			Pattern:  `(?i)secret`,
			Replace:  "[REDACTED]",
			Type:     "regex",
			Compiled: nil,
		},
		{
			Pattern:  `(?i)ssn`,
			Replace:  "[REDACTED]",
			Type:     "regex",
			Compiled: nil,
		},
		{
			Pattern:  `(?i)credit[_-]?card`,
			Replace:  "[REDACTED]",
			Type:     "regex",
			Compiled: nil,
		},
	}
}

// NewConfig creates a new configuration builder with defaults
func NewConfig() *ConfigBuilder {
	return &ConfigBuilder{
		env:             "",
		level:           types.InfoLevel,
		colorized:       true,
		prettyPrint:     true,
		adapters:        []types.Adapter{},
		policyRules:     []PolicyRule{},
		debugBufferSize: 1000,
		maxDebugHistory: 1000,
		lineInDebug:     false,
		lineInError:     true,
		verbose:         false,
		exitFunc:        os.Exit,                         // Default exit function
		panicFunc:       func(msg string) { panic(msg) }, // Default panic function

		// integrations enabled by default
		enableMetrics:    true,
		enablePooling:    true,
		enableAutoFields: true,

		// logging defaults
		formatter:     "text",
		consoleOutput: "stdout",
	}
}

// --- ConfigBuilder Methods (Fluent Interface) ---

// WithEnv sets the environment
func (c *ConfigBuilder) WithEnv(env string) *ConfigBuilder {
	c.env = env
	c.envSet = true
	return c
}

// WithLevel sets the minimum log level
func (c *ConfigBuilder) WithLevel(level types.LogLevel) *ConfigBuilder {
	c.level = level
	c.levelSet = true
	return c
}

// WithColorized sets whether to use colors
func (c *ConfigBuilder) WithColorized(colorized bool) *ConfigBuilder {
	c.colorized = colorized
	c.colorizedSet = true
	return c
}

// WithPrettyPrint sets whether to pretty print
func (c *ConfigBuilder) WithPrettyPrint(pretty bool) *ConfigBuilder {
	c.prettyPrint = pretty
	c.prettyPrintSet = true
	return c
}

// WithOutput adds an output adapter
func (c *ConfigBuilder) WithOutput(adapter types.Adapter) *ConfigBuilder {
	c.adapters = append(c.adapters, adapter)
	return c
}

// WithPolicyRule adds a policy rule
func (c *ConfigBuilder) WithPolicyRule(rule PolicyRule) *ConfigBuilder {
	c.policyRules = append(c.policyRules, rule)
	return c
}

// WithDebugBufferSize sets the debug buffer size
func (c *ConfigBuilder) WithDebugBufferSize(size int) *ConfigBuilder {
	c.debugBufferSize = size
	return c
}

// WithMaxDebugHistory sets the maximum debug history
func (c *ConfigBuilder) WithMaxDebugHistory(history int) *ConfigBuilder {
	c.maxDebugHistory = history
	return c
}

// WithLineInDebug sets whether to include line info in debug logs
func (c *ConfigBuilder) WithLineInDebug(line bool) *ConfigBuilder {
	c.lineInDebug = line
	c.lineInDebugSet = true
	return c
}

// WithLineInfoInDebug sets whether to include line info in debug logs (alias for WithLineInDebug)
func (c *ConfigBuilder) WithLineInfoInDebug(enable bool) *ConfigBuilder {
	return c.WithLineInDebug(enable)
}

// WithLineInError sets whether to include line info in error logs
func (c *ConfigBuilder) WithLineInError(line bool) *ConfigBuilder {
	c.lineInError = line
	c.lineInErrorSet = true
	return c
}

// WithVerbose sets verbose mode
func (c *ConfigBuilder) WithVerbose(verbose bool) *ConfigBuilder {
	c.verbose = verbose
	return c
}

// WithErrorHandler sets the error handler callback
func (c *ConfigBuilder) WithErrorHandler(handler interfaces.ErrorHandler) *ConfigBuilder {
	c.errorHandler = handler
	return c
}

// WithExitFunc sets the function to call on fatal/panic logs that terminate execution
func (c *ConfigBuilder) WithExitFunc(fn func(int)) *ConfigBuilder {
	c.exitFunc = fn
	return c
}

// WithPanicFunc sets the function to call on panic logs
func (c *ConfigBuilder) WithPanicFunc(fn func(string)) *ConfigBuilder {
	c.panicFunc = fn
	return c
}

// WithMetrics enables or disables automatic metrics collection
func (c *ConfigBuilder) WithMetrics(enabled bool) *ConfigBuilder {
	c.enableMetrics = enabled
	return c
}

// WithPooling enables or disables buffer/field pooling
func (c *ConfigBuilder) WithPooling(enabled bool) *ConfigBuilder {
	c.enablePooling = enabled
	return c
}

// WithAutoFields enables or disables automatic standard field population
func (c *ConfigBuilder) WithAutoFields(enabled bool) *ConfigBuilder {
	c.enableAutoFields = enabled
	return c
}

// WithJSONFormatter configures JSON formatting for logging outputs
func (c *ConfigBuilder) WithJSONFormatter() *ConfigBuilder {
	c.formatter = "json"
	return c
}

// WithTextFormatter configures text formatting for logging outputs
func (c *ConfigBuilder) WithTextFormatter() *ConfigBuilder {
	c.formatter = "text"
	return c
}

// WithFileOutput configures file output with rotation
func (c *ConfigBuilder) WithFileOutput(path string, maxSizeMB int, maxBackups int, maxAgeDays int) *ConfigBuilder {
	c.fileOutput = &types.FileOutputConfig{
		Path:       path,
		MaxSize:    int64(maxSizeMB) * 1024 * 1024, // Convert MB to bytes
		MaxBackups: maxBackups,
		MaxAge:     maxAgeDays,
	}
	return c
}

// WithConsoleOutput configures console output
func (c *ConfigBuilder) WithConsoleOutput(target string) *ConfigBuilder {
	c.consoleOutput = target // "stdout", "stderr", or ""
	return c
}

// WithAsyncBuffer configures async buffering for performance
func (c *ConfigBuilder) WithAsyncBuffer(bufferSize int, flushInterval time.Duration) *ConfigBuilder {
	c.asyncBuffer = &types.AsyncBufferConfig{
		BufferSize:    bufferSize,
		FlushInterval: flushInterval,
	}
	c.enableAsync = true
	return c
}

// WithAsync enables or disables async buffering
func (c *ConfigBuilder) WithAsync(enabled bool) *ConfigBuilder {
	c.enableAsync = enabled
	return c
}

// WithFieldBasedMasking enables zero-allocation field-based PII masking
func (c *ConfigBuilder) WithFieldBasedMasking(enabled bool) *ConfigBuilder {
	// Field-based masking will be enabled if sensitive fields are configured
	// This method is for explicit API clarity
	return c
}

// WithSensitiveFields configures sensitive fields for zero-allocation masking
func (c *ConfigBuilder) WithSensitiveFields(fields map[string]string) *ConfigBuilder {
	// Store fields for later processing in Build()
	c.sensitiveFields = fields
	return c
}

// WithSensitiveField adds a single sensitive field configuration
func (c *ConfigBuilder) WithSensitiveField(fieldName, maskValue string) *ConfigBuilder {
	if c.sensitiveFields == nil {
		c.sensitiveFields = make(map[string]string)
	}
	c.sensitiveFields[fieldName] = maskValue
	return c
}

// WithStartupRegistration configures startup field registration for O(1) access
func (c *ConfigBuilder) WithStartupRegistration(config *StartupConfig) *ConfigBuilder {
	c.startupRegistration = config
	return c
}

// WithCustomFields registers custom business fields at startup
func (c *ConfigBuilder) WithCustomFields(fields []string) *ConfigBuilder {
	c.customFields = fields
	return c
}

// WithFieldColors configures field color mapping for O(1) lookup
func (c *ConfigBuilder) WithFieldColors(colors map[string]types.Color) *ConfigBuilder {
	c.fieldColors = colors
	return c
}

// WithOptimalBufferSize sets the optimal buffer size calculated from profiling
func (c *ConfigBuilder) WithOptimalBufferSize(size int) *ConfigBuilder {
	c.optimalBufferSize = size
	return c
}

// WithContextFieldsConfig configures context field auto-loading
func (c *ConfigBuilder) WithContextFieldsConfig(config *ContextFieldsConfig) *ConfigBuilder {
	c.contextFieldsConfig = config
	return c
}

// WithContextFields enables context field auto-loading with specified fields
func (c *ConfigBuilder) WithContextFields(fields []string) *ConfigBuilder {
	c.contextFieldsConfig = &ContextFieldsConfig{
		Enabled: true,
		Fields:  fields,
	}
	return c
}

func (c *ConfigBuilder) applyEnvironmentSettings() {
	// Applies environment-specific defaults if not explicitly set
	// This maintains pure configurations
	if c.env == "" {
		return
	}

	// Apply environment-specific defaults
	switch strings.ToLower(c.env) {
	case "prod", "production":
		// Production defaults: minimal output, no colors, no pretty print
		if !c.colorizedSet {
			c.colorized = false
		}
		if !c.prettyPrintSet {
			c.prettyPrint = false
		}
		// Production uses INFO level only if not explicitly set via environment variables or defaults
		if !c.levelFromEnv && !c.levelFromDefaults {
			c.level = types.InfoLevel
		}
	case "dev", "development":
		// Development defaults: verbose output, colors, pretty print, line info
		if !c.colorizedSet {
			c.colorized = true
		}
		if !c.prettyPrintSet {
			c.prettyPrint = true
		}
		if !c.lineInDebugSet {
			c.lineInDebug = true
		}
		if !c.levelSet {
			c.level = types.DebugLevel
		}
	case "staging":
		// Staging defaults: like development but with production level
		if !c.colorizedSet {
			c.colorized = true
		}
		if !c.prettyPrintSet {
			c.prettyPrint = true
		}
		if !c.levelSet {
			c.level = types.InfoLevel
		}
	default:
		// Default: development-like settings
		if !c.colorizedSet {
			c.colorized = true
		}
		if !c.prettyPrintSet {
			c.prettyPrint = true
		}
		if !c.levelSet {
			c.level = types.InfoLevel
		}
	}

	c.envSet = true
}

// LoadFromEnv creates a ConfigBuilder from environment variables
// This provides a simple way to configure logging via environment variables
// for 12-factor app compliance and containerized deployments
func LoadFromEnv() *ConfigBuilder {
	builder := NewConfig()

	// Log level - set before environment to allow explicit override
	if level := os.Getenv("LOG_LEVEL"); level != "" {
		if logLevel := ParseLogLevel(level); logLevel.IsDefined() {
			builder.WithLevel(logLevel)
			builder.levelFromEnv = true // Mark that level came from environment
		}
	}

	// Environment - set after level to allow environment defaults to apply only if level not set
	if env := os.Getenv("APP_ENV"); env != "" {
		builder.WithEnv(env)
	}

	// Colorized output - production vs development
	if colorized := os.Getenv("LOG_COLORIZED"); colorized != "" {
		builder.WithColorized(strings.ToLower(colorized) == "true")
	}

	// Pretty print - production vs development
	if pretty := os.Getenv("LOG_PRETTY"); pretty != "" {
		builder.WithPrettyPrint(strings.ToLower(pretty) == "true")
	}

	// Line in debug - development debugging
	if lineInDebug := os.Getenv("LOG_LINE_IN_DEBUG"); lineInDebug != "" {
		builder.WithLineInDebug(strings.ToLower(lineInDebug) == "true")
	}

	// Line in error - production error tracking
	if lineInError := os.Getenv("LOG_LINE_IN_ERROR"); lineInError != "" {
		builder.WithLineInError(strings.ToLower(lineInError) == "true")
	}

	// Debug buffer size for high-throughput scenarios
	if size := os.Getenv("LOG_DEBUG_BUFFER_SIZE"); size != "" {
		var bufferSize int
		if _, err := fmt.Sscanf(size, "%d", &bufferSize); err == nil && bufferSize > 0 {
			builder.WithDebugBufferSize(bufferSize)
		}
	}

	// Max debug history for memory management
	if history := os.Getenv("LOG_MAX_DEBUG_HISTORY"); history != "" {
		var maxHistory int
		if _, err := fmt.Sscanf(history, "%d", &maxHistory); err == nil && maxHistory > 0 {
			builder.WithMaxDebugHistory(maxHistory)
		}
	}

	// Verbose mode for detailed logging
	if verbose := os.Getenv("LOG_VERBOSE"); verbose != "" {
		builder.WithVerbose(strings.ToLower(verbose) == "true")
	}

	// Auto-fields for automatic metadata population
	if autoFields := os.Getenv("LOG_AUTO_FIELDS"); autoFields != "" {
		builder.WithAutoFields(strings.ToLower(autoFields) == "true")
	}

	// Pooling for performance optimization
	if pooling := os.Getenv("LOG_POOLING"); pooling != "" {
		builder.WithPooling(strings.ToLower(pooling) == "true")
	}

	// Metrics for observability
	if metrics := os.Getenv("LOG_METRICS"); metrics != "" {
		builder.WithMetrics(strings.ToLower(metrics) == "true")
	}

	return builder
}

// Interface methods for ImmutableConfig - implements interfaces.Config

// GetLevel returns the log level
func (c *ImmutableConfig) GetLevel() types.LogLevel {
	return c.Level
}

// GetEnvironment returns the environment name
func (c *ImmutableConfig) GetEnvironment() string {
	// Environment is not stored in ImmutableConfig, it's used during build
	return ""
}

// IsColorized returns whether colorized output is enabled
func (c *ImmutableConfig) IsColorized() bool {
	return c.EnableColorized
}

// IsPrettyPrint returns whether pretty print is enabled
func (c *ImmutableConfig) IsPrettyPrint() bool {
	return c.EnablePrettyPrint
}

// GetAdapters returns the configured adapters
func (c *ImmutableConfig) GetAdapters() []interfaces.Adapter {
	// Convert types.Adapter to interfaces.Adapter
	adapters := make([]interfaces.Adapter, len(c.Adapters))
	for i, adapter := range c.Adapters {
		adapters[i] = adapter
	}
	return adapters
}

// GetMaskingRules returns the masking rules
func (c *ImmutableConfig) GetMaskingRules() []types.MaskingRule {
	// Masking rules are applied to the masker, not stored separately
	return nil
}

// GetPolicyRules returns the policy rules
func (c *ImmutableConfig) GetPolicyRules() []interfaces.PolicyRule {
	return c.PolicyRules
}

// GetDebugBufferSize returns the debug buffer size
func (c *ImmutableConfig) GetDebugBufferSize() int {
	return c.OptimalBufferSize
}

// GetMaxDebugHistory returns the max debug history
func (c *ImmutableConfig) GetMaxDebugHistory() int {
	// Not stored in ImmutableConfig
	return 0
}

// IsLineInDebug returns whether line info is included in debug logs
func (c *ImmutableConfig) IsLineInDebug() bool {
	return c.LineInDebug
}

// IsLineInError returns whether line info is included in error logs
func (c *ImmutableConfig) IsLineInError() bool {
	return c.LineInError
}

// IsVerbose returns whether verbose mode is enabled
func (c *ImmutableConfig) IsVerbose() bool {
	// Not stored in ImmutableConfig
	return false
}

// GetErrorHandler returns the error handler
func (c *ImmutableConfig) GetErrorHandler() interfaces.ErrorHandler {
	// Not stored in ImmutableConfig
	return nil
}

// GetExitFunc returns the exit function
func (c *ImmutableConfig) GetExitFunc() func(int) {
	// Not stored in ImmutableConfig
	return nil
}

// GetPanicFunc returns the panic function
func (c *ImmutableConfig) GetPanicFunc() func(string) {
	return c.PanicFunc
}

// IsMetricsEnabled returns whether metrics are enabled
func (c *ImmutableConfig) IsMetricsEnabled() bool {
	// Not stored in ImmutableConfig - used during build
	return false
}

// IsPoolingEnabled returns whether pooling is enabled
func (c *ImmutableConfig) IsPoolingEnabled() bool {
	// Not stored in ImmutableConfig - used during build
	return false
}

// IsAutoFieldsEnabled returns whether auto fields are enabled
func (c *ImmutableConfig) IsAutoFieldsEnabled() bool {
	// Not stored in ImmutableConfig - used during build
	return false
}

// GetFormatter returns the formatter type
func (c *ImmutableConfig) GetFormatter() string {
	if c.EnableJSON {
		return "json"
	}
	return "text"
}

// GetFileOutput returns the file output configuration
func (c *ImmutableConfig) GetFileOutput() *interfaces.FileOutputConfig {
	// Not stored in ImmutableConfig
	return nil
}

// GetConsoleOutput returns the console output target
func (c *ImmutableConfig) GetConsoleOutput() string {
	// Not stored in ImmutableConfig
	return ""
}

// GetAsyncBuffer returns the async buffer configuration
func (c *ImmutableConfig) GetAsyncBuffer() *interfaces.AsyncBufferConfig {
	// Not stored in ImmutableConfig
	return nil
}

// IsAsyncEnabled returns whether async mode is enabled
func (c *ImmutableConfig) IsAsyncEnabled() bool {
	return c.EnableAsync
}

// GetPIIMasker returns the PII masker
func (c *ImmutableConfig) GetPIIMasker() interfaces.PIIMasker {
	return c.Masker
}

// GetSampler returns the sampler
func (c *ImmutableConfig) GetSampler() interfaces.Sampler {
	// types.Sampler and interfaces.Sampler have different method signatures
	// Return nil for now as they are not directly compatible
	return nil
}
