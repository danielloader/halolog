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

// Package core provides the high-performance HaloLog logger implementation.
// Author: Admilson B. F. Cossa

package core

import (
	"github.com/go-gen-ecosystem/halolog/types"
)

// FieldStyler provides O(1) field styling operations with zero allocations.
// All operations modify the Entry's fields in-place.
type FieldStyler struct{}

// NewFieldStyler creates a new field styler.
func NewFieldStyler() *FieldStyler {
	return &FieldStyler{}
}

// Color applies foreground color to the last field in the entry.
//
//go:inline
func (fs *FieldStyler) Color(entry *Entry, color types.Color) {
	if entry == nil || entry.fieldCount == 0 {
		return
	}
	n := entry.fieldCount - 1
	entry.staticFields[n].FgColor = color
}

// BgColor applies background color to the last field in the entry.
//
//go:inline
func (fs *FieldStyler) BgColor(entry *Entry, color types.Color) {
	if entry == nil || entry.fieldCount == 0 {
		return
	}
	n := entry.fieldCount - 1
	entry.staticFields[n].BgColor = color
}

// Bold applies bold style to the last field in the entry.
//
//go:inline
func (fs *FieldStyler) Bold(entry *Entry) {
	if entry == nil || entry.fieldCount == 0 {
		return
	}
	n := entry.fieldCount - 1
	entry.staticFields[n].Style |= types.StyleBold
}

// Italic applies italic style to the last field in the entry.
//
//go:inline
func (fs *FieldStyler) Italic(entry *Entry) {
	if entry == nil || entry.fieldCount == 0 {
		return
	}
	n := entry.fieldCount - 1
	entry.staticFields[n].Style |= types.StyleItalic
}

// Highlight applies highlight style to the last field in the entry.
//
//go:inline
func (fs *FieldStyler) Highlight(entry *Entry) {
	if entry == nil || entry.fieldCount == 0 {
		return
	}
	n := entry.fieldCount - 1
	entry.staticFields[n].Style |= types.StyleHighlight
}

// Mask marks the last field as sensitive for masking operations.
//
//go:inline
func (fs *FieldStyler) Mask(entry *Entry) {
	if entry == nil || entry.fieldCount == 0 {
		return
	}
	n := entry.fieldCount - 1
	entry.staticFields[n].IsSensitive = true
}

// SetColorAt sets foreground color for a field by index.
//
//go:inline
func (fs *FieldStyler) SetColorAt(entry *Entry, index int, color types.Color) {
	if entry == nil || index < 0 || index >= int(entry.fieldCount) {
		return
	}
	entry.staticFields[index].FgColor = color
}

// SetSensitiveAt marks a field as sensitive by index.
//
//go:inline
func (fs *FieldStyler) SetSensitiveAt(entry *Entry, index int) {
	if entry == nil || index < 0 || index >= int(entry.fieldCount) {
		return
	}
	entry.staticFields[index].IsSensitive = true
}

// GetStyle returns the style of a field by index.
//
//go:inline
func (fs *FieldStyler) GetStyle(entry *Entry, index int) uint8 {
	if entry == nil || index < 0 || index >= int(entry.fieldCount) {
		return 0
	}
	return entry.staticFields[index].Style
}

// IsSensitive checks if a field is sensitive by index.
//
//go:inline
func (fs *FieldStyler) IsSensitive(entry *Entry, index int) bool {
	if entry == nil || index < 0 || index >= int(entry.fieldCount) {
		return false
	}
	return entry.staticFields[index].IsSensitive
}

// ClearStyles removes all styles from all fields.
func (fs *FieldStyler) ClearStyles(entry *Entry) {
	if entry == nil {
		return
	}
	for i := uint8(0); i < entry.fieldCount; i++ {
		entry.staticFields[i].Style = 0
		entry.staticFields[i].FgColor = 0
		entry.staticFields[i].BgColor = 0
	}
}
