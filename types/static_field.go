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

/* =====================================================================
   IMMUTABLE FIELD STORAGE (Zero Allocation)
   ===================================================================== */

// StaticField is a stack-allocated field (16 bytes)
type StaticField struct {
	key   [32]byte // Fixed-size key buffer
	value int64    // Numeric values
	str   *string  // String pointer (escapes but rare)
	_type uint8    // Type tag
}

// ConvertTypedFieldToStatic converts a TypedField to a StaticField for zero-allocation storage
func ConvertTypedFieldToStatic(field TypedField) StaticField {
	sf := StaticField{}

	// Copy key (truncate if necessary)
	keyBytes := []byte(field.GetKey())
	if len(keyBytes) > 31 {
		keyBytes = keyBytes[:31]
	}
	copy(sf.key[:], keyBytes)
	sf.key[len(keyBytes)] = 0 // Null terminator

	// Convert value based on type
	switch field.GetType() {
	case TypedFieldString:
		if str, ok := field.GetValue().(string); ok {
			sf.str = &str
			sf._type = uint8(TypedFieldString)
		}
	case TypedFieldInt64:
		if val, ok := field.GetValue().(int64); ok {
			sf.value = val
			sf._type = uint8(TypedFieldInt64)
		} else if val, ok := field.GetValue().(int); ok {
			sf.value = int64(val)
			sf._type = uint8(TypedFieldInt64)
		}
	case TypedFieldFloat64:
		if val, ok := field.GetValue().(float64); ok {
			sf.value = int64(val)
			sf._type = uint8(TypedFieldFloat64)
		}
	case TypedFieldBool:
		if val, ok := field.GetValue().(bool); ok {
			if val {
				sf.value = 1
			}
			sf._type = uint8(TypedFieldBool)
		}
	default:
		sf._type = uint8(TypedFieldString)
	}

	return sf
}

// Accessor methods for StaticField

// Key returns the key as a string
func (sf *StaticField) Key() string {
	// Find null terminator
	keyLen := 0
	for i := 0; i < len(sf.key) && sf.key[i] != 0; i++ {
		keyLen++
	}
	return string(sf.key[:keyLen])
}

// KeyBytes returns the key as a byte slice without allocation
func (sf *StaticField) KeyBytes() []byte {
	keyLen := 0
	for i := 0; i < len(sf.key) && sf.key[i] != 0; i++ {
		keyLen++
	}
	return sf.key[:keyLen]
}

// Type returns the field type
func (sf *StaticField) Type() TypedFieldType {
	return TypedFieldType(sf._type)
}

// StringValue returns the string value if type is string, nil otherwise
func (sf *StaticField) StringValue() *string {
	if sf._type == uint8(TypedFieldString) {
		return sf.str
	}
	return nil
}

// IntValue returns the int64 value if type is int64, 0 otherwise
func (sf *StaticField) IntValue() int64 {
	if sf._type == uint8(TypedFieldInt64) {
		return sf.value
	}
	return 0
}

// UintValue returns the uint64 value if type is uint64, 0 otherwise
func (sf *StaticField) UintValue() uint64 {
	if sf._type == uint8(TypedFieldUint64) {
		return uint64(sf.value)
	}
	return 0
}

// FloatValue returns the float64 value if type is float64, 0.0 otherwise
func (sf *StaticField) FloatValue() float64 {
	if sf._type == uint8(TypedFieldFloat64) {
		return float64(sf.value)
	}
	return 0.0
}

// BoolValue returns the bool value if type is bool, false otherwise
func (sf *StaticField) BoolValue() bool {
	if sf._type == uint8(TypedFieldBool) {
		return sf.value != 0
	}
	return false
}

// FieldSet is a fixed-size field container (escapes to the heap only if it grows
// beyond its stack-backed capacity). Its backing storage is provided externally
// by callers on the zero-allocation path.
type FieldSet struct{}

// ConvertTypedFieldToTypedFieldData converts a TypedField to TypedFieldData for static field storage
func ConvertTypedFieldToTypedFieldData(field TypedField) TypedFieldData {
	return TypedFieldData{
		Key:   field.GetKey(),
		Value: field.GetValue(),
		Type:  field.GetType(),
	}
}
