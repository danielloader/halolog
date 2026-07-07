// @author Admilson B. F. Cossa

package json

import (
	"strconv"
	"testing"

	"github.com/go-gen-ecosystem/halolog/types"
)

// TestAppendIntMatchesStrconv checks the cached small-int fast path renders
// identically to strconv across and beyond the cache boundary.
func TestAppendIntMatchesStrconv(t *testing.T) {
	for _, v := range []int64{0, 1, 9, 10, 99, 100, 200, 255, 404, 511, 512, 1000, -1, -511, 9223372036854775807} {
		if got, want := string(appendInt(nil, v)), strconv.FormatInt(v, 10); got != want {
			t.Errorf("appendInt(%d) = %q, want %q", v, got, want)
		}
	}
}

// TestAppendUintMatchesStrconv checks the unsigned fast path.
func TestAppendUintMatchesStrconv(t *testing.T) {
	for _, v := range []uint64{0, 1, 99, 100, 404, 511, 512, 1000, 18446744073709551615} {
		if got, want := string(appendUint(nil, v)), strconv.FormatUint(v, 10); got != want {
			t.Errorf("appendUint(%d) = %q, want %q", v, got, want)
		}
	}
}

// TestFormatterIntFieldsAcrossBoundary asserts full field output is correct for
// int and uint values on both sides of the cache boundary, typed and via Value.
func TestFormatterIntFieldsAcrossBoundary(t *testing.T) {
	f := NewJsonFormatter()
	for _, v := range []int{0, 42, 100, 404, 511, 512, 8080} {
		entry := &types.LogEntry{
			Level: types.InfoLevel, Message: "m", TimestampUnix: 1_700_000_000_000_000_000,
			StaticFields: []types.TypedFieldData{
				{Key: "typed", Val: types.IntValue(v)},
				{Key: "boxed", Value: v},
				{Key: "u", Value: uint16(v)},
			},
			StaticFieldCount: 3,
		}
		out := string(f.Format(entry, nil))
		exp := `"typed":` + strconv.Itoa(v) + `,"boxed":` + strconv.Itoa(v) + `,"u":` + strconv.Itoa(v)
		if !contains(out, exp) {
			t.Errorf("v=%d: output %q missing %q", v, out, exp)
		}
	}
}
