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

// Package sampling — backpressure sampler behavior across the fill regimes.
// @author Admilson B. F. Cossa

package sampling

import (
	"testing"

	"github.com/go-gen-ecosystem/halolog/adapters/outputs/asyncring"
	"github.com/go-gen-ecosystem/halolog/types"
)

// The async ring adapter is the canonical occupancy source.
var _ OccupancySource = (*asyncring.RingAdapter)(nil)

// stubOccupancy is a fixed-fill OccupancySource for deterministic tests.
type stubOccupancy struct{ used, capacity int }

func (s stubOccupancy) Occupancy() (int, int) { return s.used, s.capacity }

func keepCount(s *BackpressureSampler, level types.LogLevel, n int) int {
	entry := &types.LogEntry{Level: level}
	kept := 0
	for i := 0; i < n; i++ {
		if s.ShouldSample(entry) {
			kept++
		}
	}
	return kept
}

func TestBackpressure_IdleAndBelowLowWaterKeepEverything(t *testing.T) {
	for _, used := range []int{0, 100, 500} { // 0%, 10%, 50% of 1000
		s := NewBackpressureSampler(stubOccupancy{used, 1000}, 0.5, 0.9)
		if kept := keepCount(s, types.InfoLevel, 1000); kept != 1000 {
			t.Fatalf("fill %d/1000: kept %d, want all 1000", used, kept)
		}
	}
}

func TestBackpressure_AtHighWaterFloorsAtOneInSixteen(t *testing.T) {
	s := NewBackpressureSampler(stubOccupancy{950, 1000}, 0.5, 0.9)
	kept := keepCount(s, types.InfoLevel, 1600)
	if kept != 100 {
		t.Fatalf("at high water: kept %d of 1600, want exactly 100 (1 in 16, deterministic)", kept)
	}
}

func TestBackpressure_MidFillShedsProportionally(t *testing.T) {
	// fill 0.7 with watermarks (0.5, 0.9): p = (0.9-0.7)/0.4 = 0.5 → keep 1 in 2.
	s := NewBackpressureSampler(stubOccupancy{700, 1000}, 0.5, 0.9)
	kept := keepCount(s, types.InfoLevel, 1000)
	if kept != 500 {
		t.Fatalf("mid fill: kept %d of 1000, want exactly 500 (deterministic 1-in-2)", kept)
	}
}

func TestBackpressure_ErrorsAlwaysPass(t *testing.T) {
	s := NewBackpressureSampler(stubOccupancy{1000, 1000}, 0.5, 0.9) // saturated
	for _, lvl := range []types.LogLevel{types.ErrorLevel, types.FatalLevel, types.PanicLevel} {
		if kept := keepCount(s, lvl, 200); kept != 200 {
			t.Fatalf("level %v under saturation: kept %d of 200, want all", lvl, kept)
		}
	}
	// And Info under the same saturation is being shed, proving the contrast.
	if kept := keepCount(s, types.InfoLevel, 1600); kept >= 1600 {
		t.Fatal("info must be shed under saturation")
	}
}

func TestBackpressure_InvalidWatermarksFallBackToDefaults(t *testing.T) {
	s := NewBackpressureSampler(stubOccupancy{0, 1000}, 0.9, 0.5) // inverted
	if s.lowWater != DefaultLowWater || s.highWater != DefaultHighWater {
		t.Fatalf("watermarks = (%v,%v), want defaults", s.lowWater, s.highWater)
	}
}

func TestBackpressure_GetRateTracksOccupancy(t *testing.T) {
	cases := []struct {
		used int
		want float64
	}{
		{0, 1}, {400, 1}, {500, 1},
		{900, 1.0 / 16}, {1000, 1.0 / 16},
	}
	for _, tc := range cases {
		s := NewBackpressureSampler(stubOccupancy{tc.used, 1000}, 0.5, 0.9)
		if got := s.GetRate(); got != tc.want {
			t.Fatalf("fill %d/1000: rate %v, want %v", tc.used, got, tc.want)
		}
	}
	// Mid fill is strictly between the extremes.
	s := NewBackpressureSampler(stubOccupancy{700, 1000}, 0.5, 0.9)
	if r := s.GetRate(); r <= 1.0/16 || r >= 1 {
		t.Fatalf("mid-fill rate %v not in (1/16, 1)", r)
	}
}
