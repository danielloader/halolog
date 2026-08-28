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

// Package edge exercises HaloLog's hostile-input and API-misuse edges through
// the public API only. Every test here pins behavior that once regressed:
// level-filter bypass on the classic fluent API, an unconsulted sampler,
// non-finite floats emitting invalid JSON, pooled-state corruption from
// builder reuse, and pre-epoch timestamp rendering.
// @author Admilson B. F. Cossa
package edge

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"sync"
	"testing"

	"log/slog"

	"github.com/go-gen-ecosystem/halolog"
	jsonfmt "github.com/go-gen-ecosystem/halolog/adapters/formatters/json"
	"github.com/go-gen-ecosystem/halolog/adapters/outputs/console"
	"github.com/go-gen-ecosystem/halolog/adapters/outputs/discard"
	"github.com/go-gen-ecosystem/halolog/core"
	"github.com/go-gen-ecosystem/halolog/slogbridge"
	"github.com/go-gen-ecosystem/halolog/types"
)

// syncBuf is a goroutine-safe buffer sink for concurrent probes.
type syncBuf struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *syncBuf) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *syncBuf) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

// newBufLogger builds a single-adapter (direct-path-eligible) JSON logger.
func newBufLogger(level types.LogLevel) (*core.Logger, *syncBuf) {
	buf := &syncBuf{}
	ad := console.NewWithWriter(buf, jsonfmt.NewJsonFormatter())
	return core.New().Level(level).Adapter(ad).MustBuild(), buf
}

// newCaptureLogger forces the slice-capture path (a second adapter disables
// direct-append eligibility).
func newCaptureLogger(level types.LogLevel) (*core.Logger, *syncBuf) {
	buf := &syncBuf{}
	ad := console.NewWithWriter(buf, jsonfmt.NewJsonFormatter())
	return core.New().Level(level).Adapter(ad).Adapter(discard.New()).MustBuild(), buf
}

func lines(s string) []string {
	var out []string
	for _, ln := range strings.Split(s, "\n") {
		if strings.TrimSpace(ln) != "" {
			out = append(out, ln)
		}
	}
	return out
}

// --- Level filtering must hold on every fluent API -------------------------

func TestLevelFilter_AllAPIs(t *testing.T) {
	l, buf := newBufLogger(types.WarnLevel)
	l.Info("plain")
	l.Typed().WithString("k", "v").Info("typed")
	l.InfoLine().WithString("k", "v").Msg("line")
	l.WithField("k", "v").Info("fieldbuilder")
	l.WithField("k", "v").Debug("fieldbuilder-debug")
	l.WithField("k", "v").Trace("fieldbuilder-trace")
	if got := buf.String(); got != "" {
		t.Errorf("level-filtered output leaked: %q", got)
	}
	// At-threshold lines still pass on every API.
	l.Warn("w1")
	l.WithField("k", "v").Warn("w2")
	l.Typed().WithString("k", "v").Warn("w3")
	l.WarnLine().WithString("k", "v").Msg("w4")
	if n := len(lines(buf.String())); n != 4 {
		t.Errorf("expected 4 at-threshold lines, got %d", n)
	}
}

// --- The sampler must be consulted and honored -----------------------------

type fixedSampler struct {
	mu    sync.Mutex
	calls int
	allow bool
}

func (s *fixedSampler) ShouldSample(*types.LogEntry) bool {
	s.mu.Lock()
	s.calls++
	s.mu.Unlock()
	return s.allow
}
func (s *fixedSampler) GetRate() float64 { return 0 }
func (s *fixedSampler) SetRate(float64)  {}

func (s *fixedSampler) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls
}

func TestSamplerConsultedAndHonored(t *testing.T) {
	// A drop-everything sampler: consulted for every line, nothing written.
	buf := &syncBuf{}
	drop := &fixedSampler{allow: false}
	l := core.New().Level(types.InfoLevel).
		Adapter(console.NewWithWriter(buf, jsonfmt.NewJsonFormatter())).
		Sampling(drop).MustBuild()
	for i := 0; i < 10; i++ {
		l.Info("m")
		l.Typed().WithInt("i", i).Info("mt")
		l.WithField("i", i).Info("mf")
	}
	if got := len(lines(buf.String())); got != 0 {
		t.Errorf("drop-all sampler: %d lines written, want 0", got)
	}
	if drop.count() != 30 {
		t.Errorf("drop-all sampler consulted %d times, want 30", drop.count())
	}

	// A pass-everything sampler: all lines written.
	buf2 := &syncBuf{}
	pass := &fixedSampler{allow: true}
	l2 := core.New().Level(types.InfoLevel).
		Adapter(console.NewWithWriter(buf2, jsonfmt.NewJsonFormatter())).
		Sampling(pass).MustBuild()
	for i := 0; i < 5; i++ {
		l2.Info("m")
	}
	if got := len(lines(buf2.String())); got != 5 {
		t.Errorf("pass-all sampler: %d lines written, want 5", got)
	}
}

