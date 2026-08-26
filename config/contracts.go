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

// Package config — configuration-facing contracts. These previously lived in
// a separate `interfaces` package that duplicated several config-owned structs
// and near-duplicated engine contracts from `types`; folding them here gives
// each name one owner: `types` holds the engine vocabulary (types.Sampler is
// the entry-based sampler the logger consults), while `config` holds the
// narrow, read-only views its own builders and loaders traffic in.
// Author: Admilson B. F. Cossa

package config

import (
	"github.com/go-gen-ecosystem/halolog/types"
)

// Config is the read-only view of one immutable configuration value. It
// contains ONLY getters, so a whole configuration can be swapped atomically
// via atomic.Pointer without any caller observing a partial update.
//
//nolint:interfacebloat // Cohesive read-only view of one immutable config value; the getters must live together so the whole config can be swapped atomically.
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

// Sampler is the configuration-level sampling contract: a coarse per-level
// decision. The engine's per-entry sampler is types.Sampler; the two are
// deliberately distinct and the package qualifier keeps them unambiguous.
type Sampler interface {
	ShouldSample(level types.LogLevel) bool
}

// PIIMasker is the narrow masking view configuration consumers need.
type PIIMasker interface {
	Apply(entry *types.LogEntry)
}

// Adapter is the narrow adapter view configuration consumers need.
type Adapter interface {
	Write(entry *types.LogEntry) error
	Close() error
}

// SimpleSampler is a basic level-based sampler implementation.
type SimpleSampler struct {
	rate float64
}

// NewSimpleSampler creates a new simple sampler.
func NewSimpleSampler(rate float64) *SimpleSampler {
	return &SimpleSampler{rate: rate}
}

// ShouldSample determines if a log should be sampled.
func (s *SimpleSampler) ShouldSample(types.LogLevel) bool {
	// Rate-based sampling is a planned refinement; today every level passes.
	return true
}
