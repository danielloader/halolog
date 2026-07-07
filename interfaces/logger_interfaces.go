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
// Package interfaces provides interface definitions
// Author: Admilson B. F. Cossa

package interfaces

import (
	"context"
	"time"

	"github.com/go-gen-ecosystem/halolog/types"
)

// Logger is the unified, high-performance interface for application logging.
// It combines structured logging, legacy formatting, context awareness,
// operational control (sampling/rate-limiting), and output management.
//
// Implementation Note: Methods starting with 'With' should return a generic
// immutable copy of the logger with the new context applied.
//
//nolint:interfacebloat // Single hot-path logging contract; splitting it would force call-site type assertions and break the zero-alloc fluent chain that is HaloLog's core value.
type Logger interface {
	// -------------------------------------------------------------------------
	// Core Structured Logging
	// -------------------------------------------------------------------------

	// Trace logs a message at Trace level with optional typed fields.
	Trace(msg string, fields ...types.TypedFieldData)
	// Debug logs a message at Debug level with optional typed fields.
	Debug(msg string, fields ...types.TypedFieldData)
	// Info logs a message at Info level with optional typed fields.
	Info(msg string, fields ...types.TypedFieldData)
	// Warn logs a message at Warn level with optional typed fields.
	Warn(msg string, fields ...types.TypedFieldData)
	// Error logs a message at Error level with optional typed fields.
	Error(msg string, fields ...types.TypedFieldData)
	// Fatal logs a message at Fatal level and calls os.Exit(1).
	Fatal(msg string, fields ...types.TypedFieldData)
	// Panic logs a message at Panic level and calls panic().
	Panic(msg string, fields ...types.TypedFieldData)

	// -------------------------------------------------------------------------
	// Core Context-Aware Logging
	// -------------------------------------------------------------------------

	// TraceContext logs at Trace level, extracting trace IDs/spans from context.
	TraceContext(ctx context.Context, msg string, fields ...types.TypedFieldData)
	// DebugContext logs at Debug level, extracting trace IDs/spans from context.
	DebugContext(ctx context.Context, msg string, fields ...types.TypedFieldData)
	// InfoContext logs at Info level, extracting trace IDs/spans from context.
	InfoContext(ctx context.Context, msg string, fields ...types.TypedFieldData)
	// WarnContext logs at Warn level, extracting trace IDs/spans from context.
	WarnContext(ctx context.Context, msg string, fields ...types.TypedFieldData)
	// ErrorContext logs at Error level, extracting trace IDs/spans from context.
	ErrorContext(ctx context.Context, msg string, fields ...types.TypedFieldData)
	// FatalContext logs at Fatal level, extracting trace IDs/spans from context.
	FatalContext(ctx context.Context, msg string, fields ...types.TypedFieldData)
	// PanicContext logs at Panic level, extracting trace IDs/spans from context.
	PanicContext(ctx context.Context, msg string, fields ...types.TypedFieldData)

	// -------------------------------------------------------------------------
	// Legacy Formatted Logging (fmt.Sprintf style)
	// -------------------------------------------------------------------------

	Tracef(format string, args ...interface{})
	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	Fatalf(format string, args ...interface{})
	Panicf(format string, args ...interface{})

	TracefContext(ctx context.Context, format string, args ...interface{})
	DebugfContext(ctx context.Context, format string, args ...interface{})
	InfofContext(ctx context.Context, format string, args ...interface{})
	WarnfContext(ctx context.Context, format string, args ...interface{})
	ErrorfContext(ctx context.Context, format string, args ...interface{})
	FatalfContext(ctx context.Context, format string, args ...interface{})
	PanicfContext(ctx context.Context, format string, args ...interface{})

	// -------------------------------------------------------------------------
	// Error Handling Extensions
	// -------------------------------------------------------------------------

	// ErrorWith logs an error with an explicit error object and message.
	ErrorWith(err error, msg string, fields ...types.TypedFieldData)
	// ErrorWithContext logs an error with context, explicit error object, and message.
	ErrorWithContext(ctx context.Context, err error, msg string, fields ...types.TypedFieldData)

	// -------------------------------------------------------------------------
	// Field Enrichment (Fluent API)
	// -------------------------------------------------------------------------

	// WithField adds a single untyped key-value pair.
	WithField(key string, value interface{}) Logger
	// WithFields adds multiple typed fields from key-value pairs.
	WithFields(fields ...interface{}) Logger
	// WithTypedField adds a single strongly-typed field (zero allocation preference).
	WithTypedField(field types.TypedFieldData) Logger
	// WithTypedFields adds multiple strongly-typed fields.
	WithTypedFields(fields []types.TypedFieldData) Logger
	// WithStyledFields adds fields with specific visual styling preferences.
	WithStyledFields(fields ...*types.StyledField) Logger
	// WithError adds a standard "error" field derived from the error object.
	WithError(err error) Logger

	// -------------------------------------------------------------------------
	// Metadata & Context Management
	// -------------------------------------------------------------------------

	// WithContext embeds a context into the logger instance (useful for middleware).
	WithContext(ctx context.Context) Logger
	// WithStrategy applies a specific field handling strategy (e.g., overwrite vs append).
	WithStrategy(strategy types.FieldStrategy) Logger
	// WithCaller adds source code location (file/line) to the log entry.
	WithCaller(skip int) Logger
	// WithCallerSkip adjusts the stack frame offset for caller reporting.
	WithCallerSkip(skip int) Logger
	// WithAutoComponent attempts to automatically detect the calling component/package.
	WithAutoComponent() Logger
	// WithUserID adds a standard "user_id" field.
	WithUserID(userID interface{}) Logger
	// WithRequestID adds a standard "request_id" field.
	WithRequestID(requestID string) Logger
	// WithDuration adds a standard "duration" field (useful for access logs).
	WithDuration(duration time.Duration) Logger
	// SetComponent manually sets the component name.
	SetComponent(component string)
	// GetName returns the named instance of the logger.
	GetName() string

	// -------------------------------------------------------------------------
	// Security & Privacy (PII/Redaction)
	// -------------------------------------------------------------------------

	// MaskField ensures the values of the specified field key are redacted.
	MaskField(fieldName string) Logger
	// MaskFields ensures values for multiple keys are redacted.
	MaskFields(fieldNames ...string) Logger
	// AutoMask applies global PII masking rules to the current logger.
	AutoMask() Logger

	// -------------------------------------------------------------------------
	// Operational Control (Sampling & Rate Limiting)
	// -------------------------------------------------------------------------

	// SampleSuccess forces sampling rate for non-error logs.
	SampleSuccess(rate float64) Logger
	// WithSampling applies a sampling rate (0.0 to 1.0) to reduce volume.
	WithSampling(rate float64) Logger
	// WithSamplingFilter applies sampling logic based on a custom filter function.
	WithSamplingFilter(rate float64, filter func(*types.LogEntry) bool) Logger
	// WithRateLimit restricts the number of logs logged per second.
	WithRateLimit(maxPerSecond int) Logger

	// Captures debug logs in memory and only writes them if an error occurs later.
	WithFlightRecorder(bufferSize int) Logger

	// 2. Crypto-Privacy
	// Encrypts the field value so it is safe to store but recoverable with a private key.
	WithEncryptedField(key string, value interface{}) Logger

	// 3. Contextual Spans (Metrics + Tracing + Logging)
	// Wraps an operation in a trace span, logs its duration, and adds context.
	TraceOp(name string, fn func(ctxLogger Logger))

	// 4. Smart De-duplication
	// Prevents log flooding by collapsing identical errors.
	// WithSmartSampling(strategy types.SmartSampleStrategy) Logger // TODO: Implement SmartSampleStrategy type

	// -------------------------------------------------------------------------
	// Output & Sinks
	// -------------------------------------------------------------------------

	// ToConsole configures the logger to write to stdout/stderr.
	ToConsole() Logger
	// ToFile configures the logger to write to a specific file path synchronously.
	ToFile(path string) Logger
	// ToFileAsync configures the logger to write to a file asynchronously.
	ToFileAsync(path string, options ...types.OutputOption) Logger
	// ToMultiple sends logs to multiple outputs (fan-out).
	ToMultiple(outputs ...types.LogOutput) Logger
	// AddOutput appends a new output sink.
	AddOutput(output types.LogOutput) Logger
	// RemoveOutput removes an existing output sink.
	RemoveOutput(output types.LogOutput) Logger
	// WithHook attaches a hook to fire on specific log events.
	WithHook(hook types.LogHook) Logger

	// -------------------------------------------------------------------------
	// UX & Formatting
	// -------------------------------------------------------------------------

	// Colorize enables ANSI color output.
	Colorize() Logger
	// ColorizeWith enables ANSI color output with specific options.
	ColorizeWith(options types.ColorOptions) Logger
	// PrettyPrint enables human-readable JSON (multi-line).
	PrettyPrint() Logger
	// PrettyPrintWith enables human-readable JSON with options.
	PrettyPrintWith(options types.PrettyOptions) Logger
	// HighlightField emphasizes a specific field in console output.
	HighlightField(fieldName string) Logger

	// -------------------------------------------------------------------------
	// Observability Integration
	// -------------------------------------------------------------------------

	// WithMetric logs a message and simultaneously emits a metric.
	WithMetric(name string, value interface{}, tags ...string) Logger

	// WithAlert adds a context-aware alert configuration to the logger.
	WithAlert(name string, config types.AlertConfig) Logger

	// -------------------------------------------------------------------------
	// Configuration & Introspection
	// -------------------------------------------------------------------------

	// SetLevel modifies the logging level dynamically.
	SetLevel(level types.LogLevel)
	// GetLevel returns the current logging level.
	GetLevel() types.LogLevel
	// IsLevelEnabled checks if a specific level would be logged.
	IsLevelEnabled(level types.LogLevel) bool
	// GetEngine returns the underlying implementation (e.g., zap, logrus) for casting.
	GetEngine() interface{}
}

// LogHook represents a logging hook interface
type LogHook interface {
	Fire(entry types.LogEntry) error
	Levels() []types.LogLevel
}

// LogOutput represents a log output interface
type LogOutput interface {
	Write(entry types.LogEntry) error
	Close() error
	String() string
}

// LoggerConfig represents the single configuration for the logger
// All configurations in one interface - no separation between basic and extended
//
//nolint:interfacebloat // Cohesive read-only view of one logger configuration value; the getters must stay together so the whole config can be treated as a single immutable snapshot.
type LoggerConfig interface {
	// Basic configuration
	GetLevel() types.LogLevel
	GetOutputs() []types.LogOutput
	GetHooks() []types.LogHook
	GetSamplingRate() float64

	// Performance tuning
	GetBufferSize() int
	GetFlushInterval() int64
	GetBatchSize() int
	GetBatchTimeout() int64

	// Zero-allocation optimizations
	IsFastPathEnabled() bool
	GetFieldPoolSize() int
	GetStaticFieldCount() int

	// Security and compliance
	GetMaskingRules() []types.MaskingRule
	IsAutoMaskEnabled() bool

	// Observability
	IsMetricsEnabled() bool
	GetHealthCheckInterval() int64
}
