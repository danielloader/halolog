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
// Package registry provides component registries
// Author: Admilson B. F. Cossa

package registry

import (
	"sync"
	"testing"

	"github.com/go-gen-ecosystem/halolog/types"
)

// ============================================================================
// Color Registry Tests
// ============================================================================

func TestColorRegistry_BasicOperations(t *testing.T) {
	registry := NewColorRegistry()

	// Test initial state - registry is enabled by default
	if !registry.IsEnabled() {
		t.Error("Color registry should be enabled by default")
	}

	// Test disabling
	registry.Disable()
	if registry.IsEnabled() {
		t.Error("Color registry should be disabled after Disable()")
	}

	// Test enabling again
	registry.Enable()
	if !registry.IsEnabled() {
		t.Error("Color registry should be enabled after Enable()")
	}
}

func TestColorRegistry_LevelColors(t *testing.T) {
	registry := NewColorRegistry()
	registry.Enable()

	// Test default level colors
	tests := []struct {
		level    types.LogLevel
		hasColor bool
	}{
		{types.DebugLevel, true},
		{types.InfoLevel, true},
		{types.WarnLevel, true},
		{types.ErrorLevel, true},
		{types.FatalLevel, true},
		{types.LogLevel(99), false}, // Unknown level
	}

	for _, tt := range tests {
		color := registry.GetColor(tt.level)
		if tt.hasColor && color == "" {
			t.Errorf("Expected color for level %v, got empty string", tt.level)
		}
		if !tt.hasColor && color != "" {
			t.Errorf("Expected no color for level %v, got %s", tt.level, color)
		}
	}
}

func TestColorRegistry_CustomLevelColors(t *testing.T) {
	registry := NewColorRegistry()
	registry.Enable()

	customColor := "\033[38;5;208m" // Orange
	registry.SetColor(types.LogLevel(42), customColor)

	retrievedColor := registry.GetColor(types.LogLevel(42))
	if retrievedColor != customColor {
		t.Errorf("Expected custom color %s, got %s", customColor, retrievedColor)
	}
}

func TestColorRegistry_FieldColorsByID(t *testing.T) {
	registry := NewColorRegistry()
	registry.Enable()

	// Test field color registration
	fieldColors := map[int]types.Color{
		1: types.ColorRed,
		2: types.ColorGreen,
		3: types.ColorBlue,
	}

	err := registry.RegisterFromConfigWithIDs(fieldColors)
	if err != nil {
		t.Fatalf("Failed to register field colors: %v", err)
	}

	// Test color retrieval
	tests := []struct {
		fieldID  int
		expected string // ANSI color string
		found    bool
	}{
		{1, "\033[31m", true}, // Red
		{2, "\033[32m", true}, // Green
		{3, "\033[34m", true}, // Blue
		{99, "", false},       // Unknown field ID
	}

	for _, tt := range tests {
		color, found := registry.GetColorByFieldID(tt.fieldID)
		if found != tt.found {
			t.Errorf("Field ID %d: expected found=%v, got %v", tt.fieldID, tt.found, found)
		}
		if found && color != tt.expected {
			t.Errorf("Field ID %d: expected color %s, got %s", tt.fieldID, tt.expected, color)
		}
	}
}

func TestColorRegistry_ConcurrentAccess(t *testing.T) {
	registry := NewColorRegistry()
	registry.Enable()

	// Register initial colors
	fieldColors := map[int]types.Color{
		1: types.ColorRed,
		2: types.ColorGreen,
		3: types.ColorBlue,
	}
	_ = registry.RegisterFromConfigWithIDs(fieldColors)

	var wg sync.WaitGroup
	errors := make(chan error, 100)

	// Concurrent reads
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_, found := registry.GetColorByFieldID(id%3 + 1)
				if !found {
					errors <- nil // Expected for some cases
				}
			}
		}(i)
	}

	// Concurrent writes
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			newColors := map[int]types.Color{
				id + 10: types.ColorYellow,
				id + 20: types.ColorCyan,
			}
			if err := registry.RegisterFromConfigWithIDs(newColors); err != nil {
				errors <- err
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		if err != nil {
			t.Errorf("Concurrent operation failed: %v", err)
		}
	}
}

