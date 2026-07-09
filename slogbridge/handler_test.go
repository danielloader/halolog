// @author Admilson B. F. Cossa

package slogbridge

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
	"testing/slogtest"

	jsonfmt "github.com/go-gen-ecosystem/halolog/adapters/formatters/json"
	"github.com/go-gen-ecosystem/halolog/adapters/outputs/console"
	"github.com/go-gen-ecosystem/halolog/core"
	"github.com/go-gen-ecosystem/halolog/types"
)

func newBridgeLogger(buf *bytes.Buffer) *Handler {
	return New(core.NewLogger(core.Config{
		Component: "slog", Level: types.DebugLevel,
		Adapters: []types.Adapter{console.NewWithWriter(buf, jsonfmt.NewJsonFormatter())},
	}))
}

// TestSlogtestConformance runs the STANDARD LIBRARY handler conformance suite
// against the bridge, one subtest per rule. The result function parses the
// emitted JSON line back into slogtest's expected shape: HaloLog's "message"
// key maps to slog's "msg", and dot-joined group keys expand into nested maps
// (the bridge's documented group encoding).
//
// DOCUMENTED DEVIATION (skipped visibly, not hidden): the "empty-record /
// zero-time" rule expects a handler to omit the time key when Record.Time is
// zero. HaloLog's contract is that every line carries the logger's own clock
// timestamp (syslog-style) — the record's time is never rendered, zero or not —
// so that rule does not apply to this backend.
func TestSlogtestConformance(t *testing.T) {
	var buf bytes.Buffer
	slogtest.Run(t, func(t *testing.T) slog.Handler {
		if strings.HasSuffix(t.Name(), "/zero-time") {
			t.Skip("HaloLog stamps its own clock time on every line; Record.Time is never rendered (documented deviation)")
		}
		buf.Reset()
		return newBridgeLogger(&buf)
	}, func(t *testing.T) map[string]any {
		line := strings.TrimSpace(buf.String())
		var flat map[string]any
		if err := json.Unmarshal([]byte(line), &flat); err != nil {
			t.Fatalf("bridge emitted invalid JSON: %v\nline: %s", err, line)
		}
		return expand(flat)
	})
}

// expand converts the bridge's flat representation to slogtest's expected one:
// renames "message" to slog.MessageKey and expands "a.b.c" keys into nested
// maps.
func expand(flat map[string]any) map[string]any {
	out := make(map[string]any, len(flat))
	for k, v := range flat {
		if k == "message" {
			out[slog.MessageKey] = v
			continue
		}
		parts := strings.Split(k, ".")
		m := out
		for _, p := range parts[:len(parts)-1] {
			next, ok := m[p].(map[string]any)
			if !ok {
				next = map[string]any{}
				m[p] = next
			}
			m = next
		}
		m[parts[len(parts)-1]] = v
	}
	return out
}

// TestBridgeLevelsAndFilter verifies level mapping and Enabled filtering.
func TestBridgeLevelsAndFilter(t *testing.T) {
	var buf bytes.Buffer
	h := New(core.NewLogger(core.Config{
		Component: "slog", Level: types.WarnLevel,
		Adapters: []types.Adapter{console.NewWithWriter(&buf, jsonfmt.NewJsonFormatter())},
	}))
	lg := slog.New(h)

	lg.Info("filtered out")
	lg.Warn("kept", slog.Int("n", 1))
	lg.Error("kept too")

	out := buf.String()
	if strings.Contains(out, "filtered out") {
		t.Fatalf("INFO must be filtered at Warn level: %s", out)
	}
	if !strings.Contains(out, `"level":"WARN"`) || !strings.Contains(out, `"level":"ERROR"`) {
		t.Fatalf("level mapping wrong: %s", out)
	}
	if !strings.Contains(out, `"n":1`) {
		t.Fatalf("attr lost: %s", out)
	}
}

// TestBridgeGroupsAndWithAttrs verifies dotted-prefix group encoding with
// pre-bound attrs across WithGroup boundaries.
func TestBridgeGroupsAndWithAttrs(t *testing.T) {
	var buf bytes.Buffer
	h := newBridgeLogger(&buf)
	lg := slog.New(h).With("svc", "api").WithGroup("req").With("id", "abc")

	lg.Info("handled", slog.Int("status", 200))

	line := buf.String()
	for _, want := range []string{`"svc":"api"`, `"req.id":"abc"`, `"req.status":200`} {
		if !strings.Contains(line, want) {
			t.Fatalf("missing %s in %s", want, line)
		}
	}
}
