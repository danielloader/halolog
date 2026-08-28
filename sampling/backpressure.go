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

// Package sampling — backpressure-driven sampling.
//
// Classic samplers drop a fixed fraction regardless of whether the pipeline
// is keeping up; under overload every logger in the field degrades the same
// way — indiscriminately. BackpressureSampler closes the loop instead: it
// reads the async sink's live occupancy and sheds load proportionally to how
// full the pipeline actually is, while errors and worse always pass. Below
// the low watermark it is a no-op; between the watermarks the keep rate
// falls linearly; at the high watermark it floors at keep-1-in-16. Every
// decision is O(1), allocation-free, and deterministic (a counter, not a
// RNG), so drops spread evenly instead of clustering.
// Author: Admilson B. F. Cossa

package sampling

import (
	"sync/atomic"

	"github.com/go-gen-ecosystem/halolog/types"
)

// OccupancySource reports a queue's current fill. asyncring.RingAdapter
// implements it; any bounded sink can.
type OccupancySource interface {
	Occupancy() (used, capacity int)
}

// Backpressure watermarks and floor, exported for documentation and tests.
const (
	// DefaultLowWater is the fill fraction below which nothing is dropped.
	DefaultLowWater = 0.50
	// DefaultHighWater is the fill fraction at which shedding floors.
	DefaultHighWater = 0.90
	// backpressureFloorKeep is the 1-in-N keep rate at/above the high
	// watermark. Power of two so the modulo is a mask.
	backpressureFloorKeep = 16
)

// BackpressureSampler sheds Trace..Warn lines in proportion to a bounded
// sink's occupancy. Error and above ALWAYS pass — the lines an operator
// needs most are exactly the ones an overloaded system emits.
type BackpressureSampler struct {
	src       OccupancySource
	lowWater  float64
	highWater float64
	counter   atomic.Uint64
}

// NewBackpressureSampler builds a sampler bound to src. Watermarks are fill
// fractions in (0,1]; out-of-order or out-of-range values fall back to the
// defaults (low 0.50, high 0.90).
func NewBackpressureSampler(src OccupancySource, lowWater, highWater float64) *BackpressureSampler {
	if !(lowWater > 0 && lowWater < highWater && highWater <= 1) {
		lowWater, highWater = DefaultLowWater, DefaultHighWater
	}
	return &BackpressureSampler{src: src, lowWater: lowWater, highWater: highWater}
}

// ShouldSample implements types.Sampler.
func (s *BackpressureSampler) ShouldSample(entry *types.LogEntry) bool {
	if entry != nil && entry.Level >= types.ErrorLevel {
		return true
	}
	used, capacity := s.src.Occupancy()
	if capacity <= 0 || used <= 0 {
		return true
	}
	fill := float64(used) / float64(capacity)
	if fill <= s.lowWater {
		return true
	}
	if fill >= s.highWater {
		return s.counter.Add(1)&(backpressureFloorKeep-1) == 0
	}
	// Linear shed between the watermarks: keep probability p goes 1 → 1/floor
	// as fill goes low → high, realized deterministically as keep-1-in-N.
	p := (s.highWater - fill) / (s.highWater - s.lowWater)
	n := uint64(1.0/p + 0.5)
	if n < 1 {
		n = 1
	}
	if n > backpressureFloorKeep {
		n = backpressureFloorKeep
	}
	return s.counter.Add(1)%n == 0
}

// GetRate reports the sampler's CURRENT keep rate given live occupancy —
// 1.0 when idle, down to 1/16 at the high watermark.
func (s *BackpressureSampler) GetRate() float64 {
	used, capacity := s.src.Occupancy()
	if capacity <= 0 {
		return 1
	}
	fill := float64(used) / float64(capacity)
	switch {
	case fill <= s.lowWater:
		return 1
	case fill >= s.highWater:
		return 1.0 / backpressureFloorKeep
	default:
		return (s.highWater - fill) / (s.highWater - s.lowWater)
	}
}

// SetRate is accepted for types.Sampler compatibility and is a no-op: this
// sampler's rate is driven by the control loop, not configuration.
func (s *BackpressureSampler) SetRate(float64) {}
