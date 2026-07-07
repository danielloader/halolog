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

// Package core coverage tests for the Builder fluent configuration API.
// @author Admilson B. F. Cossa

package core

import (
	"testing"

	"github.com/go-gen-ecosystem/halolog/types"
)

// TestBuilder_LevelConvenienceMethods covers Info/Warn/Error/Debug level setters.
func TestBuilder_LevelConvenienceMethods(t *testing.T) {
	tests := []struct {
		name  string
		apply func(*Builder) *Builder
		want  types.LogLevel
	}{
		{"debug", (*Builder).Debug, types.DebugLevel},
		{"info", (*Builder).Info, types.InfoLevel},
		{"warn", (*Builder).Warn, types.WarnLevel},
		{"error", (*Builder).Error, types.ErrorLevel},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			logger := tc.apply(New().Component(tc.name).Discard()).MustBuild()
			if logger.Level() != tc.want {
				t.Errorf("level = %v, want %v", logger.Level(), tc.want)
			}
		})
	}
}

// TestBuilder_ExplicitLevel covers Level() with a concrete value.
func TestBuilder_ExplicitLevel(t *testing.T) {
	logger := New().Component("explicit").Level(types.TraceLevel).Discard().MustBuild()
	if logger.Level() != types.TraceLevel {
		t.Errorf("level = %v, want Trace", logger.Level())
	}
}

// TestBuilder_MaskingWiring covers Builder.Masking and asserts the masker is
// active by observing its mutation via the adapter.
func TestBuilder_MaskingWiring(t *testing.T) {
	var gotMsg string
	adapter := &types.FuncAdapter{
		WriteFunc: func(entry *types.LogEntry) error {
			gotMsg = entry.Message
			return nil
		},
	}
	m := &upperMasker{}
	logger := New().
		Component("mask-builder").
		Info().
		Adapter(adapter).
		Masking(m).
		MustBuild()

	logger.Info("secret")

	if m.applied != 1 {
		t.Errorf("masker applied %d times, want 1", m.applied)
	}
	if gotMsg != "[MASKED] secret" {
		t.Errorf("message = %q, want masked", gotMsg)
	}
}

// TestBuilder_SamplingWiring covers Builder.Sampling.
func TestBuilder_SamplingWiring(t *testing.T) {
	s := &fixedSampler{sample: true}
	logger := New().
		Component("sample-builder").
		Discard().
		Sampling(s).
		MustBuild()
	if logger.sampler == nil {
		t.Fatal("expected sampler to be wired via Builder.Sampling")
	}
}

// TestBuilder_AlertsAggregationFlags covers Alerts and Aggregation, which set
// config flags consumed at build time.
func TestBuilder_AlertsAggregationFlags(t *testing.T) {
	b := New().
		Component("flags").
		Discard().
		Alerts().
		Aggregation()

	if !b.config.EnableAlerts {
		t.Error("EnableAlerts not set by Alerts()")
	}
	if !b.config.EnableAggregation {
		t.Error("EnableAggregation not set by Aggregation()")
	}

	// Build must still succeed with these flags set.
	logger := b.MustBuild()
	if logger == nil {
		t.Fatal("MustBuild returned nil")
	}
}

// TestBuilder_SingleAdapter covers Builder.Adapter (single) path.
func TestBuilder_SingleAdapter(t *testing.T) {
	a := &countingAdapter{}
	logger := New().Component("single").Debug().Adapter(a).MustBuild()

	logger.Info("one")
	if len(a.levels) != 1 {
		t.Errorf("expected 1 write, got %d", len(a.levels))
	}
}