// --- JSON validity under hostile values ------------------------------------

func TestNonFiniteFloatsAreValidJSON(t *testing.T) {
	l, buf := newBufLogger(types.InfoLevel)
	l.Typed().WithFloat64("nan", math.NaN()).Info("m1")
	l.Typed().WithFloat64("pos", math.Inf(1)).Info("m2")
	l.Typed().WithFloat64("neg", math.Inf(-1)).Info("m3")
	l.Typed().WithAny("nan32", float32(math.NaN())).Info("m4")
	out := lines(buf.String())
	if len(out) != 4 {
		t.Fatalf("expected 4 lines, got %d", len(out))
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(out[0]), &m); err != nil {
		t.Fatalf("NaN line invalid JSON: %v — %q", err, out[0])
	}
	if m["nan"] != "NaN" {
		t.Errorf(`NaN renders as %v, want "NaN"`, m["nan"])
	}
	for i, ln := range out {
		if !json.Valid([]byte(ln)) {
			t.Errorf("line %d invalid JSON: %s", i, ln)
		}
	}
}

func TestFloatEdgeFormats(t *testing.T) {
	l, buf := newBufLogger(types.InfoLevel)
	l.Typed().
		WithFloat64("big", 1e21).
		WithFloat64("negzero", math.Copysign(0, -1)).
		WithFloat64("denorm", 5e-324).
		WithFloat64("maxf", math.MaxFloat64).
		Info("floats")
	for _, ln := range lines(buf.String()) {
		if !json.Valid([]byte(ln)) {
			t.Errorf("float edge produced invalid JSON: %s", ln)
		}
	}
}

func TestControlCharAndQuoteEscaping(t *testing.T) {
	nasty := "a\"b\\c\nd\te\rf\x00g\x1fh"
	l, buf := newBufLogger(types.InfoLevel)
	l.Typed().WithString("k", nasty).Info(nasty)
	out := lines(buf.String())
	if len(out) != 1 {
		t.Fatalf("expected 1 line, got %d", len(out))
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(out[0]), &m); err != nil {
		t.Fatalf("control chars broke JSON: %v in %q", err, out[0])
	}
	if m["message"] != nasty || m["k"] != nasty {
		t.Errorf("escaping not round-trip-safe: message=%q k=%q want %q", m["message"], m["k"], nasty)
	}
}

func TestUnicodePassthrough(t *testing.T) {
	msg := "héllo 世界 🎉   " // includes JS-hostile line/paragraph separators
	l, buf := newBufLogger(types.InfoLevel)
	l.Typed().WithString("k", msg).Info(msg)
	var m map[string]any
	out := lines(buf.String())
	if err := json.Unmarshal([]byte(out[0]), &m); err != nil || m["message"] != msg {
		t.Errorf("unicode round-trip failed: err=%v got=%q", err, m["message"])
	}
}

func TestPredeclaredKeyEscaping(t *testing.T) {
	weird := halolog.Key("we\"ird\\key\n")
	l, buf := newBufLogger(types.InfoLevel)
	l.Typed().Str(weird, "v").Info("m")
	out := lines(buf.String())
	var m map[string]any
	if err := json.Unmarshal([]byte(out[0]), &m); err != nil {
		t.Fatalf("pre-declared key with specials breaks JSON: %v in %q", err, out[0])
	}
	if m["we\"ird\\key\n"] != "v" {
		t.Errorf("key not round-tripped: %v", m)
	}
}

func TestEmptyKeyAndMessage(t *testing.T) {
	l, buf := newBufLogger(types.InfoLevel)
	l.Typed().WithString("", "emptykey").Info("")
	l.InfoLine().Send()
	for _, ln := range lines(buf.String()) {
		if !json.Valid([]byte(ln)) {
			t.Errorf("empty key/message invalid JSON: %q", ln)
		}
	}
}

// --- Field overflow past the 64-slot static buffer -------------------------

