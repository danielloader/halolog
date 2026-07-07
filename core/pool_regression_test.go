// @author Admilson B. F. Cossa
package core

import (
	"testing"

	"github.com/go-gen-ecosystem/halolog/types"
)

// TestPerPPool_StaticFieldsNotShrunkAcrossReuse guards a pooling regression:
// a dispatch reslices the pooled entry's StaticFields to [:count]; if the pool
// does not restore the full backing buffer on the next borrow, a subsequent
// larger field chain silently drops fields beyond the previous count.
func TestPerPPool_StaticFieldsNotShrunkAcrossReuse(t *testing.T) {
	var got int
	adapter := &types.FuncAdapter{
		WriteFunc: func(e *types.LogEntry) error {
			got = e.StaticFieldCount
			return nil
		},
	}
	logger := New().Adapter(adapter).MustBuild()

	// First a two-field entry (reslices the pooled StaticFields to len 2).
	logger.WithField("a", 1).WithField("b", 2).Info("two")
	if got != 2 {
		t.Fatalf("two-field entry: StaticFieldCount = %d, want 2", got)
	}

	// Reusing the recycled state, a four-field entry must keep all four.
	logger.WithField("a", 1).WithField("b", 2).WithField("c", 3).WithField("d", 4).Info("four")
	if got != 4 {
		t.Fatalf("four-field entry after reuse: StaticFieldCount = %d, want 4 (fields were dropped)", got)
	}
}