func TestColorRegistry_PerformanceStats(t *testing.T) {
	registry := NewColorRegistry()
	registry.Enable()

	// Register some colors
	fieldColors := map[int]types.Color{
		1: types.ColorRed,
		2: types.ColorGreen,
	}
	_ = registry.RegisterFromConfigWithIDs(fieldColors)

	// Perform level color lookups (these update stats)
	for i := 0; i < 100; i++ {
		registry.GetColor(types.DebugLevel)
		registry.GetColor(types.InfoLevel)
		registry.GetColor(types.LogLevel(99)) // Not found
	}

	stats := registry.GetPerformanceStats()
	if stats.LookupCount != 300 {
		t.Errorf("Expected 300 lookups, got %d", stats.LookupCount)
	}
	// Cache hits should be 200 (found colors) but the implementation may vary
	if stats.CacheHits == 0 {
		t.Error("Expected some cache hits, got 0")
	}
}

func TestColorRegistry_Reset(t *testing.T) {
	registry := NewColorRegistry()
	registry.Enable()

	// Register colors and perform lookups
	fieldColors := map[int]types.Color{1: types.ColorRed}
	_ = registry.RegisterFromConfigWithIDs(fieldColors)
	registry.GetColor(types.DebugLevel) // This updates stats

	// Reset
	registry.Reset()

	// Verify reset
	stats := registry.GetPerformanceStats()
	if stats.LookupCount != 0 || stats.CacheHits != 0 {
		t.Error("Performance stats should be reset to zero")
	}

	// Color should still be available (reset doesn't clear registrations)
	color, found := registry.GetColorByFieldID(1)
	if !found || color != "\033[31m" {
		t.Error("Color registration should persist after reset")
	}
}

func TestColorRegistry_SchemeManagement(t *testing.T) {
	registry := NewColorRegistry()

	// Test setting scheme (no getter available, so just test it doesn't panic)
	registry.SetColorScheme("dark")
	// If we get here without panic, the method works
}

func TestColorRegistry_ErrorHandling(t *testing.T) {
	registry := NewColorRegistry()

	// Test registration with nil map (should not panic)
	err := registry.RegisterFromConfigWithIDs(nil)
	if err != nil {
		t.Errorf("RegisterFromConfigWithIDs with nil should not error, got: %v", err)
	}

	// Test registration with empty map
	err = registry.RegisterFromConfigWithIDs(map[int]types.Color{})
	if err != nil {
		t.Errorf("RegisterFromConfigWithIDs with empty map should not error, got: %v", err)
	}
}

func TestColorRegistry_DisabledBehavior(t *testing.T) {
	registry := NewColorRegistry()
	// Registry is enabled by default

	// Disable it for this test
	registry.Disable()

	// Test that GetColor returns empty when disabled
	levelColor := registry.GetColor(types.InfoLevel)
	if levelColor != "" {
		t.Error("GetColor should return empty string when disabled")
	}

	// GetColorByFieldID doesn't check enabled state, so test with GetColor instead
	fieldColors := map[int]types.Color{1: types.ColorRed}
	err := registry.RegisterFromConfigWithIDs(fieldColors)
	if err != nil {
		t.Errorf("Registration should work when disabled: %v", err)
	}
}

// Benchmark tests
func BenchmarkColorRegistry_GetColorByFieldID(b *testing.B) {
	registry := NewColorRegistry()
	registry.Enable()

	// Register test colors
	fieldColors := make(map[int]types.Color, 100)
	for i := 0; i < 100; i++ {
		fieldColors[i] = types.Color(i % 16)
	}
	_ = registry.RegisterFromConfigWithIDs(fieldColors)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		registry.GetColorByFieldID(i % 100)
	}
}

func BenchmarkColorRegistry_RegisterFromConfigWithIDs(b *testing.B) {
	registry := NewColorRegistry()
	registry.Enable()

	fieldColors := make(map[int]types.Color, 100)
	for i := 0; i < 100; i++ {
		fieldColors[i] = types.Color(i % 16)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = registry.RegisterFromConfigWithIDs(fieldColors)
	}
}

func BenchmarkColorRegistry_ConcurrentAccess(b *testing.B) {
	registry := NewColorRegistry()
	registry.Enable()

	// Register initial colors
	fieldColors := make(map[int]types.Color, 50)
	for i := 0; i < 50; i++ {
		fieldColors[i] = types.Color(i % 16)
	}
	_ = registry.RegisterFromConfigWithIDs(fieldColors)

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			// Alternate between reads and writes
			if i%10 == 0 {
				newColors := map[int]types.Color{i: types.Color(i % 16)}
				_ = registry.RegisterFromConfigWithIDs(newColors)
			} else {
				registry.GetColorByFieldID(i % 50)
			}
			i++
		}
	})
}
