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
// Package pipeline provides logging pipeline functionality
// Author: Admilson B. F. Cossa

package pipeline

import (
	"sync"
	"sync/atomic"
	"time"

	clock "github.com/go-gen-ecosystem/halolog/cache"
	fielddict "github.com/go-gen-ecosystem/halolog/fielddict"
	pool "github.com/go-gen-ecosystem/halolog/pool"
	"github.com/go-gen-ecosystem/halolog/race"
	registry "github.com/go-gen-ecosystem/halolog/registry"
	"github.com/go-gen-ecosystem/halolog/types"
)

// PipelineWriter defines the interface for pipeline write operations.
//
//nolint:revive // Public write-path interface; "Pipeline" prefix names the abstraction and renaming would break consumers.
type PipelineWriter interface {
	Write(*clock.CachedClock, types.LogLevel, string, []types.TypedFieldData, int)
}

// EnhancedPipelineStrategy implements pipeline selection
type EnhancedPipelineStrategy struct {
	mu sync.RWMutex

	// Current active pipeline (atomic for lock-free reads)
	currentPipeline atomic.Value // types.Pipeline

	// Pipeline instances
	directPipeline *DirectPipeline
	simplePipeline *SimplePipeline
	fullPipeline   *FullPipeline

	// Configuration
	config    *PipelineConfig
	adapters  []types.Adapter
	fieldDict *fielddict.FieldDictionary

	// Performance tracking
	selectionCount  atomic.Int64
	pipelineMetrics map[string]*PipelineMetrics

	// Feature flags (cached for O(1) checks)
	hasMasking  atomic.Bool
	hasSampling atomic.Bool
	hasAsync    atomic.Bool
	hasColors   atomic.Bool

	// Optimization state
	optimalBufferSize int
	startupTime       time.Time

	// Race prevention state
	strategyID  uint64
	atomicState atomic.Uint64
}

// PipelineConfig contains configuration for pipeline strategy.
//
//nolint:revive // Public API config type; "Pipeline" prefix documents scope and renaming would break consumers.
type PipelineConfig struct {
	// Feature configuration
	EnableMasking  bool
	EnableSampling bool
	SampleRate     int32
	EnableAsync    bool
	EnableColors   bool
	ColorScheme    string

	// Performance configuration
	OptimalBufferSize int // Calculated at startup
	MaxBufferSize     int // Safety limit
	AsyncQueueSize    int // Async queue capacity
	AsyncWorkers      int // Number of async workers

	// Sensitive field configuration
	SensitiveFields   []string // Explicit sensitive field names
	SensitivePatterns []string // Pattern-based sensitive field matching

	// Adapter configuration
	Adapters []types.Adapter

	// Runtime configuration
	AutoOptimize      bool // Enable automatic optimization
	MetricsCollection bool // Enable metrics collection
}

// NewEnhancedPipelineStrategy creates a new enhanced pipeline strategy
func NewEnhancedPipelineStrategy(config *PipelineConfig, adapters []types.Adapter) *EnhancedPipelineStrategy {
	if config == nil {
		config = getDefaultPipelineConfig()
	}

	strategy := &EnhancedPipelineStrategy{
		config:          config,
		adapters:        adapters,
		fieldDict:       fielddict.GlobalFieldDictionary,
		pipelineMetrics: make(map[string]*PipelineMetrics),
		startupTime:     time.Now(),
		strategyID:      race.GlobalRacePrevention.GenerateStrategyID(),
	}

	// Initialize atomic state
	strategy.atomicState.Store(0)

	// Calculate optimal buffer size
	strategy.optimalBufferSize = calculateOptimalBufferSize(config)

	// Initialize feature flags
	strategy.hasMasking.Store(config.EnableMasking)
	strategy.hasSampling.Store(config.EnableSampling)
	strategy.hasAsync.Store(config.EnableAsync)
	strategy.hasColors.Store(config.EnableColors)

	// Initialize pipelines with race prevention
	strategy.initializePipelines()

	// Configure registries
	strategy.configureRegistries()

	// Select initial pipeline with race prevention
	strategy.selectOptimalPipeline()

	// Register strategy with race prevention
	if race.GlobalRacePrevention != nil {
		race.GlobalRacePrevention.RegisterStrategy(strategy.strategyID)
	}

	return strategy
}

