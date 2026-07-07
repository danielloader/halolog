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
// Package pipeline tests exercise the metrics collector and reporting.
// @author Admilson B. F. Cossa

package pipeline

import (
	"testing"
	"time"
)

// TestMetricsCollector_RecordAndAggregate verifies first-record and subsequent
// aggregation, including min/max/avg timing recalculation.
func TestMetricsCollector_RecordAndAggregate(t *testing.T) {
	c := NewPipelineMetricsCollector()

	// First record for "p1": seeds timing metrics.
	c.RecordPipelineMetrics("p1", PipelineMetrics{ProcessedCount: 10, MaskedCount: 2}, 30*time.Nanosecond)

	m, ok := c.GetPipelineMetrics("p1")
	if !ok {
		t.Fatal("expected p1 metrics to exist")
	}
	if m.ProcessedCount != 10 || m.MaskedCount != 2 {
		t.Fatalf("unexpected first record: %+v", m)
	}
	if m.MinTime != 30 || m.MaxTime != 30 {
		t.Fatalf("expected min=max=30 on first record, got min=%d max=%d", m.MinTime, m.MaxTime)
	}

	// Second record with a smaller duration -> updates min, aggregates counts.
	c.RecordPipelineMetrics("p1", PipelineMetrics{ProcessedCount: 5}, 10*time.Nanosecond)
	m, _ = c.GetPipelineMetrics("p1")
	if m.ProcessedCount != 15 {
		t.Fatalf("expected aggregated processed=15, got %d", m.ProcessedCount)
	}
	if m.MinTime != 10 {
		t.Fatalf("expected min updated to 10, got %d", m.MinTime)
	}
	if m.MaxTime != 30 {
		t.Fatalf("expected max stays 30, got %d", m.MaxTime)
	}
	if m.AvgTime <= 0 {
		t.Fatalf("expected positive avg time, got %f", m.AvgTime)
	}

	// Third record with a larger duration -> updates max.
	c.RecordPipelineMetrics("p1", PipelineMetrics{ProcessedCount: 1}, 100*time.Nanosecond)
	m, _ = c.GetPipelineMetrics("p1")
	if m.MaxTime != 100 {
		t.Fatalf("expected max updated to 100, got %d", m.MaxTime)
	}
}

// TestMetricsCollector_GlobalMetrics verifies global aggregation across pipelines.
func TestMetricsCollector_GlobalMetrics(t *testing.T) {
	c := NewPipelineMetricsCollector()
	c.RecordPipelineMetrics("a", PipelineMetrics{ProcessedCount: 4, ErrorCount: 1}, 20*time.Nanosecond)
	c.RecordPipelineMetrics("b", PipelineMetrics{ProcessedCount: 6, DroppedCount: 2}, 40*time.Nanosecond)

	g := c.GetGlobalMetrics()
	if g.TotalProcessed != 10 {
		t.Fatalf("expected TotalProcessed=10, got %d", g.TotalProcessed)
	}
	if g.TotalErrors != 1 {
		t.Fatalf("expected TotalErrors=1, got %d", g.TotalErrors)
	}
	if g.TotalDropped != 2 {
		t.Fatalf("expected TotalDropped=2, got %d", g.TotalDropped)
	}
	if g.MinTime != 20 || g.MaxTime != 40 {
		t.Fatalf("expected min=20 max=40, got min=%d max=%d", g.MinTime, g.MaxTime)
	}
	if g.AvgTime <= 0 {
		t.Fatalf("expected positive global avg, got %f", g.AvgTime)
	}
	if g.Uptime < 0 {
		t.Fatalf("expected non-negative uptime, got %v", g.Uptime)
	}
}

// TestMetricsCollector_GetPipelineMetrics_Missing verifies the not-found path.
func TestMetricsCollector_GetPipelineMetrics_Missing(t *testing.T) {
	c := NewPipelineMetricsCollector()
	if _, ok := c.GetPipelineMetrics("nope"); ok {
		t.Fatal("expected missing pipeline to report ok=false")
	}
}

