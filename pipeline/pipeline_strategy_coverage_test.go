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
// Package pipeline tests exercise the EnhancedPipelineStrategy selection logic.
// @author Admilson B. F. Cossa

package pipeline

import (
	"testing"

	"github.com/go-gen-ecosystem/halolog/types"
)

// resetStrategyPipelines releases the individual pipelines owned by a strategy
// WITHOUT calling ps.Shutdown().
//
// NOTE (pre-existing source bug, NOT fixed here): EnhancedPipelineStrategy.Shutdown()
// calls pool.GlobalPool.Close(), and pool.(*EntryPool).Close() spins in an infinite
// loop — it drains the sync.Pool with Get() expecting a nil sentinel, but the pool's
// New func always constructs a fresh *types.LogEntry, so Get() is never nil. Calling
// Shutdown() therefore hangs the test process. These tests deliberately avoid it and
// reset the owned pipelines directly instead.
func resetStrategyPipelines(ps *EnhancedPipelineStrategy) {
	if ps.directPipeline != nil {
		ps.directPipeline.Reset()
	}
	if ps.simplePipeline != nil {
		ps.simplePipeline.Reset()
	}
	if ps.fullPipeline != nil {
		// Full pipeline has no async started in these tests, so Reset is safe.
		ps.fullPipeline.Reset()
	}
}

// TestGetDefaultPipelineConfig verifies the documented default configuration.
func TestGetDefaultPipelineConfig(t *testing.T) {
	cfg := getDefaultPipelineConfig()
	if !cfg.EnableMasking {
		t.Fatal("expected masking enabled by default")
	}
	if cfg.EnableSampling {
		t.Fatal("expected sampling disabled by default")
	}
	if cfg.SampleRate != 100 {
		t.Fatalf("expected default sample rate 100, got %d", cfg.SampleRate)
	}
	if cfg.OptimalBufferSize != 32 {
		t.Fatalf("expected default buffer 32, got %d", cfg.OptimalBufferSize)
	}
}

// TestCalculateOptimalBufferSize_Strategy verifies the strategy-level helper.
func TestCalculateOptimalBufferSize_Strategy(t *testing.T) {
	if got := calculateOptimalBufferSize(&PipelineConfig{OptimalBufferSize: 48}); got != 48 {
		t.Fatalf("expected configured 48, got %d", got)
	}
	if got := calculateOptimalBufferSize(&PipelineConfig{OptimalBufferSize: 0}); got != 32 {
		t.Fatalf("expected default 32, got %d", got)
	}
}

// TestNewEnhancedPipelineStrategy_NilConfigDefaults verifies a nil config falls
// back to defaults and selects the FullPipeline (masking+colors on by default).
func TestNewEnhancedPipelineStrategy_NilConfigDefaults(t *testing.T) {
	var sink []captured
	adapters := []types.Adapter{newCapturingAdapter(&sink)}

	ps := NewEnhancedPipelineStrategy(nil, adapters)
	defer resetStrategyPipelines(ps)

	// Default config enables masking + colors, so the full pipeline is selected.
	if name := ps.getCurrentPipelineName(); name != "full" {
		t.Fatalf("expected 'full' pipeline for default config, got %q", name)
	}
	m := ps.GetMetrics()
	if !m.HasMasking {
		t.Fatal("expected HasMasking true for default config")
	}
	if m.SelectionCount < 1 {
		t.Fatalf("expected at least one selection, got %d", m.SelectionCount)
	}
}

// TestEnhancedStrategy_SelectsDirect verifies a feature-free, config selects the
// direct pipeline.
func TestEnhancedStrategy_SelectsDirect(t *testing.T) {
	var sink []captured
	adapters := []types.Adapter{newCapturingAdapter(&sink)}
	cfg := &PipelineConfig{
		EnableMasking:     false,
		EnableSampling:    false,
		EnableAsync:       false,
		EnableColors:      false,
		OptimalBufferSize: 32,
		MetricsCollection: false,
	}

	ps := NewEnhancedPipelineStrategy(cfg, adapters)
	defer resetStrategyPipelines(ps)

	if name := ps.getCurrentPipelineName(); name != "direct" {
		t.Fatalf("expected 'direct' pipeline, got %q", name)
	}
}

