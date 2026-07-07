// @author Admilson B. F. Cossa

package types

import (
	"testing"
	"unsafe"
)

// TestTypedFieldDataSize guards the per-field hot-path record size. A copy of
// TypedFieldData is written into the entry's field array for every logged field,
// so its size is directly on the zero-allocation hot path. The struct was slimmed
// from 144 bytes (it carried presentation/color/style/sensitivity/dictionary
// fields no shipped formatter read) to its hot fields. This test fails if the
// record grows back.
func TestTypedFieldDataSize(t *testing.T) {
	const maxBytes = 104 // Key(16)+KeyDesc(8)+Val(56)+Value(16)+Type(8)
	if got := unsafe.Sizeof(TypedFieldData{}); got > maxBytes {
		t.Fatalf("TypedFieldData grew to %d bytes (limit %d) — the per-field copy is on the 0-alloc hot path; keep it lean", got, maxBytes)
	}
}

// TestFieldValueSize guards the value union size.
func TestFieldValueSize(t *testing.T) {
	const maxBytes = 56 // Kind(8)+Int64(8)+Float64(8)+String(16)+Any(16)
	if got := unsafe.Sizeof(FieldValue{}); got > maxBytes {
		t.Fatalf("FieldValue grew to %d bytes (limit %d)", got, maxBytes)
	}
}
