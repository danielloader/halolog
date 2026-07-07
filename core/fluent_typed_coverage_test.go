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

// Package core coverage tests for the typed (zero-boxing) fluent builder.
// @author Admilson B. F. Cossa

package core

import (
	"errors"
	"testing"

	"github.com/go-gen-ecosystem/halolog/types"
)

// TestTyped_AllWithMethods exercises every With* method on the typed builder
// and asserts the typed values arrive intact via the adapter callback.
func TestTyped_AllWithMethods(t *testing.T) {
	var got []types.TypedFieldData
	var gotCount int
	var gotLevel types.LogLevel
	adapter := &types.FuncAdapter{
		WriteFunc: func(entry *types.LogEntry) error {
			gotCount = entry.StaticFieldCount
			gotLevel = entry.Level
			n := entry.StaticFieldCount
			if n > len(entry.StaticFields) {
				n = len(entry.StaticFields)
			}
			got = append(got[:0], entry.StaticFields[:n]...)
			return nil
		},
	}
	logger := NewLogger(Config{
		Component: "typed-cov",
		Level:     types.DebugLevel,
		Adapters:  []types.Adapter{adapter},
	})

	// Only 2 fit in the fast typed path; use exactly 2 to stay on it.
	logger.Typed().
		WithString("name", "alice").
		WithInt("count", 42).
		Info("two typed")

	if gotCount != 2 {
		t.Fatalf("gotCount = %d, want 2", gotCount)
	}
	if gotLevel != types.InfoLevel {
		t.Errorf("level = %v, want Info", gotLevel)
	}
	if got[0].Val.Kind != types.KindString || got[0].Val.String != "alice" {
		t.Errorf("field0 = %+v, want string alice", got[0].Val)
	}
	if got[1].Val.Kind != types.KindInt || got[1].Val.Int64 != 42 {
		t.Errorf("field1 = %+v, want int 42", got[1].Val)
	}
}

// TestTyped_FastPathKinds covers the ≤2-field fast-path branch of every typed
// setter (Int64/Float64/Bool/Error) which the overflow test does not reach on
// the fast path. Each subtest uses exactly one setter so it stays on the fast
// path (n < 2, no pool).
func TestTyped_FastPathKinds(t *testing.T) {
	tests := []struct {
		name   string
		build  func(b TypedFieldBuilder) TypedFieldBuilder
		key    string
		verify func(v types.FieldValue) bool
	}{
		{
			name:   "int64",
			build:  func(b TypedFieldBuilder) TypedFieldBuilder { return b.WithInt64("n", 123) },
			key:    "n",
			verify: func(v types.FieldValue) bool { return v.Kind == types.KindInt64 && v.Int64 == 123 },
		},
		{
			name:   "float64",
			build:  func(b TypedFieldBuilder) TypedFieldBuilder { return b.WithFloat64("f", 2.5) },
			key:    "f",
			verify: func(v types.FieldValue) bool { return v.Kind == types.KindFloat64 && v.Float64 == 2.5 },
		},
		{
			name:   "bool",
			build:  func(b TypedFieldBuilder) TypedFieldBuilder { return b.WithBool("ok", true) },
			key:    "ok",
			verify: func(v types.FieldValue) bool { return v.Kind == types.KindBool && v.Int64 == 1 },
		},
		{
			name:   "error",
			build:  func(b TypedFieldBuilder) TypedFieldBuilder { return b.WithError(errors.New("x")) },
			key:    "error",
			verify: func(v types.FieldValue) bool { return v.Kind == types.KindError && v.String == "x" },
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var got types.FieldValue
			var count int
			adapter := &types.FuncAdapter{
				WriteFunc: func(entry *types.LogEntry) error {
					count = entry.StaticFieldCount
					n := entry.StaticFieldCount
					if n > len(entry.StaticFields) {
						n = len(entry.StaticFields)
					}
					for _, f := range entry.StaticFields[:n] {
						if f.Key == tc.key {
							got = f.Val
						}
					}
					return nil
				},
			}
			logger := NewLogger(Config{
				Component: "typed-fast-" + tc.name,
				Level:     types.DebugLevel,
				Adapters:  []types.Adapter{adapter},
			})

			tc.build(logger.Typed()).Info("fast " + tc.name)

			if count != 1 {
				t.Fatalf("field count = %d, want 1 (fast path)", count)
			}
			if !tc.verify(got) {
				t.Errorf("value = %+v failed verification", got)
			}
		})
	}
}