// TestMetricsCollector_GetAllPipelineMetrics verifies the snapshot map.
func TestMetricsCollector_GetAllPipelineMetrics(t *testing.T) {
	c := NewPipelineMetricsCollector()
	c.RecordPipelineMetrics("x", PipelineMetrics{ProcessedCount: 1}, time.Nanosecond)
	c.RecordPipelineMetrics("y", PipelineMetrics{ProcessedCount: 2}, time.Nanosecond)

	all := c.GetAllPipelineMetrics()
	if len(all) != 2 {
		t.Fatalf("expected 2 pipelines, got %d", len(all))
	}
	if all["x"].ProcessedCount != 1 || all["y"].ProcessedCount != 2 {
		t.Fatalf("unexpected snapshot: %+v", all)
	}
}

// TestMetricsCollector_Reset verifies all metrics are zeroed.
func TestMetricsCollector_Reset(t *testing.T) {
	c := NewPipelineMetricsCollector()
	c.RecordPipelineMetrics("z", PipelineMetrics{ProcessedCount: 9}, 50*time.Nanosecond)

	c.Reset()

	g := c.GetGlobalMetrics()
	if g.TotalProcessed != 0 || g.MinTime != 0 || g.MaxTime != 0 {
		t.Fatalf("expected all-zero after reset, got %+v", g)
	}
	if len(c.GetAllPipelineMetrics()) != 0 {
		t.Fatal("expected per-pipeline metrics cleared after reset")
	}
}

// TestMetricsCollector_CalculateAvgTimeZero verifies avg is 0 with no data.
func TestMetricsCollector_CalculateAvgTimeZero(t *testing.T) {
	c := NewPipelineMetricsCollector()
	if got := c.GetGlobalMetrics().AvgTime; got != 0 {
		t.Fatalf("expected avg 0 with no data, got %f", got)
	}
}

// TestMetricsCollector_SetCollectionPeriod verifies the setter path runs safely.
func TestMetricsCollector_SetCollectionPeriod(t *testing.T) {
	c := NewPipelineMetricsCollector()
	c.SetCollectionPeriod(5 * time.Second)
	// No getter is exported; the assertion is that this does not panic and that
	// subsequent recording still works.
	c.RecordPipelineMetrics("p", PipelineMetrics{ProcessedCount: 1}, time.Nanosecond)
	if _, ok := c.GetPipelineMetrics("p"); !ok {
		t.Fatal("expected recording to still work after SetCollectionPeriod")
	}
}

// TestMetricsCollector_PerformanceGrades verifies grading buckets via the public
// report path by seeding controlled avg times.
func TestMetricsCollector_PerformanceGrades(t *testing.T) {
	testCases := []struct {
		name          string
		processed     int64
		durationNanos int64
		wantContains  string
	}{
		{name: "world_class", processed: 1, durationNanos: 5, wantContains: "A+"},
		{name: "excellent", processed: 1, durationNanos: 15, wantContains: "A ("},
		{name: "good", processed: 1, durationNanos: 30, wantContains: "B ("},
		{name: "acceptable", processed: 1, durationNanos: 70, wantContains: "C ("},
		{name: "needs_improvement", processed: 1, durationNanos: 200, wantContains: "D ("},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			c := NewPipelineMetricsCollector()
			c.RecordPipelineMetrics("p", PipelineMetrics{ProcessedCount: tc.processed},
				time.Duration(tc.durationNanos)*time.Nanosecond)
			report := c.GeneratePerformanceReport()
			if !contains(report.PerformanceGrade, tc.wantContains) {
				t.Fatalf("grade %q did not contain %q", report.PerformanceGrade, tc.wantContains)
			}
			if report.Timestamp.IsZero() {
				t.Fatal("expected report timestamp to be set")
			}
			if len(report.Recommendations) == 0 {
				t.Fatal("expected at least one recommendation")
			}
		})
	}
}

