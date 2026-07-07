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
// Package core provides core logging functionality
// Author: Admilson B. F. Cossa

package types

import (
	"context"
	"time"
)

// CoreLogger defines the interface for the core logger implementation.
// This interface provides the complete API surface for logging operations.
//
//nolint:interfacebloat // Complete hot-path logging API surface; splitting would fragment the public contract.
type CoreLogger interface {
	// Core logging methods
	Trace(msg string) CoreLogger
	Debug(msg string) CoreLogger
	Info(msg string) CoreLogger
	Warn(msg string) CoreLogger
	Error(msg string, err error) CoreLogger
	Fatal(msg string) CoreLogger
	Panic(msg string) CoreLogger

	// Context-aware logging
	TraceContext(ctx context.Context, msg string) CoreLogger
	DebugContext(ctx context.Context, msg string) CoreLogger
	InfoContext(ctx context.Context, msg string) CoreLogger
	WarnContext(ctx context.Context, msg string) CoreLogger
	ErrorContext(ctx context.Context, msg string, err error) CoreLogger
	FatalContext(ctx context.Context, msg string) CoreLogger
	PanicContext(ctx context.Context, msg string) CoreLogger

	// Field builders - fluent API
	WithField(key string, value interface{}) CoreLogger
	WithFields(fields map[string]interface{}) CoreLogger
	WithTypedField(field TypedFieldData) CoreLogger
	WithTypedFields(fields []TypedFieldData) CoreLogger
	WithStyledFields(fields ...*StyledField) CoreLogger
	WithError(err error) CoreLogger
	WithAutoComponent() CoreLogger

	// Field management
	WithStrategy(strategy FieldStrategy) CoreLogger
	WithCaller(skip int) CoreLogger
	WithCallerSkip(skip int) CoreLogger
	WithContext(ctx context.Context) CoreLogger
	WithUserID(userID interface{}) CoreLogger
	WithRequestID(requestID string) CoreLogger
	WithDuration(duration time.Duration) CoreLogger

	// Masking and security
	MaskField(fieldName string) CoreLogger
	MaskFields(fieldNames ...string) CoreLogger
	AutoMask() CoreLogger

	// Sampling and rate limiting
	SampleSuccess(rate float64) CoreLogger
	WithSampling(rate float64, filter func(*LogEntry) bool) CoreLogger
	WithRateLimit(maxPerSecond int) CoreLogger

	// Styling and formatting
	Colorize() CoreLogger
	ColorizeWith(options ColorOptions) CoreLogger
	PrettyPrint() CoreLogger
	PrettyPrintWith(options PrettyOptions) CoreLogger
	HighlightField(fieldName string) CoreLogger

	// Formatted logging
	Tracef(format string, args ...interface{}) CoreLogger
	Debugf(format string, args ...interface{}) CoreLogger
	Infof(format string, args ...interface{}) CoreLogger
	Warnf(format string, args ...interface{}) CoreLogger
	Errorf(format string, args ...interface{}) CoreLogger
	Fatalf(format string, args ...interface{}) CoreLogger
	Panicf(format string, args ...interface{}) CoreLogger

	// Output destinations
	ToConsole() CoreLogger
	ToFile(path string) CoreLogger
	ToFileAsync(path string, options ...OutputOption) CoreLogger
	ToMultiple(outputs ...Output) CoreLogger

	// Metrics integration
	WithMetric(name string, value interface{}, tags ...string) CoreLogger

	// Features - all included in single interface
	WithHook(hook LogHook) CoreLogger
	AddOutput(output LogOutput) CoreLogger
	RemoveOutput(output LogOutput) CoreLogger

	// Configuration (aligned with unified Logger)
	SetLevel(level LogLevel)
	GetLevel() LogLevel
	IsLevelEnabled(level LogLevel) bool
	SetComponent(component string)
	GetName() string

	// Engine introspection (aligned with unified Logger)
	GetEngine() interface{} // Returns internal engine (implementation specific)
}

// LogHook represents a logging hook interface
type LogHook interface {
	Fire(entry LogEntry) error
	Levels() []LogLevel
}

// LogOutput represents a log output interface
type LogOutput interface {
	Write(entry LogEntry) error
}

// CoreAdapter interface for internal adapter implementations
// This is now just an alias for Adapter to maintain backward compatibility
type CoreAdapter = Adapter

// Filter interface for log filtering
type Filter interface {
	ShouldLog(level LogLevel, message string) bool
}

// FieldStrategy interface for field processing strategies
type FieldStrategy interface {
	ProcessField(field *TypedFieldData) *TypedFieldData
	ProcessFields(fields []TypedFieldData) []TypedFieldData
}

// EnvironmentDetector interface for environment detection
type EnvironmentDetector interface {
	DetectEnvironment() EnvironmentType
	IsDevelopment() bool
	IsProduction() bool
	IsTesting() bool
}

// PIIMasker interface for PII masking operations.
//
//nolint:interfacebloat // Cohesive masking contract: apply + field/string masking + rule/pattern management.
type PIIMasker interface {
	Apply(entry *LogEntry)
	MaskField(field *TypedFieldData) *TypedFieldData
	MaskFields(fields []TypedFieldData) []TypedFieldData
	MaskString(input string) string
	AddRule(pattern string, replace string, ruleType string) error
	RemoveRule(pattern string) error
	AddPattern(name, patternStr, mask string) error
	RemovePattern(name string)
	GetPatterns() []string
	Clone() PIIMasker
}

// RetryManager interface for retry logic management
type RetryManager interface {
	ShouldRetry(error) bool
	NextBackoff(attempt int) time.Duration
	MaxRetries() int
}

// Sampler interface for log sampling
type Sampler interface {
	ShouldSample(entry *LogEntry) bool
	GetRate() float64
	SetRate(rate float64)
}

// type Sampler interface {
// 	ShouldSample(entry *LogEntry) bool
// 	GetRate() float64
// 	SetRate(rate float64)
// }

// OutputOption configures output behavior (function type)
type OutputOption func(interface{})