// getDefaultPipelineConfig returns default configuration
func getDefaultPipelineConfig() *PipelineConfig {
	return &PipelineConfig{
		EnableMasking:     true,
		EnableSampling:    false,
		SampleRate:        100, // Sample everything
		EnableAsync:       false,
		EnableColors:      true,
		ColorScheme:       "default",
		OptimalBufferSize: 32, // P99 for most applications
		MaxBufferSize:     1000,
		AsyncQueueSize:    1000,
		AsyncWorkers:      4,
		AutoOptimize:      true,
		MetricsCollection: true,
	}
}

// calculateOptimalBufferSize determines optimal buffer size from configuration
func calculateOptimalBufferSize(config *PipelineConfig) int {
	if config.OptimalBufferSize > 0 {
		return config.OptimalBufferSize
	}
	return 32 // Default P99
}

// initializePipelines creates all pipeline instances
func (ps *EnhancedPipelineStrategy) initializePipelines() {
	// DirectPipeline: 0 fields, no features
	ps.directPipeline = &DirectPipeline{
		Adapters: ps.adapters,
	}

	// SimplePipelineEnhanced: fields, no features
	ps.simplePipeline = NewSimplePipeline(ps.adapters, ps.optimalBufferSize)

	// FullPipelineEnhanced: all features
	ps.fullPipeline = NewFullPipeline(ps.adapters, ps.fieldDict)

	// Configure full pipeline features
	if ps.config.EnableMasking {
		// Enable masking on the full pipeline
		ps.fullPipeline.HasMasking = true

		// Register sensitive fields
		for _, field := range ps.config.SensitiveFields {
			registry.GlobalSensitiveFieldRegistry.RegisterSensitiveField(field)
		}
		for _, pattern := range ps.config.SensitivePatterns {
			registry.GlobalSensitiveFieldRegistry.RegisterSensitivePattern(pattern)
		}
	}

	if ps.config.EnableSampling {
		ps.fullPipeline.HasSampling = true
		ps.fullPipeline.SetSampleRate(ps.config.SampleRate)
	}

	if ps.config.EnableColors {
		ps.fullPipeline.HasColors = true
		registry.GlobalColorRegistry.SetColorScheme(ps.config.ColorScheme)
	}
}

// configureRegistries configures global registries
func (ps *EnhancedPipelineStrategy) configureRegistries() {
	// Configure sensitive field registry
	if len(ps.config.SensitiveFields) > 0 || len(ps.config.SensitivePatterns) > 0 {
		config := map[string]interface{}{
			"sensitive_fields":   ps.config.SensitiveFields,
			"sensitive_patterns": ps.config.SensitivePatterns,
		}
		registry.RegisterSensitiveFieldsFromConfig(config)
	}

	// Configure color registry
	if ps.config.EnableColors {
		config := map[string]interface{}{
			"colors_enabled": true,
			"color_scheme":   ps.config.ColorScheme,
		}
		registry.ConfigureColorsFromConfig(config)
	}
}

// Write implements the pipeline strategy with smart selection
func (ps *EnhancedPipelineStrategy) Write(clock *clock.CachedClock, level types.LogLevel, msg string, fields []types.TypedFieldData, fieldCount int) {
	startTime := time.Now()

	// Race prevention: safe pipeline access
	if race.GlobalRacePrevention != nil {
		race.GlobalRacePrevention.SafeStrategyAccess(ps.strategyID, "write")
	}

	// Get current pipeline (atomic read for lock-free fast path)
	pipeline := unboxPipeline(ps.currentPipeline.Load())
	if pipeline == nil {
		// Fallback to simple pipeline
		pipeline = ps.simplePipeline
	}

	// Execute pipeline with race prevention
	if p, ok := pipeline.(PipelineWriter); ok {
		// Thread-safe pipeline execution
		if race.GlobalRacePrevention != nil {
			race.GlobalRacePrevention.SafePipelineExecution(ps.strategyID, "write", pipeline)
		}
		p.Write(clock, level, msg, fields, fieldCount)
	}

	// Record metrics
	if ps.config.MetricsCollection {
		duration := time.Since(startTime)
		ps.recordMetrics(duration, fieldCount)
	}
}

// selectOptimalPipeline chooses the best pipeline based on configuration. It
// acquires ps.mu; callers already holding the lock must use
// selectOptimalPipelineLocked instead (see UpdateConfiguration).
func (ps *EnhancedPipelineStrategy) selectOptimalPipeline() {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.selectOptimalPipelineLocked()
}

