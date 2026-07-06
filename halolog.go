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

// Package halolog provides API for halolog
// Author: Admilson B. F. Cossa
// Provides global singleton access to loggers

package halolog

import (
	"sync"

	"github.com/go-gen-ecosystem/halolog/config"
	"github.com/go-gen-ecosystem/halolog/core"
	"github.com/go-gen-ecosystem/halolog/types"
)

// Logger is an alias for core.Logger to support fluent API type usage
type Logger = *core.Logger

var (
	registryMu sync.Mutex
	registry   = map[string]*core.Logger{}
)

// GetLogger returns a process-wide singleton logger for the given name,
// creating a default one (info level) on first use. It is the simplest entry
// point for applications that just want named logger access.
func GetLogger(name string) *core.Logger {
	registryMu.Lock()
	defer registryMu.Unlock()
	if l, ok := registry[name]; ok {
		return l
	}
	l := core.New().Component(name).MustBuild()
	registry[name] = l
	return l
}

// GetLoggerWithConfig returns a process-wide singleton logger for the given
// name, built from the supplied immutable configuration on first use.
func GetLoggerWithConfig(name string, cfg *config.ImmutableConfig) *core.Logger {
	registryMu.Lock()
	defer registryMu.Unlock()
	if l, ok := registry[name]; ok {
		return l
	}
	coreCfg := core.Config{Component: name}
	if cfg != nil {
		coreCfg.Level = cfg.Level
		coreCfg.Adapters = cfg.Adapters
		coreCfg.EnableMasking = cfg.EnableMasking
		coreCfg.Masker = cfg.Masker
		coreCfg.Sampler = cfg.Sampler
		coreCfg.EnableSampling = cfg.Sampler != nil
		coreCfg.EnableMetrics = cfg.EnableMetrics
	}
	l := core.NewLogger(coreCfg)
	registry[name] = l
	return l
}

// WithAdapters is a convenience function to append adapters to a config builder.
func WithAdapters(adapters ...types.Adapter) func(*config.ConfigBuilder) {
	return func(b *config.ConfigBuilder) {
		for _, adapter := range adapters {
			b.WithOutput(adapter)
		}
	}
}
