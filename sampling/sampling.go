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
	"sync/atomic"
	"time"

	"github.com/go-gen-ecosystem/halolog/types"
)

// SamplingStrategy defines how logs should be sampled.
//
//nolint:revive // SamplingStrategy is stable public API; the sampling.Strategy rename would break importers.
type SamplingStrategy int

const (
	// SampleByCount samples every Nth log message
	SampleByCount SamplingStrategy = iota
	// SampleByTime samples logs within a time window
	SampleByTime
	// SampleByLevel samples based on log level
	SampleByLevel
)

// SamplingConfig configures the sampling behavior.
//
//nolint:revive // SamplingConfig is stable public API; the sampling.Config rename would break importers.
type SamplingConfig struct {
	Strategy            SamplingStrategy
	SamplingDenominator int                    // For count-based: log every Nth message (N=1 means 100%, N=10 means 10%)
	TimeWindow          time.Duration          // For time-based: window duration
	MaxPerWindow        int                    // For time-based: max logs per window
	LevelSampling       map[types.LogLevel]int // Per-level sample denominators
}

// SamplingManager provides sampling logic and state management.
//
//nolint:revive // SamplingManager is stable public API; the sampling.Manager rename would break importers.
type SamplingManager struct {
	config SamplingConfig

	// Count-based sampling
	counter uint64

	// Time-based sampling
	mu          sync.RWMutex
	windowStart time.Time
	windowCount int

	// Per-level counters for sampling decisions
	levelCounters map[types.LogLevel]*uint64

	// Per-level counters for statistics tracking
	levelStatsCounters map[types.LogLevel]*uint64
}

// NewSamplingManager creates a new sampling manager
func NewSamplingManager(config SamplingConfig) *SamplingManager {
	// Set sane defaults and validate configuration
	if config.SamplingDenominator <= 0 {
		config.SamplingDenominator = 10 // Default: 10 (10% sampling)
	}
	if config.TimeWindow <= 0 {
		config.TimeWindow = time.Second // Default: 1 second
	}
	if config.MaxPerWindow <= 0 {
		config.MaxPerWindow = 100 // Default: Max 100 logs per window
	}

	manager := &SamplingManager{
		config:             config,
		windowStart:        time.Now(),
		levelCounters:      make(map[types.LogLevel]*uint64),
		levelStatsCounters: make(map[types.LogLevel]*uint64),
	}

	// Initialize level counters for SampleByLevel strategy
	if config.Strategy == SampleByLevel {
		for level := types.TraceLevel; level <= types.PanicLevel; level++ {
			// Initialize sampling counter using atomic.StoreUint64 to ensure it's managed atomically
			samplingCounter := uint64(0)
			manager.levelCounters[level] = &samplingCounter

			// Initialize statistics counter
			statsCounter := uint64(0)
			manager.levelStatsCounters[level] = &statsCounter
		}
	}

	return manager
}

// ShouldSample determines if a log entry should be sampled
func (s *SamplingManager) ShouldSample(entry *types.LogEntry) bool {
	// Always increment level statistics counters if they exist (for statistics tracking)
	if statsCounterPtr, ok := s.levelStatsCounters[entry.Level]; ok {
		atomic.AddUint64(statsCounterPtr, 1)
	}

	return s.shouldSample(entry.Level)
}

// shouldSample determines if a log should be written
func (s *SamplingManager) shouldSample(level types.LogLevel) bool {
	switch s.config.Strategy {
	case SampleByCount:
		return s.shouldSampleByCount(level)
	case SampleByTime:
		return s.shouldSampleByTime(level)
	case SampleByLevel:
		return s.shouldSampleByLevel(level)
	default:
		// Default to sampling everything if strategy is unknown/unset
		return true
	}
}

// shouldSampleByCount samples every Nth log message based on SamplingDenominator.
// N=1 samples 100%. N=10 samples 10%.
func (s *SamplingManager) shouldSampleByCount(level types.LogLevel) bool {
	// Always log errors, fatals, and panics, regardless of rate
	if level >= types.ErrorLevel {
		return true
	}

	// Atomically increment the counter
	count := atomic.AddUint64(&s.counter, 1)

	denom := uint64(s.config.SamplingDenominator)
	if denom == 1 {
		return true // Always sample if denominator is 1 (100%)
	}

	// Sample the log if the counter is a multiple of the denominator
	// (i.e., sample the Nth, 2Nth, 3Nth log, etc.)
	return count%denom == 0
}

func (s *SamplingManager) shouldSampleByTime(level types.LogLevel) bool {
	// Always log errors, fatals, and panics, regardless of rate
	if level >= types.ErrorLevel {
		return true
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()

	// Reset window if expired
	if now.Sub(s.windowStart) >= s.config.TimeWindow {
		s.windowStart = now
		s.windowCount = 0
	}

	// Check if within limit
	if s.windowCount >= s.config.MaxPerWindow {
		return false
	}

	s.windowCount++
	return true
}

func (s *SamplingManager) shouldSampleByLevel(level types.LogLevel) bool {
	// Always log errors, fatals, and panics, regardless of rate
	// assuming ErrorLevel includes Fatal and Panic
	if level >= types.ErrorLevel {
		return true
	}

	counterPtr, ok := s.levelCounters[level]
	if !ok {
		// If the level is not initialized for counting, fall through to default behavior (sample)
		return true
	}

	// Increment counter and get the new count for sampling decision
	count := atomic.AddUint64(counterPtr, 1)

	// Get sample denominator for this level
	sampleDenom, ok := s.config.LevelSampling[level]
	if !ok || sampleDenom <= 0 {
		// Fallback to global denominator
		sampleDenom = s.config.SamplingDenominator
	}

	denom := uint64(sampleDenom)
	if denom == 1 {
		return true // 100% sample rate
	}

	// For non-error levels, apply sampling logic
	// Sample every Nth log (where N is the denominator)
	return count%denom == 0
}

// GetSamplingStats returns statistics about sampling
func (s *SamplingManager) GetSamplingStats() map[string]interface{} {
	stats := make(map[string]interface{})

	switch s.config.Strategy {
	case SampleByCount:
		stats["strategy"] = "count"
		denom := s.config.SamplingDenominator
		stats["sample_rate"] = denom

		totalCount := atomic.LoadUint64(&s.counter)
		stats["total_count"] = totalCount
		// Calculate sampled count based on the denominator
		if denom > 0 {
			stats["sampled_count"] = totalCount / uint64(denom)
		} else {
			stats["sampled_count"] = totalCount // Safety: if denom is 0, assume 100% sampled
		}

	case SampleByTime:
		s.mu.RLock()
		stats["strategy"] = "time"
		stats["time_window"] = s.config.TimeWindow.String()
		stats["max_per_window"] = s.config.MaxPerWindow
		stats["current_window_count"] = s.windowCount
		s.mu.RUnlock()

	case SampleByLevel:
		stats["strategy"] = "level"
		levelStats := make(map[string]uint64)
		for level, counter := range s.levelStatsCounters {
			levelStats[level.String()] = atomic.LoadUint64(counter)
		}
		stats["level_counts"] = levelStats
	}

	return stats
}