// selectOptimalPipelineLocked performs pipeline selection assuming the caller
// already holds ps.mu.
func (ps *EnhancedPipelineStrategy) selectOptimalPipelineLocked() {
	// Race prevention: safe pipeline selection
	if race.GlobalRacePrevention != nil {
		race.GlobalRacePrevention.SafeStrategyAccess(ps.strategyID, "select_pipeline")
	}

	var selectedPipeline interface{}

	// Pipeline selection logic (O(1) checks)
	switch {
	case ps.shouldUseDirectPipeline():
		selectedPipeline = ps.directPipeline
	case ps.shouldUseSimplePipeline():
		selectedPipeline = ps.simplePipeline
	default:
		selectedPipeline = ps.fullPipeline
	}

	// Race prevention: validate pipeline before atomic update
	if race.GlobalRacePrevention != nil {
		race.GlobalRacePrevention.SafePipelineExecution(ps.strategyID, "select_pipeline", selectedPipeline)
	}

	// Atomic update for lock-free reads. The three pipeline types are distinct
	// concrete types, so they are boxed in a single wrapper type — an atomic.Value
	// panics if stored values have inconsistent concrete types.
	ps.currentPipeline.Store(pipelineBox{selectedPipeline})
	ps.selectionCount.Add(1)

	// Update atomic state for race detection
	ps.atomicState.Add(1)
}

// shouldUseDirectPipeline determines if DirectPipeline should be used
func (ps *EnhancedPipelineStrategy) shouldUseDirectPipeline() bool {
	// DirectPipeline: 0 fields, no features
	return !ps.hasMasking.Load() && !ps.hasSampling.Load() &&
		!ps.hasAsync.Load() && !ps.hasColors.Load()
}

// shouldUseSimplePipeline determines if SimplePipelineEnhanced should be used
func (ps *EnhancedPipelineStrategy) shouldUseSimplePipeline() bool {
	// SimplePipelineEnhanced: fields, no pipeline-level features
	// Field-level colors can use SimplePipeline for better performance
	// Only pipeline-level colors (hasColors) require FullPipeline
	return !ps.hasMasking.Load() && !ps.hasSampling.Load() &&
		!ps.hasAsync.Load()
}

// recordMetrics records pipeline execution metrics
func (ps *EnhancedPipelineStrategy) recordMetrics(duration time.Duration, fieldCount int) {
	metrics := PipelineMetrics{
		ProcessedCount: 1,
		TotalTime:      duration.Nanoseconds(),
	}

	// Determine pipeline name for metrics
	pipelineName := ps.getCurrentPipelineName()

	// Record to global collector
	GlobalMetricsCollector.RecordPipelineExecution(pipelineName, metrics, duration)
}

// pipelineBox wraps the selected pipeline so the distinct concrete pipeline
// types (*DirectPipeline, *SimplePipeline, *FullPipeline) can share a single
// atomic.Value, which requires one consistent concrete type.
type pipelineBox struct{ p any }

// unboxPipeline returns the pipeline stored in an atomic.Value (nil if unset).
func unboxPipeline(v any) any {
	if b, ok := v.(pipelineBox); ok {
		return b.p
	}
	return nil
}

// getCurrentPipelineName returns the name of the current pipeline
func (ps *EnhancedPipelineStrategy) getCurrentPipelineName() string {
	pipeline := unboxPipeline(ps.currentPipeline.Load())

	switch pipeline.(type) {
	case *DirectPipeline:
		return "direct"
	case *SimplePipeline:
		return "simple"
	case *FullPipeline:
		return "full"
	default:
		return "unknown"
	}
}

// UpdateConfiguration updates pipeline configuration and reselects optimal pipeline
func (ps *EnhancedPipelineStrategy) UpdateConfiguration(config *PipelineConfig) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	// Race prevention: safe configuration update
	if race.GlobalRacePrevention != nil {
		race.GlobalRacePrevention.SafeStrategyAccess(ps.strategyID, "update_config")
	}

	ps.config = config

	// Update feature flags
	ps.hasMasking.Store(config.EnableMasking)
	ps.hasSampling.Store(config.EnableSampling)
	ps.hasAsync.Store(config.EnableAsync)
	ps.hasColors.Store(config.EnableColors)

	// Recalculate optimal buffer size
	ps.optimalBufferSize = calculateOptimalBufferSize(config)

	// Reconfigure pipelines with race prevention
	ps.reconfigurePipelines()

	// Reselect optimal pipeline (we already hold ps.mu, so use the locked variant
	// to avoid re-acquiring the non-reentrant mutex and deadlocking).
	ps.selectOptimalPipelineLocked()

	// Update atomic state for race detection
	ps.atomicState.Add(1)
}

