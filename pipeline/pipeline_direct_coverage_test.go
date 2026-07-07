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
// Package pipeline tests exercise the DirectPipeline zero-field fast path.
// @author Admilson B. F. Cossa

package pipeline

import (
	"testing"
	"time"

	clock "github.com/go-gen-ecosystem/halolog/cache"
	"github.com/go-gen-ecosystem/halolog/types"
)

// captured is a snapshot of the fields we care about, copied INSIDE the adapter
// callback because the *types.LogEntry passed to Write is pooled and recycled.
type captured struct {
	level    types.LogLevel
	message  string
	numStat  int
	statKeys []string
}

// newCapturingAdapter returns a FuncAdapter that appends a snapshot of each entry
// to the provided slice. It copies field data inside the callback (never retains
// the pooled pointer).
func newCapturingAdapter(sink *[]captured) *types.FuncAdapter {
	return &types.FuncAdapter{
		WriteFunc: func(e *types.LogEntry) error {
			snap := captured{
				level:   e.Level,
				message: e.Message,
				numStat: e.StaticFieldCount,
			}
			for i := 0; i < e.StaticFieldCount && i < len(e.StaticFields); i++ {
				snap.statKeys = append(snap.statKeys, e.StaticFields[i].Key)
			}
			*sink = append(*sink, snap)
			return nil
		},
	}
}

// newTestClock builds a real cached clock for use in Write calls.
func newTestClock(t *testing.T) *clock.CachedClock {
	t.Helper()
	cc := clock.NewCachedClock(10 * time.Millisecond)
	t.Cleanup(cc.Close)
	return cc
}

// TestNewDirectPipeline_SingleVsMulti verifies the constructor picks the single
// adapter fast path for exactly one adapter and the slice path otherwise.
func TestNewDirectPipeline_SingleVsMulti(t *testing.T) {
	var sink []captured
	a := newCapturingAdapter(&sink)

	single := NewDirectPipeline([]types.Adapter{a})
	if !single.HasSingleAdapter {
		t.Fatal("expected HasSingleAdapter=true for one adapter")
	}
	if single.SingleAdapter == nil {
		t.Fatal("expected SingleAdapter to be populated")
	}

	multi := NewDirectPipeline([]types.Adapter{a, a})
	if multi.HasSingleAdapter {
		t.Fatal("expected HasSingleAdapter=false for two adapters")
	}
	if len(multi.Adapters) != 2 {
		t.Fatalf("expected 2 adapters in slice path, got %d", len(multi.Adapters))
	}
}

// TestDirectPipeline_WriteSingleAdapter verifies a zero-field write reaches the
// single adapter through WriteZero with correct metadata.
func TestDirectPipeline_WriteSingleAdapter(t *testing.T) {
	var sink []captured
	p := NewDirectPipeline([]types.Adapter{newCapturingAdapter(&sink)})
	cc := newTestClock(t)

	p.Write(cc, types.WarnLevel, "hello-direct", nil, 0)

	if len(sink) != 1 {
		t.Fatalf("expected 1 write, got %d", len(sink))
	}
	if sink[0].level != types.WarnLevel {
		t.Fatalf("expected WarnLevel, got %v", sink[0].level)
	}
	if sink[0].message != "hello-direct" {
		t.Fatalf("expected message 'hello-direct', got %q", sink[0].message)
	}
	if sink[0].numStat != 0 {
		t.Fatalf("expected 0 static fields on zero-field path, got %d", sink[0].numStat)
	}
}

// TestDirectPipeline_WriteMultiAdapter verifies the slice fan-out path writes to
// every adapter.
func TestDirectPipeline_WriteMultiAdapter(t *testing.T) {
	var sinkA, sinkB []captured
	p := NewDirectPipeline([]types.Adapter{
		newCapturingAdapter(&sinkA),
		newCapturingAdapter(&sinkB),
	})
	cc := newTestClock(t)

	p.Write(cc, types.InfoLevel, "fanout", nil, 0)

	if len(sinkA) != 1 || len(sinkB) != 1 {
		t.Fatalf("expected both adapters to receive one write, got A=%d B=%d", len(sinkA), len(sinkB))
	}
}