func TestManyFieldsBothPaths(t *testing.T) {
	for _, tc := range []struct {
		name  string
		build func(types.LogLevel) (*core.Logger, *syncBuf)
	}{
		{"direct", newBufLogger},
		{"capture", newCaptureLogger},
	} {
		t.Run(tc.name, func(t *testing.T) {
			l, buf := tc.build(types.InfoLevel)
			b := l.Typed()
			for i := 0; i < 100; i++ {
				b = b.WithInt(fmt.Sprintf("f%03d", i), i)
			}
			b.Info("hundred")
			out := lines(buf.String())
			if len(out) != 1 {
				t.Fatalf("expected 1 line, got %d", len(out))
			}
			var m map[string]any
			if err := json.Unmarshal([]byte(out[0]), &m); err != nil {
				t.Fatalf("invalid JSON with 100 fields: %v", err)
			}
			for i := 0; i < 100; i++ {
				if _, ok := m[fmt.Sprintf("f%03d", i)]; !ok {
					t.Fatalf("field f%03d dropped (100-field line)", i)
				}
			}
			// The pooled state must come back clean for the next line.
			l.Typed().WithInt("only", 1).Info("clean")
			out = lines(buf.String())
			var m2 map[string]any
			if err := json.Unmarshal([]byte(out[len(out)-1]), &m2); err != nil {
				t.Fatalf("follow-up line invalid: %v", err)
			}
			if len(m2) != 4 { // time, level, message, only
				t.Errorf("pooled state leaked fields across lines: %v", m2)
			}
		})
	}
}

func TestHugeMessage(t *testing.T) {
	huge := strings.Repeat("x", 2<<20) // 2 MiB
	l, buf := newBufLogger(types.InfoLevel)
	l.Typed().WithString("k", huge).Info("big")
	l.Typed().WithString("s", "small").Info("after-big") // pool must recover
	out := lines(buf.String())
	if len(out) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(out))
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(out[0]), &m); err != nil {
		t.Fatalf("huge line invalid: %v", err)
	}
	if s, ok := m["k"].(string); !ok || len(s) != 2<<20 {
		t.Errorf("huge field mangled")
	}
	if !json.Valid([]byte(out[1])) {
		t.Errorf("line after huge line invalid: %q", out[1])
	}
}

// --- Use-after-dispatch misuse is a no-op, never corruption ----------------

func TestLineDoubleMsgIsNoOp(t *testing.T) {
	l, buf := newBufLogger(types.InfoLevel)
	ln := l.InfoLine().WithString("k", "v")
	ln.Msg("first")
	ln.Msg("second") // documented misuse: must be a silent no-op, not a panic
	ln.WithString("late", "field").Msg("third")
	out := lines(buf.String())
	if len(out) != 1 {
		t.Fatalf("double Msg must emit exactly 1 line, got %d: %v", len(out), out)
	}
	var m map[string]any
	_ = json.Unmarshal([]byte(out[0]), &m)
	if m["message"] != "first" {
		t.Errorf("kept line is %v, want the first", m["message"])
	}
}

func TestFieldBuilderReuseCannotPoisonPool(t *testing.T) {
	l, buf := newBufLogger(types.InfoLevel)
	fb := l.WithField("a", 1)
	fb.Info("first")
	fb.Info("second") // misuse: state already recycled — must be a no-op

	// The regression this pins: after a double dispatch, two later builders
	// used to receive the SAME pooled state, so fb1's line carried fb2's
	// field. With the epoch guard the pool stays consistent.
	fb1 := l.WithField("first_key", "A")
	fb2 := l.WithField("second_key", "B")
	fb1.Info("from-fb1")
	fb2.Info("from-fb2")

	out := lines(buf.String())
	if len(out) != 3 { // first + from-fb1 + from-fb2 (the misuse emits nothing)
		t.Fatalf("expected 3 lines, got %d: %v", len(out), out)
	}
	var m1, m2 map[string]any
	_ = json.Unmarshal([]byte(out[1]), &m1)
	_ = json.Unmarshal([]byte(out[2]), &m2)
	if m1["first_key"] != "A" || m1["second_key"] != nil {
		t.Errorf("fb1 line corrupted: %v", m1)
	}
	if m2["second_key"] != "B" || m2["first_key"] != nil {
		t.Errorf("fb2 line corrupted: %v", m2)
	}
}

// --- Fatal / Panic semantics ------------------------------------------------

