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

// TypedField represents a strongly-typed field interface
type TypedField interface {
	GetKey() string
	GetValue() interface{}
	GetType() TypedFieldType
	IsOptimized() bool
}

// TypedFieldData represents the data structure for typed fields.
//
// A copy of this struct is written into the entry's field array on every logged
// field, so its size sits directly on the zero-allocation hot path. It carries
// only what the capture + format paths read: the key (raw or pre-declared), the
// typed value (Val, boxing-free for scalars), and the legacy interface value
// (Value) used by the WithField / auto-inference / masking paths. Presentation
// (color/style), sensitivity, and dictionary-id fields that no shipped formatter
// reads were removed to shrink the per-field copy; field styling was parked out
// of the shipped tree.
type TypedFieldData struct {
	// Key is the raw field name. Empty when KeyDesc carries a pre-declared key.
	Key string
	// KeyDesc, when non-nil, points to a pre-declared FieldKey whose pre-escaped
	// fragment lets a formatter emit the key without escaping it per call. It is
	// nil for the ordinary string-key APIs (WithField/WithString), which the
	// formatter may still serve from its own key-fragment cache.
	KeyDesc *FieldKey
	// Val stores the typed value without interface boxing for scalars.
	// This is the preferred storage mechanism for the typed builder.
	Val FieldValue
	// Value is the legacy interface{} storage, used by WithField, automatic type
	// inference, and masking; the formatter falls back to it when Val is unset.
	Value interface{}
	// Type is the inferred public field type for the legacy interface path.
	Type TypedFieldType
}

// TypedFieldType represents the type of a typed field
type TypedFieldType int

// Typed field type constants
const (
	TypedFieldString TypedFieldType = iota
	TypedFieldInt64
	TypedFieldUint64
	TypedFieldFloat64
	TypedFieldBool
	TypedFieldTime
	TypedFieldDuration
	TypedFieldError
	TypedFieldBytes
	TypedFieldNil
	TypedFieldAny
)

// String returns the string representation of the typed field type
func (t TypedFieldType) String() string {
	switch t {
	case TypedFieldString:
		return "string"
	case TypedFieldInt64:
		return "int64"
	case TypedFieldUint64:
		return "uint64"
	case TypedFieldFloat64:
		return "float64"
	case TypedFieldBool:
		return "bool"
	case TypedFieldTime:
		return "time"
	case TypedFieldDuration:
		return "duration"
	case TypedFieldError:
		return "error"
	case TypedFieldBytes:
		return "bytes"
	case TypedFieldNil:
		return "nil"
	case TypedFieldAny:
		return "any"
	default:
		return "unknown"
	}
}