// TestDirectPipeline_WriteWithFieldsFallback verifies that calling Write with
// fieldCount>0 (a misconfiguration) still delivers the fields via the pooled
// handleFields fallback rather than dropping them.
func TestDirectPipeline_WriteWithFieldsFallback(t *testing.T) {
	var sink []captured
	p := NewDirectPipeline([]types.Adapter{newCapturingAdapter(&sink)})
	cc := newTestClock(t)

	fields := []types.TypedFieldData{
		{Key: "user", Value: "alice"},
		{Key: "id", Value: 42},
	}
	p.Write(cc, types.ErrorLevel, "with-fields", fields, len(fields))

	if len(sink) != 1 {
		t.Fatalf("expected 1 write, got %d", len(sink))
	}
	if sink[0].numStat != 2 {
		t.Fatalf("expected 2 static fields in fallback path, got %d", sink[0].numStat)
	}
	if sink[0].statKeys[0] != "user" || sink[0].statKeys[1] != "id" {
		t.Fatalf("field keys not preserved: %v", sink[0].statKeys)
	}
}

// TestDirectPipeline_WriteFieldsMultiAdapter drives the multi-adapter branch of
// the handleFields fallback.
func TestDirectPipeline_WriteFieldsMultiAdapter(t *testing.T) {
	var sinkA, sinkB []captured
	p := NewDirectPipeline([]types.Adapter{
		newCapturingAdapter(&sinkA),
		newCapturingAdapter(&sinkB),
	})
	cc := newTestClock(t)

	fields := []types.TypedFieldData{{Key: "k", Value: "v"}}
	p.Write(cc, types.DebugLevel, "multi-fields", fields, 1)

	if len(sinkA) != 1 || len(sinkB) != 1 {
		t.Fatalf("expected both adapters to receive fallback write, got A=%d B=%d", len(sinkA), len(sinkB))
	}
	if sinkA[0].numStat != 1 || sinkB[0].numStat != 1 {
		t.Fatalf("expected 1 field per adapter, got A=%d B=%d", sinkA[0].numStat, sinkB[0].numStat)
	}
}

// TestDirectPipeline_HandleFieldsOverflow drives the dynamic-Fields overflow
// branch of handleFields when fieldCount exceeds the static capacity.
func TestDirectPipeline_HandleFieldsOverflow(t *testing.T) {
	// Capture via the dynamic Fields slice, since overflow stores there.
	var dynamicCounts []int
	adapter := &types.FuncAdapter{
		WriteFunc: func(e *types.LogEntry) error {
			dynamicCounts = append(dynamicCounts, len(e.Fields))
			return nil
		},
	}
	p := NewDirectPipeline([]types.Adapter{adapter})
	cc := newTestClock(t)

	// Build more fields than the static capacity (pool default static cap is 16).
	const n = 40
	fields := make([]types.TypedFieldData, n)
	for i := range fields {
		fields[i] = types.TypedFieldData{Key: "k", Value: i}
	}
	p.Write(cc, types.InfoLevel, "overflow", fields, n)

	if len(dynamicCounts) != 1 {
		t.Fatalf("expected 1 write, got %d", len(dynamicCounts))
	}
	if dynamicCounts[0] != n {
		t.Fatalf("expected %d overflow fields in dynamic slice, got %d", n, dynamicCounts[0])
	}
}

// TestDirectPipeline_Reset verifies Reset clears adapter references.
func TestDirectPipeline_Reset(t *testing.T) {
	var sink []captured
	p := NewDirectPipeline([]types.Adapter{
		newCapturingAdapter(&sink),
		newCapturingAdapter(&sink),
	})
	p.Reset()
	if len(p.Adapters) != 0 {
		t.Fatalf("expected adapters cleared after Reset, got len=%d", len(p.Adapters))
	}
}

// TestIsDirectPathEligible verifies the eligibility predicate across the feature
// matrix.
func TestIsDirectPathEligible(t *testing.T) {
	testCases := []struct {
		name       string
		fieldCount int
		masking    bool
		sampling   bool
		async      bool
		colors     bool
		want       bool
	}{
		{name: "zero_fields_no_features", fieldCount: 0, want: true},
		{name: "has_fields", fieldCount: 1, want: false},
		{name: "masking_enabled", fieldCount: 0, masking: true, want: false},
		{name: "sampling_enabled", fieldCount: 0, sampling: true, want: false},
		{name: "async_enabled", fieldCount: 0, async: true, want: false},
		{name: "colors_enabled", fieldCount: 0, colors: true, want: false},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := IsDirectPathEligible(tc.fieldCount, tc.masking, tc.sampling, tc.async, tc.colors)
			if got != tc.want {
				t.Fatalf("IsDirectPathEligible=%v, want %v", got, tc.want)
			}
		})
	}
}
