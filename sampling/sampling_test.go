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
	"github.com/go-gen-ecosystem/halolog/utils"
)

// MockAdapter for sampling tests
type samplingMockAdapter struct {
	name       string
	writeCalls int
	mu         sync.Mutex
}

func (m *samplingMockAdapter) Name() string {
	return m.name
}

func (m *samplingMockAdapter) Health() error {
	return nil
}

func (m *samplingMockAdapter) Write(entry *types.LogEntry) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.writeCalls++
	return nil
}

func (m *samplingMockAdapter) Flush() error {
	return nil
}

func (m *samplingMockAdapter) SetFormatter(formatter types.Formatter) {
	// Mock implementation
}

func (m *samplingMockAdapter) WriteZero(entry *types.LogEntry) error {
	fields := make([]types.TypedField, 0, len(entry.Fields))
	for _, field := range entry.Fields {
		fields = append(fields, field)
	}
	regularEntry := &types.LogEntry{
		Level:     types.LogLevel(entry.Level),
		Message:   entry.Message,
		Component: entry.Component,
		Fields:    entry.Fields,
		Timestamp: entry.Timestamp,
	}
	return m.Write(regularEntry)
}

func (m *samplingMockAdapter) Close() error {
	return nil
}

// TestNewSamplingManager_Defaults tests default values in constructor
func TestNewSamplingManager_Defaults(t *testing.T) {
	config := SamplingConfig{
		Strategy: SampleByCount,
	}
	manager := NewSamplingManager(config)

	if manager.config.SamplingDenominator != 10 {
		t.Errorf("Expected default SamplingDenominator=10, got %d", manager.config.SamplingDenominator)
	}

	if manager.config.TimeWindow != time.Second {
		t.Errorf("Expected default TimeWindow=1s, got %v", manager.config.TimeWindow)
	}

	if manager.config.MaxPerWindow != 100 {
		t.Errorf("Expected default MaxPerWindow=100, got %d", manager.config.MaxPerWindow)
	}

	if manager.windowStart.IsZero() {
		t.Error("Expected windowStart to be initialized")
	}
}

// TestNewSamplingManager_WithValues tests constructor with provided values
func TestNewSamplingManager_WithValues(t *testing.T) {
	config := SamplingConfig{
		Strategy:            SampleByTime,
		SamplingDenominator: 5,
		TimeWindow:          2 * time.Second,
		MaxPerWindow:        50,
	}
	manager := NewSamplingManager(config)

	if manager.config.SamplingDenominator != 5 {
		t.Errorf("Expected SamplingDenominator=5, got %d", manager.config.SamplingDenominator)
	}

	if manager.config.TimeWindow != 2*time.Second {
		t.Errorf("Expected TimeWindow=2s, got %v", manager.config.TimeWindow)
	}

	if manager.config.MaxPerWindow != 50 {
		t.Errorf("Expected MaxPerWindow=50, got %d", manager.config.MaxPerWindow)
	}
}

// TestNewSamplingManager_LevelStrategy tests level-based initialization
func TestNewSamplingManager_LevelStrategy(t *testing.T) {
	config := SamplingConfig{
		Strategy: SampleByLevel,
	}
	manager := NewSamplingManager(config)

	// Verify level counters are initialized
	if manager.levelCounters == nil {
		t.Fatal("Expected levelCounters to be initialized")
	}

	expectedLevels := []types.LogLevel{types.TraceLevel, types.DebugLevel, types.InfoLevel, types.WarnLevel, types.ErrorLevel, types.FatalLevel, types.PanicLevel}
	for _, level := range expectedLevels {
		if _, ok := manager.levelCounters[level]; !ok {
			t.Errorf("Expected counter for level %v to be initialized", level)
		}
	}
}

// TestSampling_ShouldSample_CountBased tests count-based sampling
func TestSampling_ShouldSample_CountBased(t *testing.T) {
	config := SamplingConfig{
		Strategy:            SampleByCount,
		SamplingDenominator: 3,
	}
	manager := NewSamplingManager(config)

	results := []bool{}
	for i := 0; i < 9; i++ {
		entry := &types.LogEntry{Level: types.InfoLevel}
		results = append(results, manager.ShouldSample(entry))
	}

	// Every 3rd should be sampled: positions 2, 5, 8 (0-indexed)
	expected := []bool{false, false, true, false, false, true, false, false, true}

	for i, result := range results {
		if result != expected[i] {
			t.Errorf("Position %d: expected %v, got %v", i, expected[i], result)
		}
	}
}

