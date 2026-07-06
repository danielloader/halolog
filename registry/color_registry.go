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
	"strings"
	"sync"

	"github.com/go-gen-ecosystem/halolog/types"
)

// ColorRegistry provides O(1) color lookup for log levels
// Optimized for High performance with pre-computed mappings
type ColorRegistry struct {
	mu sync.RWMutex

	// O(1) lookup arrays (faster than maps for small ranges)
	levelColors    [256]string // Direct array lookup for log levels
	levelColors256 [256]string // Extended range for custom levels

	// Fallback map for custom level names
	customColors map[string]string

	// O(1) lookup for field colors by field ID (startup registration)
	fieldColorsByID map[int]string

	// Color scheme configuration
	colorScheme string // "default", "dark", "light", "custom"
	enabled     bool   // Global color enable/disable

	// Performance metrics
	lookupCount int64
	cacheHits   int64
}

// NewColorRegistry creates a new color registry with pre-configured colors
func NewColorRegistry() *ColorRegistry {
	registry := &ColorRegistry{
		customColors:    make(map[string]string, 32),
		fieldColorsByID: make(map[int]string, 64), // Pre-allocate for field colors
		colorScheme:     "default",
		enabled:         true,
	}

	// Initialize default color mappings
	registry.initializeDefaultColors()

	return registry
}

// initializeDefaultColors sets up pre-configured color mappings
// Based on industry standard logging standards and accessibility guidelines
func (r *ColorRegistry) initializeDefaultColors() {
	// Standard log level colors (ANSI escape codes)
	// Using high-contrast colors for accessibility and readability

	// Default scheme colors
	r.levelColors[types.DebugLevel] = "\033[90m" // Bright black (gray)
	r.levelColors[types.InfoLevel] = "\033[32m"  // Green
	r.levelColors[types.WarnLevel] = "\033[33m"  // Yellow
	r.levelColors[types.ErrorLevel] = "\033[31m" // Red
	r.levelColors[types.FatalLevel] = "\033[35m" // Magenta

	// Dark scheme colors (for dark terminals)
	r.levelColors256[types.DebugLevel] = "\033[38;5;250m" // Light gray
	r.levelColors256[types.InfoLevel] = "\033[38;5;46m"   // Bright green
	r.levelColors256[types.WarnLevel] = "\033[38;5;226m"  // Bright yellow
	r.levelColors256[types.ErrorLevel] = "\033[38;5;196m" // Bright red
	r.levelColors256[types.FatalLevel] = "\033[38;5;201m" // Bright magenta
}

// GetColor returns the color for a log level (O(1) lookup)
// This is the primary fast path for color assignment
func (r *ColorRegistry) GetColor(level types.LogLevel) string {
	r.lookupCount++

	if !r.enabled {
		return "" // No colors when disabled
	}

	// High-performance path: Direct array lookup (O(1))
	if level < 256 {
		r.cacheHits++

		// Use appropriate color scheme
		if r.colorScheme == "dark" {
			return r.levelColors256[level]
		}
		return r.levelColors[level]
	}

	// Fallback: Custom level lookup (slower but still O(1) for small maps)
	return r.getCustomColor(level)
}

// getCustomColor handles custom log levels (fallback path)
func (r *ColorRegistry) getCustomColor(level types.LogLevel) string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Convert level to string for map lookup
	levelStr := r.levelToString(level)
	if color, exists := r.customColors[levelStr]; exists {
		return color
	}

	// Default to no color for unknown levels
	return ""
}

// levelToString converts log level to string representation
func (r *ColorRegistry) levelToString(level types.LogLevel) string {
	// Common level mappings
	switch level {
	case types.DebugLevel:
		return "DEBUG"
	case types.InfoLevel:
		return "INFO"
	case types.WarnLevel:
		return "WARN"
	case types.ErrorLevel:
		return "ERROR"
	case types.FatalLevel:
		return "FATAL"
	default:
		// For custom levels, use numeric representation
		return string(rune('0' + level))
	}
}

// SetColor sets a custom color for a specific log level
func (r *ColorRegistry) SetColor(level types.LogLevel, color string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Direct array assignment for fast levels
	if level < 256 {
		r.levelColors[level] = color
		r.levelColors256[level] = color
		return
	}

	// Map assignment for custom levels
	levelStr := r.levelToString(level)
	r.customColors[levelStr] = color
}

// SetColorScheme sets the global color scheme
func (r *ColorRegistry) SetColorScheme(scheme string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.colorScheme = scheme

	// Update colors based on scheme
	switch scheme {
	case "dark":
		r.applyDarkScheme()
	case "light":
		r.applyLightScheme()
	case "default":
		r.initializeDefaultColors()
	}
}

// applyDarkScheme applies colors optimized for dark terminals
func (r *ColorRegistry) applyDarkScheme() {
	r.levelColors[types.DebugLevel] = "\033[38;5;250m" // Light gray
	r.levelColors[types.InfoLevel] = "\033[38;5;46m"   // Bright green
	r.levelColors[types.WarnLevel] = "\033[38;5;226m"  // Bright yellow
	r.levelColors[types.ErrorLevel] = "\033[38;5;196m" // Bright red
	r.levelColors[types.FatalLevel] = "\033[38;5;201m" // Bright magenta
}

