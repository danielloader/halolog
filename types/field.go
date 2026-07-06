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
// Package types provides core type definitions
// Author: Admilson B. F. Cossa

package types

import (
	"sync"
)

// FieldBuilder provides fluent interface for building fields
type FieldBuilder interface {
	String(key string, value string) FieldBuilder
	Int(key string, value int) FieldBuilder
	Int64(key string, value int64) FieldBuilder
	Uint64(key string, value uint64) FieldBuilder
	Float64(key string, value float64) FieldBuilder
	Bool(key string, value bool) FieldBuilder
	Time(key string, value interface{}) FieldBuilder
	Duration(key string, value interface{}) FieldBuilder
	Error(key string, value error) FieldBuilder
	Any(key string, value interface{}) FieldBuilder
	Build() []TypedField
}

// FieldChain provides zero-allocation field chaining
type FieldChain interface {
	Add(field TypedField) FieldChain
	AddString(key string, value string) FieldChain
	AddInt64(key string, value int64) FieldChain
	AddFloat64(key string, value float64) FieldChain
	AddBool(key string, value bool) FieldChain
	GetFields() []TypedField
}

// FieldBatch provides efficient batch field operations
type FieldBatch interface {
	Add(fields ...TypedField) FieldBatch
	AddFields(fields map[string]interface{}) FieldBatch
	GetFields() []TypedField
	GetFieldMap() map[string]interface{}
	Size() int
}

// FieldPool provides field object pooling for zero-allocation
type FieldPool interface {
}

// FieldBuffer provides zero-allocation field storage
type FieldBuffer struct {
	mu     sync.RWMutex
	keys   []string
	values []interface{}
	keyMap map[string]int // O(1) key to index mapping
	intBuf [24]byte       // Buffer for integer conversions
}

// StyledField represents a field with visual styling
type StyledField struct {
	Field TypedField // Embedded typed field

	// Visual styling
	FgColor   Color
	BgColor   Color
	Bold      bool
	Italic    bool
	Underline bool
	Masked    bool
	MaskIf    bool
	Highlight bool
}

// Color sets the foreground color
func (sf *StyledField) Color(color Color) *StyledField {
	sf.FgColor = color
	return sf
}

// Background sets the background color
func (sf *StyledField) Background(color Color) *StyledField {
	sf.BgColor = color
	return sf
}

// SetBold enables bold text
func (sf *StyledField) SetBold() *StyledField {
	sf.Bold = true
	return sf
}

// SetItalic enables italic text
func (sf *StyledField) SetItalic() *StyledField {
	sf.Italic = true
	return sf
}

// SetUnderline enables underline
func (sf *StyledField) SetUnderline() *StyledField {
	sf.Underline = true
	return sf
}

// SetMasked enables masking
func (sf *StyledField) SetMasked() *StyledField {
	sf.Masked = true
	return sf
}

// SetHighlight enables highlighting
func (sf *StyledField) SetHighlight() *StyledField {
	sf.Highlight = true
	return sf
}

// Info applies info styling (cyan color)
func (sf *StyledField) Info() *StyledField {
	sf.FgColor = ColorCyan
	return sf
}

// Success applies success styling (green color)
func (sf *StyledField) Success() *StyledField {
	sf.FgColor = ColorGreen
	return sf
}

// Warning applies warning styling (yellow color)
func (sf *StyledField) Warning() *StyledField {
	sf.FgColor = ColorYellow
	return sf
}

// Error applies error styling (red color, bold)
func (sf *StyledField) Error() *StyledField {
	sf.FgColor = ColorRed
	sf.Bold = true
	return sf
}

// Critical applies critical styling (bright red on black, bold, underline)
func (sf *StyledField) Critical() *StyledField {
	sf.FgColor = ColorBrightRed
	sf.BgColor = ColorBlack
	sf.Bold = true
	sf.Underline = true
	return sf
}

// Muted applies muted styling (gray color)
func (sf *StyledField) Muted() *StyledField {
	sf.FgColor = ColorGray
	return sf
}

// ToTypedField converts styled field to typed field (for engine compatibility)
func (sf *StyledField) ToTypedField() TypedField {
	return sf.Field
}
