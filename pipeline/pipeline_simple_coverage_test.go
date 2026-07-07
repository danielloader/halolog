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
// Package pipeline tests exercise the SimplePipeline field fast path.
// @author Admilson B. F. Cossa

package pipeline

import (
	"testing"

	"github.com/go-gen-ecosystem/halolog/types"
)

// TestNewSimplePipeline_BufferSizeClamping verifies the constructor normalizes
// the optimal buffer size (default when non-positive, cap at 1000).
func TestNewSimplePipeline_BufferSizeClamping(t *testing.T) {
	testCases := []struct {
		name string
		in   int
		want int
	}{
		{name: "zero_defaults_to_32", in: 0, want: 32},
		{name: "negative_defaults_to_32", in: -10, want: 32},
		{name: "in_range_preserved", in: 64, want: 64},
		{name: "over_cap_clamped", in: 5000, want: 1000},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var sink []captured
			p := NewSimplePipeline([]types.Adapter{newCapturingAdapter(&sink)}, tc.in)
			if p.OptimalBufferSize != tc.want {
				t.Fatalf("OptimalBufferSize=%d, want %d", p.OptimalBufferSize, tc.want)
			}
		})
	}
}

// TestSimplePipeline_WriteFields verifies fields flow through to the adapter via
// the static-buffer common path.
func TestSimplePipeline_WriteFields(t *testing.T) {
	var sink []captured
	p := NewSimplePipeline([]types.Adapter{newCapturingAdapter(&sink)}, 32)
	cc := newTestClock(t)

	fields := []types.TypedFieldData{
		{Key: "svc", Value: "auth"},
		{Key: "latency_ms", Value: 12},
	}
	p.Write(cc, types.InfoLevel, "request", fields, len(fields))

	if len(sink) != 1 {
		t.Fatalf("expected 1 write, got %d", len(sink))
	}
	if sink[0].message != "request" {
		t.Fatalf("message mismatch: %q", sink[0].message)
	}
	if sink[0].numStat != 2 {
		t.Fatalf("expected 2 static fields, got %d", sink[0].numStat)
	}
	if sink[0].statKeys[0] != "svc" || sink[0].statKeys[1] != "latency_ms" {
		t.Fatalf("field keys not preserved: %v", sink[0].statKeys)
	}
}

// TestSimplePipeline_WriteZeroFields verifies the pipeline still works with an
// empty field set (edge of the field path).
func TestSimplePipeline_WriteZeroFields(t *testing.T) {
	var sink []captured
	p := NewSimplePipeline([]types.Adapter{newCapturingAdapter(&sink)}, 32)
	cc := newTestClock(t)

	p.Write(cc, types.TraceLevel, "empty", nil, 0)

	if len(sink) != 1 || sink[0].numStat != 0 {
		t.Fatalf("expected 1 write with 0 fields, got %d writes", len(sink))
	}
}

// TestSimplePipeline_WriteOverflow verifies that field counts exceeding the
// static capacity spill into the dynamic Fields slice.
func TestSimplePipeline_WriteOverflow(t *testing.T) {
	var dynamicCount int
	var writes int
	adapter := &types.FuncAdapter{
		WriteFunc: func(e *types.LogEntry) error {
			writes++
			dynamicCount = len(e.Fields)
			return nil
		},
	}
	p := NewSimplePipeline([]types.Adapter{adapter}, 32)
	cc := newTestClock(t)

	const n = 50 // exceeds the pool's static capacity of 16
	fields := make([]types.TypedFieldData, n)
	for i := range fields {
		fields[i] = types.TypedFieldData{Key: "f", Value: i}
	}
	p.Write(cc, types.InfoLevel, "big", fields, n)

	if writes != 1 {
		t.Fatalf("expected 1 write, got %d", writes)
	}
	if dynamicCount != n {
		t.Fatalf("expected %d overflow fields, got %d", n, dynamicCount)
	}
}