// applyLightScheme applies colors optimized for light terminals
func (r *ColorRegistry) applyLightScheme() {
	r.levelColors[types.DebugLevel] = "\033[38;5;240m" // Dark gray
	r.levelColors[types.InfoLevel] = "\033[38;5;22m"   // Dark green
	r.levelColors[types.WarnLevel] = "\033[38;5;130m"  // Dark orange
	r.levelColors[types.ErrorLevel] = "\033[38;5;124m" // Dark red
	r.levelColors[types.FatalLevel] = "\033[38;5;91m"  // Dark magenta
}

// Enable enables color output globally
func (r *ColorRegistry) Enable() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.enabled = true
}

// Disable disables color output globally
func (r *ColorRegistry) Disable() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.enabled = false
}

// IsEnabled returns whether colors are currently enabled
func (r *ColorRegistry) IsEnabled() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.enabled
}

// GetPerformanceStats returns registry performance statistics
func (r *ColorRegistry) GetPerformanceStats() ColorStats {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return ColorStats{
		LookupCount: r.lookupCount,
		CacheHits:   r.cacheHits,
		HitRate:     float64(r.cacheHits) / float64(r.lookupCount) * 100,
		Enabled:     r.enabled,
		Scheme:      r.colorScheme,
	}
}

// Reset clears all custom colors and resets to defaults
func (r *ColorRegistry) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Clear custom colors
	r.customColors = make(map[string]string, 32)

	// Reset to default scheme
	r.colorScheme = "default"
	r.enabled = true

	// Reinitialize default colors
	r.initializeDefaultColors()

	// Reset metrics
	r.lookupCount = 0
	r.cacheHits = 0
}

// ColorStats contains performance statistics for the color registry
type ColorStats struct {
	LookupCount int64
	CacheHits   int64
	HitRate     float64
	Enabled     bool
	Scheme      string
}

// GlobalColorRegistry provides a singleton instance for system-wide use
var GlobalColorRegistry = NewColorRegistry()

// ConfigureColorsFromConfig configures colors from application configuration
func ConfigureColorsFromConfig(config map[string]interface{}) {
	if config == nil {
		return
	}

	// Enable/disable colors
	if enabled, ok := config["colors_enabled"].(bool); ok {
		if enabled {
			GlobalColorRegistry.Enable()
		} else {
			GlobalColorRegistry.Disable()
		}
	}

	// Set color scheme
	if scheme, ok := config["color_scheme"].(string); ok {
		GlobalColorRegistry.SetColorScheme(scheme)
	}

	// Set custom colors
	if customColors, ok := config["custom_colors"].(map[string]interface{}); ok {
		for level, color := range customColors {
			if colorStr, ok := color.(string); ok {
				// Convert level string to LogLevel
				var levelInt types.LogLevel
				switch strings.ToUpper(level) {
				case "DEBUG":
					levelInt = types.DebugLevel
				case "INFO":
					levelInt = types.InfoLevel
				case "WARN":
					levelInt = types.WarnLevel
				case "ERROR":
					levelInt = types.ErrorLevel
				case "FATAL":
					levelInt = types.FatalLevel
				default:
					continue
				}
				GlobalColorRegistry.SetColor(levelInt, colorStr)
			}
		}
	}
}

// RegisterFromConfig registers field colors from startup configuration
func (r *ColorRegistry) RegisterFromConfig(fieldColors map[string]types.Color) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if fieldColors == nil {
		return nil
	}

	// Register each field color mapping
	for fieldName, color := range fieldColors {
		// Convert color to ANSI string
		colorStr := r.colorToANSI(color)
		if colorStr != "" {
			r.customColors[fieldName] = colorStr
		}
	}

	return nil
}

// RegisterFromConfigWithIDs registers field colors with field IDs for O(1) lookup
func (r *ColorRegistry) RegisterFromConfigWithIDs(fieldColorsByID map[int]types.Color) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if fieldColorsByID == nil {
		return nil
	}

	// Register each field color mapping by field ID
	for fieldID, color := range fieldColorsByID {
		// Convert color to ANSI string
		colorStr := r.colorToANSI(color)
		if colorStr != "" {
			r.fieldColorsByID[fieldID] = colorStr
		}
	}

	return nil
}

// GetColorByFieldID returns color for a field by its ID (O(1) lookup)
func (r *ColorRegistry) GetColorByFieldID(fieldID int) (string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// O(1) direct map lookup by field ID
	if color, exists := r.fieldColorsByID[fieldID]; exists {
		return color, true
	}

	return "", false
}

// colorToANSI converts types.Color to ANSI escape code
func (r *ColorRegistry) colorToANSI(color types.Color) string {
	switch color {
	case types.ColorRed:
		return "\033[31m"
	case types.ColorGreen:
		return "\033[32m"
	case types.ColorYellow:
		return "\033[33m"
	case types.ColorBlue:
		return "\033[34m"
	case types.ColorMagenta:
		return "\033[35m"
	case types.ColorCyan:
		return "\033[36m"
	case types.ColorWhite:
		return "\033[37m"
	case types.ColorBlack:
		return "\033[30m"
	default:
		return ""
	}
}
