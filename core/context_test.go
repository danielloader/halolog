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

// Package core — bound-context (child logger) behavior tests: output
// correctness on both encode paths, byte-identity between them, chaining,
// masking visibility, terminal semantics, and the construction-time cap.
// @author Admilson B. F. Cossa

package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	jsonfmt "github.com/go-gen-ecosystem/halolog/adapters/formatters/json"
	"github.com/go-gen-ecosystem/halolog/adapters/outputs/console"
	"github.com/go-gen-ecosystem/halolog/types"
)

var ctxKeyTenant = &types.FieldKey{Name: "tenant", JSONFragment: jsonfmt.KeyFragment("tenant")}

// newDirectBufLogger builds a direct-eligible JSON logger into buf.
func newDirectBufLogger(buf *bytes.Buffer) *Logger {
	return NewLogger(Config{
		Level:    types.InfoLevel,
		Adapters: []types.Adapter{console.NewWithWriter(buf, jsonfmt.NewJsonFormatter())},
		ExitFunc: func(int) {},
	})
}

// newCaptureBufLogger forces the capture path (two adapters).
func newCaptureBufLogger(buf *bytes.Buffer) *Logger {
	return NewLogger(Config{
		Level: types.InfoLevel,
		Adapters: []types.Adapter{
			console.NewWithWriter(buf, jsonfmt.NewJsonFormatter()),
			console.NewWithWriter(&bytes.Buffer{}, jsonfmt.NewJsonFormatter()),
		},
		ExitFunc: func(int) {},
	})
}

func lastLine(buf *bytes.Buffer) string {
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	return lines[len(lines)-1]
}

func parseLine(t *testing.T, line string) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(line), &m); err != nil {
		t.Fatalf("invalid JSON line: %v — %q", err, line)
	}
	return m
}

// TestContext_AllAPIsCarryBoundFields verifies that every logging API on a
// child logger emits the bound context: message-only, FieldBuilder, Typed,
// and Line — on the direct path.
func TestContext_AllAPIsCarryBoundFields(t *testing.T) {
	var buf bytes.Buffer
	child := newDirectBufLogger(&buf).With().
		Str(ctxKeyTenant, "acme").
		WithString("region", "eu").
		WithInt("shard", 7).
		Logger()

	child.Info("plain")
	child.WithField("k", "v").Info("fieldbuilder")
	child.Typed().WithInt("status", 200).Info("typed")
	child.InfoLine().WithBool("ok", true).Msg("line")

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 4 {
		t.Fatalf("expected 4 lines, got %d", len(lines))
	}
	for i, ln := range lines {
		m := parseLine(t, ln)
		if m["tenant"] != "acme" || m["region"] != "eu" || m["shard"] != float64(7) {
			t.Errorf("line %d missing bound context: %v", i, m)
		}
	}
	// Per-line fields also present, after the bound context.
	m := parseLine(t, lines[2])
	if m["status"] != float64(200) {
		t.Errorf("typed line lost its own field: %v", m)
	}
	if !strings.Contains(lines[2], `"tenant":"acme","region":"eu","shard":7,"status":200`) {
		t.Errorf("bound context must precede line fields: %q", lines[2])
	}
}

// TestContext_ByteIdentityAcrossPaths pins the core invariant: the same child
// context and line render byte-identically on the direct path and the capture
// path (ignoring the timestamp header, which differs only by clock reads).
func TestContext_ByteIdentityAcrossPaths(t *testing.T) {
	var direct, capture bytes.Buffer
	dl := newDirectBufLogger(&direct).With().
		Str(ctxKeyTenant, "acme").WithString("esc", "a\"b").WithFloat64("r", 2.5).Logger()
	cl := newCaptureBufLogger(&capture).With().
		Str(ctxKeyTenant, "acme").WithString("esc", "a\"b").WithFloat64("r", 2.5).Logger()

	dl.Typed().WithInt("n", 42).Info("m")
	cl.Typed().WithInt("n", 42).Info("m")

	trim := func(s string) string {
		i := strings.Index(s, `,"level"`)
		return s[i:]
	}
	d, c := trim(lastLine(&direct)), trim(lastLine(&capture))
	if d != c {
		t.Fatalf("bound-context output diverges across paths:\ndirect:  %s\ncapture: %s", d, c)
	}
}