// TestSimplePipeline_MultiAdapter verifies fan-out to multiple adapters.
func TestSimplePipeline_MultiAdapter(t *testing.T) {
	var sinkA, sinkB []captured
	p := NewSimplePipeline([]types.Adapter{
		newCapturingAdapter(&sinkA),
		newCapturingAdapter(&sinkB),
	}, 32)
	cc := newTestClock(t)

	fields := []types.TypedFieldData{{Key: "k", Value: "v"}}
	p.Write(cc, types.InfoLevel, "m", fields, 1)

	if len(sinkA) != 1 || len(sinkB) != 1 {
		t.Fatalf("expected both adapters to write, got A=%d B=%d", len(sinkA), len(sinkB))
	}
}

// TestSimplePipeline_HandleFieldOverflowStaticBranch drives handleFieldOverflow
// directly on a controlled entry to cover its static-capacity branch, which the
// main Write path does not normally reach.
func TestSimplePipeline_HandleFieldOverflowStaticBranch(t *testing.T) {
	var sink []captured
	p := NewSimplePipeline([]types.Adapter{newCapturingAdapter(&sink)}, 32)

	entry := &types.LogEntry{
		StaticFields: make([]types.TypedFieldData, 8),
	}
	fields := []types.TypedFieldData{
		{Key: "a", Value: 1},
		{Key: "b", Value: 2},
	}
	// fieldCount (2) <= cap(StaticFields) (8): static branch.
	p.handleFieldOverflow(entry, fields, len(fields))
	if entry.StaticFieldCount != 2 {
		t.Fatalf("expected static branch to set count=2, got %d", entry.StaticFieldCount)
	}
	if entry.StaticFields[0].Key != "a" || entry.StaticFields[1].Key != "b" {
		t.Fatalf("static fields not copied correctly: %+v", entry.StaticFields[:2])
	}

	// fieldCount (10) > cap(StaticFields) (8): dynamic branch.
	big := make([]types.TypedFieldData, 10)
	for i := range big {
		big[i] = types.TypedFieldData{Key: "x", Value: i}
	}
	p.handleFieldOverflow(entry, big, len(big))
	if entry.StaticFieldCount != 0 {
		t.Fatalf("expected dynamic branch to reset static count to 0, got %d", entry.StaticFieldCount)
	}
	if len(entry.Fields) != 10 {
		t.Fatalf("expected 10 dynamic fields, got %d", len(entry.Fields))
	}
}

// TestSimplePipeline_Reset verifies Reset clears adapters.
func TestSimplePipeline_Reset(t *testing.T) {
	var sink []captured
	p := NewSimplePipeline([]types.Adapter{
		newCapturingAdapter(&sink),
		newCapturingAdapter(&sink),
	}, 32)
	p.Reset()
	if len(p.Adapters) != 0 {
		t.Fatalf("expected adapters cleared, got %d", len(p.Adapters))
	}
}

// TestSimplePipeline_FieldDictNilSafe verifies the dict-integration helpers are
// safe no-ops when no global integration is configured.
func TestSimplePipeline_FieldDictNilSafe(t *testing.T) {
	var sink []captured
	p := NewSimplePipeline([]types.Adapter{newCapturingAdapter(&sink)}, 32)

	// Force the field-integration off to exercise the nil-guard branches
	// regardless of global initialization order in other tests.
	p.FieldIntegration = nil

	// RegisterFieldsWithDict: nil integration -> no-op, no panic.
	p.RegisterFieldsWithDict([]string{"a", "b"})

	// GetFieldID: nil integration -> (0, false).
	if id, ok := p.GetFieldID("anything"); ok || id != 0 {
		t.Fatalf("expected (0,false) with nil integration, got (%d,%v)", id, ok)
	}

	// ProcessFieldsWithDict: nil integration -> returns input unchanged.
	in := []types.TypedFieldData{{Key: "k", Value: 1}}
	out := p.ProcessFieldsWithDict(in)
	if len(out) != 1 || out[0].Key != "k" {
		t.Fatalf("expected input returned unchanged, got %+v", out)
	}

	// Empty inputs are handled as no-ops too.
	p.RegisterFieldsWithDict(nil)
	if same := p.ProcessFieldsWithDict(nil); same != nil {
		t.Fatalf("expected nil returned for nil input, got %+v", same)
	}
}

