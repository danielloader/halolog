// @author Admilson B. F. Cossa

package json

import (
	"strconv"
	"sync"
	"testing"

	"github.com/go-gen-ecosystem/halolog/types"
)

// TestKeyFragment verifies the pre-rendered `,"key":` fragment, including escaping.
func TestKeyFragment(t *testing.T) {
	cases := []struct{ name, in, want string }{
		{"plain", "user_id", `,"user_id":`},
		{"needs-escape-quote", `a"b`, `,"a\"b":`},
		{"needs-escape-backslash", `a\b`, `,"a\\b":`},
		{"empty", "", `,"":`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := string(KeyFragment(c.in)); got != c.want {
				t.Fatalf("KeyFragment(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

// TestKeyedFieldMatchesStringField asserts that a field emitted via a pre-declared
// FieldKey (KeyDesc) is byte-identical to the same field emitted via the plain
// string key path — the two APIs must never diverge in output.
func TestKeyedFieldMatchesStringField(t *testing.T) {
	f := NewJsonFormatter()

	keyed := &types.LogEntry{
		Level: types.InfoLevel, Message: "m", TimestampUnix: 1_700_000_000_000_000_000,
		StaticFields: []types.TypedFieldData{{
			Key: "user_id", KeyDesc: &types.FieldKey{Name: "user_id", JSONFragment: KeyFragment("user_id")},
			Val: types.StringValue("alice"),
		}},
		StaticFieldCount: 1,
	}
	stringKey := &types.LogEntry{
		Level: types.InfoLevel, Message: "m", TimestampUnix: 1_700_000_000_000_000_000,
		StaticFields:     []types.TypedFieldData{{Key: "user_id", Val: types.StringValue("alice")}},
		StaticFieldCount: 1,
	}

	if a, b := string(f.Format(keyed, nil)), string(f.Format(stringKey, nil)); a != b {
		t.Fatalf("keyed vs string-key output differ:\n keyed=%s\n strkey=%s", a, b)
	}
}

// TestStringKeyCacheIsCorrectAndStable exercises the transparent auto-cache: the
// same string key formatted repeatedly must always render identically (first call
// escapes+caches, later calls hit the cache).
func TestStringKeyCacheIsCorrectAndStable(t *testing.T) {
	f := NewJsonFormatter()
	entry := &types.LogEntry{
		Level: types.InfoLevel, Message: "m", TimestampUnix: 1_700_000_000_000_000_000,
		StaticFields:     []types.TypedFieldData{{Key: "svc", Val: types.StringValue("auth")}},
		StaticFieldCount: 1,
	}
	first := string(f.Format(entry, nil))
	for i := 0; i < 5; i++ {
		if got := string(f.Format(entry, nil)); got != first {
			t.Fatalf("call %d rendered %q, want stable %q", i, got, first)
		}
	}
	if want := `"svc":"auth"`; !contains(first, want) {
		t.Fatalf("output %q missing %q", first, want)
	}
}

// TestZeroAllocKeyPaths guards that both the pre-declared-key and the cached
// string-key formatting paths allocate nothing once warm.
func TestZeroAllocKeyPaths(t *testing.T) {
	f := NewJsonFormatter()
	key := &types.FieldKey{Name: "user_id", JSONFragment: KeyFragment("user_id")}
	keyed := &types.LogEntry{
		Level: types.InfoLevel, Message: "m", TimestampUnix: 1_700_000_000_000_000_000,
		StaticFields:     []types.TypedFieldData{{Key: "user_id", KeyDesc: key, Val: types.StringValue("alice")}},
		StaticFieldCount: 1,
	}
	strKey := &types.LogEntry{
		Level: types.InfoLevel, Message: "m", TimestampUnix: 1_700_000_000_000_000_000,
		StaticFields:     []types.TypedFieldData{{Key: "user_id", Val: types.StringValue("alice")}},
		StaticFieldCount: 1,
	}
	dst := make([]byte, 0, 256)
	_ = f.Format(strKey, dst) // warm the cache

	if a := testing.AllocsPerRun(1000, func() { _ = f.Format(keyed, dst[:0]) }); a != 0 {
		t.Fatalf("pre-declared key path: %.2f allocs/op, want 0", a)
	}
	if a := testing.AllocsPerRun(1000, func() { _ = f.Format(strKey, dst[:0]) }); a != 0 {
		t.Fatalf("cached string-key path: %.2f allocs/op, want 0", a)
	}
}

// TestKeyFragmentCacheConcurrent hammers the copy-on-write cache from many
// goroutines with overlapping and distinct keys; run with -race to prove the
// atomic-load reads and copy-on-write inserts are data-race free and that every
// key resolves to the correct fragment.
func TestKeyFragmentCacheConcurrent(t *testing.T) {
	const goroutines = 32
	keys := make([]string, 200)
	for i := range keys {
		keys[i] = "field_" + strconv.Itoa(i)
	}
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for g := 0; g < goroutines; g++ {
		go func(seed int) {
			defer wg.Done()
			for i := 0; i < len(keys); i++ {
				k := keys[(i+seed)%len(keys)]
				got := string(cachedKeyFragment(k))
				want := `,"` + k + `":`
				if got != want {
					t.Errorf("cachedKeyFragment(%q) = %q, want %q", k, got, want)
					return
				}
			}
		}(g)
	}
	wg.Wait()
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