// TestContext_Chaining verifies grandchildren re-bind the full ancestor
// context plus their own additions, in order.
func TestContext_Chaining(t *testing.T) {
	var buf bytes.Buffer
	parent := newDirectBufLogger(&buf).With().WithString("svc", "auth").Logger()
	childLog := parent.With().WithString("req", "r-1").Logger()

	childLog.Info("nested")
	m := parseLine(t, lastLine(&buf))
	if m["svc"] != "auth" || m["req"] != "r-1" {
		t.Fatalf("chained context incomplete: %v", m)
	}
	if !strings.Contains(lastLine(&buf), `"svc":"auth","req":"r-1"`) {
		t.Fatalf("chained order wrong: %q", lastLine(&buf))
	}

	// The parent is unaffected by the child's additions.
	buf.Reset()
	parent.Info("parent-only")
	if strings.Contains(buf.String(), "r-1") {
		t.Fatal("child binding leaked into parent")
	}
}

// redactMasker blanks the value of every field named "secret".
type redactMasker struct{}

func (redactMasker) Apply(entry *types.LogEntry) {
	n := entry.StaticFieldCount
	if n > len(entry.StaticFields) {
		n = len(entry.StaticFields)
	}
	for i := 0; i < n; i++ {
		if entry.StaticFields[i].Key == "secret" {
			entry.StaticFields[i].Val = types.StringValue("[REDACTED]")
			entry.StaticFields[i].Value = nil
		}
	}
}
func (redactMasker) MaskField(f *types.TypedFieldData) *types.TypedFieldData    { return f }
func (redactMasker) MaskFields(f []types.TypedFieldData) []types.TypedFieldData { return f }
func (redactMasker) MaskString(s string) string                                 { return s }
func (redactMasker) AddRule(_, _, _ string) error                               { return nil }
func (redactMasker) RemoveRule(string) error                                    { return nil }
func (redactMasker) AddPattern(_, _, _ string) error                            { return nil }
func (redactMasker) RemovePattern(string)                                       {}
func (redactMasker) GetPatterns() []string                                      { return nil }
func (redactMasker) Clone() types.PIIMasker                                     { return redactMasker{} }

// TestContext_MaskerSeesBoundFields pins the security invariant: bound fields
// travel as structured data on the capture path, so a configured masker can
// redact them — binding a field is never a masking bypass.
func TestContext_MaskerSeesBoundFields(t *testing.T) {
	var buf bytes.Buffer
	l := NewLogger(Config{
		Level:         types.InfoLevel,
		Adapters:      []types.Adapter{console.NewWithWriter(&buf, jsonfmt.NewJsonFormatter())},
		EnableMasking: true,
		Masker:        redactMasker{},
		ExitFunc:      func(int) {},
	})
	child := l.With().WithString("secret", "hunter2").WithString("ok", "visible").Logger()

	child.Info("masked?")
	m := parseLine(t, lastLine(&buf))
	if m["secret"] != "[REDACTED]" {
		t.Fatalf("masker did not see bound field: %v", m)
	}
	if m["ok"] != "visible" {
		t.Fatalf("masker over-applied: %v", m)
	}
}

// TestContext_FatalCarriesContextAndExits verifies terminal semantics on a
// bound logger and that the fatal line still carries the context.
func TestContext_FatalCarriesContextAndExits(t *testing.T) {
	var buf bytes.Buffer
	exitCode := -1
	l := NewLogger(Config{
		Level:    types.InfoLevel,
		Adapters: []types.Adapter{console.NewWithWriter(&buf, jsonfmt.NewJsonFormatter())},
		ExitFunc: func(code int) { exitCode = code },
	})
	child := l.With().WithString("svc", "auth").Logger()

	child.Fatal("bye")
	if exitCode != 1 {
		t.Fatalf("bound Fatal exit code = %d, want 1", exitCode)
	}
	if m := parseLine(t, lastLine(&buf)); m["svc"] != "auth" {
		t.Fatalf("fatal line lost bound context: %v", m)
	}
}

// TestDirectPath_TypedFatalExits is the regression test for the shared-tail
// fix: Typed().Fatal on a DIRECT-eligible logger must honor the
// flush-and-exit contract (it previously wrote the line and returned).
func TestDirectPath_TypedFatalExits(t *testing.T) {
	var buf bytes.Buffer
	exited := 0
	l := NewLogger(Config{
		Level:    types.InfoLevel,
		Adapters: []types.Adapter{console.NewWithWriter(&buf, jsonfmt.NewJsonFormatter())},
		ExitFunc: func(int) { exited++ },
	})
	l.Typed().WithInt("n", 1).Fatal("boom")
	if exited != 1 {
		t.Fatalf("direct-path Typed().Fatal must exit exactly once, got %d", exited)
	}
	func() {
		defer func() {
			if recover() != "p" {
				t.Fatal("direct-path Typed().Panic must panic with the message")
			}
		}()
		l.Typed().WithInt("n", 2).Panic("p")
	}()
}

