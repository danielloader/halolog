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
// @author Admilson B. F. Cossa

package registry

import (
	"testing"

	"github.com/go-gen-ecosystem/halolog/types"
)

// ============================================================================
// ColorRegistry Coverage Tests (branches not exercised by color_registry_test.go)
// ============================================================================

func TestColorRegistry_DarkSchemeGetColor(t *testing.T) {
	r := NewColorRegistry()
	r.SetColorScheme("dark")

	// In dark scheme, GetColor for a level < 256 returns the levelColors256 value.
	got := r.GetColor(types.InfoLevel)
	if got != "\033[38;5;46m" {
		t.Errorf("dark scheme Info color = %q, want bright green 256-color", got)
	}
}

func TestColorRegistry_LightSchemeGetColor(t *testing.T) {
	r := NewColorRegistry()
	r.SetColorScheme("light")

	// Light scheme rewrites levelColors (not levelColors256); default (non-dark)
	// path returns levelColors, so Error should be the dark-red 256-color.
	got := r.GetColor(types.ErrorLevel)
	if got != "\033[38;5;124m" {
		t.Errorf("light scheme Error color = %q, want dark red 256-color", got)
	}
}

func TestColorRegistry_DefaultSchemeReinitialises(t *testing.T) {
	r := NewColorRegistry()
	r.SetColorScheme("light")
	r.SetColorScheme("default")

	// Back to default: Info must be plain green ANSI again.
	if got := r.GetColor(types.InfoLevel); got != "\033[32m" {
		t.Errorf("default scheme Info color = %q, want plain green", got)
	}
}

func TestColorRegistry_UnknownSchemeIsInert(t *testing.T) {
	r := NewColorRegistry()
	// An unrecognised scheme records the name but leaves colours untouched.
	r.SetColorScheme("solarized")
	if got := r.GetColor(types.InfoLevel); got != "\033[32m" {
		t.Errorf("unknown scheme should not change colours, got %q", got)
	}
	stats := r.GetPerformanceStats()
	if stats.Scheme != "solarized" {
		t.Errorf("scheme name should be recorded, got %q", stats.Scheme)
	}
}

func TestColorRegistry_CustomLevelBeyond256(t *testing.T) {
	r := NewColorRegistry()

	// LogLevel is a uint8 in most builds; guard by only running when the type
	// can represent values >= 256. If it cannot, the getCustomColor path is
	// unreachable and this sub-test is a no-op.
	const customLevel = types.LogLevel(300)
	if int(customLevel) < 256 {
		t.Skip("LogLevel cannot represent values >= 256; custom path unreachable")
	}

	// Unknown custom level -> empty string via getCustomColor fallback.
	if got := r.GetColor(customLevel); got != "" {
		t.Errorf("unknown custom level should yield empty color, got %q", got)
	}

	// Registering a color for a >=256 level stores it in the custom map,
	// keyed by levelToString's numeric fallback, and is retrievable.
	custom := "\033[38;5;208m"
	r.SetColor(customLevel, custom)
	if got := r.GetColor(customLevel); got != custom {
		t.Errorf("custom >=256 level color = %q, want %q", got, custom)
	}
}

func TestColorRegistry_ColorToANSI_AllColors(t *testing.T) {
	r := NewColorRegistry()

	// RegisterFromConfig routes every color through colorToANSI and stores the
	// resulting ANSI string keyed by field name in customColors.
	fieldColors := map[string]types.Color{
		"f_red":     types.ColorRed,
		"f_green":   types.ColorGreen,
		"f_yellow":  types.ColorYellow,
		"f_blue":    types.ColorBlue,
		"f_magenta": types.ColorMagenta,
		"f_cyan":    types.ColorCyan,
		"f_white":   types.ColorWhite,
		"f_black":   types.ColorBlack,
	}
	if err := r.RegisterFromConfig(fieldColors); err != nil {
		t.Fatalf("RegisterFromConfig error: %v", err)
	}

	// Verify the ANSI mapping indirectly via colorToANSI is exercised by
	// asserting one representative from each end of the switch.
	cases := []struct {
		color types.Color
		want  string
	}{
		{types.ColorRed, "\033[31m"},
		{types.ColorGreen, "\033[32m"},
		{types.ColorYellow, "\033[33m"},
		{types.ColorBlue, "\033[34m"},
		{types.ColorMagenta, "\033[35m"},
		{types.ColorCyan, "\033[36m"},
		{types.ColorWhite, "\033[37m"},
		{types.ColorBlack, "\033[30m"},
	}
	for _, c := range cases {
		if got := r.colorToANSI(c.color); got != c.want {
			t.Errorf("colorToANSI(%v) = %q, want %q", c.color, got, c.want)
		}
	}

	// An out-of-range color yields "" and is therefore skipped by RegisterFromConfig.
	if got := r.colorToANSI(types.Color(250)); got != "" {
		t.Errorf("unknown color should map to empty ANSI, got %q", got)
	}
}