func TestPanicPanicsAndFatalExits(t *testing.T) {
	buf := &syncBuf{}
	exitCode := -1
	l := core.NewLogger(core.Config{
		Level:    types.InfoLevel,
		Adapters: []types.Adapter{console.NewWithWriter(buf, jsonfmt.NewJsonFormatter())},
		ExitFunc: func(code int) { exitCode = code },
	})

	func() {
		defer func() {
			if r := recover(); r != "boom" {
				t.Errorf("Panic must panic with the message, got %v", r)
			}
		}()
		l.Panic("boom")
	}()

	l.Fatal("bye")
	if exitCode != 1 {
		t.Errorf("Fatal exit code = %d, want 1", exitCode)
	}

	// Both lines must have been written before terminating.
	out := lines(buf.String())
	if len(out) != 2 {
		t.Fatalf("expected 2 lines (panic + fatal), got %d", len(out))
	}
}

// --- Timestamp edges --------------------------------------------------------

func TestPreEpochTimestampRendersDigits(t *testing.T) {
	f := jsonfmt.NewJsonFormatterWithPrecision(jsonfmt.PrecisionMilli)
	e := &types.LogEntry{TimestampUnix: -1_500_000_000, Level: types.InfoLevel, Message: "pre-epoch"}
	out := string(f.Format(e, nil))
	i := strings.Index(out, `"time":"`)
	if i < 0 {
		t.Fatalf("no time field in %q", out)
	}
	rest := out[i+8:]
	j := strings.IndexByte(rest, '.')
	if j < 0 {
		t.Fatalf("no fractional digits in %q", out)
	}
	frac := rest[j+1 : j+4]
	for _, c := range frac {
		if c < '0' || c > '9' {
			t.Fatalf("pre-epoch timestamp renders garbage fractional digits %q in %q", frac, out)
		}
	}
	// -1.5s floor-divides to second -2 with +500ms, i.e. …58.500 local time.
	if frac != "500" {
		t.Errorf("fractional digits = %q, want 500 (floored remainder)", frac)
	}
	if !json.Valid([]byte(out)) {
		t.Errorf("pre-epoch line invalid JSON: %q", out)
	}
}

func TestZeroValueEntryFormat(t *testing.T) {
	f := jsonfmt.NewJsonFormatter()
	out := string(f.Format(&types.LogEntry{}, nil))
	if !json.Valid([]byte(out)) {
		t.Errorf("zero-value entry invalid JSON: %q", out)
	}
}

// --- Concurrency integrity --------------------------------------------------

func TestConcurrentLineIntegrity(t *testing.T) {
	l, buf := newBufLogger(types.InfoLevel)
	const goroutines, perG = 32, 200
	var wg sync.WaitGroup
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < perG; i++ {
				l.Typed().WithInt("g", g).WithInt("i", i).Info("cc")
			}
		}(g)
	}
	wg.Wait()
	out := lines(buf.String())
	if len(out) != goroutines*perG {
		t.Errorf("lost/duplicated lines under concurrency: got %d want %d", len(out), goroutines*perG)
	}
	for _, ln := range out {
		if !json.Valid([]byte(ln)) {
			t.Fatalf("torn line under concurrency: %q", ln)
		}
	}
}

func TestGetLoggerSingletonConcurrent(t *testing.T) {
	const goroutines = 64
	var wg sync.WaitGroup
	ptrs := make([]*core.Logger, goroutines)
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			ptrs[g] = halolog.GetLogger("edge-shared")
		}(g)
	}
	wg.Wait()
	for g := 1; g < goroutines; g++ {
		if ptrs[g] != ptrs[0] {
			t.Fatalf("GetLogger returned different instances")
		}
	}
}

// --- slog bridge group flattening -------------------------------------------

func TestSlogBridgeGroups(t *testing.T) {
	l, buf := newBufLogger(types.DebugLevel)
	sl := slog.New(slogbridge.New(l))
	sl.WithGroup("req").With("id", 7).Info("handled", "dur", "5ms", slog.Group("db", "rows", 3))
	out := lines(buf.String())
	var m map[string]any
	if err := json.Unmarshal([]byte(out[0]), &m); err != nil {
		t.Fatalf("slog bridge produced invalid JSON: %v", err)
	}
	for _, want := range []string{"req.id", "req.dur", "req.db.rows"} {
		if _, ok := m[want]; !ok {
			t.Errorf("slog group flattening missing %q: %v", want, m)
		}
	}
}
