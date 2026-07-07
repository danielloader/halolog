// @author Admilson B. F. Cossa

package json

import (
	"errors"
	"testing"

	"github.com/go-gen-ecosystem/halolog/types"
)

// goldenEntry builds a fixed LogEntry exercising every field kind and both the
// typed (Val) and legacy (Value) storage paths, plus a pre-declared keyed field
// and an escape-needing string. TimestampUnix is fixed so the output is fully
// deterministic. This is the byte-for-byte safety net that gates the FieldValue
// / TypedFieldData slimming: the JSON output must not change.
func goldenEntry() *types.LogEntry {
	kUser := &types.FieldKey{Name: "user", JSONFragment: KeyFragment("user")}
	sf := []types.TypedFieldData{
		{KeyDesc: kUser, Val: types.StringValue("alice")},         // keyed string
		{Key: "status", Val: types.IntValue(200)},                 // typed int (in small-int cache)
		{Key: "bytes", Val: types.Int64Value(1048576)},            // typed int64 (out of cache)
		{Key: "ratio", Val: types.Float64Value(3.5)},              // typed float
		{Key: "ok", Val: types.BoolValue(true)},                   // typed bool
		{Key: "off", Val: types.BoolValue(false)},                 // typed bool false
		{Key: "err", Val: types.ErrorValue(errors.New("boom"))},   // typed error
		{Key: "any", Val: types.AnyValue(map[string]int{"a": 1})}, // typed any
		{Key: "legacy", Value: "plain"},                           // legacy WithField string
		{Key: "legacynum", Value: 42},                             // legacy WithField int
		{Key: "esc", Val: types.StringValue("a\"b\n\tc")},         // escaping required
	}
	return &types.LogEntry{
		TimestampUnix:    1_600_000_000_000_000_000,
		Level:            types.InfoLevel,
		Message:          "golden message",
		StaticFields:     sf,
		StaticFieldCount: len(sf),
	}
}

// goldenFields is the exact byte sequence the formatter must emit for the field
// portion of goldenEntry() (everything after the message value through the closing
// brace + newline). It is asserted verbatim: any change to FieldValue layout or
// TypedFieldData slimming that alters a single byte here fails the test. The
// timestamp prefix is intentionally excluded because it carries a machine-local
// timezone offset and is not what these optimizations touch.
const goldenFields = `,"user":"alice","status":200,"bytes":1048576,"ratio":3.5,"ok":true,"off":false,"err":"boom","any":"map[a:1]","legacy":"plain","legacynum":42,"esc":"a\"b\n\tc"}` + "\n"

func TestFormatterGolden(t *testing.T) {
	got := string(NewJsonFormatter().Format(goldenEntry(), nil))
	const marker = `"message":"golden message"`
	i := indexOf(got, marker)
	if i < 0 {
		t.Fatalf("message marker not found in output: %q", got)
	}
	fields := got[i+len(marker):]
	if fields != goldenFields {
		t.Fatalf("field rendering changed.\n got: %q\nwant: %q", fields, goldenFields)
	}
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
