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

func TestDefaultConfig(t *testing.T) {
	builder := NewConfig()
	if builder == nil {
		t.Fatal("NewConfig() returned nil")
	}

	config := builder.Build()
	if config == nil {
		t.Fatal("Build() returned nil")
	}

	if config.Level == 0 {
		t.Error("Default config should have non-zero level")
	}

	if !config.EnableColorized {
		t.Error("Default config should be colorized")
	}

	if !config.EnablePrettyPrint {
		t.Error("Default config should be pretty printed")
	}
}

func TestProductionDefaults(t *testing.T) {
	builder := ProductionDefaults()
	if builder == nil {
		t.Fatal("ProductionDefaults() returned nil")
	}

	// Build the config to test the actual values
	config := builder.Build()
	if config == nil {
		t.Fatal("Build() returned nil")
	}

	// Test that production defaults are set correctly
	if config.Level != types.WarnLevel {
		t.Errorf("Expected production level to be WarnLevel, got %v", config.Level)
	}

	if config.EnableColorized {
		t.Error("Expected production colorized to be false")
	}

	if config.EnablePrettyPrint {
		t.Error("Expected production pretty print to be false")
	}
}

func TestDevelopmentDefaults(t *testing.T) {
	builder := DevelopmentDefaults()
	if builder == nil {
		t.Fatal("DevelopmentDefaults() returned nil")
	}

	// Build the config to test the actual values
	config := builder.Build()
	if config == nil {
		t.Fatal("Build() returned nil")
	}

	// Test that development defaults are set correctly
	if config.Level != types.DebugLevel {
		t.Errorf("Expected development level to be DebugLevel, got %v", config.Level)
	}

	if !config.EnableColorized {
		t.Error("Expected development colorized to be true")
	}

	if !config.EnablePrettyPrint {
		t.Error("Expected development pretty print to be true")
	}
}
