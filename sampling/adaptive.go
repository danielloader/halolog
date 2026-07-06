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

// AdaptiveConfig configures adaptive sampling behavior
type AdaptiveConfig struct {
	// BaseRate is the default sampling rate (0.0-1.0)
	BaseRate float64

	// ErrorBoost multiplier for error-level logs (e.g., 10.0 = 10x more likely)
	ErrorBoost float64

	// MinRate is the minimum sampling rate even under high load
	MinRate float64

	// MaxRate is the maximum sampling rate (typically 1.0)
	MaxRate float64

	// TargetThroughput is the target logs/second before rate adjustment
	TargetThroughput int64

	// AdjustmentInterval is how often to recalculate rates
	AdjustmentInterval time.Duration

	// PreservePatterns are field patterns that bypass sampling
	PreservePatterns []string
}

// DefaultAdaptiveConfig returns sensible defaults
func DefaultAdaptiveConfig() AdaptiveConfig {
	return AdaptiveConfig{
		BaseRate:           0.1,   // 10% sampling
		ErrorBoost:         10.0,  // 10x for errors
		MinRate:            0.01,  // Minimum 1%
		MaxRate:            1.0,   // Maximum 100%
		TargetThroughput:   10000, // 10K logs/sec target
		AdjustmentInterval: time.Second,
		PreservePatterns:   nil,
	}
}

// AdaptiveSampler implements intelligent sampling that adjusts based on load
type AdaptiveSampler struct {
	config AdaptiveConfig

	// Current effective rate (adjusted dynamically)
	currentRate atomic.Value // float64

	// Throughput tracking
	mu             sync.RWMutex
	windowStart    time.Time
	windowCount    int64
	lastAdjustment time.Time
	recentRates    []float64 // Ring buffer of recent rates
	rateIndex      int
	totalSampled   int64
	totalDropped   int64
	errorsSampled  int64
	errorsTotal    int64

	// Counter for sampling decisions
	counter uint64
}

// NewAdaptiveSampler creates an adaptive sampler
func NewAdaptiveSampler(config AdaptiveConfig) *AdaptiveSampler {
	if config.BaseRate <= 0 {
		config.BaseRate = 0.1
	}
	if config.MinRate <= 0 {
		config.MinRate = 0.01
	}
	if config.MaxRate <= 0 {
		config.MaxRate = 1.0
	}
	if config.ErrorBoost <= 0 {
		config.ErrorBoost = 10.0
	}
	if config.TargetThroughput <= 0 {
		config.TargetThroughput = 10000
	}
	if config.AdjustmentInterval <= 0 {
		config.AdjustmentInterval = time.Second
	}

	s := &AdaptiveSampler{
		config:         config,
		windowStart:    time.Now(),
		lastAdjustment: time.Now(),
		recentRates:    make([]float64, 10), // Keep last 10 rate samples
	}
	s.currentRate.Store(config.BaseRate)
	return s
}

// ShouldSample determines if a log entry should be sampled
func (s *AdaptiveSampler) ShouldSample(entry *types.LogEntry) bool {
	// Track all entries
	s.trackEntry(entry)

	// Always sample errors and above (with boost consideration)
	if entry.Level >= types.ErrorLevel {
		atomic.AddInt64(&s.errorsTotal, 1)
		if s.shouldSampleWithRate(s.getErrorRate()) {
			atomic.AddInt64(&s.errorsSampled, 1)
			return true
		}
		return false
	}

	// Regular sampling for other levels
	return s.shouldSampleWithRate(s.getCurrentRate())
}

// getCurrentRate returns the current adaptive rate
func (s *AdaptiveSampler) getCurrentRate() float64 {
	rate := s.currentRate.Load()
	if rate == nil {
		return s.config.BaseRate
	}
	return rate.(float64)
}

// getErrorRate returns the boosted rate for errors
func (s *AdaptiveSampler) getErrorRate() float64 {
	baseRate := s.getCurrentRate()
	boostedRate := baseRate * s.config.ErrorBoost
	if boostedRate > s.config.MaxRate {
		return s.config.MaxRate
	}
	return boostedRate
}

