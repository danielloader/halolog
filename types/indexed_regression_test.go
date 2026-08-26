// @author Admilson B. F. Cossa
package types

import (
	"testing"
	"time"
)

// TestAddIndexedField_DoesNotHang guards the fix for a hang: with no field
// dictionary, field IDs were hash-derived (~4 billion), so the chunk-growth loop
// tried to allocate billions of chunks and never returned. IDs are now dense.
func TestAddIndexedField_DoesNotHang(t *testing.T) {
	done := make(chan struct{})
	go func() {
		e := &LogEntry{}
		for i := 0; i < 100; i++ {
			e.AddIndexedField("field", "value")
			e.AddIndexedField("other", "value2")
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("AddIndexedField hung (unbounded chunk growth)")
	}
}

// TestFieldBuffer_FromMapThenGet guards the fix for a stale key index: FromMap
// rebuilt keys/values but not keyMap, so Get returned (nil,false) afterwards.
func TestFieldBuffer_FromMapThenGet(t *testing.T) {
	fb := &FieldBuffer{}
	fb.FromMap(map[string]interface{}{"alpha": 1, "beta": "two"})

	if v, ok := fb.Get("alpha"); !ok {
		t.Errorf("Get(alpha) = (%v, %v), want found", v, ok)
	}
	if v, ok := fb.Get("beta"); !ok {
		t.Errorf("Get(beta) = (%v, %v), want found", v, ok)
	}
	if _, ok := fb.Get("missing"); ok {
		t.Error("Get(missing) should not be found")
	}
}

// NOTE: IndexedFieldStore's fast-path presence bitmask now uses an
// atomic OR instead of atomic.Add (which corrupted the mask when a field was
// re-set). A full Set/Get round-trip is deliberately not asserted here: the
// store's Get path has separate pre-existing defects (it does not resolve set
// values), and the store is unused by the shipped logger — it is retained as an
// experimental component and slated for repair in a later release.
