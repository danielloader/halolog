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

// Package core coverage tests for the generic FieldBuilder fluent API.
// @author Admilson B. F. Cossa

package core

import (
	"errors"
	"testing"

	"github.com/go-gen-ecosystem/halolog/types"
)

// capturedEntry is a snapshot of the fields we care about, taken INSIDE the
// adapter callback because the *types.LogEntry is recycled after Write returns.
//
// fieldCount is the TOTAL number of fields (static + dynamic overflow). The
// per-P pool's StaticFields slice can be resliced shorter by a prior multi-field
// dispatch (see the pool-reslicing note in the return summary), so overflow may
// spill into entry.Fields at a lower threshold than the nominal 64. Capturing
// the union keeps these tests order-independent while still asserting real
// end-to-end field delivery.
type capturedEntry struct {
	level      types.LogLevel
	message    string
	component  string
	fieldCount int
	fields     []types.TypedFieldData
}

// newCapturingLogger builds a logger whose single adapter snapshots each entry
// into the returned slice (copying fields eagerly to survive pooling).
func newCapturingLogger(t *testing.T, level types.LogLevel) (*Logger, *[]capturedEntry) {
	t.Helper()
	captures := &[]capturedEntry{}
	adapter := &types.FuncAdapter{
		WriteFunc: func(entry *types.LogEntry) error {
			snap := capturedEntry{
				level:     entry.Level,
				message:   entry.Message,
				component: entry.Component,
			}
			// Copy the active static fields so the assertion outlives recycling.
			n := entry.StaticFieldCount
			if n > len(entry.StaticFields) {
				n = len(entry.StaticFields)
			}
			snap.fields = append(snap.fields, entry.StaticFields[:n]...)
			// Include any dynamic-overflow fields so total counts are accurate.
			snap.fields = append(snap.fields, entry.Fields...)
			snap.fieldCount = len(snap.fields)
			*captures = append(*captures, snap)
			return nil
		},
	}
	logger := NewLogger(Config{
		Component: "fluent-cov",
		Level:     level,
		Adapters:  []types.Adapter{adapter},
	})
	return logger, captures
}

// TestFieldBuilder_TypedHelpers exercises String/Int/Int64/Float64/Bool/Err
// which all funnel through WithField, and verifies the final entry.
func TestFieldBuilder_TypedHelpers(t *testing.T) {
	logger, captures := newCapturingLogger(t, types.DebugLevel)

	logger.WithField("base", "v").
		String("s", "hello").
		Int("i", 7).
		Int64("i64", 99).
		Float64("f", 3.5).
		Bool("b", true).
		Err(errors.New("boom")).
		Info("typed helpers")

	if len(*captures) != 1 {
		t.Fatalf("expected 1 capture, got %d", len(*captures))
	}
	c := (*captures)[0]
	if c.message != "typed helpers" {
		t.Errorf("message = %q", c.message)
	}
	if c.level != types.InfoLevel {
		t.Errorf("level = %v", c.level)
	}
	// base + 6 helper fields = 7 fields.
	if c.fieldCount != 7 {
		t.Errorf("fieldCount = %d, want 7", c.fieldCount)
	}
	// Verify a representative subset landed with correct keys/values.
	got := map[string]interface{}{}
	for _, f := range c.fields {
		got[f.Key] = f.Value
	}
	if got["s"] != "hello" {
		t.Errorf("field s = %v, want hello", got["s"])
	}
	if got["i"] != 7 {
		t.Errorf("field i = %v, want 7", got["i"])
	}
	if got["error"] != "boom" {
		t.Errorf("field error = %v, want boom", got["error"])
	}
}

// TestFieldBuilder_ErrNil confirms Err(nil) is a no-op (adds no field).
func TestFieldBuilder_ErrNil(t *testing.T) {
	logger, captures := newCapturingLogger(t, types.DebugLevel)

	logger.WithField("only", 1).Err(nil).Info("no err field")

	if len(*captures) != 1 {
		t.Fatalf("expected 1 capture, got %d", len(*captures))
	}
	if (*captures)[0].fieldCount != 1 {
		t.Errorf("fieldCount = %d, want 1 (Err(nil) must not add)", (*captures)[0].fieldCount)
	}
}

// TestFieldBuilder_AllLevelsMultiField drives Debug/Warn/Error/Trace/Fatal/Panic
// through the multi-field path (2 fields) so the non-info branches are covered.
func TestFieldBuilder_AllLevelsMultiField(t *testing.T) {
	tests := []struct {
		name  string
		level types.LogLevel
		call  func(fb FieldBuilder, msg string)
		want  types.LogLevel
	}{
		{"debug", types.DebugLevel, func(fb FieldBuilder, m string) { fb.Debug(m) }, types.DebugLevel},
		{"warn", types.DebugLevel, func(fb FieldBuilder, m string) { fb.Warn(m) }, types.WarnLevel},
		{"error", types.DebugLevel, func(fb FieldBuilder, m string) { fb.Error(m) }, types.ErrorLevel},
		{"trace", types.TraceLevel, func(fb FieldBuilder, m string) { fb.Trace(m) }, types.TraceLevel},
		{"fatal", types.DebugLevel, func(fb FieldBuilder, m string) { fb.Fatal(m) }, types.FatalLevel},
		{"panic", types.DebugLevel, func(fb FieldBuilder, m string) { fb.Panic(m) }, types.PanicLevel},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			logger, captures := newCapturingLogger(t, tc.level)
			fb := logger.WithField("k1", "v1").WithField("k2", 2)
			tc.call(fb, "multi "+tc.name)

			if len(*captures) != 1 {
				t.Fatalf("expected 1 capture, got %d", len(*captures))
			}
			c := (*captures)[0]
			if c.level != tc.want {
				t.Errorf("level = %v, want %v", c.level, tc.want)
			}
			if c.fieldCount != 2 {
				t.Errorf("fieldCount = %d, want 2", c.fieldCount)
			}
			if c.message != "multi "+tc.name {
				t.Errorf("message = %q", c.message)
			}
		})
	}
}

