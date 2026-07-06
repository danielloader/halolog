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

package interfaces

import (
	"time"

	"github.com/go-gen-ecosystem/halolog/types"
)

// Config interface for logger configuration
// NOTE: This interface contains ONLY getters, ensuring immutability for atomic.Pointer usage.
type Config interface {
	GetLevel() types.LogLevel
	GetEnvironment() string
	IsColorized() bool
	IsPrettyPrint() bool
	GetAdapters() []Adapter
	GetMaskingRules() []types.MaskingRule
	GetPolicyRules() []PolicyRule
	GetDebugBufferSize() int
	GetMaxDebugHistory() int
	IsLineInDebug() bool
	IsLineInError() bool
	IsVerbose() bool
	GetErrorHandler() ErrorHandler
	GetExitFunc() func(int)
	GetPanicFunc() func(string)

	// Integration methods
	IsMetricsEnabled() bool
	IsPoolingEnabled() bool
	IsAutoFieldsEnabled() bool

	// Logging-focused configuration getters
	GetFormatter() string
	GetFileOutput() *FileOutputConfig
	GetConsoleOutput() string
	GetAsyncBuffer() *AsyncBufferConfig
	IsAsyncEnabled() bool

	// Performance optimization getters
	GetPIIMasker() PIIMasker
	GetSampler() Sampler
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

// Sampler determines whether a log entry should be sampled
type Sampler interface {
	ShouldSample(level types.LogLevel) bool
}

// PIIMasker applies PII masking to log entries
type PIIMasker interface {
	Apply(entry *types.LogEntry)
}

// Adapter represents a log output adapter
type Adapter interface {
	Write(entry *types.LogEntry) error
	Close() error
}
