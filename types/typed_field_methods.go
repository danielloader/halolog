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

// GetKey returns the field key
func (f TypedFieldData) GetKey() string {
	return f.Key
}

// GetValue returns the field value
func (f TypedFieldData) GetValue() interface{} {
	return f.Value
}

// GetType returns the field type
func (f TypedFieldData) GetType() TypedFieldType {
	return f.Type
}

// IsOptimized returns true if the field is stored in optimized storage
func (f TypedFieldData) IsOptimized() bool {
	return f.Optimized
}

// Accessor methods for formatters (pointer receivers for compatibility with StaticField)

// IntValue returns the value as int64 (for TypedFieldInt64, TypedFieldTime, TypedFieldDuration)
func (tfd *TypedFieldData) IntValue() int64 {
	if v, ok := tfd.Value.(int64); ok {
		return v
	}
	if v, ok := tfd.Value.(int); ok {
		return int64(v)
	}
	return 0
}

// UintValue returns the value as uint64 (for TypedFieldUint64)
func (tfd *TypedFieldData) UintValue() uint64 {
	if v, ok := tfd.Value.(uint64); ok {
		return v
	}
	if v, ok := tfd.Value.(uint); ok {
		return uint64(v)
	}
	return 0
}

// FloatValue returns the value as float64 (for TypedFieldFloat64)
func (tfd *TypedFieldData) FloatValue() float64 {
	if v, ok := tfd.Value.(float64); ok {
		return v
	}
	if v, ok := tfd.Value.(float32); ok {
		return float64(v)
	}
	return 0.0
}

// BoolValue returns the value as bool (for TypedFieldBool)
func (tfd *TypedFieldData) BoolValue() bool {
	if v, ok := tfd.Value.(bool); ok {
		return v
	}
	return false
}

// StringValue returns the value as *string (for TypedFieldString, TypedFieldError, TypedFieldBytes)
func (tfd *TypedFieldData) StringValue() *string {
	if v, ok := tfd.Value.(string); ok {
		return &v
	}
	if v, ok := tfd.Value.(error); ok {
		s := v.Error()
		return &s
	}
	if v, ok := tfd.Value.([]byte); ok {
		s := string(v)
		return &s
	}
	return nil
}