// TestSampling_ShouldSample_TimeBased tests time-based sampling
func TestSampling_ShouldSample_TimeBased(t *testing.T) {
	config := SamplingConfig{
		Strategy:     SampleByTime,
		TimeWindow:   time.Hour, // Long window
		MaxPerWindow: 3,
	}
	manager := NewSamplingManager(config)

	// First 3 should pass
	entry1 := &types.LogEntry{Level: types.InfoLevel}
	if !manager.ShouldSample(entry1) {
		t.Error("Expected 1st message to be sampled")
	}
	entry2 := &types.LogEntry{Level: types.InfoLevel}
	if !manager.ShouldSample(entry2) {
		t.Error("Expected 2nd message to be sampled")
	}
	entry3 := &types.LogEntry{Level: types.InfoLevel}
	if !manager.ShouldSample(entry3) {
		t.Error("Expected 3rd message to be sampled")
	}

	// 4th should fail (exceeds limit)
	entry4 := &types.LogEntry{Level: types.InfoLevel}
	if manager.ShouldSample(entry4) {
		t.Error("Expected 4th message NOT to be sampled")
	}
	entry5 := &types.LogEntry{Level: types.InfoLevel}
	if manager.ShouldSample(entry5) {
		t.Error("Expected 5th message NOT to be sampled")
	}
}

// TestSampling_ShouldSample_TimeBasedWindowReset tests window reset
func TestSampling_ShouldSample_TimeBasedWindowReset(t *testing.T) {
	config := SamplingConfig{
		Strategy:     SampleByTime,
		TimeWindow:   50 * time.Millisecond,
		MaxPerWindow: 2,
	}
	manager := NewSamplingManager(config)

	// First 2 should pass
	entry1 := &types.LogEntry{Level: types.InfoLevel}
	if !manager.ShouldSample(entry1) {
		t.Error("Expected 1st message to be sampled")
	}
	entry2 := &types.LogEntry{Level: types.InfoLevel}
	if !manager.ShouldSample(entry2) {
		t.Error("Expected 2nd message to be sampled")
	}

	// 3rd should fail
	entry3 := &types.LogEntry{Level: types.InfoLevel}
	if manager.ShouldSample(entry3) {
		t.Error("Expected 3rd message NOT to be sampled")
	}

	// Wait for window to expire
	time.Sleep(60 * time.Millisecond)

	// Should sample again after window reset
	entry4 := &types.LogEntry{Level: types.InfoLevel}
	if !manager.ShouldSample(entry4) {
		t.Error("Expected message to be sampled after window reset")
	}
}

// TestSampling_ShouldSample_LevelBased_AlwaysLogErrors tests error-level sampling
func TestSampling_ShouldSample_LevelBased_AlwaysLogErrors(t *testing.T) {
	config := SamplingConfig{
		Strategy:            SampleByLevel,
		SamplingDenominator: 10,
	}
	manager := NewSamplingManager(config)

	// Errors should ALWAYS be sampled
	for i := 0; i < 20; i++ {
		entry := &types.LogEntry{Level: types.ErrorLevel}
		if !manager.ShouldSample(entry) {
			t.Errorf("Expected types.ErrorLevel message %d to be sampled", i)
		}
	}

	// Fatal should ALWAYS be sampled
	for i := 0; i < 20; i++ {
		entry := &types.LogEntry{Level: types.FatalLevel}
		if !manager.ShouldSample(entry) {
			t.Errorf("Expected types.FatalLevel message %d to be sampled", i)
		}
	}

	// Panic should ALWAYS be sampled
	for i := 0; i < 20; i++ {
		entry := &types.LogEntry{Level: types.PanicLevel}
		if !manager.ShouldSample(entry) {
			t.Errorf("Expected types.PanicLevel message %d to be sampled", i)
		}
	}
}

// TestSampling_ShouldSample_LevelBased_SampleInfo tests info-level sampling
func TestSampling_ShouldSample_LevelBased_SampleInfo(t *testing.T) {
	config := SamplingConfig{
		Strategy:            SampleByLevel,
		SamplingDenominator: 5,
	}
	manager := NewSamplingManager(config)

	sampledCount := 0
	totalCount := 100

	for i := 0; i < totalCount; i++ {
		entry := &types.LogEntry{Level: types.InfoLevel}
		if manager.ShouldSample(entry) {
			sampledCount++
		}
	}

	// Should sample approximately 20% (1 in 5)
	expectedMin := 15
	expectedMax := 25
	if sampledCount < expectedMin || sampledCount > expectedMax {
		t.Errorf("Expected between %d and %d samples, got %d", expectedMin, expectedMax, sampledCount)
	}
}