// TestEnhancedStrategy_WriteRoutesToPipeline verifies Write actually dispatches to
// the selected pipeline and reaches the adapter.
func TestEnhancedStrategy_WriteRoutesToPipeline(t *testing.T) {
	var got int
	var lastMsg string
	adapter := &types.FuncAdapter{
		WriteFunc: func(e *types.LogEntry) error {
			got++
			lastMsg = e.Message
			return nil
		},
	}
	cfg := &PipelineConfig{
		EnableMasking:     false,
		EnableSampling:    false,
		EnableAsync:       false,
		EnableColors:      false,
		OptimalBufferSize: 32,
		MetricsCollection: true, // exercise the recordMetrics path
	}
	ps := NewEnhancedPipelineStrategy(cfg, []types.Adapter{adapter})
	defer resetStrategyPipelines(ps)
	cc := newTestClock(t)

	ps.Write(cc, types.InfoLevel, "routed", nil, 0)

	if got != 1 {
		t.Fatalf("expected 1 adapter write, got %d", got)
	}
	if lastMsg != "routed" {
		t.Fatalf("expected message 'routed', got %q", lastMsg)
	}
}

// TestEnhancedStrategy_UpdateConfiguration verifies that reconfiguring a strategy
// applies the new settings and, critically, does not deadlock. (It previously
// self-deadlocked: UpdateConfiguration held ps.mu while calling selectOptimalPipeline,
// which re-acquired the same non-reentrant mutex. Now it uses the *Locked variant.)
func TestEnhancedStrategy_UpdateConfiguration(t *testing.T) {
	var sink []captured
	ps := NewEnhancedPipelineStrategy(&PipelineConfig{OptimalBufferSize: 32}, []types.Adapter{newCapturingAdapter(&sink)})
	defer resetStrategyPipelines(ps)

	// Must complete (no hang) and apply the new feature flags.
	ps.UpdateConfiguration(&PipelineConfig{
		OptimalBufferSize: 64,
		EnableMasking:     true,
		EnableSampling:    true,
		SampleRate:        50,
	})

	if !ps.hasMasking.Load() {
		t.Error("UpdateConfiguration should have enabled masking")
	}
	if !ps.hasSampling.Load() {
		t.Error("UpdateConfiguration should have enabled sampling")
	}
	if ps.optimalBufferSize != calculateOptimalBufferSize(ps.config) {
		t.Error("UpdateConfiguration should have recalculated the optimal buffer size")
	}
}

// TestEnhancedStrategy_ShouldUsePredicates directly exercises the predicate
// helpers across feature combinations.
func TestEnhancedStrategy_ShouldUsePredicates(t *testing.T) {
	var sink []captured
	ps := NewEnhancedPipelineStrategy(&PipelineConfig{OptimalBufferSize: 32}, []types.Adapter{newCapturingAdapter(&sink)})
	defer resetStrategyPipelines(ps)

	// Feature-free: both direct and simple eligible.
	if !ps.shouldUseDirectPipeline() {
		t.Fatal("expected direct eligible with no features")
	}
	if !ps.shouldUseSimplePipeline() {
		t.Fatal("expected simple eligible with no features")
	}

	// Enable masking: neither direct nor simple.
	ps.hasMasking.Store(true)
	if ps.shouldUseDirectPipeline() {
		t.Fatal("expected direct ineligible with masking")
	}
	if ps.shouldUseSimplePipeline() {
		t.Fatal("expected simple ineligible with masking")
	}
	ps.hasMasking.Store(false)

	// Colors: not direct, but still simple (field-level colors allowed).
	ps.hasColors.Store(true)
	if ps.shouldUseDirectPipeline() {
		t.Fatal("expected direct ineligible with colors")
	}
	if !ps.shouldUseSimplePipeline() {
		t.Fatal("expected simple still eligible with colors")
	}
}

