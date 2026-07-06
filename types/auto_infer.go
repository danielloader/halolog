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

func inferTypeFromValue(value any) TypedFieldType {
	switch value.(type) {
	case string:
		return TypedFieldString
	case int, int8, int16, int32, int64:
		return TypedFieldInt64
	case uint, uint8, uint16, uint32, uint64:
		return TypedFieldUint64
	case float32, float64:
		return TypedFieldFloat64
	case bool:
		return TypedFieldBool
	case time.Time:
		return TypedFieldTime
	case time.Duration:
		return TypedFieldDuration
	case []byte:
		return TypedFieldBytes
	default:
		return TypedFieldString
	}
}

/* =====================================================================
   AUTOMATIC TYPE INFERENCE - THE MAGIC
   ===================================================================== */

// AutoInferType converts any Go type to types.TypedField automatically
// This is the key to making the API clean and type-safe!
func AutoInferType(key string, value interface{}) *TypedFieldData {
	field := &TypedFieldData{Key: key}

	switch v := value.(type) {
	case string:
		field.Value = v
		field.Type = TypedFieldString

	// All integer types → int64
	case int:
		field.Value = int64(v)
		field.Type = TypedFieldInt64
	case int8:
		field.Value = int64(v)
		field.Type = TypedFieldInt64
	case int16:
		field.Value = int64(v)
		field.Type = TypedFieldInt64
	case int32:
		field.Value = int64(v)
		field.Type = TypedFieldInt64
	case int64:
		field.Value = v
		field.Type = TypedFieldInt64

	// All unsigned integer types → uint64
	case uint:
		field.Value = uint64(v)
		field.Type = TypedFieldUint64
	case uint8:
		field.Value = uint64(v)
		field.Type = TypedFieldUint64
	case uint16:
		field.Value = uint64(v)
		field.Type = TypedFieldUint64
	case uint32:
		field.Value = uint64(v)
		field.Type = TypedFieldUint64
	case uint64:
		field.Value = v
		field.Type = TypedFieldUint64

	// All float types → float64
	case float32:
		field.Value = float64(v)
		field.Type = TypedFieldFloat64
	case float64:
		field.Value = v
		field.Type = TypedFieldFloat64

	case bool:
		field.Value = v
		field.Type = TypedFieldBool

	case time.Time:
		field.Value = v
		field.Type = TypedFieldTime

	case time.Duration:
		field.Value = v
		field.Type = TypedFieldDuration

	case []byte:
		field.Value = v
		field.Type = TypedFieldBytes

	case error:
		if v == nil {
			field.Value = nil
			field.Type = TypedFieldNil
		} else {
			field.Value = v.Error()
			field.Type = TypedFieldError
		}

	case nil:
		field.Value = nil
		field.Type = TypedFieldNil

	default:
		// Fallback: store as-is
		field.Value = v
		field.Type = TypedFieldString
	}

	return field
}