// TestSampling_ShouldSample_LevelBased_CustomRate tests custom per-level rates
func TestSampling_ShouldSample_LevelBased_CustomRate(t *testing.T) {
	config := SamplingConfig{
		Strategy:            SampleByLevel,
		SamplingDenominator: 10, // default
		LevelSampling: map[types.LogLevel]int{
			types.DebugLevel: 2, // 50% for debug
			types.InfoLevel:  5, // 20% for info
		},
	}
	manager := NewSamplingManager(config)

	// Test debug level (50% sampling)
	debugSampled := 0
	for i := 0; i < 100; i++ {
		entry := &types.LogEntry{Level: types.DebugLevel}
		if manager.ShouldSample(entry) {
			debugSampled++
		}
	}

	if debugSampled < 40 || debugSampled > 60 {
		t.Errorf("Expected ~50 debug samples, got %d", debugSampled)
	}

	// Test info level (20% sampling)
	infoSampled := 0
	for i := 0; i < 100; i++ {
		entry := &types.LogEntry{Level: types.InfoLevel}
		if manager.ShouldSample(entry) {
			infoSampled++
		}
	}

	if infoSampled < 15 || infoSampled > 25 {
		t.Errorf("Expected ~20 info samples, got %d", infoSampled)
	}
}

// TestSampling_ShouldSample_UnknownStrategy tests fallback behavior
func TestSampling_ShouldSample_UnknownStrategy(t *testing.T) {
	config := SamplingConfig{
		Strategy: SamplingStrategy(999),
	}
	manager := NewSamplingManager(config)

	// Unknown strategy should default to true (sample everything)
	for i := 0; i < 10; i++ {
		entry := &types.LogEntry{Level: types.InfoLevel}
		if !manager.ShouldSample(entry) {
			t.Error("Expected unknown strategy to sample everything")
		}
	}
}

// TestSampling_GetSamplingStats_CountBased tests count-based statistics
func TestSampling_GetSamplingStats_CountBased(t *testing.T) {
	config := SamplingConfig{
		Strategy:            SampleByCount,
		SamplingDenominator: 4,
	}
	manager := NewSamplingManager(config)

	// Generate 20 samples
	for i := 0; i < 20; i++ {
		entry := &types.LogEntry{Level: types.InfoLevel}
		manager.ShouldSample(entry)
	}

	stats := manager.GetSamplingStats()

	if stats["strategy"] != "count" {
		t.Errorf("Expected strategy='count', got %v", stats["strategy"])
	}

	if stats["sample_rate"] != 4 {
		t.Errorf("Expected sample_rate=4, got %v", stats["sample_rate"])
	}

	totalCount, ok := stats["total_count"].(uint64)
	if !ok || totalCount != 20 {
		t.Errorf("Expected total_count=20, got %v", stats["total_count"])
	}

	sampledCount, ok := stats["sampled_count"].(uint64)
	if !ok || sampledCount != 5 { // 20/4 = 5
		t.Errorf("Expected sampled_count=5, got %v", stats["sampled_count"])
	}
}

// TestSampling_GetSamplingStats_TimeBased tests time-based statistics
func TestSampling_GetSamplingStats_TimeBased(t *testing.T) {
	config := SamplingConfig{
		Strategy:     SampleByTime,
		TimeWindow:   2 * time.Second,
		MaxPerWindow: 15,
	}
	manager := NewSamplingManager(config)

	// Sample 7 messages
	entry := &types.LogEntry{Level: types.InfoLevel}
	for i := 0; i < 7; i++ {
		manager.ShouldSample(entry)
	}

	stats := manager.GetSamplingStats()

	if stats["strategy"] != "time" {
		t.Errorf("Expected strategy='time', got %v", stats["strategy"])
	}

	if stats["time_window"] != (2 * time.Second).String() {
		t.Errorf("Expected time_window='2s', got %v", stats["time_window"])
	}

	if stats["max_per_window"] != 15 {
		t.Errorf("Expected max_per_window=15, got %v", stats["max_per_window"])
	}

	currentCount, ok := stats["current_window_count"].(int)
	if !ok || currentCount != 7 {
		t.Errorf("Expected current_window_count=7, got %v", stats["current_window_count"])
	}
}

