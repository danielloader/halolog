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
	"time"

	"github.com/go-gen-ecosystem/halolog/types"
)

// CollectorAdapter adapts metrics.Collector to implement types.MetricsCollector
type CollectorAdapter struct {
	collector *Collector
}

// NewCollectorAdapter creates a new adapter for the metrics collector
func NewCollectorAdapter(collector *Collector) types.MetricsCollector {
	return &CollectorAdapter{
		collector: collector,
	}
}

// RecordLog records a log entry
func (a *CollectorAdapter) RecordLog(level types.LogLevel) {
	levelStr := levelToString(level)
	a.collector.RecordLog(levelStr, 0, false, false)
}

// RecordError records a write error
func (a *CollectorAdapter) RecordError() {
	a.collector.RecordLog("ERROR", 0, false, false)
}

// RecordErrorHandled records a handled error
func (a *CollectorAdapter) RecordErrorHandled() {
	// metrics.Collector doesn't distinguish handled errors
	// We can track this in error logs
	a.collector.RecordLog("ERROR", 0, false, false)
}

// RecordDropped records a dropped log
func (a *CollectorAdapter) RecordDropped() {
	a.collector.RecordLog("", 0, true, false)
}

// GetDroppedCount returns the number of dropped logs
func (a *CollectorAdapter) GetDroppedCount() int64 {
	snapshot := a.collector.GetSnapshot()
	return int64(snapshot.DroppedLogs)
}

// Reset resets all metrics
func (a *CollectorAdapter) Reset() {
	// metrics.Collector doesn't have a Reset method
	// We would need to create a new collector or add Reset to metrics.Collector
	// For now, this is a no-op
}

// RecordComponent records a log for a component
func (a *CollectorAdapter) RecordComponent(component string) {
	// metrics.Collector doesn't track component stats
	// This is a feature only in core/metrics_collector.go
	// For now, this is a no-op
}

// GetStats returns a snapshot of the current statistics
func (a *CollectorAdapter) GetStats() *types.MetricsStats {
	snapshot := a.collector.GetSnapshot()

	return &types.MetricsStats{
		TotalLogs:      int64(snapshot.TotalLogs),
		DebugCount:     int64(snapshot.DebugLogs),
		InfoCount:      int64(snapshot.InfoLogs),
		WarnCount:      int64(snapshot.WarnLogs),
		ErrorCount:     int64(snapshot.ErrorLogs),
		FatalCount:     0, // metrics.Collector doesn't separate Fatal
		ErrorsHandled:  0, // metrics.Collector doesn't track handled errors separately
		ErrorsTotal:    int64(snapshot.ErrorLogs),
		DroppedCount:   int64(snapshot.DroppedLogs),
		ComponentStats: make(map[string]int64), // Not supported by metrics.Collector
	}
}

// Clone creates a deep copy of the metrics collector
func (a *CollectorAdapter) Clone() types.MetricsCollector {
	// metrics.Collector doesn't have a Clone method
	// Create a new collector with same enabled state
	newCollector := NewCollector(a.collector.IsEnabled())
	return NewCollectorAdapter(newCollector)
}

// levelToString converts types.LogLevel to string
func levelToString(level types.LogLevel) string {
	switch level {
	case types.TraceLevel:
		return "TRACE"
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
	case types.PanicLevel:
		return "PANIC"
	default:
		return "INFO"
	}
}

// RecordLogWithLatency records a log entry with latency tracking
// This is an extended method that leverages metrics.Collector's latency tracking
func (a *CollectorAdapter) RecordLogWithLatency(level types.LogLevel, latency time.Duration) {
	levelStr := levelToString(level)
	a.collector.RecordLog(levelStr, latency, false, false)
}
