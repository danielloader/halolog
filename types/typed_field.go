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

// TypedFieldData represents the data structure for typed fields
type TypedFieldData struct {
	Key string
	// Val stores the typed value without allocation.
	// This is the preferred storage mechanism.
	Val FieldValue
	// Value is the legacy interface{} storage.
	// Used for fallback or when Val is KindUnknown.
	Value     interface{}
	Type      TypedFieldType
	Optimized bool
	FieldID   int // Field dictionary ID for O(1) lookups

	// Fluent API fields (High-performance zero-allocation)
	FgColor     Color
	BgColor     Color
	IsSensitive bool
	Style       uint8
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