// shouldSampleWithRate uses probabilistic sampling
func (s *AdaptiveSampler) shouldSampleWithRate(rate float64) bool {
	if rate >= 1.0 {
		atomic.AddInt64(&s.totalSampled, 1)
		return true
	}
	if rate <= 0 {
		atomic.AddInt64(&s.totalDropped, 1)
		return false
	}

	// Use denominator-based sampling: rate 0.01 = sample 1 in 100
	count := atomic.AddUint64(&s.counter, 1)
	denominator := uint64(1.0 / rate)
	if denominator == 0 {
		denominator = 1
	}

	// Sample if counter is divisible by denominator
	if count%denominator == 0 {
		atomic.AddInt64(&s.totalSampled, 1)
		return true
	}
	atomic.AddInt64(&s.totalDropped, 1)
	return false
}

// trackEntry tracks throughput for adaptive adjustments
func (s *AdaptiveSampler) trackEntry(entry *types.LogEntry) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	s.windowCount++

	// Check if we should adjust the rate
	if now.Sub(s.lastAdjustment) >= s.config.AdjustmentInterval {
		s.adjustRate(now)
	}
}

// adjustRate recalculates the sampling rate based on throughput
func (s *AdaptiveSampler) adjustRate(now time.Time) {
	elapsed := now.Sub(s.windowStart).Seconds()
	if elapsed <= 0 {
		return
	}

	// Calculate current throughput
	currentThroughput := float64(s.windowCount) / elapsed

	// Calculate new rate based on throughput vs target
	var newRate float64
	if currentThroughput <= 0 {
		newRate = s.config.MaxRate
	} else if currentThroughput > float64(s.config.TargetThroughput) {
		// Reduce rate proportionally to how much we're over target
		ratio := float64(s.config.TargetThroughput) / currentThroughput
		newRate = s.config.BaseRate * ratio
	} else {
		// Under target, can increase towards base rate
		ratio := currentThroughput / float64(s.config.TargetThroughput)
		// Inverse: lower throughput = higher rate
		newRate = s.config.BaseRate + (s.config.MaxRate-s.config.BaseRate)*(1-ratio)
	}

	// Clamp to bounds
	if newRate < s.config.MinRate {
		newRate = s.config.MinRate
	}
	if newRate > s.config.MaxRate {
		newRate = s.config.MaxRate
	}

	// Smooth the rate change using moving average
	s.recentRates[s.rateIndex] = newRate
	s.rateIndex = (s.rateIndex + 1) % len(s.recentRates)

	// Calculate average of recent rates
	var sum float64
	var count int
	for _, r := range s.recentRates {
		if r > 0 {
			sum += r
			count++
		}
	}
	if count > 0 {
		smoothedRate := sum / float64(count)
		s.currentRate.Store(smoothedRate)
	}

	// Reset window
	s.windowStart = now
	s.windowCount = 0
	s.lastAdjustment = now
}

// GetStats returns current sampling statistics
func (s *AdaptiveSampler) GetStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return map[string]interface{}{
		"strategy":       "adaptive",
		"current_rate":   s.getCurrentRate(),
		"base_rate":      s.config.BaseRate,
		"error_boost":    s.config.ErrorBoost,
		"total_sampled":  atomic.LoadInt64(&s.totalSampled),
		"total_dropped":  atomic.LoadInt64(&s.totalDropped),
		"errors_sampled": atomic.LoadInt64(&s.errorsSampled),
		"errors_total":   atomic.LoadInt64(&s.errorsTotal),
		"window_count":   s.windowCount,
	}
}

// Reset resets the sampler state
func (s *AdaptiveSampler) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.currentRate.Store(s.config.BaseRate)
	s.windowStart = time.Now()
	s.windowCount = 0
	s.lastAdjustment = time.Now()
	s.recentRates = make([]float64, 10)
	s.rateIndex = 0
	atomic.StoreInt64(&s.totalSampled, 0)
	atomic.StoreInt64(&s.totalDropped, 0)
	atomic.StoreInt64(&s.errorsSampled, 0)
	atomic.StoreInt64(&s.errorsTotal, 0)
	atomic.StoreUint64(&s.counter, 0)
}
