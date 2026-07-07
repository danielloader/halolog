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

func fieldKey(name string) *types.FieldKey {
	return &types.FieldKey{Name: name, JSONFragment: jsonfmt.KeyFragment(name)}
}

// TestTypedKeyed_OutputMatchesStringKeyed asserts the pre-declared-key methods
// (Str/Int/Bool) produce exactly the same JSON as the string-key methods.
func TestTypedKeyed_OutputMatchesStringKeyed(t *testing.T) {
	var keyed, str bytes.Buffer
	lk := NewLogger(Config{Component: "b", Level: types.DebugLevel,
		Adapters: []types.Adapter{console.NewWithWriter(&keyed, jsonfmt.NewJsonFormatter())}})
	ls := NewLogger(Config{Component: "b", Level: types.DebugLevel,
		Adapters: []types.Adapter{console.NewWithWriter(&str, jsonfmt.NewJsonFormatter())}})

	uid, cnt, ok := fieldKey("user_id"), fieldKey("count"), fieldKey("ok")
	lk.Typed().Str(uid, "alice").Int(cnt, 42).Bool(ok, true).Info("login")
	ls.Typed().WithString("user_id", "alice").WithInt("count", 42).WithBool("ok", true).Info("login")

	if keyed.String() != str.String() {
		t.Fatalf("keyed vs string output differ:\n keyed=%s str=%s", keyed.String(), str.String())
	}
	if !strings.Contains(keyed.String(), `"user_id":"alice"`) || !strings.Contains(keyed.String(), `"count":42`) {
		t.Fatalf("keyed output missing expected fields: %s", keyed.String())
	}
}

// TestTypedKeyed_ZeroAlloc guards that the pre-declared-key path allocates
// nothing through a real (serializing) adapter.
func TestTypedKeyed_ZeroAlloc(t *testing.T) {
	lg := NewLogger(Config{Component: "b", Level: types.DebugLevel,
		Adapters: []types.Adapter{&benchmarkAdapter{}}})
	uid, cnt := fieldKey("user_id"), fieldKey("count")

	if a := testing.AllocsPerRun(1000, func() {
		lg.Typed().Str(uid, "alice").Int(cnt, 42).Info("login")
	}); a != 0 {
		t.Fatalf("keyed path must be 0 allocs/op, got %.2f", a)
	}
}

// TestTypedKeyed_NilErrIsNoop verifies Err with a nil error adds no field.
func TestTypedKeyed_NilErrIsNoop(t *testing.T) {
	var buf bytes.Buffer
	lg := NewLogger(Config{Component: "b", Level: types.DebugLevel,
		Adapters: []types.Adapter{console.NewWithWriter(&buf, jsonfmt.NewJsonFormatter())}})
	lg.Typed().Err(fieldKey("error"), nil).Info("no error")
	if strings.Contains(buf.String(), `"error"`) {
		t.Fatalf("nil Err should add no field: %s", buf.String())
	}
}