// TestMetricsCollector_Recommendations exercises the recommendation branches.
func TestMetricsCollector_Recommendations(t *testing.T) {
	t.Run("healthy_default", func(t *testing.T) {
		c := NewPipelineMetricsCollector()
		c.RecordPipelineMetrics("p", PipelineMetrics{ProcessedCount: 1000}, 5*time.Nanosecond)
		recs := c.GeneratePerformanceReport().Recommendations
		if !anyContains(recs, "within acceptable parameters") {
			t.Fatalf("expected healthy recommendation, got %v", recs)
		}
	})

	t.Run("slow_and_errors_and_drops_and_allocs", func(t *testing.T) {
		c := NewPipelineMetricsCollector()
		// Global AvgTime = totalTime / totalProcessed. With ProcessedCount=1 and a
		// 200ns duration, AvgTime=200 (>50ns) triggers the perf recommendation.
		// Error/drop/alloc thresholds are relative to TotalProcessed(=1):
		//   errors > 1/100 (=0), drops > 1/1000 (=0), allocs > 1*2 (=2).
		c.RecordPipelineMetrics("p", PipelineMetrics{
			ProcessedCount: 1,
			ErrorCount:     1,  // >1% error rate
			DroppedCount:   1,  // >0.1% drop rate
			Allocations:    10, // >2 allocs per log
		}, 200*time.Nanosecond)
		recs := c.GeneratePerformanceReport().Recommendations
		if !anyContains(recs, "optimizing pipeline configuration") {
			t.Fatalf("expected perf rec, got %v", recs)
		}
		if !anyContains(recs, "error rate") {
			t.Fatalf("expected error-rate rec, got %v", recs)
		}
		if !anyContains(recs, "drops detected") {
			t.Fatalf("expected drop rec, got %v", recs)
		}
		if !anyContains(recs, "allocation rate") {
			t.Fatalf("expected allocation rec, got %v", recs)
		}
	})

	t.Run("high_variance", func(t *testing.T) {
		c := NewPipelineMetricsCollector()
		// The per-pipeline variance branch fires when MaxTime > AvgTime*10, where
		// per-pipeline AvgTime = TotalTime/ProcessedCount. Seed many fast records
		// then one large outlier so the spread satisfies the inequality.
		for i := 0; i < 50; i++ {
			c.RecordPipelineMetrics("hv", PipelineMetrics{ProcessedCount: 1}, 1*time.Nanosecond)
		}
		c.RecordPipelineMetrics("hv", PipelineMetrics{ProcessedCount: 1}, 5000*time.Nanosecond)
		recs := c.GeneratePerformanceReport().Recommendations
		if !anyContains(recs, "timing variance") {
			t.Fatalf("expected variance rec, got %v", recs)
		}
	})
}

// TestRecordPipelineExecution_GlobalAndMethod verifies both the package-level
// convenience function and the method delegate to the collector.
func TestRecordPipelineExecution_GlobalAndMethod(t *testing.T) {
	// Snapshot the global before to compute deltas (avoids cross-test coupling).
	before := GlobalMetricsCollector.GetGlobalMetrics().TotalProcessed

	RecordPipelineExecution("global-conv", PipelineMetrics{ProcessedCount: 3}, time.Nanosecond)
	GlobalMetricsCollector.RecordPipelineExecution("global-method", PipelineMetrics{ProcessedCount: 4}, time.Nanosecond)

	after := GlobalMetricsCollector.GetGlobalMetrics().TotalProcessed
	if after-before != 7 {
		t.Fatalf("expected global processed delta of 7, got %d", after-before)
	}
}

func contains(s, sub string) bool {
	return len(sub) == 0 || indexOf(s, sub) >= 0
}

func anyContains(list []string, sub string) bool {
	for _, s := range list {
		if contains(s, sub) {
			return true
		}
	}
	return false
}

// indexOf is a tiny substring search to avoid importing strings for a single use.
func indexOf(s, sub string) int {
	n, m := len(s), len(sub)
	if m == 0 {
		return 0
	}
	for i := 0; i+m <= n; i++ {
		if s[i:i+m] == sub {
			return i
		}
	}
	return -1
}
