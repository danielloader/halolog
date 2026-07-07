// @author Admilson B. F. Cossa

package middleware

import (
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-gen-ecosystem/halolog/types"
)

// recorder is a base adapter that records message|firstFieldValue for each entry
// it receives, so a test can verify what the async adapter actually delivered.
type recorder struct {
	mu   sync.Mutex
	seen []string
}

func (r *recorder) Name() string { return "recorder" }
func (r *recorder) Write(e *types.LogEntry) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	v := ""
	if e.StaticFieldCount > 0 {
		v = e.StaticFields[0].Val.String
	}
	r.seen = append(r.seen, e.Message+"|"+v)
	return nil
}
func (r *recorder) WriteZero(e *types.LogEntry) error { return r.Write(e) }
func (r *recorder) Flush() error                      { return nil }
func (r *recorder) Close() error                      { return nil }
func (r *recorder) SetFormatter(_ types.Formatter)    {}
func (r *recorder) Health() error                     { return nil }

func (r *recorder) snapshot() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.seen...)
}

// TestAsyncAdapter_UseAfterRecycleSafe is the regression guard: after Write
// returns, mutating (recycling) the source entry must not change what the base
// adapter eventually receives, because the async adapter copies the record.
func TestAsyncAdapter_UseAfterRecycleSafe(t *testing.T) {
	rec := &recorder{}
	a := NewAsyncAdapter(rec, &AsyncAdapterOptions{BufferSize: 16, BatchSize: 1, FlushInterval: 5 * time.Millisecond})

	e := &types.LogEntry{
		Level: types.InfoLevel, Message: "login",
		StaticFields:     []types.TypedFieldData{{Key: "user", Val: types.StringValue("alice")}},
		StaticFieldCount: 1,
	}
	if err := a.Write(e); err != nil {
		t.Fatalf("Write: %v", err)
	}
	// Recycle/overwrite the source entry immediately, as the pool would.
	e.StaticFields[0].Val = types.StringValue("MALLORY")
	e.Message = "overwritten"
	e.StaticFieldCount = 0

	_ = a.Close() // drains everything to the base adapter

	got := rec.snapshot()
	if len(got) != 1 {
		t.Fatalf("expected 1 delivered record, got %d: %v", len(got), got)
	}
	if got[0] != "login|alice" {
		t.Fatalf("record corrupted by source recycle: got %q, want %q", got[0], "login|alice")
	}
	if strings.Contains(got[0], "MALLORY") || strings.Contains(got[0], "overwritten") {
		t.Fatalf("leaked recycled state: %q", got[0])
	}
}