// TestSampling_GetSamplingStats_LevelBased tests level-based statistics
// FIXED: Create new entries instead of reusing and mutating one entry
func TestSampling_GetSamplingStats_LevelBased(t *testing.T) {
	config := SamplingConfig{
		Strategy:            SampleByLevel,
		SamplingDenominator: 2,
	}
	manager := NewSamplingManager(config)

	// Sample different levels - CREATE NEW ENTRIES EACH TIME
	// The bug was: reusing entry and mutating Level before ShouldSample increments counter
	for i := 0; i < 5; i++ {
		infoEntry := &types.LogEntry{Level: types.InfoLevel}
		manager.ShouldSample(infoEntry)

		errorEntry := &types.LogEntry{Level: types.ErrorLevel}
		manager.ShouldSample(errorEntry)

		debugEntry := &types.LogEntry{Level: types.DebugLevel}
		manager.ShouldSample(debugEntry)
	}

	stats := manager.GetSamplingStats()

	if stats["strategy"] != "level" {
		t.Errorf("Expected strategy='level', got %v", stats["strategy"])
	}

	levelStats, ok := stats["level_counts"].(map[string]uint64)
	if !ok {
		t.Fatal("Expected level_counts to be map[string]uint64")
	}

	// Each level should have been called 5 times (counter increments every time)
	if levelStats[types.InfoLevel.String()] != 5 {
		t.Errorf("Expected types.InfoLevel count=5, got %v", levelStats[types.InfoLevel.String()])
	}

	if levelStats[types.ErrorLevel.String()] != 5 {
		t.Errorf("Expected types.ErrorLevel count=5, got %v", levelStats[types.ErrorLevel.String()])
	}

	if levelStats[types.DebugLevel.String()] != 5 {
		t.Errorf("Expected types.DebugLevel count=5, got %v", levelStats[types.DebugLevel.String()])
	}
}

// TestSampling_CountBased_Integration tests complete count-based flow
func TestSampling_CountBased_Integration(t *testing.T) {
	mock := &samplingMockAdapter{name: "test"}
	config := SamplingConfig{
		Strategy:            SampleByCount,
		SamplingDenominator: 10,
	}
	manager := NewSamplingManager(config)

	// Log 100 messages
	for i := 0; i < 100; i++ {
		entry := &types.LogEntry{
			Level:   types.InfoLevel,
			Message: utils.FormatIntWithPrefix("message", i),
		}
		if manager.ShouldSample(entry) {
			mock.Write(entry)
		}
	}

	// Should sample 10% (1 in 10)
	expectedMin := 8
	expectedMax := 12
	if mock.writeCalls < expectedMin || mock.writeCalls > expectedMax {
		t.Errorf("Expected between %d and %d sampled logs, got %d", expectedMin, expectedMax, mock.writeCalls)
	}
}

// TestSampling_ProbabilityBased_Integration tests high-frequency sampling
func TestSampling_ProbabilityBased_Integration(t *testing.T) {
	mock := &samplingMockAdapter{name: "test"}
	config := SamplingConfig{
		Strategy:            SampleByCount,
		SamplingDenominator: 2, // 50%
	}
	manager := NewSamplingManager(config)

	// Log 1000 messages
	for i := 0; i < 1000; i++ {
		entry := &types.LogEntry{
			Level:   types.InfoLevel,
			Message: utils.FormatIntWithPrefix("message", i),
		}
		if manager.ShouldSample(entry) {
			mock.Write(entry)
		}
	}

	// Should sample approximately 50%
	expectedMin := 450
	expectedMax := 550
	if mock.writeCalls < expectedMin || mock.writeCalls > expectedMax {
		t.Errorf("Expected between %d and %d sampled logs, got %d", expectedMin, expectedMax, mock.writeCalls)
	}
}

// TestSampling_LevelBased_Integration tests level-based sampling
func TestSampling_LevelBased_Integration(t *testing.T) {
	mock := &samplingMockAdapter{name: "test"}
	config := SamplingConfig{
		Strategy:            SampleByLevel,
		SamplingDenominator: 10,
	}
	manager := NewSamplingManager(config)

	// Log 100 errors and 100 infos
	for i := 0; i < 100; i++ {
		errorEntry := &types.LogEntry{Level: types.ErrorLevel, Message: "error"}
		if manager.ShouldSample(errorEntry) {
			mock.Write(errorEntry)
		}

		infoEntry := &types.LogEntry{Level: types.InfoLevel, Message: "info"}
		if manager.ShouldSample(infoEntry) {
			mock.Write(infoEntry)
		}
	}

	// Should have ~110 logs (100 errors + 10 infos)
	expectedMin := 105
	expectedMax := 115
	if mock.writeCalls < expectedMin || mock.writeCalls > expectedMax {
		t.Errorf("Expected between %d and %d total logs, got %d", expectedMin, expectedMax, mock.writeCalls)
	}
}

