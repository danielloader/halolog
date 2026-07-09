// @author Admilson B. F. Cossa

package core

import (
	"bytes"
	"strings"
	"testing"

	jsonfmt "github.com/go-gen-ecosystem/halolog/adapters/formatters/json"
	"github.com/go-gen-ecosystem/halolog/adapters/outputs/console"
	"github.com/go-gen-ecosystem/halolog/types"
)

// TestLine_MatchesTypedBuilder proves the level-first Line API produces the
// exact bytes the typed builder produces for the same fields (they share
// addField/dispatchLine, and this pins that).
func TestLine_MatchesTypedBuilder(t *testing.T) {
	var lineBuf, typedBuf bytes.Buffer
	lg := func(w *bytes.Buffer) *Logger {
		return NewLogger(Config{Component: "t", Level: types.InfoLevel,
			Adapters: []types.Adapter{console.NewWithWriter(w, jsonfmt.NewJsonFormatter())}})
	}
	l1, l2 := lg(&lineBuf), lg(&typedBuf)

	l1.InfoLine().Str(benchKey, "alice").WithInt("status", 200).
		WithFloat64("r", 1.5).WithBool("ok", true).Msg("m")
	l2.Typed().Str(benchKey, "alice").WithInt("status", 200).
		WithFloat64("r", 1.5).WithBool("ok", true).Info("m")

	after := func(s string) string {
		i := strings.Index(s, `"level"`)
		if i < 0 {
			t.Fatalf("no level in %q", s)
		}
		return s[i:]
	}
	if got, want := after(lineBuf.String()), after(typedBuf.String()); got != want {
		t.Fatalf("Line output differs from Typed output.\nline:  %q\ntyped: %q", got, want)
	}
}

// TestLine_DisabledLevelIsFreeAndSilent proves a filtered level writes nothing,
// acquires nothing, and allocates nothing.
func TestLine_DisabledLevelIsFreeAndSilent(t *testing.T) {
	spy := newRawSpy()
	l := NewLogger(Config{Component: "t", Level: types.WarnLevel, Adapters: []types.Adapter{spy}})

	allocs := testing.AllocsPerRun(200, func() {
		l.DebugLine().Str(benchKey, "alice").WithInt("n", 1).Msg("dropped")
	})
	if allocs != 0 {
		t.Fatalf("disabled line allocates: %v", allocs)
	}
	if len(spy.raw) != 0 || spy.zeroCalls != 0 {
		t.Fatalf("disabled line produced output: raw=%d zero=%d", len(spy.raw), spy.zeroCalls)
	}
	if ln := l.DebugLine(); ln.s != nil {
		t.Fatal("disabled line must not acquire pooled state")
	}
}

// TestLine_ErrAndSendEdges covers nil-error no-ops and Send's empty message.
func TestLine_ErrAndSendEdges(t *testing.T) {
	spy := newRawSpy()
	l := NewLogger(Config{Component: "t", Level: types.InfoLevel, Adapters: []types.Adapter{spy}})

	l.InfoLine().Err(benchKey, nil).WithError(nil).WithInt("n", 1).Send()
	if len(spy.raw) != 1 {
		t.Fatalf("expected 1 raw line, got %d", len(spy.raw))
	}
	line := string(spy.raw[0])
	if strings.Contains(line, "error") || !strings.Contains(line, `"n":1`) {
		t.Fatalf("nil errors must be omitted, n kept: %q", line)
	}
	if !strings.Contains(line, `"message":""`) {
		t.Fatalf("Send must emit an empty message: %q", line)
	}
}

// TestZeroAlloc_Line guards the sacred invariant on the Line API (direct path).
func TestZeroAlloc_Line(t *testing.T) {
	l := NewLogger(Config{Component: "t", Level: types.InfoLevel,
		Adapters: []types.Adapter{&noRecordRaw{enc: jsonfmt.NewJsonFormatter()}}})

	allocs := testing.AllocsPerRun(1000, func() {
		l.InfoLine().Str(benchKey, "alice").WithInt("status", 200).WithBool("ok", true).Msg("m")
	})
	if allocs != 0 {
		t.Fatalf("Line direct path allocates: %v allocs/op (must be 0)", allocs)
	}
}
