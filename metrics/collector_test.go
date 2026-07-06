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
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewCollector(t *testing.T) {
	collector := NewCollector(true)
	if collector == nil {
		t.Fatal("NewCollector returned nil")
	}

	if !collector.enabled {
		t.Error("Collector should be enabled")
	}
}

func TestRecordLog(t *testing.T) {
	collector := NewCollector(true)

	// Record some logs
	collector.RecordLog("INFO", time.Millisecond*10, false, false)
	collector.RecordLog("ERROR", time.Millisecond*20, false, true)
	collector.RecordLog("WARN", time.Millisecond*15, true, false)

	// Get snapshot
	snapshot := collector.GetSnapshot()

	if snapshot.TotalLogs != 3 {
		t.Errorf("Expected 3 total logs, got %d", snapshot.TotalLogs)
	}

	if snapshot.InfoLogs != 1 {
		t.Errorf("Expected 1 info log, got %d", snapshot.InfoLogs)
	}

	if snapshot.ErrorLogs != 1 {
		t.Errorf("Expected 1 error log, got %d", snapshot.ErrorLogs)
	}

	if snapshot.WarnLogs != 1 {
		t.Errorf("Expected 1 warn log, got %d", snapshot.WarnLogs)
	}

	if snapshot.DroppedLogs != 1 {
		t.Errorf("Expected 1 dropped log, got %d", snapshot.DroppedLogs)
	}

	if snapshot.RetriedLogs != 1 {
		t.Errorf("Expected 1 retried log, got %d", snapshot.RetriedLogs)
	}
}

func TestRecordLogDisabled(t *testing.T) {
	collector := NewCollector(false)

	// Record some logs - should be no-op
	collector.RecordLog("INFO", time.Millisecond*10, false, false)
	collector.RecordLog("ERROR", time.Millisecond*20, false, false)

	// Get snapshot - should be empty
	snapshot := collector.GetSnapshot()

	if snapshot.TotalLogs != 0 {
		t.Errorf("Expected 0 total logs when disabled, got %d", snapshot.TotalLogs)
	}
}

func TestLatencyMetrics(t *testing.T) {
	collector := NewCollector(true)

	// Record logs with different latencies
	collector.RecordLog("INFO", time.Millisecond*10, false, false)
	collector.RecordLog("INFO", time.Millisecond*50, false, false)
	collector.RecordLog("INFO", time.Millisecond*30, false, false)

	// Get snapshot
	snapshot := collector.GetSnapshot()

	// Average should be around 30ms (10+50+30)/3
	expectedAvg := uint64(30000000) // 30ms in nanoseconds
	tolerance := uint64(10000000)   // 10ms tolerance

	if snapshot.AvgLatency < expectedAvg-tolerance || snapshot.AvgLatency > expectedAvg+tolerance {
		t.Errorf("Expected avg latency around %d ns, got %d", expectedAvg, snapshot.AvgLatency)
	}

	if snapshot.MinLatency != uint64(10000000) { // 10ms
		t.Errorf("Expected min latency 10000000 ns, got %d", snapshot.MinLatency)
	}

	if snapshot.MaxLatency != uint64(50000000) { // 50ms
		t.Errorf("Expected max latency 50000000 ns, got %d", snapshot.MaxLatency)
	}
}

func TestContentionAndGoroutines(t *testing.T) {
	collector := NewCollector(true)

	// Test contention
	collector.IncrementContention()
	collector.IncrementContention()

	// Test goroutine count
	collector.SetActiveGoroutines(10)

	// Get snapshot
	snapshot := collector.GetSnapshot()

	if snapshot.ContentionCount != 2 {
		t.Errorf("Expected 2 contentions, got %d", snapshot.ContentionCount)
	}

	if snapshot.ActiveGoroutines != 10 {
		t.Errorf("Expected 10 active goroutines, got %d", snapshot.ActiveGoroutines)
	}
}

func TestMinuteMetrics(t *testing.T) {
	collector := NewCollector(true)

	// Record logs to update current minute metrics
	collector.RecordLog("INFO", time.Millisecond*10, false, false)
	collector.RecordLog("ERROR", time.Millisecond*20, false, false)

	// Get snapshot
	snapshot := collector.GetSnapshot()

	// Should have recent minute metrics
	if len(snapshot.RecentMinutes) == 0 {
		t.Error("Expected recent minute metrics")
	}

	// Check that at least one minute has data
	foundData := false
	for _, minute := range snapshot.RecentMinutes {
		if minute.TotalLogs > 0 {
			foundData = true
			if minute.TotalLogs != 2 {
				t.Errorf("Expected 2 logs in minute, got %d", minute.TotalLogs)
			}
			break
		}
	}

	if !foundData {
		t.Error("Expected at least one minute with data")
	}
}