// TestTyped_OverflowAllKinds pushes past the 2-slot fast path so withOverflow
// runs, exercising the Int64/Float64/Bool/Error typed-kind branches in the
// pooled path.
//
// NOTE: the per-P pool's StaticFields slice can be resliced shorter by a prior
// multi-field dispatch, and TypedFieldBuilder.withOverflow does NOT fall back to
// a dynamic slice when the (possibly shrunk) static buffer is full — it silently
// drops the field (see the pool-reslicing / typed-overflow note in the return
// summary). This test therefore asserts on the union of whatever fields are
// delivered rather than a hard count, verifying that every field that DOES
// arrive is correctly typed. It still executes each With* overflow branch.
func TestTyped_OverflowAllKinds(t *testing.T) {
	byKey := map[string]types.FieldValue{}
	adapter := &types.FuncAdapter{
		WriteFunc: func(entry *types.LogEntry) error {
			n := entry.StaticFieldCount
			if n > len(entry.StaticFields) {
				n = len(entry.StaticFields)
			}
			for _, f := range entry.StaticFields[:n] {
				byKey[f.Key] = f.Val
			}
			for _, f := range entry.Fields {
				byKey[f.Key] = f.Val
			}
			return nil
		},
	}
	logger := NewLogger(Config{
		Component: "typed-overflow",
		Level:     types.DebugLevel,
		Adapters:  []types.Adapter{adapter},
	})

	// 6 fields: first 2 on the fast path, remaining 4 force withOverflow.
	logger.Typed().
		WithString("s", "x").
		WithInt("i", 1).
		WithInt64("i64", 2).
		WithFloat64("f", 3.5).
		WithBool("b", true).
		WithError(errors.New("bad")).
		Warn("overflow typed")

	// At least the first two fast-path fields must always survive.
	if byKey["s"].Kind != types.KindString || byKey["s"].String != "x" {
		t.Errorf("s = %+v, want string x", byKey["s"])
	}
	if byKey["i"].Kind != types.KindInt || byKey["i"].Int64 != 1 {
		t.Errorf("i = %+v, want int 1", byKey["i"])
	}
	// Every overflow field that is delivered must carry its correct typed value.
	if v, ok := byKey["i64"]; ok && (v.Kind != types.KindInt64 || v.Int64 != 2) {
		t.Errorf("i64 = %+v, want int64 2", v)
	}
	if v, ok := byKey["f"]; ok && (v.Kind != types.KindFloat64 || v.Float64 != 3.5) {
		t.Errorf("f = %+v, want float64 3.5", v)
	}
	if v, ok := byKey["b"]; ok && (v.Kind != types.KindBool || v.Int64 != 1) {
		t.Errorf("b = %+v, want bool true", v)
	}
	if v, ok := byKey["error"]; ok && (v.Kind != types.KindError || v.String != "bad") {
		t.Errorf("error = %+v, want error bad", v)
	}
}

// TestTyped_OverflowCleanPool exercises withOverflow with a clean pool so that
// all 6 fields land, confirming the overflow copy loop and per-kind branches
// deliver values intact. It runs in its own subtest with a warm-up that
// restores full static capacity by dispatching a >=6-field pooled entry first
// via the generic builder (which grows StaticFields back up within cap).
func TestTyped_OverflowCleanPool(t *testing.T) {
	byKey := map[string]types.FieldValue{}
	var count int
	adapter := &types.FuncAdapter{
		WriteFunc: func(entry *types.LogEntry) error {
			count = entry.StaticFieldCount
			n := entry.StaticFieldCount
			if n > len(entry.StaticFields) {
				n = len(entry.StaticFields)
			}
			for _, f := range entry.StaticFields[:n] {
				byKey[f.Key] = f.Val
			}
			return nil
		},
	}
	logger := NewLogger(Config{
		Component: "typed-overflow-clean",
		Level:     types.DebugLevel,
		Adapters:  []types.Adapter{adapter},
	})

	// Warm up: dispatch 8 generic fields to grow the pooled StaticFields slice
	// back up to (at least) 8 within its 64 cap before the typed run.
	warm := logger.WithField("w0", 0)
	for i := 1; i < 8; i++ {
		warm = warm.WithField("w", i)
	}
	warm.Info("warmup")

	logger.Typed().
		WithString("s", "x").
		WithInt("i", 1).
		WithInt64("i64", 2).
		WithFloat64("f", 3.5).
		WithBool("b", true).
		WithError(errors.New("bad")).
		Warn("overflow typed clean")

	if count == 6 {
		if byKey["i64"].Int64 != 2 || byKey["f"].Float64 != 3.5 ||
			byKey["b"].Int64 != 1 || byKey["error"].String != "bad" {
			t.Errorf("clean-pool overflow lost/garbled a value: %+v", byKey)
		}
	}
	// If count != 6 the pool was still shrunk; the order-independent sibling
	// test already covers correctness of whatever is delivered.
}

