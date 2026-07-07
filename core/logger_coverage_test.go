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

// Package core coverage tests for the Logger dispatch, masking, and lifecycle.
// @author Admilson B. F. Cossa

package core

import (
	"errors"
	"testing"

	"github.com/go-gen-ecosystem/halolog/adapters/outputs/discard"
	"github.com/go-gen-ecosystem/halolog/types"
)

// countingAdapter records level-tagged Write calls and any lifecycle errors.
type countingAdapter struct {
	levels     []types.LogLevel
	flushErr   error
	closeErr   error
	flushCalls int
	closeCalls int
}

func (a *countingAdapter) Name() string { return "counting" }
func (a *countingAdapter) Write(entry *types.LogEntry) error {
	a.levels = append(a.levels, entry.Level)
	return nil
}
func (a *countingAdapter) WriteZero(entry *types.LogEntry) error {
	a.levels = append(a.levels, entry.Level)
	return nil
}
func (a *countingAdapter) Flush() error {
	a.flushCalls++
	return a.flushErr
}
func (a *countingAdapter) Close() error {
	a.closeCalls++
	return a.closeErr
}
func (a *countingAdapter) SetFormatter(f types.Formatter) {}
func (a *countingAdapter) Health() error                  { return nil }

// upperMasker is a real PIIMasker that rewrites the message so we can assert it
// was actually applied. Only Apply carries behaviour; the remaining methods
// satisfy the full contract with sensible defaults.
type upperMasker struct{ applied int }

func (m *upperMasker) Apply(entry *types.LogEntry) {
	m.applied++
	entry.Message = "[MASKED] " + entry.Message
}
func (m *upperMasker) MaskField(field *types.TypedFieldData) *types.TypedFieldData { return field }
func (m *upperMasker) MaskFields(fields []types.TypedFieldData) []types.TypedFieldData {
	return fields
}
func (m *upperMasker) MaskString(input string) string                  { return input }
func (m *upperMasker) AddRule(pattern, replace, ruleType string) error { return nil }
func (m *upperMasker) RemoveRule(pattern string) error                 { return nil }
func (m *upperMasker) AddPattern(name, patternStr, mask string) error  { return nil }
func (m *upperMasker) RemovePattern(name string)                       {}
func (m *upperMasker) GetPatterns() []string                           { return nil }
func (m *upperMasker) Clone() types.PIIMasker                          { return m }

// TestLogger_TraceFatalPanicRegular drives the generic realTrace/realFatal/realPanic
// paths (regular adapter, no masking, single adapter).
func TestLogger_TraceFatalPanicRegular(t *testing.T) {
	a := &countingAdapter{}
	logger := NewLogger(Config{
		Component: "lifecycle",
		Level:     types.TraceLevel,
		Adapters:  []types.Adapter{a},
	})

	logger.Trace("t")
	logger.Fatal("f")
	logger.Panic("p")

	want := []types.LogLevel{types.TraceLevel, types.FatalLevel, types.PanicLevel}
	if len(a.levels) != 3 {
		t.Fatalf("got %d writes, want 3: %v", len(a.levels), a.levels)
	}
	for i, w := range want {
		if a.levels[i] != w {
			t.Errorf("write %d level = %v, want %v", i, a.levels[i], w)
		}
	}
}

// TestLogger_TraceFatalPanicMulti covers the multi-adapter loop inside the
// generic realTrace/realFatal/realPanic functions.
func TestLogger_TraceFatalPanicMulti(t *testing.T) {
	a1 := &countingAdapter{}
	a2 := &countingAdapter{}
	logger := NewLogger(Config{
		Component: "lifecycle-multi",
		Level:     types.TraceLevel,
		Adapters:  []types.Adapter{a1, a2},
	})

	logger.Trace("t")
	logger.Fatal("f")
	logger.Panic("p")

	if len(a1.levels) != 3 || len(a2.levels) != 3 {
		t.Fatalf("adapter writes a1=%d a2=%d, want 3 each", len(a1.levels), len(a2.levels))
	}
}