func TestColorRegistry_RegisterFromConfig_NilAndSkips(t *testing.T) {
	r := NewColorRegistry()

	// nil map is a no-op returning nil.
	if err := r.RegisterFromConfig(nil); err != nil {
		t.Errorf("nil RegisterFromConfig should not error: %v", err)
	}

	// A color that maps to "" (unknown) must be skipped, not stored.
	if err := r.RegisterFromConfig(map[string]types.Color{"skip": types.Color(200)}); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestColorRegistry_SetColorSchemeAppliesDarkAndLight(t *testing.T) {
	r := NewColorRegistry()

	// applyDarkScheme rewrites levelColors to 256-color codes; because the scheme
	// is "dark", GetColor reads levelColors256 (also 256-color). Verify Fatal.
	r.SetColorScheme("dark")
	if got := r.GetColor(types.FatalLevel); got != "\033[38;5;201m" {
		t.Errorf("dark Fatal = %q, want bright magenta 256", got)
	}

	// applyLightScheme rewrites levelColors; scheme is "light" (non-dark) so
	// GetColor reads levelColors. Verify Warn is the dark-orange 256 code.
	r.SetColorScheme("light")
	if got := r.GetColor(types.WarnLevel); got != "\033[38;5;130m" {
		t.Errorf("light Warn = %q, want dark orange 256", got)
	}
}

func TestConfigureColorsFromConfig(t *testing.T) {
	// nil config is a no-op.
	ConfigureColorsFromConfig(nil)

	// Snapshot and restore the global registry's enabled state so this test
	// does not leak side effects into other tests using the singleton.
	prevEnabled := GlobalColorRegistry.IsEnabled()
	defer func() {
		if prevEnabled {
			GlobalColorRegistry.Enable()
		} else {
			GlobalColorRegistry.Disable()
		}
		GlobalColorRegistry.SetColorScheme("default")
	}()

	cfg := map[string]interface{}{
		"colors_enabled": true,
		"color_scheme":   "dark",
		"custom_colors": map[string]interface{}{
			"DEBUG":   "\033[90m",
			"INFO":    "\033[32m",
			"WARN":    "\033[33m",
			"ERROR":   "\033[31m",
			"FATAL":   "\033[35m",
			"UNKNOWN": "\033[40m", // hits the default: continue branch
			"BAD":     12345,      // non-string value ignored
		},
	}
	ConfigureColorsFromConfig(cfg)

	if !GlobalColorRegistry.IsEnabled() {
		t.Error("colors_enabled=true should enable the global registry")
	}
	// A known custom color should now be set on the global registry.
	if got := GlobalColorRegistry.GetColor(types.DebugLevel); got == "" {
		t.Error("DEBUG custom color should be applied")
	}

	// Now disable via config to exercise the false branch.
	ConfigureColorsFromConfig(map[string]interface{}{"colors_enabled": false})
	if GlobalColorRegistry.IsEnabled() {
		t.Error("colors_enabled=false should disable the global registry")
	}
}

func TestColorRegistry_SetColorFastLevel(t *testing.T) {
	r := NewColorRegistry()

	// SetColor on a level < 256 writes both the plain and 256 arrays.
	custom := "\033[38;5;99m"
	r.SetColor(types.WarnLevel, custom)

	if got := r.GetColor(types.WarnLevel); got != custom {
		t.Errorf("fast-level SetColor not reflected, got %q want %q", got, custom)
	}
	// Dark scheme reads levelColors256 which SetColor also updated.
	r.SetColorScheme("dark")
	// dark scheme reinit does not run for SetColor, but SetColorScheme("dark")
	// calls applyDarkScheme which overwrites Warn; so just assert no panic and
	// that a value is present.
	if got := r.GetColor(types.WarnLevel); got == "" {
		t.Error("expected a non-empty dark-scheme color for Warn")
	}
}
