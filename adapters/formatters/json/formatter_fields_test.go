// @author Admilson B. F. Cossa

package json

import (
	"strings"
	"testing"

	"github.com/go-gen-ecosystem/halolog/types"
)

// TestFormat_EmitsStaticFieldsAndTimestamp guards a formatter defect where JSON
// output dropped every WithField field (only the dynamic Fields overflow slice
// was rendered) and printed a zero timestamp (it read entry.Timestamp instead of
// the hot-path entry.TimestampUnix). It also guards value typing: numbers and
// bools must be unquoted, strings quoted.
func TestFormat_EmitsStaticFieldsAndTimestamp(t *testing.T) {
	entry := &types.LogEntry{
		Level:            types.InfoLevel,
		Message:          "hello",
		TimestampUnix:    1_700_000_000_000_000_000, // 2023-11-14T…, unix nanos
		StaticFields:     make([]types.TypedFieldData, 4),
		StaticFieldCount: 4,
	}
	// Legacy interface-based (WithField) values.
	entry.StaticFields[0] = types.TypedFieldData{Key: "svc", Value: "auth"}
	entry.StaticFields[1] = types.TypedFieldData{Key: "count", Value: 42}
	entry.StaticFields[2] = types.TypedFieldData{Key: "ok", Value: true}
	// Typed (boxing-free) value.
	entry.StaticFields[3] = types.TypedFieldData{Key: "ratio", Val: types.Float64Value(1.5)}

	out := string(NewJsonFormatter().Format(entry, nil))

	checks := []string{
		`"message":"hello"`,
		`"svc":"auth"`, // string quoted
		`"count":42`,   // int unquoted
		`"ok":true`,    // bool unquoted
		`"ratio":1.5`,  // typed float unquoted
	}
	for _, c := range checks {
		if !strings.Contains(out, c) {
			t.Errorf("JSON output missing %s\n  got: %s", c, out)
		}
	}
	// A real timestamp — never the zero-value year 0001.
	if strings.Contains(out, "0001-01-01") || !strings.Contains(out, `"time":"20`) {
		t.Errorf("timestamp not rendered from TimestampUnix: %s", out)
	}
}