// TestLogger_MaskingSingleAdapter covers the *MaskOne specialized paths and the
// generic realTrace masking branch. The masker mutates the message.
func TestLogger_MaskingSingleAdapter(t *testing.T) {
	var gotMessages []string
	adapter := &types.FuncAdapter{
		WriteFunc: func(entry *types.LogEntry) error {
			gotMessages = append(gotMessages, entry.Message)
			return nil
		},
	}
	m := &upperMasker{}
	logger := NewLogger(Config{
		Component:     "mask-one",
		Level:         types.TraceLevel,
		Adapters:      []types.Adapter{adapter},
		EnableMasking: true,
		Masker:        m,
	})

	logger.Info("hello")
	logger.Debug("dbg")
	logger.Warn("warn")
	logger.Error("err")
	logger.Trace("trc")

	if m.applied != 5 {
		t.Errorf("masker applied %d times, want 5", m.applied)
	}
	for _, msg := range gotMessages {
		if len(msg) < len("[MASKED] ") || msg[:len("[MASKED] ")] != "[MASKED] " {
			t.Errorf("message %q not masked", msg)
		}
	}
}

// TestLogger_MaskingMultiAdapter covers the *MaskMulti specialized paths.
func TestLogger_MaskingMultiAdapter(t *testing.T) {
	a1 := &countingAdapter{}
	a2 := &countingAdapter{}
	m := &upperMasker{}
	logger := NewLogger(Config{
		Component:     "mask-multi",
		Level:         types.DebugLevel,
		Adapters:      []types.Adapter{a1, a2},
		EnableMasking: true,
		Masker:        m,
	})

	logger.Info("i")
	logger.Debug("d")
	logger.Warn("w")
	logger.Error("e")

	// 4 log calls, applied once per call (masker runs before fan-out).
	if m.applied != 4 {
		t.Errorf("masker applied %d times, want 4", m.applied)
	}
	if len(a1.levels) != 4 || len(a2.levels) != 4 {
		t.Errorf("writes a1=%d a2=%d, want 4 each", len(a1.levels), len(a2.levels))
	}
}

// TestLogger_DiscardAllLevels drives the concrete discard adapter fast paths for
// every level, ensuring no panic and that the discard branch is selected.
func TestLogger_DiscardAllLevels(t *testing.T) {
	logger := NewLogger(Config{
		Component: "discard-all",
		Level:     types.TraceLevel,
		Adapters:  []types.Adapter{discard.New()},
	})
	if logger.discardAdapter == nil {
		t.Fatal("expected discardAdapter to be detected")
	}

	logger.Trace("t")
	logger.Debug("d")
	logger.Info("i")
	logger.Warn("w")
	logger.Error("e")
	logger.Fatal("f")
	logger.Panic("p")
}

// TestLogger_DiscardLevelFiltered ensures disabled levels map to noopLog on the
// discard path (level above the message level).
func TestLogger_DiscardLevelFiltered(t *testing.T) {
	logger := NewLogger(Config{
		Component: "discard-filtered",
		Level:     types.ErrorLevel,
		Adapters:  []types.Adapter{discard.New()},
	})
	// These are below Error; they map to noopLog. Just ensure no panic.
	logger.Trace("t")
	logger.Debug("d")
	logger.Info("i")
	logger.Warn("w")
	// At/above threshold.
	logger.Error("e")
	logger.Fatal("f")
}

// TestLogger_FlushClose covers the happy path and error-propagation branches of
// Flush and Close.
func TestLogger_FlushClose(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		a := &countingAdapter{}
		logger := NewLogger(Config{
			Component: "fc-ok",
			Level:     types.InfoLevel,
			Adapters:  []types.Adapter{a},
		})
		if err := logger.Flush(); err != nil {
			t.Errorf("Flush err = %v, want nil", err)
		}
		if err := logger.Close(); err != nil {
			t.Errorf("Close err = %v, want nil", err)
		}
		if a.flushCalls != 1 || a.closeCalls != 1 {
			t.Errorf("flushCalls=%d closeCalls=%d, want 1 each", a.flushCalls, a.closeCalls)
		}
	})

	t.Run("flush error propagates", func(t *testing.T) {
		wantErr := errors.New("flush failed")
		a := &countingAdapter{flushErr: wantErr}
		logger := NewLogger(Config{
			Component: "fc-flush-err",
			Level:     types.InfoLevel,
			Adapters:  []types.Adapter{a},
		})
		if err := logger.Flush(); !errors.Is(err, wantErr) {
			t.Errorf("Flush err = %v, want %v", err, wantErr)
		}
	})

	t.Run("close error propagates", func(t *testing.T) {
		wantErr := errors.New("close failed")
		a := &countingAdapter{closeErr: wantErr}
		logger := NewLogger(Config{
			Component: "fc-close-err",
			Level:     types.InfoLevel,
			Adapters:  []types.Adapter{a},
		})
		if err := logger.Close(); !errors.Is(err, wantErr) {
			t.Errorf("Close err = %v, want %v", err, wantErr)
		}
	})
}

