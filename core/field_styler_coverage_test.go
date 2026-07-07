// Copyright 2025 Admilson B. F. Cossa
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package core coverage tests for FieldStyler.
// @author Admilson B. F. Cossa

package core

import (
	"testing"

	"github.com/go-gen-ecosystem/halolog/types"
)

// styledEntry builds an Entry with n string fields for styler tests.
func styledEntry(n int) *Entry {
	e := &Entry{}
	for i := 0; i < n; i++ {
		e.AddString("k", "v")
	}
	return e
}

// TestFieldStyler_LastFieldOps covers Color/BgColor/Bold/Italic/Highlight/Mask
// which all operate on the last field.
func TestFieldStyler_LastFieldOps(t *testing.T) {
	fs := NewFieldStyler()
	e := styledEntry(3)

	fs.Color(e, types.Color(2))
	fs.BgColor(e, types.Color(5))
	fs.Bold(e)
	fs.Italic(e)
	fs.Highlight(e)
	fs.Mask(e)

	last := e.staticFields[e.fieldCount-1]
	if last.FgColor != types.Color(2) {
		t.Errorf("FgColor = %v, want 2", last.FgColor)
	}
	if last.BgColor != types.Color(5) {
		t.Errorf("BgColor = %v, want 5", last.BgColor)
	}
	wantStyle := types.StyleBold | types.StyleItalic | types.StyleHighlight
	if last.Style != wantStyle {
		t.Errorf("Style = %08b, want %08b", last.Style, wantStyle)
	}
	if !last.IsSensitive {
		t.Error("expected last field to be marked sensitive")
	}

	// Earlier fields must remain untouched.
	if e.staticFields[0].Style != 0 || e.staticFields[0].FgColor != 0 {
		t.Error("non-last field was mutated")
	}
}

// TestFieldStyler_IndexOps covers SetColorAt/SetSensitiveAt/GetStyle/IsSensitive.
func TestFieldStyler_IndexOps(t *testing.T) {
	fs := NewFieldStyler()
	e := styledEntry(3)

	fs.SetColorAt(e, 1, types.Color(9))
	fs.SetSensitiveAt(e, 1)
	fs.Bold(e) // sets style on last (index 2)

	if e.staticFields[1].FgColor != types.Color(9) {
		t.Errorf("index-1 FgColor = %v, want 9", e.staticFields[1].FgColor)
	}
	if !fs.IsSensitive(e, 1) {
		t.Error("index 1 should be sensitive")
	}
	if fs.IsSensitive(e, 0) {
		t.Error("index 0 should not be sensitive")
	}
	if fs.GetStyle(e, 2) != types.StyleBold {
		t.Errorf("index-2 style = %08b, want bold", fs.GetStyle(e, 2))
	}
	if fs.GetStyle(e, 0) != 0 {
		t.Errorf("index-0 style = %08b, want 0", fs.GetStyle(e, 0))
	}
}

// TestFieldStyler_ClearStyles zeroes style + colors across all fields.
func TestFieldStyler_ClearStyles(t *testing.T) {
	fs := NewFieldStyler()
	e := styledEntry(2)
	fs.SetColorAt(e, 0, types.Color(3))
	fs.Bold(e)
	fs.Highlight(e)

	fs.ClearStyles(e)

	for i := uint8(0); i < e.fieldCount; i++ {
		f := e.staticFields[i]
		if f.Style != 0 || f.FgColor != 0 || f.BgColor != 0 {
			t.Errorf("field %d not cleared: %+v", i, f)
		}
	}
}

// TestFieldStyler_GuardsNilAndBounds covers the defensive early-returns for
// nil entries, empty entries, and out-of-range indices.
func TestFieldStyler_GuardsNilAndBounds(t *testing.T) {
	fs := NewFieldStyler()

	// nil entry — every method must be a safe no-op.
	fs.Color(nil, types.Color(1))
	fs.BgColor(nil, types.Color(1))
	fs.Bold(nil)
	fs.Italic(nil)
	fs.Highlight(nil)
	fs.Mask(nil)
	fs.SetColorAt(nil, 0, types.Color(1))
	fs.SetSensitiveAt(nil, 0)
	fs.ClearStyles(nil)
	if fs.GetStyle(nil, 0) != 0 {
		t.Error("GetStyle(nil) should be 0")
	}
	if fs.IsSensitive(nil, 0) {
		t.Error("IsSensitive(nil) should be false")
	}

	// Empty entry — last-field ops must early-return (fieldCount == 0).
	empty := &Entry{}
	fs.Color(empty, types.Color(1))
	fs.Bold(empty)
	fs.Mask(empty)

	// Out-of-range indices on a populated entry.
	e := styledEntry(2)
	fs.SetColorAt(e, -1, types.Color(1))
	fs.SetColorAt(e, 99, types.Color(1))
	fs.SetSensitiveAt(e, 99)
	if fs.GetStyle(e, 99) != 0 {
		t.Error("GetStyle out-of-range should be 0")
	}
	if fs.IsSensitive(e, -1) {
		t.Error("IsSensitive out-of-range should be false")
	}
	// Nothing should have changed.
	if e.staticFields[0].FgColor != 0 || e.staticFields[1].IsSensitive {
		t.Error("out-of-range ops mutated the entry")
	}
}