// TestSampling_TimeBased_Integration tests time-based sampling
func TestSampling_TimeBased_Integration(t *testing.T) {
	mock := &samplingMockAdapter{name: "test"}
	config := SamplingConfig{
		Strategy:     SampleByTime,
		TimeWindow:   100 * time.Millisecond,
		MaxPerWindow: 5,
	}
	manager := NewSamplingManager(config)

	// Write 20 entries with delays
	for i := 0; i < 20; i++ {
		entry := &types.LogEntry{
			Level:   types.InfoLevel,
			Message: utils.FormatIntWithPrefix("message", i),
		}
		if manager.ShouldSample(entry) {
			mock.Write(entry)
		}
		time.Sleep(10 * time.Millisecond)
	}

	// Should have sampled some entries (at least 5)
	if mock.writeCalls < 5 {
		t.Errorf("Expected at least 5 sampled logs, got %d", mock.writeCalls)
	}

	// Should not sample all entries
	if mock.writeCalls >= 20 {
		t.Errorf("Expected sampling to reduce log count, got %d", mock.writeCalls)
	}
}

// TestSampling_ConcurrentAccess_CountBased tests thread safety for count-based
func TestSampling_ConcurrentAccess_CountBased(t *testing.T) {
	mock := &samplingMockAdapter{name: "test"}
	config := SamplingConfig{
		Strategy:            SampleByCount,
		SamplingDenominator: 10,
	}
	manager := NewSamplingManager(config)

	var wg sync.WaitGroup
	numGoroutines := 10
	writesPerGoroutine := 100

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < writesPerGoroutine; j++ {
				entry := &types.LogEntry{
					Level:   types.InfoLevel,
					Message: utils.FormatTwoIntsSimple("concurrent", id, '-', j, ""),
				}
				if manager.ShouldSample(entry) {
					mock.Write(entry)
				}
			}
		}(i)
	}

	wg.Wait()

	// Should sample ~10% of 1000
	expectedMin := 80
	expectedMax := 120
	if mock.writeCalls < expectedMin || mock.writeCalls > expectedMax {
		t.Errorf("Expected between %d and %d concurrent samples, got %d", expectedMin, expectedMax, mock.writeCalls)
	}
}

// TestSampling_ConcurrentAccess_TimeBased tests thread safety for time-based
func TestSampling_ConcurrentAccess_TimeBased(t *testing.T) {
	mock := &samplingMockAdapter{name: "test"}
	config := SamplingConfig{
		Strategy:     SampleByTime,
		TimeWindow:   time.Second,
		MaxPerWindow: 50,
	}
	manager := NewSamplingManager(config)

	var wg sync.WaitGroup
	numGoroutines := 10

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				entry := &types.LogEntry{
					Level:   types.InfoLevel,
					Message: utils.FormatTwoIntsSimple("concurrent", id, '-', j, ""),
				}
				if manager.ShouldSample(entry) {
					mock.Write(entry)
				}
			}
		}(i)
	}

	wg.Wait()

	// Should not exceed MaxPerWindow
	if mock.writeCalls > 50 {
		t.Errorf("Expected at most 50 samples in window, got %d", mock.writeCalls)
	}

	// Should have sampled something
	if mock.writeCalls == 0 {
		t.Error("Expected at least some samples")
	}
}

// TestSampling_ConcurrentAccess_LevelBased tests thread safety for level-based
func TestSampling_ConcurrentAccess_LevelBased(t *testing.T) {
	mock := &samplingMockAdapter{name: "test"}
	config := SamplingConfig{
		Strategy:            SampleByLevel,
		SamplingDenominator: 5,
	}
	manager := NewSamplingManager(config)

	var wg sync.WaitGroup
	numGoroutines := 10

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				// Mix of info and error
				if j%2 == 0 {
					if manager.ShouldSample(&types.LogEntry{Level: types.InfoLevel, Message: "info"}) {
						mock.Write(&types.LogEntry{Level: types.InfoLevel, Message: "info"})
					}
				} else {
					if manager.ShouldSample(&types.LogEntry{Level: types.ErrorLevel, Message: "error"}) {
						mock.Write(&types.LogEntry{Level: types.ErrorLevel, Message: "error"})
					}
				}
			}
		}(i)
	}

	wg.Wait()

	// Should have sampled some entries
	if mock.writeCalls == 0 {
		t.Error("Expected at least some samples")
	}

	// Verify stats work under concurrent load
	stats := manager.GetSamplingStats()
	if stats["strategy"] != "level" {
		t.Errorf("Expected strategy='level', got %v", stats["strategy"])
	}
}

