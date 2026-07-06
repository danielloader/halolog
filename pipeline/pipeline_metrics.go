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
)

// PipelineMetrics contains performance metrics for pipeline monitoring
type PipelineMetrics struct {
	ProcessedCount int64
	MaskedCount    int64
	SampledCount   int64
	QueueSize      int

	// Timing metrics (nanoseconds)
	TotalTime int64
	MinTime   int64
	MaxTime   int64
	AvgTime   float64

	// Error metrics
	ErrorCount   int64
	DroppedCount int64

	// Memory metrics
	Allocations    int64
	BytesAllocated int64
}

// PipelineMetricsCollector collects and aggregates metrics across all pipelines
type PipelineMetricsCollector struct {
	mu sync.RWMutex

	// Global metrics
	totalProcessed   atomic.Int64
	totalMasked      atomic.Int64
	totalSampled     atomic.Int64
	totalErrors      atomic.Int64
	totalDropped     atomic.Int64
	totalAllocations atomic.Int64
	totalBytes       atomic.Int64

	// Timing metrics
	totalTime atomic.Int64
	minTime   atomic.Int64
	maxTime   atomic.Int64

	// Per-pipeline metrics
	pipelineMetrics map[string]*PipelineMetrics

	// Collection state
	startTime        time.Time
	lastResetTime    time.Time
	collectionPeriod time.Duration
}

// NewPipelineMetricsCollector creates a new metrics collector
func NewPipelineMetricsCollector() *PipelineMetricsCollector {
	now := time.Now()
	return &PipelineMetricsCollector{
		pipelineMetrics:  make(map[string]*PipelineMetrics),
		startTime:        now,
		lastResetTime:    now,
		collectionPeriod: 60 * time.Second, // 1 minute default
	}
}

// RecordPipelineMetrics records metrics from a pipeline execution
func (c *PipelineMetricsCollector) RecordPipelineMetrics(pipelineName string, metrics PipelineMetrics, duration time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Update per-pipeline metrics
	if existing, exists := c.pipelineMetrics[pipelineName]; exists {
		// Aggregate with existing metrics
		existing.ProcessedCount += metrics.ProcessedCount
		existing.MaskedCount += metrics.MaskedCount
		existing.SampledCount += metrics.SampledCount
		existing.ErrorCount += metrics.ErrorCount
		existing.DroppedCount += metrics.DroppedCount
		existing.Allocations += metrics.Allocations
		existing.BytesAllocated += metrics.BytesAllocated

		// Update timing metrics
		existing.TotalTime += duration.Nanoseconds()
		if existing.MinTime == 0 || duration.Nanoseconds() < existing.MinTime {
			existing.MinTime = duration.Nanoseconds()
		}
		if duration.Nanoseconds() > existing.MaxTime {
			existing.MaxTime = duration.Nanoseconds()
		}

		// Recalculate average
		if existing.ProcessedCount > 0 {
			existing.AvgTime = float64(existing.TotalTime) / float64(existing.ProcessedCount)
		}
	} else {
		// First time recording for this pipeline
		metrics.TotalTime = duration.Nanoseconds()
		metrics.MinTime = duration.Nanoseconds()
		metrics.MaxTime = duration.Nanoseconds()
		metrics.AvgTime = float64(duration.Nanoseconds())
		c.pipelineMetrics[pipelineName] = &metrics
	}

	// Update global atomic counters
	c.totalProcessed.Add(metrics.ProcessedCount)
	c.totalMasked.Add(metrics.MaskedCount)
	c.totalSampled.Add(metrics.SampledCount)
	c.totalErrors.Add(metrics.ErrorCount)
	c.totalDropped.Add(metrics.DroppedCount)
	c.totalAllocations.Add(metrics.Allocations)
	c.totalBytes.Add(metrics.BytesAllocated)

	// Update timing metrics
	c.totalTime.Add(duration.Nanoseconds())

	// Update min/max timing (atomic operations)
	currentMin := c.minTime.Load()
	if currentMin == 0 || duration.Nanoseconds() < currentMin {
		c.minTime.Store(duration.Nanoseconds())
	}

	currentMax := c.maxTime.Load()
	if duration.Nanoseconds() > currentMax {
		c.maxTime.Store(duration.Nanoseconds())
	}
}

// GetGlobalMetrics returns aggregated metrics across all pipelines
func (c *PipelineMetricsCollector) GetGlobalMetrics() GlobalPipelineMetrics {
	return GlobalPipelineMetrics{
		TotalProcessed:   c.totalProcessed.Load(),
		TotalMasked:      c.totalMasked.Load(),
		TotalSampled:     c.totalSampled.Load(),
		TotalErrors:      c.totalErrors.Load(),
		TotalDropped:     c.totalDropped.Load(),
		TotalAllocations: c.totalAllocations.Load(),
		TotalBytes:       c.totalBytes.Load(),
		MinTime:          c.minTime.Load(),
		MaxTime:          c.maxTime.Load(),
		AvgTime:          c.calculateGlobalAvgTime(),
		Uptime:           time.Since(c.startTime),
	}
}

// GetPipelineMetrics returns metrics for a specific pipeline
func (c *PipelineMetricsCollector) GetPipelineMetrics(pipelineName string) (PipelineMetrics, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if metrics, exists := c.pipelineMetrics[pipelineName]; exists {
		return *metrics, true
	}

	return PipelineMetrics{}, false
}