// TestConfigureFromApplicationProfile exercises the profile-to-config translation
// across the application-type branches.
func TestConfigureFromApplicationProfile(t *testing.T) {
	// nil profile -> defaults.
	if cfg := ConfigureFromApplicationProfile(nil); cfg == nil || !cfg.EnableMasking {
		t.Fatal("expected default config for nil profile")
	}

	t.Run("high_performance", func(t *testing.T) {
		cfg := ConfigureFromApplicationProfile(map[string]interface{}{
			"application_type": "high_performance",
			"p99_field_count":  10,
		})
		if cfg.EnableAsync || cfg.EnableColors || cfg.EnableSampling {
			t.Fatalf("expected all speed features off, got %+v", cfg)
		}
		if cfg.OptimalBufferSize != 15 {
			t.Fatalf("expected buffer p99+5=15, got %d", cfg.OptimalBufferSize)
		}
	})

	t.Run("web_api", func(t *testing.T) {
		cfg := ConfigureFromApplicationProfile(map[string]interface{}{
			"application_type": "web_api",
		})
		if !cfg.EnableColors || !cfg.EnableMasking {
			t.Fatalf("expected colors+masking on for web_api, got %+v", cfg)
		}
	})

	t.Run("batch_processor", func(t *testing.T) {
		cfg := ConfigureFromApplicationProfile(map[string]interface{}{
			"application_type": "batch_processor",
		})
		if !cfg.EnableAsync {
			t.Fatal("expected async on for batch_processor")
		}
		if cfg.AsyncQueueSize != 10000 {
			t.Fatalf("expected async queue 10000, got %d", cfg.AsyncQueueSize)
		}
	})

	t.Run("sensitive_fields", func(t *testing.T) {
		cfg := ConfigureFromApplicationProfile(map[string]interface{}{
			"sensitive_fields": []string{"ssn", "card"},
		})
		if len(cfg.SensitiveFields) != 2 {
			t.Fatalf("expected 2 sensitive fields, got %d", len(cfg.SensitiveFields))
		}
	})
}

// TestInitializeEnhancedStrategy verifies the global singleton initializer is
// idempotent.
func TestInitializeEnhancedStrategy(t *testing.T) {
	// Reset the global for a clean, deterministic assertion.
	GlobalEnhancedStrategy = nil
	var sink []captured
	adapters := []types.Adapter{newCapturingAdapter(&sink)}

	InitializeEnhancedStrategy(&PipelineConfig{OptimalBufferSize: 32}, adapters)
	first := GlobalEnhancedStrategy
	if first == nil {
		t.Fatal("expected global strategy to be initialized")
	}

	// Second call must be a no-op (idempotent).
	InitializeEnhancedStrategy(&PipelineConfig{OptimalBufferSize: 64}, adapters)
	if GlobalEnhancedStrategy != first {
		t.Fatal("expected InitializeEnhancedStrategy to be idempotent")
	}

	// Avoid first.Shutdown() (hangs via pool.Close, see resetStrategyPipelines note).
	resetStrategyPipelines(first)
	GlobalEnhancedStrategy = nil
}

// TestEnhancedStrategy_SelectsSimple verifies the SimplePipeline selection branch:
// a config with colors on but no masking/sampling/async selects 'simple' on the
// strategy's FIRST (constructor) selection.
//
// NOTE: this uses a fresh strategy whose very first stored pipeline is the simple
// one. It intentionally does NOT re-run selectOptimalPipeline after a type change,
// because that triggers a pre-existing source bug: selectOptimalPipeline stores
// different concrete pointer types (*DirectPipeline / *SimplePipeline / *FullPipeline)
// into the same atomic.Value, which panics with "store of inconsistently typed value"
// on any pipeline-type transition. See the package report.
func TestEnhancedStrategy_SelectsSimple(t *testing.T) {
	var sink []captured
	cfg := &PipelineConfig{
		EnableColors:      true, // colors-only -> simple path on first selection
		ColorScheme:       "default",
		OptimalBufferSize: 32,
	}
	ps := NewEnhancedPipelineStrategy(cfg, []types.Adapter{newCapturingAdapter(&sink)})
	defer resetStrategyPipelines(ps)

	if name := ps.getCurrentPipelineName(); name != "simple" {
		t.Fatalf("expected 'simple' pipeline with colors-only config, got %q", name)
	}
}

// TestEnhancedStrategy_WithMaskingConfigured verifies that constructing with
// sensitive fields/patterns configured drives the masking-registration branch.
func TestEnhancedStrategy_WithMaskingConfigured(t *testing.T) {
	var sink []captured
	cfg := &PipelineConfig{
		EnableMasking:     true,
		EnableColors:      true,
		ColorScheme:       "default",
		OptimalBufferSize: 32,
		SensitiveFields:   []string{"password", "token"},
		SensitivePatterns: []string{".*secret.*"},
	}
	ps := NewEnhancedPipelineStrategy(cfg, []types.Adapter{newCapturingAdapter(&sink)})
	defer resetStrategyPipelines(ps)

	if name := ps.getCurrentPipelineName(); name != "full" {
		t.Fatalf("expected 'full' pipeline with masking configured, got %q", name)
	}
	if !ps.fullPipeline.HasMasking {
		t.Fatal("expected full pipeline masking enabled")
	}
	if !ps.fullPipeline.HasColors {
		t.Fatal("expected full pipeline colors enabled")
	}
}
