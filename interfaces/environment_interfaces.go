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

import "github.com/go-gen-ecosystem/halolog/types"

// EnvironmentProvider provides environment detection and configuration
type EnvironmentProvider interface {
	GetEnvironment() types.EnvironmentType
	IsDevelopment() bool
	IsTesting() bool
	IsStaging() bool
	IsProduction() bool
}

// EnvironmentFilter provides environment-based log filtering
type EnvironmentFilter interface {
	ShouldFilter(level types.LogLevel) bool
	GetFilterLevel() types.LogLevel
	SetFilterLevel(level types.LogLevel)
	GetEnvironment() types.EnvironmentType
}

// EnvironmentManager combines environment detection and filtering
type EnvironmentManager interface {
	EnvironmentProvider
	EnvironmentFilter
	ConfigureForEnvironment(env types.EnvironmentType)
}

// EnvironmentDetector interface for environment detection (High-performance)
type EnvironmentDetector interface {
	IsDevelopment() bool
	IsTesting() bool
	IsStaging() bool
	IsProduction() bool
}

// ConfigurableEnvironment interface combining filter and detector with configuration (High-performance)
type ConfigurableEnvironment interface {
	EnvironmentFilter
	EnvironmentDetector
	SetEnvironment(env types.EnvironmentType)
}
