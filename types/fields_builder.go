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
	"strconv"
)

// FieldsBuilder - zero-allocation field builder for callback pattern
type FieldsBuilder struct {
	keys   []string
	values []any
}

// NewFieldsBuilder creates a new field builder
func NewFieldsBuilder() *FieldsBuilder {
	return &FieldsBuilder{
		keys:   make([]string, 0, 8),
		values: make([]any, 0, 8),
	}
}

// AddInt adds an integer field
func (fb *FieldsBuilder) AddInt(key string, value int) *FieldsBuilder {
	fb.keys = append(fb.keys, key)
	fb.values = append(fb.values, value)
	return fb
}

// AddInt64 adds an int64 field
func (fb *FieldsBuilder) AddInt64(key string, value int64) *FieldsBuilder {
	fb.keys = append(fb.keys, key)
	fb.values = append(fb.values, strconv.FormatInt(value, 10))
	return fb
}

// AddFloat64 adds a float64 field
func (fb *FieldsBuilder) AddFloat64(key string, value float64) *FieldsBuilder {
	fb.keys = append(fb.keys, key)
	fb.values = append(fb.values, strconv.FormatFloat(value, 'f', 6, 64))
	return fb
}

// AddBool adds a boolean field
func (fb *FieldsBuilder) AddBool(key string, value bool) *FieldsBuilder {
	fb.keys = append(fb.keys, key)
	fb.values = append(fb.values, strconv.FormatBool(value))
	return fb
}

// AddString adds a string field
func (fb *FieldsBuilder) AddString(key string, value string) *FieldsBuilder {
	fb.keys = append(fb.keys, key)
	fb.values = append(fb.values, value)
	return fb
}

// Build creates a slice of TypedField
func (fb *FieldsBuilder) Build() []TypedFieldData {
	fields := make([]TypedFieldData, len(fb.keys))
	for i := range fb.keys {
		fields[i] = TypedFieldData{Key: fb.keys[i], Value: fb.values[i]}
	}
	return fields
}
