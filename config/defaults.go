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

import "github.com/go-gen-ecosystem/halolog/types"

// DefaultConfig returns a ConfigBuilder with default settings
func DefaultConfig() *ConfigBuilder {
	return NewConfig().
		WithEnv("development").
		WithLevel(types.InfoLevel).
		WithColorized(true).
		WithPrettyPrint(true)
}

// ProductionDefaults returns a ConfigBuilder with production-optimized settings
func ProductionDefaults() *ConfigBuilder {
	builder := NewConfig().
		WithEnv("production").
		WithLevel(types.WarnLevel).
		WithColorized(false).
		WithPrettyPrint(false)
	// Mark that this level comes from defaults, not builder pattern
	builder.levelFromDefaults = true
	return builder
}

// DevelopmentDefaults returns a ConfigBuilder with development-optimized settings
func DevelopmentDefaults() *ConfigBuilder {
	builder := NewConfig().
		WithEnv("development").
		WithLevel(types.DebugLevel).
		WithColorized(true).
		WithPrettyPrint(true)
	// Mark that this level comes from defaults, not builder pattern
	builder.levelFromDefaults = true
	return builder
}
