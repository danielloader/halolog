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
// Package interfaces provides interface definitions
// Author: Admilson B. F. Cossa

package interfaces

import "github.com/go-gen-ecosystem/halolog/types"

// SimpleSampler is a basic sampler implementation
type SimpleSampler struct {
	rate float64
}

// NewSimpleSampler creates a new simple sampler
func NewSimpleSampler(rate float64) *SimpleSampler {
	return &SimpleSampler{rate: rate}
}

// ShouldSample determines if a log should be sampled
func (s *SimpleSampler) ShouldSample(level types.LogLevel) bool {
	// For now, always return true (no sampling)
	// In production, this would use random sampling based on rate
	return true
}
