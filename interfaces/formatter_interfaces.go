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
	"github.com/go-gen-ecosystem/halolog/types"
)

// FormatterFactory creates formatters based on configuration
type FormatterFactory interface {
	CreateFormatter(config FormatterConfig) types.Formatter
}

// FormatterConfig holds formatter configuration
type FormatterConfig struct {
	Type          string // "json" or "text"
	PrettyPrint   bool
	TimeFormat    string
	IncludeLevel  bool
	IncludeCaller bool
	ColorEnabled  bool
}

// DefaultFormatterFactory provides default formatter creation
type DefaultFormatterFactory struct{}

// CreateFormatter creates a formatter based on config
func (f *DefaultFormatterFactory) CreateFormatter(config FormatterConfig) types.Formatter {
	// Return default console formatter
	// Actual JSON/Text formatters are in adapters/formatters/ package
	return types.NewDefaultConsoleFormatter()
}
