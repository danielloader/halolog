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

import (
	"fmt"
	"testing"

	"github.com/go-gen-ecosystem/halolog/types"
)

func TestDebugStartupConfig(t *testing.T) {
	startupConfig := &StartupConfig{
		CustomFields: []string{"user_id", "request_id", "response_time"},
		SensitiveFields: []SensitiveFieldConfig{
			{Name: "api_key", FieldID: 100, Mask: "***API_KEY***"},
			{Name: "token", FieldID: 101, Mask: "***TOKEN***"},
		},
		FieldColors: map[string]types.Color{
			"error":   types.ColorRed,
			"success": types.ColorGreen,
		},
		OptimalBufferSize: 48,
	}

	builder := NewConfig().
		WithStartupRegistration(startupConfig)

	config := builder.Build()

	fmt.Printf("RegisteredFieldIDs: %v\n", config.RegisteredFieldIDs)
	fmt.Printf("SensitiveFieldMap: %v\n", config.SensitiveFieldMap)
	fmt.Printf("FieldColorMap: %v\n", config.FieldColorMap)
	fmt.Printf("StartupRegistration.CustomFields: %v\n", config.StartupRegistration.CustomFields)
	fmt.Printf("StartupRegistration.SensitiveFields: %v\n", config.StartupRegistration.SensitiveFields)
	fmt.Printf("StartupRegistration.FieldColors: %v\n", config.StartupRegistration.FieldColors)
}
