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

// rawSpy is a types.Adapter that accepts raw lines and records which write path
// was used, so tests can PROVE the direct fast path engaged (or did not).
type rawSpy struct {
	raw       [][]byte
	zeroCalls int
	enc       types.DirectFieldEncoder
}

func newRawSpy() *rawSpy { return &rawSpy{enc: jsonfmt.NewJsonFormatter()} }

func (s *rawSpy) Name() string                    { return "rawspy" }
func (s *rawSpy) Write(*types.LogEntry) error     { s.zeroCalls++; return nil }
func (s *rawSpy) WriteZero(*types.LogEntry) error { s.zeroCalls++; return nil }
func (s *rawSpy) Flush() error                    { return nil }
func (s *rawSpy) Close() error                    { return nil }
func (s *rawSpy) SetFormatter(types.Formatter)    {}
func (s *rawSpy) Health() error                   { return nil }
func (s *rawSpy) WriteRaw(line []byte) error {
	s.raw = append(s.raw, append([]byte(nil), line...))
	return nil
}
func (s *rawSpy) DirectEncoder() types.DirectFieldEncoder { return s.enc }

var benchKey = &types.FieldKey{Name: "user", JSONFragment: jsonfmt.KeyFragment("user")}

// TestDirectPath_Engages proves a single raw-capable JSON adapter with no
// masking/sampling routes typed lines through WriteRaw, not WriteZero.
func TestDirectPath_Engages(t *testing.T) {
	spy := newRawSpy()
	l := NewLogger(Config{Component: "t", Level: types.InfoLevel, Adapters: []types.Adapter{spy}})

	l.Typed().Str(benchKey, "alice").WithInt("n", 7).Info("hello")

	if len(spy.raw) != 1 || spy.zeroCalls != 0 {
		t.Fatalf("direct path did not engage: raw=%d zero=%d", len(spy.raw), spy.zeroCalls)
	}
	line := string(spy.raw[0])
	if !strings.HasSuffix(line, `,"user":"alice","n":7}`+"\n") {
		t.Fatalf("direct line wrong: %q", line)
	}
	if !strings.Contains(line, `"level":"INFO","message":"hello"`) {
		t.Fatalf("direct header wrong: %q", line)
	}
}

// TestDirectPath_MatchesCapturePath logs the same typed line through the direct
// path (single console+JSON adapter) and the capture path (two adapters →
// ineligible) and asserts identical bytes after the timestamp.
func TestDirectPath_MatchesCapturePath(t *testing.T) {
	var directBuf, capA, capB bytes.Buffer
	direct := NewLogger(Config{Component: "t", Level: types.InfoLevel,
		Adapters: []types.Adapter{console.NewWithWriter(&directBuf, jsonfmt.NewJsonFormatter())}})
	capture := NewLogger(Config{Component: "t", Level: types.InfoLevel,
		Adapters: []types.Adapter{
			console.NewWithWriter(&capA, jsonfmt.NewJsonFormatter()),
			console.NewWithWriter(&capB, jsonfmt.NewJsonFormatter()),
		}})

	log := func(l *Logger) {
		l.Typed().Str(benchKey, "alice").WithInt("status", 200).
			WithFloat64("ratio", 2.5).WithBool("ok", true).
			WithString("esc", "a\"b").Info("m")
	}
	log(direct)
	log(capture)

	after := func(s string) string {
		i := strings.Index(s, `"level"`)
		if i < 0 {
			t.Fatalf("no level in %q", s)
		}
		return s[i:]
	}
	if got, want := after(directBuf.String()), after(capA.String()); got != want {
		t.Fatalf("direct output differs from capture output.\ndirect:  %q\ncapture: %q", got, want)
	}
}

// TestDirectPath_FallsBackWithMultipleAdapters proves ineligible configs use the
// capture path unchanged.
func TestDirectPath_FallsBackWithMultipleAdapters(t *testing.T) {
	spy1, spy2 := newRawSpy(), newRawSpy()
	l := NewLogger(Config{Component: "t", Level: types.InfoLevel, Adapters: []types.Adapter{spy1, spy2}})

	l.Typed().WithString("k", "v").Info("m")

	if len(spy1.raw) != 0 || len(spy2.raw) != 0 {
		t.Fatal("multi-adapter config must not use WriteRaw")
	}
	if spy1.zeroCalls != 1 || spy2.zeroCalls != 1 {
		t.Fatalf("capture path not used: %d/%d", spy1.zeroCalls, spy2.zeroCalls)
	}
}

// TestZeroAlloc_DirectPath guards the sacred invariant on the new path: a keyed
// + string-keyed typed line through the direct encoder is 0 allocs/op.
func TestZeroAlloc_DirectPath(t *testing.T) {
	spy := newRawSpy()
	spy.raw = nil // avoid the spy's own recording allocation skewing the count
	l := NewLogger(Config{Component: "t", Level: types.InfoLevel, Adapters: []types.Adapter{&noRecordRaw{enc: spy.enc}}})

	allocs := testing.AllocsPerRun(1000, func() {
		l.Typed().Str(benchKey, "alice").WithInt("status", 200).WithBool("ok", true).Info("m")
	})
	if allocs != 0 {
		t.Fatalf("direct path allocates: %v allocs/op (must be 0)", allocs)
	}
}

// noRecordRaw is a raw-capable adapter whose WriteRaw does nothing, so the alloc
// guard measures only the logger's own path.
type noRecordRaw struct{ enc types.DirectFieldEncoder }

func (s *noRecordRaw) Name() string                            { return "noraw" }
func (s *noRecordRaw) Write(*types.LogEntry) error             { return nil }
func (s *noRecordRaw) WriteZero(*types.LogEntry) error         { return nil }
func (s *noRecordRaw) Flush() error                            { return nil }
func (s *noRecordRaw) Close() error                            { return nil }
func (s *noRecordRaw) SetFormatter(types.Formatter)            {}
func (s *noRecordRaw) Health() error                           { return nil }
func (s *noRecordRaw) WriteRaw([]byte) error                   { return nil }
func (s *noRecordRaw) DirectEncoder() types.DirectFieldEncoder { return s.enc }