// TestFieldBuilder_SingleFieldFastPath drives the Debug/Warn/Error single-field
// path (StaticFieldCount == 1, single adapter, no masking), verifying it captures
// exactly one field via the pooled per-P entry dispatch.
func TestFieldBuilder_SingleFieldFastPath(t *testing.T) {
	tests := []struct {
		name string
		call func(fb FieldBuilder, msg string)
		want types.LogLevel
	}{
		{"debug", func(fb FieldBuilder, m string) { fb.Debug(m) }, types.DebugLevel},
		{"warn", func(fb FieldBuilder, m string) { fb.Warn(m) }, types.WarnLevel},
		{"error", func(fb FieldBuilder, m string) { fb.Error(m) }, types.ErrorLevel},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			logger, captures := newCapturingLogger(t, types.DebugLevel)
			logger.WithField("solo", "field").call2(tc.call, "single "+tc.name)
			if len(*captures) != 1 {
				t.Fatalf("expected 1 capture, got %d", len(*captures))
			}
			c := (*captures)[0]
			if c.level != tc.want {
				t.Errorf("level = %v, want %v", c.level, tc.want)
			}
			if c.fieldCount != 1 {
				t.Errorf("fieldCount = %d, want 1", c.fieldCount)
			}
		})
	}
}

// call2 is a tiny adapter so table rows can invoke a level method on a builder.
func (fb FieldBuilder) call2(fn func(FieldBuilder, string), msg string) {
	fn(fb, msg)
}

// TestFieldBuilder_NoStateFallsThrough verifies that a FieldBuilder with no
// accumulated state delegates straight to the logger's level methods.
func TestFieldBuilder_NoStateFallsThrough(t *testing.T) {
	logger, captures := newCapturingLogger(t, types.TraceLevel)

	// WithError(nil) returns a stateless FieldBuilder.
	fb := logger.WithError(nil)
	fb.Info("i")
	fb.Debug("d")
	fb.Warn("w")
	fb.Error("e")
	fb.Trace("t")

	if len(*captures) != 5 {
		t.Fatalf("expected 5 captures via fallthrough, got %d", len(*captures))
	}
	wantLevels := []types.LogLevel{
		types.InfoLevel, types.DebugLevel, types.WarnLevel,
		types.ErrorLevel, types.TraceLevel,
	}
	for i, want := range wantLevels {
		if (*captures)[i].level != want {
			t.Errorf("capture %d level = %v, want %v", i, (*captures)[i].level, want)
		}
	}
}

// TestFieldBuilder_Overflow pushes well past the static buffer so the
// dynamic-overflow branch in WithField (append to entry.Fields) runs, and
// asserts that no fields are dropped. We assert on the union of static +
// dynamic fields (via the capturing helper) so the test is independent of
// where exactly the static/dynamic boundary falls for the pooled state.
func TestFieldBuilder_Overflow(t *testing.T) {
	logger, captures := newCapturingLogger(t, types.DebugLevel)

	const total = 80 // comfortably above the 64-slot nominal static buffer
	fb := logger.WithField("f0", 0)
	for i := 1; i < total; i++ {
		fb = fb.WithField("k", i)
	}
	fb.Info("overflow entry")

	if len(*captures) != 1 {
		t.Fatalf("expected 1 capture, got %d", len(*captures))
	}
	c := (*captures)[0]
	// Every field must be delivered somewhere (static + dynamic overflow).
	if c.fieldCount != total {
		t.Errorf("total delivered fields = %d, want %d", c.fieldCount, total)
	}
	// First and last field values must survive the overflow.
	if c.fields[0].Value != 0 {
		t.Errorf("first field value = %v, want 0", c.fields[0].Value)
	}
	if c.fields[total-1].Value != total-1 {
		t.Errorf("last field value = %v, want %d", c.fields[total-1].Value, total-1)
	}
}

// TestFieldBuilder_WithErrorField verifies logger.WithError populates the
// "error" field for a real error.
func TestFieldBuilder_WithErrorField(t *testing.T) {
	logger, captures := newCapturingLogger(t, types.DebugLevel)

	logger.WithError(errors.New("db down")).Error("failed op")

	if len(*captures) != 1 {
		t.Fatalf("expected 1 capture, got %d", len(*captures))
	}
	c := (*captures)[0]
	if c.fieldCount != 1 {
		t.Fatalf("fieldCount = %d, want 1", c.fieldCount)
	}
	if c.fields[0].Key != "error" || c.fields[0].Value != "db down" {
		t.Errorf("error field = %+v, want key=error value=db down", c.fields[0])
	}
}
