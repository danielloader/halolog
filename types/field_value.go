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

// FieldKind identifies the type of a field value without interface{} boxing.
type FieldKind uint8

// Field kind identifiers used to store typed field values without interface{} boxing.
const (
	KindUnknown FieldKind = iota
	KindString
	KindInt
	KindInt64
	KindFloat64
	KindBool
	KindError
	KindAny // fallback for interface{}
)

// FieldValue stores a typed field value without interface{} boxing.
// This eliminates the type word + data word overhead of interface{}.
type FieldValue struct {
	Kind    FieldKind
	Int64   int64
	Float64 float64
	String  string
	Any     interface{} // only used for KindAny
}

// StringValue creates a string FieldValue.
//
//go:inline
func StringValue(s string) FieldValue {
	return FieldValue{Kind: KindString, String: s}
}

// IntValue creates an int FieldValue.
//
//go:inline
func IntValue(i int) FieldValue {
	return FieldValue{Kind: KindInt, Int64: int64(i)}
}

// Int64Value creates an int64 FieldValue.
//
//go:inline
func Int64Value(i int64) FieldValue {
	return FieldValue{Kind: KindInt64, Int64: i}
}

// Float64Value creates a float64 FieldValue.
//
//go:inline
func Float64Value(f float64) FieldValue {
	return FieldValue{Kind: KindFloat64, Float64: f}
}

// BoolValue creates a bool FieldValue.
//
//go:inline
func BoolValue(b bool) FieldValue {
	if b {
		return FieldValue{Kind: KindBool, Int64: 1}
	}
	return FieldValue{Kind: KindBool, Int64: 0}
}

// ErrorValue creates an error FieldValue.
//
//go:inline
func ErrorValue(err error) FieldValue {
	if err == nil {
		return FieldValue{Kind: KindError}
	}
	return FieldValue{Kind: KindError, String: err.Error()}
}

// AnyValue creates an interface{} FieldValue (fallback).
//
//go:inline
func AnyValue(v interface{}) FieldValue {
	return FieldValue{Kind: KindAny, Any: v}
}

// ToInterface converts FieldValue to interface{} for adapter compatibility.
//
//go:inline
func (v FieldValue) ToInterface() interface{} {
	switch v.Kind {
	case KindString:
		return v.String
	case KindInt:
		return int(v.Int64)
	case KindInt64:
		return v.Int64
	case KindFloat64:
		return v.Float64
	case KindBool:
		return v.Int64 != 0
	case KindError:
		return v.String
	case KindAny:
		return v.Any
	default:
		return nil
	}
}
