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
// Package metrics provides metrics collection
// Author: Admilson B. F. Cossa

package metrics

import (
	"encoding/json"
	"net/http"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// Collector provides comprehensive logging metrics with minimal overhead
type Collector struct {
	mu sync.RWMutex

	// Core metrics - atomic for lock-free reads
	totalLogs atomic.Uint64
	errorLogs atomic.Uint64
	warnLogs  atomic.Uint64
	infoLogs  atomic.Uint64
	debugLogs atomic.Uint64

	// Performance metrics
	totalLatency atomic.Uint64 // nanoseconds
	maxLatency   atomic.Uint64 // nanoseconds
	minLatency   atomic.Uint64 // nanoseconds

	// Concurrency metrics
	activeGoroutines atomic.Int32
	contentionCount  atomic.Uint64

	// Reliability metrics
	droppedLogs atomic.Uint64
	retriedLogs atomic.Uint64

	// Time-based metrics (circular buffer for recent data)
	minuteMetrics [60]*MinuteMetrics // Last 60 minutes
	currentMinute atomic.Int32

	// Configuration
	enabled bool
}

// MinuteMetrics holds metrics for a single minute
type MinuteMetrics struct {
	Timestamp   time.Time
	TotalLogs   uint64
	ErrorLogs   uint64
	WarnLogs    uint64
	InfoLogs    uint64
	DebugLogs   uint64
	AvgLatency  uint64 // nanoseconds
	MaxLatency  uint64 // nanoseconds
	MinLatency  uint64 // nanoseconds
	DroppedLogs uint64
	RetriedLogs uint64
}

// MetricsSnapshot represents a point-in-time snapshot of metrics.
//
//nolint:revive // Established public API name; the "Metrics" prefix documents the payload and renaming would break external consumers of the metrics package.
type MetricsSnapshot struct {
	Timestamp        time.Time        `json:"timestamp"`
	TotalLogs        uint64           `json:"total_logs"`
	ErrorLogs        uint64           `json:"error_logs"`
	WarnLogs         uint64           `json:"warn_logs"`
	InfoLogs         uint64           `json:"info_logs"`
	DebugLogs        uint64           `json:"debug_logs"`
	AvgLatency       uint64           `json:"avg_latency_ns"`
	MaxLatency       uint64           `json:"max_latency_ns"`
	MinLatency       uint64           `json:"min_latency_ns"`
	DroppedLogs      uint64           `json:"dropped_logs"`
	RetriedLogs      uint64           `json:"retried_logs"`
	ActiveGoroutines int32            `json:"active_goroutines"`
	ContentionCount  uint64           `json:"contention_count"`
	MemoryUsage      *MemoryMetrics   `json:"memory_usage,omitempty"`
	RecentMinutes    []*MinuteMetrics `json:"recent_minutes,omitempty"`
}

// MemoryMetrics contains memory usage statistics
type MemoryMetrics struct {
	Alloc        uint64 `json:"alloc"`
	TotalAlloc   uint64 `json:"total_alloc"`
	Sys          uint64 `json:"sys"`
	NumGC        uint32 `json:"num_gc"`
	NumGoroutine int    `json:"num_goroutine"`
}

// NewCollector creates a new metrics collector
func NewCollector(enabled bool) *Collector {
	c := &Collector{
		enabled: enabled,
	}

	// Initialize minute metrics
	now := time.Now()
	currentMinute := now.Minute()
	for i := 0; i < 60; i++ {
		c.minuteMetrics[i] = &MinuteMetrics{
			Timestamp: now.Add(time.Duration(i-currentMinute) * time.Minute),
		}
	}
	c.currentMinute.Store(int32(currentMinute))

	return c
}

// RecordLog records metrics for a log entry
func (c *Collector) RecordLog(level string, latency time.Duration, dropped bool, retried bool) {
	if !c.enabled {
		return
	}

	// Update core metrics
	c.totalLogs.Add(1)

	switch level {
	case "ERROR":
		c.errorLogs.Add(1)
	case "WARN":
		c.warnLogs.Add(1)
	case "INFO":
		c.infoLogs.Add(1)
	case "DEBUG":
		c.debugLogs.Add(1)
	}

	// Update performance metrics
	latencyNs := uint64(latency.Nanoseconds())
	c.totalLatency.Add(latencyNs)

	// Update min/max latency
	for {
		currentMax := c.maxLatency.Load()
		if latencyNs <= currentMax || c.maxLatency.CompareAndSwap(currentMax, latencyNs) {
			break
		}
	}

	// Update min latency (handle zero case)
	for {
		currentMin := c.minLatency.Load()
		if currentMin == 0 || latencyNs < currentMin {
			if c.minLatency.CompareAndSwap(currentMin, latencyNs) {
				break
			}
		} else {
			break
		}
	}

	// Update reliability metrics
	if dropped {
		c.droppedLogs.Add(1)
	}
	if retried {
		c.retriedLogs.Add(1)
	}

	// Update current minute metrics
	c.updateMinuteMetrics(level, latencyNs, dropped, retried)
}

// updateMinuteMetrics updates the current minute's metrics
func (c *Collector) updateMinuteMetrics(level string, latency uint64, dropped bool, retried bool) {
	currentMinute := int(c.currentMinute.Load())
	minute := c.minuteMetrics[currentMinute]

	minute.TotalLogs++
	switch level {
	case "ERROR":
		minute.ErrorLogs++
	case "WARN":
		minute.WarnLogs++
	case "INFO":
		minute.InfoLogs++
	case "DEBUG":
		minute.DebugLogs++
	}

	// Update latency metrics
	if minute.TotalLogs == 1 {
		minute.MinLatency = latency
		minute.MaxLatency = latency
	} else {
		if latency < minute.MinLatency {
			minute.MinLatency = latency
		}
		if latency > minute.MaxLatency {
			minute.MaxLatency = latency
		}
	}

	// Calculate average latency
	minute.AvgLatency = (minute.AvgLatency*(minute.TotalLogs-1) + latency) / minute.TotalLogs

	if dropped {
		minute.DroppedLogs++
	}
	if retried {
		minute.RetriedLogs++
	}
}

// GetSnapshot returns a snapshot of current metrics
func (c *Collector) GetSnapshot() *MetricsSnapshot {
	if !c.enabled {
		return &MetricsSnapshot{
			Timestamp: time.Now(),
		}
	}

	// Get memory stats
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	totalLogs := c.totalLogs.Load()
	var avgLatency uint64
	if totalLogs > 0 {
		avgLatency = c.totalLatency.Load() / totalLogs
	}

	snapshot := &MetricsSnapshot{
		Timestamp:        time.Now(),
		TotalLogs:        totalLogs,
		ErrorLogs:        c.errorLogs.Load(),
		WarnLogs:         c.warnLogs.Load(),
		InfoLogs:         c.infoLogs.Load(),
		DebugLogs:        c.debugLogs.Load(),
		AvgLatency:       avgLatency,
		MaxLatency:       c.maxLatency.Load(),
		MinLatency:       c.minLatency.Load(),
		DroppedLogs:      c.droppedLogs.Load(),
		RetriedLogs:      c.retriedLogs.Load(),
		ActiveGoroutines: c.activeGoroutines.Load(),
		ContentionCount:  c.contentionCount.Load(),
		MemoryUsage: &MemoryMetrics{
			Alloc:        memStats.Alloc,
			TotalAlloc:   memStats.TotalAlloc,
			Sys:          memStats.Sys,
			NumGC:        memStats.NumGC,
			NumGoroutine: runtime.NumGoroutine(),
		},
	}

	// Copy recent minute metrics
	c.mu.RLock()
	recentMinutes := make([]*MinuteMetrics, 0, 10)
	currentMinute := int(c.currentMinute.Load())
	for i := 0; i < 10; i++ {
		idx := (currentMinute - i + 60) % 60
		if c.minuteMetrics[idx].TotalLogs > 0 {
			recentMinutes = append(recentMinutes, c.minuteMetrics[idx])
		}
	}
	c.mu.RUnlock()

	snapshot.RecentMinutes = recentMinutes
	return snapshot
}

// ServeHTTP implements http.Handler for metrics endpoint
func (c *Collector) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	snapshot := c.GetSnapshot()
	w.Header().Set("Content-Type", "application/json")

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(snapshot); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// SetEnabled enables or disables the collector
func (c *Collector) SetEnabled(enabled bool) {
	c.mu.Lock()
	c.enabled = enabled
	c.mu.Unlock()
}

// IsEnabled returns whether the collector is enabled
func (c *Collector) IsEnabled() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.enabled
}

// IncrementContention increments the contention counter
func (c *Collector) IncrementContention() {
	if c.enabled {
		c.contentionCount.Add(1)
	}
}

// SetActiveGoroutines sets the current number of active goroutines
func (c *Collector) SetActiveGoroutines(count int32) {
	if c.enabled {
		c.activeGoroutines.Store(count)
	}
}