// GetAllPipelineMetrics returns metrics for all pipelines
func (c *PipelineMetricsCollector) GetAllPipelineMetrics() map[string]PipelineMetrics {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make(map[string]PipelineMetrics, len(c.pipelineMetrics))
	for name, metrics := range c.pipelineMetrics {
		result[name] = *metrics
	}

	return result
}

// Reset resets all metrics
func (c *PipelineMetricsCollector) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Reset atomic counters
	c.totalProcessed.Store(0)
	c.totalMasked.Store(0)
	c.totalSampled.Store(0)
	c.totalErrors.Store(0)
	c.totalDropped.Store(0)
	c.totalAllocations.Store(0)
	c.totalBytes.Store(0)
	c.totalTime.Store(0)
	c.minTime.Store(0)
	c.maxTime.Store(0)

	// Reset per-pipeline metrics
	c.pipelineMetrics = make(map[string]*PipelineMetrics)
	c.lastResetTime = time.Now()
}

// calculateGlobalAvgTime calculates the global average execution time
func (c *PipelineMetricsCollector) calculateGlobalAvgTime() float64 {
	totalProcessed := c.totalProcessed.Load()
	totalTime := c.totalTime.Load()

	if totalProcessed == 0 {
		return 0
	}

	return float64(totalTime) / float64(totalProcessed)
}

// SetCollectionPeriod sets the metrics collection period
func (c *PipelineMetricsCollector) SetCollectionPeriod(period time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.collectionPeriod = period
}

// GlobalPipelineMetrics contains aggregated metrics across all pipelines
type GlobalPipelineMetrics struct {
	TotalProcessed   int64
	TotalMasked      int64
	TotalSampled     int64
	TotalErrors      int64
	TotalDropped     int64
	TotalAllocations int64
	TotalBytes       int64
	MinTime          int64
	MaxTime          int64
	AvgTime          float64
	Uptime           time.Duration
}

// PerformanceReport generates a comprehensive performance report
func (c *PipelineMetricsCollector) GeneratePerformanceReport() PerformanceReport {
	global := c.GetGlobalMetrics()
	pipelines := c.GetAllPipelineMetrics()

	return PerformanceReport{
		Timestamp:        time.Now(),
		GlobalMetrics:    global,
		PipelineMetrics:  pipelines,
		PerformanceGrade: c.calculatePerformanceGrade(global),
		Recommendations:  c.generateRecommendations(global, pipelines),
	}
}

// PerformanceReport contains comprehensive performance analysis
type PerformanceReport struct {
	Timestamp        time.Time
	GlobalMetrics    GlobalPipelineMetrics
	PipelineMetrics  map[string]PipelineMetrics
	PerformanceGrade string
	Recommendations  []string
}

// calculatePerformanceGrade evaluates overall performance
func (c *PipelineMetricsCollector) calculatePerformanceGrade(metrics GlobalPipelineMetrics) string {
	// Performance grading based on industry benchmarks
	avgTime := metrics.AvgTime

	if avgTime < 10 { // Sub-10ns: World-class
		return "A+ (World Class)"
	} else if avgTime < 20 { // Sub-20ns: Excellent
		return "A (Excellent)"
	} else if avgTime < 50 { // Sub-50ns: Good
		return "B (Good)"
	} else if avgTime < 100 { // Sub-100ns: Acceptable
		return "C (Acceptable)"
	} else { // >100ns: Needs improvement
		return "D (Needs Improvement)"
	}
}

// generateRecommendations provides optimization suggestions
func (c *PipelineMetricsCollector) generateRecommendations(global GlobalPipelineMetrics, pipelines map[string]PipelineMetrics) []string {
	var recommendations []string

	// Check for performance issues
	if global.AvgTime > 50 {
		recommendations = append(recommendations, "Consider optimizing pipeline configuration for better performance")
	}

	if global.TotalErrors > global.TotalProcessed/100 { // >1% error rate
		recommendations = append(recommendations, "High error rate detected - check adapter configurations")
	}

	if global.TotalDropped > global.TotalProcessed/1000 { // >0.1% drop rate
		recommendations = append(recommendations, "Log drops detected - consider increasing buffer sizes or async queue capacity")
	}

	if global.TotalAllocations > global.TotalProcessed*2 { // >2 allocations per log
		recommendations = append(recommendations, "High allocation rate - optimize object pooling configuration")
	}

	// Pipeline-specific recommendations
	for name, metrics := range pipelines {
		if float64(metrics.MaxTime) > metrics.AvgTime*10 { // High variance
			recommendations = append(recommendations,
				"High timing variance in "+name+" pipeline - check for resource contention")
		}
	}

	if len(recommendations) == 0 {
		recommendations = append(recommendations, "Performance is within acceptable parameters")
	}

	return recommendations
}

// GlobalMetricsCollector is the singleton instance for system-wide use
var GlobalMetricsCollector = NewPipelineMetricsCollector()

// RecordPipelineExecution records pipeline execution metrics
func (c *PipelineMetricsCollector) RecordPipelineExecution(pipelineName string, metrics PipelineMetrics, duration time.Duration) {
	c.RecordPipelineMetrics(pipelineName, metrics, duration)
}

// RecordPipelineExecution is a convenience function for recording pipeline execution
func RecordPipelineExecution(pipelineName string, metrics PipelineMetrics, duration time.Duration) {
	GlobalMetricsCollector.RecordPipelineMetrics(pipelineName, metrics, duration)
}