// TestSampling_EdgeCase_ZeroSampleRate tests behavior with invalid config
func TestSampling_EdgeCase_ZeroSampleRate(t *testing.T) {
	config := SamplingConfig{
		Strategy:            SampleByCount,
		SamplingDenominator: 0, // Will be set to default 10
	}
	manager := NewSamplingManager(config)

	if manager.config.SamplingDenominator != 10 {
		t.Errorf("Expected default SamplingDenominator=10, got %d", manager.config.SamplingDenominator)
	}
}

// TestSampling_EdgeCase_LevelBasedWithoutCounters tests missing level counter
func TestSampling_EdgeCase_LevelBasedWithoutCounters(t *testing.T) {
	config := SamplingConfig{
		Strategy:            SampleByLevel,
		SamplingDenominator: 5,
	}
	manager := NewSamplingManager(config)

	// Manually clear a counter to test the fallback
	delete(manager.levelCounters, types.InfoLevel)

	// Should still work without panicking
	result := manager.ShouldSample(&types.LogEntry{Level: types.InfoLevel, Message: "info"})

	// With no counter, it should default to true (last return in shouldSampleByLevel)
	if !result {
		t.Error("Expected ShouldSample to return true when counter is missing")
	}
}

// TestSampling_shouldSampleByCount_DirectCall tests internal method
func TestSampling_shouldSampleByCount_DirectCall(t *testing.T) {
	config := SamplingConfig{
		Strategy:            SampleByCount,
		SamplingDenominator: 3,
	}
	manager := NewSamplingManager(config)

	results := []bool{}
	for i := 0; i < 6; i++ {
		results = append(results, manager.shouldSampleByCount(types.InfoLevel))
	}

	// Should match pattern: false, false, true, false, false, true
	expected := []bool{false, false, true, false, false, true}
	for i, result := range results {
		if result != expected[i] {
			t.Errorf("Position %d: expected %v, got %v", i, expected[i], result)
		}
	}
}

// TestSampling_shouldSampleByTime_DirectCall tests internal method
func TestSampling_shouldSampleByTime_DirectCall(t *testing.T) {
	config := SamplingConfig{
		Strategy:     SampleByTime,
		TimeWindow:   100 * time.Millisecond,
		MaxPerWindow: 2,
	}
	manager := NewSamplingManager(config)

	// First 2 should pass
	if !manager.shouldSampleByTime(types.InfoLevel) {
		t.Error("Expected 1st call to return true")
	}
	if !manager.shouldSampleByTime(types.InfoLevel) {
		t.Error("Expected 2nd call to return true")
	}

	// 3rd should fail
	if manager.shouldSampleByTime(types.InfoLevel) {
		t.Error("Expected 3rd call to return false")
	}

	// Wait for window reset
	time.Sleep(150 * time.Millisecond)

	// Should pass again
	if !manager.shouldSampleByTime(types.InfoLevel) {
		t.Error("Expected call after window reset to return true")
	}
}

// TestSampling_shouldSampleByLevel_DirectCall tests internal method
func TestSampling_shouldSampleByLevel_DirectCall(t *testing.T) {
	config := SamplingConfig{
		Strategy:            SampleByLevel,
		SamplingDenominator: 2,
	}
	manager := NewSamplingManager(config)

	// Errors always pass
	for i := 0; i < 10; i++ {
		if !manager.shouldSampleByLevel(types.ErrorLevel) {
			t.Errorf("Expected types.ErrorLevel call %d to return true", i)
		}
	}

	// Info samples at 50%
	results := []bool{}
	for i := 0; i < 4; i++ {
		results = append(results, manager.shouldSampleByLevel(types.InfoLevel))
	}

	// Should follow pattern: false, true, false, true
	expected := []bool{false, true, false, true}
	for i, result := range results {
		if result != expected[i] {
			t.Errorf("types.InfoLevel position %d: expected %v, got %v", i, expected[i], result)
		}
	}
}