// reconfigurePipelines updates pipeline configurations
func (ps *EnhancedPipelineStrategy) reconfigurePipelines() {
	// Update simple pipeline buffer size
	ps.simplePipeline.OptimalBufferSize = ps.optimalBufferSize

	// Update full pipeline features
	if ps.config.EnableSampling {
		ps.fullPipeline.SetSampleRate(ps.config.SampleRate)
	}

	if ps.config.EnableColors {
		registry.GlobalColorRegistry.SetColorScheme(ps.config.ColorScheme)
	}
}

// GetMetrics returns current pipeline metrics
func (ps *EnhancedPipelineStrategy) GetMetrics() StrategyMetrics {
	return StrategyMetrics{
		SelectionCount:    ps.selectionCount.Load(),
		CurrentPipeline:   ps.getCurrentPipelineName(),
		OptimalBufferSize: ps.optimalBufferSize,
		HasMasking:        ps.hasMasking.Load(),
		HasSampling:       ps.hasSampling.Load(),
		HasAsync:          ps.hasAsync.Load(),
		HasColors:         ps.hasColors.Load(),
		Uptime:            time.Since(ps.startupTime),
		GlobalMetrics:     GlobalMetricsCollector.GetGlobalMetrics(),
	}
}

// StrategyMetrics contains strategy performance metrics
type StrategyMetrics struct {
	SelectionCount    int64
	CurrentPipeline   string
	OptimalBufferSize int
	HasMasking        bool
	HasSampling       bool
	HasAsync          bool
	HasColors         bool
	Uptime            time.Duration
	GlobalMetrics     GlobalPipelineMetrics
}

// Shutdown gracefully shuts down all pipelines
func (ps *EnhancedPipelineStrategy) Shutdown() {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	// Race prevention: safe shutdown
	if race.GlobalRacePrevention != nil {
		race.GlobalRacePrevention.SafeStrategyAccess(ps.strategyID, "shutdown")
	}

	// Shutdown pipelines with race prevention
	if ps.directPipeline != nil {
		if race.GlobalRacePrevention != nil {
			race.GlobalRacePrevention.SafePipelineExecution(ps.strategyID, "shutdown", ps.directPipeline)
		}
		ps.directPipeline.Reset()
	}
	if ps.simplePipeline != nil {
		if race.GlobalRacePrevention != nil {
			race.GlobalRacePrevention.SafePipelineExecution(ps.strategyID, "shutdown", ps.simplePipeline)
		}
		ps.simplePipeline.Reset()
	}
	if ps.fullPipeline != nil {
		if race.GlobalRacePrevention != nil {
			race.GlobalRacePrevention.SafePipelineExecution(ps.strategyID, "shutdown", ps.fullPipeline)
		}
		ps.fullPipeline.Reset()
	}

	// Close enhanced pool if initialized
	if pool.GlobalPool != nil {
		pool.GlobalPool.Close()
	}

	// Unregister strategy from race prevention
	if race.GlobalRacePrevention != nil {
		race.GlobalRacePrevention.UnregisterStrategy(ps.strategyID)
	}
}

// GlobalEnhancedStrategy is the singleton instance for system-wide use
var GlobalEnhancedStrategy *EnhancedPipelineStrategy

// InitializeEnhancedStrategy initializes the global enhanced pipeline strategy
func InitializeEnhancedStrategy(config *PipelineConfig, adapters []types.Adapter) {
	if GlobalEnhancedStrategy != nil {
		return // Already initialized
	}

	GlobalEnhancedStrategy = NewEnhancedPipelineStrategy(config, adapters)
}

// ConfigureFromApplicationProfile configures the strategy based on application profiling
func ConfigureFromApplicationProfile(profile map[string]interface{}) *PipelineConfig {
	config := getDefaultPipelineConfig()

	if profile == nil {
		return config
	}

	// Extract profiling data
	if p99Fields, ok := profile["p99_field_count"].(int); ok {
		config.OptimalBufferSize = p99Fields + 5
	}

	if appType, ok := profile["application_type"].(string); ok {
		// Adjust configuration based on application type
		switch appType {
		case "high_performance":
			config.EnableAsync = false    // Sync for lowest latency
			config.EnableColors = false   // No colors for speed
			config.EnableSampling = false // Sample everything
		case "web_api":
			config.EnableColors = true
			config.EnableMasking = true
		case "batch_processor":
			config.EnableAsync = true
			config.AsyncQueueSize = 10000
		}
	}

	if sensitiveFields, ok := profile["sensitive_fields"].([]string); ok {
		config.SensitiveFields = sensitiveFields
	}

	return config
}