// TestContext_BoundCap verifies the construction-time cap: fields beyond
// maxBoundFields are dropped consistently on BOTH representations.
func TestContext_BoundCap(t *testing.T) {
	var buf bytes.Buffer
	c := newDirectBufLogger(&buf).With()
	for i := 0; i < maxBoundFields+8; i++ {
		c = c.WithInt(fmt.Sprintf("b%02d", i), i)
	}
	child := c.Logger()
	if len(child.boundFields) != maxBoundFields {
		t.Fatalf("bound fields = %d, want cap %d", len(child.boundFields), maxBoundFields)
	}
	child.Info("capped")
	m := parseLine(t, lastLine(&buf))
	if _, ok := m[fmt.Sprintf("b%02d", maxBoundFields-1)]; !ok {
		t.Fatal("last in-cap bound field missing")
	}
	if _, ok := m[fmt.Sprintf("b%02d", maxBoundFields)]; ok {
		t.Fatal("over-cap bound field leaked into output")
	}
}

// TestContext_BranchingDoesNotAlias pins the copy-on-append contract: a
// Context is a VALUE that may be branched, and two children derived from the
// same intermediate context must never share writable backing — neither for
// the structured fields nor for the pre-encoded bytes. (The original
// implementation appended into shared backing; with spare capacity at the
// branch point — guaranteed for children of bound loggers, which seed
// headroom — the second branch overwrote the first branch's field, and a
// different-length encoding could tear the first child's bytes entirely.)
func TestContext_BranchingDoesNotAlias(t *testing.T) {
	var buf bytes.Buffer
	base := newDirectBufLogger(&buf)

	// Case 1: branch a first-level context off a BOUND logger (the seeded
	// +4 capacity made this deterministically broken before the fix).
	bound := base.With().WithString("svc", "auth").Logger()
	mid := bound.With() // seeds fields cap n+4, bytes cap +64 — spare capacity
	c1 := mid.WithString("branch", "one").Logger()
	c2 := mid.WithString("branch", "two-longer-value").Logger()

	c1.Info("from-c1")
	m1 := parseLine(t, lastLine(&buf))
	if m1["branch"] != "one" {
		t.Fatalf("c1 corrupted by c2's branch: %v", m1)
	}
	c2.Info("from-c2")
	m2 := parseLine(t, lastLine(&buf))
	if m2["branch"] != "two-longer-value" {
		t.Fatalf("c2 wrong: %v", m2)
	}

	// Case 2: branch an intermediate chain link (spare capacity from append
	// growth). Also verifies the bytes are not torn: both lines must parse.
	mid2 := base.With().WithString("a", "1")
	d1 := mid2.WithString("b", "bee").Logger()
	d2 := mid2.WithString("c", "sea").Logger()
	d1.Info("d1")
	md1 := parseLine(t, lastLine(&buf))
	if md1["b"] != "bee" || md1["c"] != nil {
		t.Fatalf("d1 corrupted by d2's branch: %v", md1)
	}
	d2.Info("d2")
	md2 := parseLine(t, lastLine(&buf))
	if md2["c"] != "sea" || md2["b"] != nil {
		t.Fatalf("d2 wrong: %v", md2)
	}
}

// TestContext_LevelInheritedAtDerivation documents level semantics: the child
// takes the parent's level at With().Logger() time.
func TestContext_LevelInheritedAtDerivation(t *testing.T) {
	var buf bytes.Buffer
	l := NewLogger(Config{
		Level:    types.WarnLevel,
		Adapters: []types.Adapter{console.NewWithWriter(&buf, jsonfmt.NewJsonFormatter())},
		ExitFunc: func(int) {},
	})
	child := l.With().WithString("svc", "auth").Logger()
	child.Info("filtered")
	child.WithField("k", 1).Info("filtered too")
	child.Typed().WithInt("k", 2).Info("filtered too")
	if got := strings.TrimSpace(buf.String()); got != "" {
		t.Fatalf("below-level bound lines leaked: %q", got)
	}
	child.Warn("passes")
	if m := parseLine(t, lastLine(&buf)); m["svc"] != "auth" {
		t.Fatalf("warn line lost context: %v", m)
	}
}
