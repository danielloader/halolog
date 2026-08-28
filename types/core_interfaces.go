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
	"time"
)

// NOTE: a 60-method `CoreLogger` god-interface used to live here, promising
// an API (Colorize, Tracef, ToFile, WithUserID, …) that nothing implemented
// and nothing referenced. It was removed outright — the real public surface
// is the concrete core.Logger and its fluent builders; adapters and
// transforms are typed by the small, purpose-built interfaces in this
// package (Adapter, Formatter, Sampler, PIIMasker, …).

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
