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

// Package sampling provides log sampling functionality
// Author: Admilson B. F. Cossa

package sampling

import (
	"sync"
	"testing"
	"time"

	"github.com/go-gen-ecosystem/halolog/types"
)

func TestDefaultAdaptiveConfig(t *testing.T) {
	cfg := DefaultAdaptiveConfig()

	if cfg.BaseRate != 0.1 {
		t.Errorf("Expected BaseRate 0.1, got %v", cfg.BaseRate)
	}
	if cfg.ErrorBoost != 10.0 {
		t.Errorf("Expected ErrorBoost 10.0, got %v", cfg.ErrorBoost)
	}
	if cfg.MinRate != 0.01 {
		t.Errorf("Expected MinRate 0.01, got %v", cfg.MinRate)
	}
	if cfg.MaxRate != 1.0 {
		t.Errorf("Expected MaxRate 1.0, got %v", cfg.MaxRate)
	}
}

func TestNewAdaptiveSampler(t *testing.T) {
	t.Run("with defaults", func(t *testing.T) {
		sampler := NewAdaptiveSampler(DefaultAdaptiveConfig())
		if sampler == nil {
			t.Fatal("Expected non-nil sampler")
		}
		if sampler.getCurrentRate() != 0.1 {
			t.Errorf("Expected initial rate 0.1, got %v", sampler.getCurrentRate())
		}
	})

	t.Run("with zero values", func(t *testing.T) {
		sampler := NewAdaptiveSampler(AdaptiveConfig{})
		if sampler == nil {
			t.Fatal("Expected non-nil sampler")
		}
		// Should use defaults for zero values
		if sampler.config.BaseRate != 0.1 {
			t.Errorf("Expected default base rate")
		}
	})

	t.Run("with custom config", func(t *testing.T) {
		cfg := AdaptiveConfig{
			BaseRate:   0.5,
			ErrorBoost: 2.0,
			MinRate:    0.1,
			MaxRate:    0.9,
		}
		sampler := NewAdaptiveSampler(cfg)
		if sampler.getCurrentRate() != 0.5 {
			t.Errorf("Expected initial rate 0.5, got %v", sampler.getCurrentRate())
		}
	})
}

func TestAdaptiveSampler_ShouldSample(t *testing.T) {
	cfg := AdaptiveConfig{
		BaseRate:   1.0, // 100% for predictable testing
		ErrorBoost: 1.0,
		MinRate:    0.0,
		MaxRate:    1.0,
	}
	sampler := NewAdaptiveSampler(cfg)

	t.Run("samples all at 100% rate", func(t *testing.T) {
		entry := &types.LogEntry{
			Level:   types.InfoLevel,
			Message: "test",
		}
		// With 100% rate, should always sample
		for i := 0; i < 100; i++ {
			if !sampler.ShouldSample(entry) {
				t.Error("Expected all entries to be sampled at 100% rate")
			}
		}
	})
}

func TestAdaptiveSampler_ErrorBoost(t *testing.T) {
	cfg := AdaptiveConfig{
		BaseRate:   0.1,  // 10%
		ErrorBoost: 10.0, // Should boost to 100%
		MinRate:    0.01,
		MaxRate:    1.0,
	}
	sampler := NewAdaptiveSampler(cfg)

	// Create error entries
	errorEntry := &types.LogEntry{
		Level:   types.ErrorLevel,
		Message: "error",
	}

	// With 10% base rate and 10x boost, errors should sample at 100%
	sampled := 0
	for i := 0; i < 100; i++ {
		if sampler.ShouldSample(errorEntry) {
			sampled++
		}
	}

	// Should sample nearly all errors (allowing some variance)
	if sampled < 90 {
		t.Errorf("Expected ~100%% of errors sampled with boost, got %d%%", sampled)
	}
}

func TestAdaptiveSampler_LowRate(t *testing.T) {
	cfg := AdaptiveConfig{
		BaseRate:   0.01, // 1%
		ErrorBoost: 1.0,
		MinRate:    0.01,
		MaxRate:    1.0,
	}
	sampler := NewAdaptiveSampler(cfg)

	entry := &types.LogEntry{
		Level:   types.InfoLevel,
		Message: "test",
	}

	// With 1% rate, should sample approximately 1%
	sampled := 0
	total := 10000
	for i := 0; i < total; i++ {
		if sampler.ShouldSample(entry) {
			sampled++
		}
	}

	// Allow 0.5% to 2% variance
	rate := float64(sampled) / float64(total)
	if rate < 0.005 || rate > 0.02 {
		t.Errorf("Expected ~1%% sampling rate, got %.2f%%", rate*100)
	}
}

