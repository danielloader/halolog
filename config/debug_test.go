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

	t.Logf("RegisteredFieldIDs: %v", config.RegisteredFieldIDs)
	t.Logf("SensitiveFieldMap: %v", config.SensitiveFieldMap)
	t.Logf("FieldColorMap: %v", config.FieldColorMap)
	t.Logf("StartupRegistration.CustomFields: %v", config.StartupRegistration.CustomFields)
	t.Logf("StartupRegistration.SensitiveFields: %v", config.StartupRegistration.SensitiveFields)
	t.Logf("StartupRegistration.FieldColors: %v", config.StartupRegistration.FieldColors)
}
