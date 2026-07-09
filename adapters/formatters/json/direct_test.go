// @author Admilson B. F. Cossa

package json

import (
	"errors"
	"testing"

	"github.com/go-gen-ecosystem/halolog/types"
)

// TestDirectMatchesFormat is the oracle for the direct-append fast path: encoding
// a log line field-by-field with the Append* helpers must be byte-for-byte
// identical to capturing the same fields and running Formatter.Format. It renders
// the exact fields of goldenEntry() both ways and compares. If they ever diverge,
// the fast path would emit different bytes than the normal path — a correctness
// bug this test forbids.
func TestDirectMatchesFormat(t *testing.T) {
	entry := goldenEntry()

	// Capture path.
	want := NewJsonFormatter().Format(entry, nil)

	// Direct path: same fields, appended one at a time. The two legacy
	// interface-valued fields are passed as AnyValue — the direct path serves
	// only the typed builder, and KindAny renders through the same appendAny the
	// capture path uses for legacy values, so the bytes must still match.
	kUser := &types.FieldKey{Name: "user", JSONFragment: KeyFragment("user")}
	var got []byte
	got = AppendHeader(got, entry.TimestampUnix, entry.Level, entry.Message)
	got = AppendField(got, kUser, "user", types.StringValue("alice"))
	got = AppendField(got, nil, "status", types.IntValue(200))
	got = AppendField(got, nil, "bytes", types.Int64Value(1048576))
	got = AppendField(got, nil, "ratio", types.Float64Value(3.5))
	got = AppendField(got, nil, "ok", types.BoolValue(true))
	got = AppendField(got, nil, "off", types.BoolValue(false))
	got = AppendField(got, nil, "err", types.ErrorValue(errors.New("boom")))
	got = AppendField(got, nil, "any", types.AnyValue(map[string]int{"a": 1}))
	got = AppendField(got, nil, "legacy", types.AnyValue("plain"))
	got = AppendField(got, nil, "legacynum", types.AnyValue(42))
	got = AppendField(got, nil, "esc", types.StringValue("a\"b\n\tc"))
	got = AppendCloser(got)

	if string(got) != string(want) {
		t.Fatalf("direct output differs from format output.\ndirect: %q\nformat: %q", got, want)
	}
}