func TestAdaptiveSampler_GetStats(t *testing.T) {
	sampler := NewAdaptiveSampler(DefaultAdaptiveConfig())

	// Sample some entries
	for i := 0; i < 100; i++ {
		entry := &types.LogEntry{Level: types.InfoLevel, Message: "test"}
		sampler.ShouldSample(entry)
	}

	stats := sampler.GetStats()

	if stats["strategy"] != "adaptive" {
		t.Error("Expected strategy 'adaptive'")
	}
	if stats["base_rate"] != 0.1 {
		t.Error("Expected base_rate 0.1")
	}
	if stats["error_boost"] != 10.0 {
		t.Error("Expected error_boost 10.0")
	}
}

func TestAdaptiveSampler_Reset(t *testing.T) {
	sampler := NewAdaptiveSampler(DefaultAdaptiveConfig())

	// Sample some entries
	for i := 0; i < 100; i++ {
		entry := &types.LogEntry{Level: types.InfoLevel}
		sampler.ShouldSample(entry)
	}

	// Reset
	sampler.Reset()

	stats := sampler.GetStats()
	if stats["total_sampled"].(int64) != 0 {
		t.Error("Expected total_sampled to be reset")
	}
	if stats["total_dropped"].(int64) != 0 {
		t.Error("Expected total_dropped to be reset")
	}
}

func TestAdaptiveSampler_Concurrency(t *testing.T) {
	sampler := NewAdaptiveSampler(DefaultAdaptiveConfig())

	const goroutines = 100
	const iterations = 100

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				level := types.InfoLevel
				if j%10 == 0 {
					level = types.ErrorLevel
				}
				entry := &types.LogEntry{Level: level, Message: "concurrent"}
				sampler.ShouldSample(entry)
			}
		}(i)
	}

	wg.Wait()

	// Verify no panics and stats are reasonable
	stats := sampler.GetStats()
	totalSampled := stats["total_sampled"].(int64)
	totalDropped := stats["total_dropped"].(int64)

	total := totalSampled + totalDropped
	if total != goroutines*iterations {
		t.Errorf("Expected %d total entries, got %d", goroutines*iterations, total)
	}
}

func TestAdaptiveSampler_RateAdjustment(t *testing.T) {
	cfg := AdaptiveConfig{
		BaseRate:           0.1,
		ErrorBoost:         1.0,
		MinRate:            0.01,
		MaxRate:            1.0,
		TargetThroughput:   100, // Low target for testing
		AdjustmentInterval: 10 * time.Millisecond,
	}
	sampler := NewAdaptiveSampler(cfg)

	entry := &types.LogEntry{Level: types.InfoLevel}

	// Generate high throughput
	for i := 0; i < 1000; i++ {
		sampler.ShouldSample(entry)
	}

	// Wait for adjustment
	time.Sleep(20 * time.Millisecond)

	// Sample more to trigger adjustment
	for i := 0; i < 100; i++ {
		sampler.ShouldSample(entry)
	}

	// Rate should have adjusted (either up or down based on throughput)
	// Just verify it's within bounds
	rate := sampler.getCurrentRate()
	if rate < cfg.MinRate || rate > cfg.MaxRate {
		t.Errorf("Rate %v out of bounds [%v, %v]", rate, cfg.MinRate, cfg.MaxRate)
	}
}

func BenchmarkAdaptiveSampler_ShouldSample(b *testing.B) {
	sampler := NewAdaptiveSampler(DefaultAdaptiveConfig())
	entry := &types.LogEntry{Level: types.InfoLevel, Message: "benchmark"}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		sampler.ShouldSample(entry)
	}
}

func BenchmarkAdaptiveSampler_Parallel(b *testing.B) {
	sampler := NewAdaptiveSampler(DefaultAdaptiveConfig())

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		entry := &types.LogEntry{Level: types.InfoLevel, Message: "benchmark"}
		for pb.Next() {
			sampler.ShouldSample(entry)
		}
	})
}