// TestLogger_SamplingConfigured verifies the sampler is wired when enabled.
// (Sampler is stored on the logger; this exercises the NewLogger sampling
// branch without asserting hot-path behaviour.)
func TestLogger_SamplingConfigured(t *testing.T) {
	s := &fixedSampler{sample: true}
	logger := NewLogger(Config{
		Component:      "sampled",
		Level:          types.InfoLevel,
		Adapters:       []types.Adapter{discard.New()},
		EnableSampling: true,
		Sampler:        s,
	})
	if logger.sampler == nil {
		t.Fatal("expected sampler to be configured")
	}
}

// fixedSampler is a deterministic Sampler for wiring tests.
type fixedSampler struct {
	sample bool
	rate   float64
}

func (s *fixedSampler) ShouldSample(entry *types.LogEntry) bool { return s.sample }
func (s *fixedSampler) GetRate() float64                        { return s.rate }
func (s *fixedSampler) SetRate(rate float64)                    { s.rate = rate }

// TestLogger_LevelDisabledNoWrite confirms that below-threshold levels on a
// regular adapter map to noopLog and never reach the adapter.
func TestLogger_LevelDisabledNoWrite(t *testing.T) {
	a := &countingAdapter{}
	logger := NewLogger(Config{
		Component: "disabled",
		Level:     types.WarnLevel,
		Adapters:  []types.Adapter{a},
	})

	logger.Trace("t")
	logger.Debug("d")
	logger.Info("i")
	if len(a.levels) != 0 {
		t.Fatalf("expected no writes below threshold, got %v", a.levels)
	}

	logger.Warn("w")
	logger.Error("e")
	if len(a.levels) != 2 {
		t.Errorf("expected 2 writes at/above threshold, got %v", a.levels)
	}
}

// TestLogger_DebugWarnErrorMultiNoMask covers the *NoMaskMulti specialized
// paths for Debug/Warn/Error (multiple adapters, no masking) which the
// single-adapter tests do not reach.
func TestLogger_DebugWarnErrorMultiNoMask(t *testing.T) {
	a1 := &countingAdapter{}
	a2 := &countingAdapter{}
	logger := NewLogger(Config{
		Component: "multi-nomask",
		Level:     types.DebugLevel,
		Adapters:  []types.Adapter{a1, a2},
	})

	logger.Debug("d")
	logger.Warn("w")
	logger.Error("e")

	if len(a1.levels) != 3 || len(a2.levels) != 3 {
		t.Fatalf("writes a1=%d a2=%d, want 3 each", len(a1.levels), len(a2.levels))
	}
	want := []types.LogLevel{types.DebugLevel, types.WarnLevel, types.ErrorLevel}
	for i, w := range want {
		if a1.levels[i] != w {
			t.Errorf("a1 write %d = %v, want %v", i, a1.levels[i], w)
		}
	}
}

// TestLogger_InfoMultiNoMask covers infoNoMaskMulti explicitly.
func TestLogger_InfoMultiNoMask(t *testing.T) {
	a1 := &countingAdapter{}
	a2 := &countingAdapter{}
	logger := NewLogger(Config{
		Component: "info-multi-nomask",
		Level:     types.InfoLevel,
		Adapters:  []types.Adapter{a1, a2},
	})

	logger.Info("i")
	if len(a1.levels) != 1 || len(a2.levels) != 1 {
		t.Errorf("writes a1=%d a2=%d, want 1 each", len(a1.levels), len(a2.levels))
	}
	if a1.levels[0] != types.InfoLevel {
		t.Errorf("level = %v, want Info", a1.levels[0])
	}
}

// TestLogger_WithFieldMultiAdapterNoMask covers the FieldBuilder multi-adapter
// fan-out (len(adapters) > 1) with 2 fields.
func TestLogger_WithFieldMultiAdapterNoMask(t *testing.T) {
	a1 := &countingAdapter{}
	a2 := &countingAdapter{}
	logger := NewLogger(Config{
		Component: "wf-multi",
		Level:     types.DebugLevel,
		Adapters:  []types.Adapter{a1, a2},
	})

	logger.WithField("k1", "v1").WithField("k2", 2).Info("multi write")

	if len(a1.levels) != 1 || len(a2.levels) != 1 {
		t.Errorf("writes a1=%d a2=%d, want 1 each", len(a1.levels), len(a2.levels))
	}
}
