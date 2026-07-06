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
	"time"
)

// Fields creates multiple typed fields from key-value pairs
// Clean, map-like syntax with automatic type inference
//
// Example:
//
//	Fields(
//	    "user_id", 12345,
//	    "name", "John",
//	    "active", true,
//	    "score", 98.5,
//	)

func Fields(pairs ...interface{}) []TypedField {
	if len(pairs)%2 != 0 {
		return nil // Odd number of arguments
	}

	fields := make([]TypedField, 0, len(pairs)/2)

	for i := 0; i < len(pairs); i += 2 {
		key, ok := pairs[i].(string)
		if !ok {
			continue // Skip invalid key
		}

		value := pairs[i+1]
		// autoInferType returns *TypedFieldData, but TypedField is TypedFieldData (value type)
		// We need to dereference the pointer
		fieldPtr := AutoInferType(key, value)
		if fieldPtr != nil {
			fields = append(fields, *fieldPtr)
		}
	}

	return fields
}

// FieldMap creates typed fields from a map with automatic type inference
//
// Example:
//
//	data := map[string]interface{}{
//	    "user_id": 12345,
//	    "name": "John",
//	}
//	fields := FieldMap(data)
func FieldMap(m map[string]interface{}) []TypedField {
	if len(m) == 0 {
		return nil
	}

	fields := make([]TypedField, 0, len(m))
	for key, value := range m {
		// autoInferType returns *TypedFieldData, but TypedField is TypedFieldData (value type)
		// We need to dereference the pointer
		fieldPtr := AutoInferType(key, value)
		if fieldPtr != nil {
			fields = append(fields, *fieldPtr)
		}
	}

	return fields
}

// SF creates a styled field with automatic type inference
// Combines field creation and styling in one fluent API
//
// Example:
//
//	SF("status", "success").Success()
//	SF("amount", 99.99).Color(ColorGreen).SetBold()
func SF(key string, value interface{}) *StyledField {
	return &StyledField{
		Field: AutoInferType(key, value),
	}
}

/* =====================================================================
   EXPLICIT TYPE BUILDERS (OPTIONAL)
   ===================================================================== */

// String creates a string field explicitly
func String(key string, value string) TypedFieldData {
	return TypedFieldData{Key: key, Value: value, Type: TypedFieldString}
}

// Int creates an int64 field
func Int(key string, value int64) TypedFieldData {
	return TypedFieldData{Key: key, Value: value, Type: TypedFieldInt64}
}

// Uint creates a uint64 field
func Uint(key string, value uint64) TypedFieldData {
	return TypedFieldData{Key: key, Value: value, Type: TypedFieldUint64}
}

// Float creates a float64 field
func Float(key string, value float64) TypedFieldData {
	return TypedFieldData{Key: key, Value: value, Type: TypedFieldFloat64}
}

// Bool creates a boolean field
func Bool(key string, value bool) TypedFieldData {
	return TypedFieldData{Key: key, Value: value, Type: TypedFieldBool}
}

// Time creates a time field
func Time(key string, value time.Time) TypedFieldData {
	return TypedFieldData{Key: key, Value: value, Type: TypedFieldTime}
}

// Duration creates a duration field
func Duration(key string, value time.Duration) TypedFieldData {
	return TypedFieldData{Key: key, Value: value, Type: TypedFieldDuration}
}

// Error creates an error field
func Error(key string, value error) TypedFieldData {
	if value == nil {
		return TypedFieldData{Key: key, Value: nil, Type: TypedFieldNil}
	}
	return TypedFieldData{Key: key, Value: value.Error(), Type: TypedFieldError}
}

// Bytes creates a bytes field
func Bytes(key string, value []byte) TypedFieldData {
	return TypedFieldData{Key: key, Value: value, Type: TypedFieldBytes}
}

// Nil creates a nil field
func Nil(key string) TypedFieldData {
	return TypedFieldData{Key: key, Value: nil, Type: TypedFieldNil}
}

/* =====================================================================
   COMMON FIELD SHORTCUTS
   ===================================================================== */

// UserID creates a user_id field (common convention)
func UserID(value interface{}) TypedFieldData {
	fieldPtr := AutoInferType("user_id", value)
	if fieldPtr != nil {
		return *fieldPtr
	}
	return TypedFieldData{}
}

// RequestID creates a request_id field (common convention)
func RequestID(value string) TypedFieldData {
	fieldPtr := AutoInferType("request_id", value)
	if fieldPtr != nil {
		return *fieldPtr
	}
	return TypedFieldData{}
}

// TraceID creates a trace_id field (common convention)
func TraceID(value string) TypedFieldData {
	fieldPtr := AutoInferType("trace_id", value)
	if fieldPtr != nil {
		return *fieldPtr
	}
	return TypedFieldData{}
}

// SpanID creates a span_id field (common convention)
func SpanID(value string) TypedFieldData {
	fieldPtr := AutoInferType("span_id", value)
	if fieldPtr != nil {
		return *fieldPtr
	}
	return TypedFieldData{}
}

// Err creates an error field (common convention)
func Err(value error) TypedFieldData {
	return Error("error", value)
}

// Msg creates a msg field (common in structured logging)
func Msg(value string) TypedFieldData {
	return String("msg", value)
}

// Method creates a method field (for HTTP requests)
func Method(value string) TypedFieldData {
	return String("method", value)
}

// Path creates a path field (for HTTP requests)
func Path(value string) TypedFieldData {
	return String("path", value)
}

// Status creates a status field (for HTTP responses)
func Status(value int) TypedFieldData {
	return Int("status", int64(value))
}

// DurationMs creates a duration_ms field (common metric)
func DurationMs(value int64) TypedFieldData {
	return Int("duration_ms", value)
}