// TestSimplePipeline_FieldDictIntegrationActive exercises the dict-integration
// happy path when a real integration is wired in.
func TestSimplePipeline_FieldDictIntegrationActive(t *testing.T) {
	InitializeFieldDictIntegration()
	if GlobalFieldDictIntegration == nil {
		t.Skip("global field dict integration unavailable")
	}

	var sink []captured
	p := NewSimplePipeline([]types.Adapter{newCapturingAdapter(&sink)}, 32)
	p.FieldIntegration = GlobalFieldDictIntegration

	// Register keys; then a lookup for a registered key should succeed.
	p.RegisterFieldsWithDict([]string{"simple_pipeline_test_key"})
	if _, ok := p.GetFieldID("simple_pipeline_test_key"); !ok {
		t.Fatal("expected registered key to be found via GetFieldID")
	}

	// ProcessFieldsWithDict returns a same-length processed slice.
	in := []types.TypedFieldData{
		{Key: "simple_pipeline_test_key", Value: 1},
		{Key: "another_key", Value: 2},
	}
	out := p.ProcessFieldsWithDict(in)
	if len(out) != len(in) {
		t.Fatalf("expected %d processed fields, got %d", len(in), len(out))
	}
}

// TestIsSimplePathEligible verifies the eligibility predicate.
func TestIsSimplePathEligible(t *testing.T) {
	testCases := []struct {
		name       string
		fieldCount int
		masking    bool
		sampling   bool
		async      bool
		colors     bool
		want       bool
	}{
		{name: "fields_no_features", fieldCount: 2, want: true},
		{name: "zero_fields", fieldCount: 0, want: false},
		{name: "masking", fieldCount: 1, masking: true, want: false},
		{name: "sampling", fieldCount: 1, sampling: true, want: false},
		{name: "async", fieldCount: 1, async: true, want: false},
		{name: "colors_not_simple", fieldCount: 1, colors: true, want: false},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := IsSimplePathEligible(tc.fieldCount, tc.masking, tc.sampling, tc.async, tc.colors)
			if got != tc.want {
				t.Fatalf("IsSimplePathEligible=%v, want %v", got, tc.want)
			}
		})
	}
}

// TestCalculateOptimalBufferSize verifies the priority order of buffer sizing.
func TestCalculateOptimalBufferSize(t *testing.T) {
	testCases := []struct {
		name       string
		configured int
		profiled   int
		want       int
	}{
		{name: "configured_wins", configured: 40, profiled: 20, want: 40},
		{name: "profiled_plus_slack", configured: 0, profiled: 20, want: 25},
		{name: "configured_out_of_range_falls_to_profiled", configured: 500, profiled: 30, want: 35},
		{name: "all_defaults", configured: 0, profiled: 0, want: 32},
		{name: "profiled_out_of_range_defaults", configured: 0, profiled: 500, want: 32},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := CalculateOptimalBufferSize(tc.configured, tc.profiled)
			if got != tc.want {
				t.Fatalf("CalculateOptimalBufferSize(%d,%d)=%d, want %d",
					tc.configured, tc.profiled, got, tc.want)
			}
		})
	}
}

// TestGetOptimalBufferSizeForApp verifies profile-driven buffer sizing.
func TestGetOptimalBufferSizeForApp(t *testing.T) {
	if got := GetOptimalBufferSizeForApp(nil); got != 32 {
		t.Fatalf("nil profile: expected 32, got %d", got)
	}
	if got := GetOptimalBufferSizeForApp(map[string]interface{}{}); got != 32 {
		t.Fatalf("empty profile: expected 32, got %d", got)
	}
	if got := GetOptimalBufferSizeForApp(map[string]interface{}{"p99_field_count": 15}); got != 20 {
		t.Fatalf("p99=15 profile: expected 20, got %d", got)
	}
	// Wrong type or non-positive value falls back to default.
	if got := GetOptimalBufferSizeForApp(map[string]interface{}{"p99_field_count": "bad"}); got != 32 {
		t.Fatalf("bad-type profile: expected 32, got %d", got)
	}
	if got := GetOptimalBufferSizeForApp(map[string]interface{}{"p99_field_count": 0}); got != 32 {
		t.Fatalf("zero p99 profile: expected 32, got %d", got)
	}
}