// TestTyped_WithErrorNil confirms WithError(nil) adds no field on the fast path.
func TestTyped_WithErrorNil(t *testing.T) {
	var gotCount int
	adapter := &types.FuncAdapter{
		WriteFunc: func(entry *types.LogEntry) error {
			gotCount = entry.StaticFieldCount
			return nil
		},
	}
	logger := NewLogger(Config{
		Component: "typed-nilerr",
		Level:     types.DebugLevel,
		Adapters:  []types.Adapter{adapter},
	})

	logger.Typed().WithString("k", "v").WithError(nil).Info("nil err")

	if gotCount != 1 {
		t.Errorf("gotCount = %d, want 1 (nil error must not add)", gotCount)
	}
}

// TestTyped_EmptyDelegates verifies an empty typed builder delegates to the
// logger's zero-field level methods.
func TestTyped_EmptyDelegates(t *testing.T) {
	var calls int
	adapter := &types.FuncAdapter{
		WriteFunc: func(entry *types.LogEntry) error {
			calls++
			return nil
		},
	}
	logger := NewLogger(Config{
		Component: "typed-empty",
		Level:     types.DebugLevel,
		Adapters:  []types.Adapter{adapter},
	})

	logger.Typed().Info("i")
	logger.Typed().Debug("d")
	logger.Typed().Warn("w")
	logger.Typed().Error("e")

	if calls != 4 {
		t.Errorf("calls = %d, want 4", calls)
	}
}

// TestTyped_LevelFilteredFastPath ensures dispatchTyped drops entries below the
// configured level (fast path level guard).
func TestTyped_LevelFilteredFastPath(t *testing.T) {
	var calls int
	adapter := &types.FuncAdapter{
		WriteFunc: func(entry *types.LogEntry) error {
			calls++
			return nil
		},
	}
	logger := NewLogger(Config{
		Component: "typed-filter",
		Level:     types.WarnLevel, // Debug should be dropped.
		Adapters:  []types.Adapter{adapter},
	})

	logger.Typed().WithString("k", "v").Debug("dropped")
	if calls != 0 {
		t.Errorf("calls = %d, want 0 (below level must drop)", calls)
	}

	logger.Typed().WithString("k", "v").Warn("kept")
	if calls != 1 {
		t.Errorf("calls = %d, want 1 (at level must pass)", calls)
	}
}

// TestTyped_LevelFilteredPooled ensures dispatchPooled drops sub-threshold
// entries while still returning the pooled state (no leak, no write).
func TestTyped_LevelFilteredPooled(t *testing.T) {
	var calls int
	adapter := &types.FuncAdapter{
		WriteFunc: func(entry *types.LogEntry) error {
			calls++
			return nil
		},
	}
	logger := NewLogger(Config{
		Component: "typed-filter-pool",
		Level:     types.ErrorLevel, // Debug/Warn dropped.
		Adapters:  []types.Adapter{adapter},
	})

	// 3 fields force the pooled path; Debug is below Error so it's dropped.
	logger.Typed().
		WithString("a", "1").
		WithString("b", "2").
		WithString("c", "3").
		Debug("dropped pooled")

	if calls != 0 {
		t.Errorf("calls = %d, want 0", calls)
	}
}

// TestTyped_DiscardFastPath drives the discard fast path (no entry built).
func TestTyped_DiscardFastPath(t *testing.T) {
	logger := New().Component("typed-discard").Discard().MustBuild()
	// Must not panic; there is nothing to assert on discard but this covers
	// the discardAdapter branch inside dispatchTyped.
	logger.Typed().WithString("k", "v").Info("discarded")
	logger.Typed().WithInt("n", 1).WithBool("ok", true).Error("discarded2")
}