func TestMemoryMetrics(t *testing.T) {
	collector := NewCollector(true)

	// Get snapshot
	snapshot := collector.GetSnapshot()

	// Memory metrics should be present
	if snapshot.MemoryUsage == nil {
		t.Fatal("Expected memory usage metrics")
	}

	// Basic sanity checks
	if snapshot.MemoryUsage.NumGoroutine < 1 {
		t.Error("Expected at least 1 goroutine")
	}

	if snapshot.MemoryUsage.NumGC < 0 {
		t.Error("GC count should not be negative")
	}
}

func TestServeHTTP(t *testing.T) {
	collector := NewCollector(true)

	// Record some logs to have data
	collector.RecordLog("INFO", time.Millisecond*10, false, false)
	collector.RecordLog("ERROR", time.Millisecond*20, true, false)

	// Create test request
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()

	// Serve the request
	collector.ServeHTTP(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Check content type
	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}

	// Parse response
	var snapshot MetricsSnapshot
	if err := json.Unmarshal(w.Body.Bytes(), &snapshot); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}

	// Verify data
	if snapshot.TotalLogs != 2 {
		t.Errorf("Expected 2 total logs in response, got %d", snapshot.TotalLogs)
	}

	if snapshot.DroppedLogs != 1 {
		t.Errorf("Expected 1 dropped log in response, got %d", snapshot.DroppedLogs)
	}
}

func TestServeHTTP_MethodNotAllowed(t *testing.T) {
	collector := NewCollector(true)

	// Test with POST method
	req := httptest.NewRequest(http.MethodPost, "/metrics", nil)
	w := httptest.NewRecorder()

	collector.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}

func TestServeHTTP_Disabled(t *testing.T) {
	collector := NewCollector(false)

	// Record logs while disabled
	collector.RecordLog("INFO", time.Millisecond*10, false, false)

	// Create test request
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()

	collector.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Parse response
	var snapshot MetricsSnapshot
	if err := json.Unmarshal(w.Body.Bytes(), &snapshot); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}

	// Should have timestamp but no data when disabled
	if snapshot.TotalLogs != 0 {
		t.Errorf("Expected 0 total logs when disabled, got %d", snapshot.TotalLogs)
	}

	if snapshot.Timestamp.IsZero() {
		t.Error("Expected non-zero timestamp")
	}
}

func TestSetEnabled(t *testing.T) {
	collector := NewCollector(false)

	// Initially disabled
	if collector.IsEnabled() {
		t.Error("Collector should be disabled initially")
	}

	// Enable
	collector.SetEnabled(true)
	if !collector.IsEnabled() {
		t.Error("Collector should be enabled after SetEnabled(true)")
	}

	// Disable again
	collector.SetEnabled(false)
	if collector.IsEnabled() {
		t.Error("Collector should be disabled after SetEnabled(false)")
	}
}

func TestIsEnabled(t *testing.T) {
	collector := NewCollector(true)

	if !collector.IsEnabled() {
		t.Error("IsEnabled should return true when enabled")
	}

	collector.SetEnabled(false)
	if collector.IsEnabled() {
		t.Error("IsEnabled should return false when disabled")
	}
}

func TestSetEnabled_RecordsData(t *testing.T) {
	collector := NewCollector(false)

	// Record logs while disabled
	collector.RecordLog("INFO", time.Millisecond*10, false, false)
	snapshot := collector.GetSnapshot()
	if snapshot.TotalLogs != 0 {
		t.Error("Should not record logs when disabled")
	}

	// Enable and record more logs
	collector.SetEnabled(true)
	collector.RecordLog("ERROR", time.Millisecond*20, false, false)
	snapshot = collector.GetSnapshot()
	if snapshot.TotalLogs != 1 {
		t.Errorf("Expected 1 total log after enabling, got %d", snapshot.TotalLogs)
	}

	if snapshot.ErrorLogs != 1 {
		t.Errorf("Expected 1 error log, got %d", snapshot.ErrorLogs)
	}
}
