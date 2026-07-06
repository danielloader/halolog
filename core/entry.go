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

// Entry represents a log entry optimized for stack allocation.
// For synchronous logging, Entry lives entirely on the stack (0 allocations).
// For async logging, Entry is pooled.
type Entry struct {
	// Hot data (frequently accessed)
	Timestamp int64
	Level     types.LogLevel
	Component string
	Message   string

	// Static field storage (covers 95% of use cases)
	staticFields [16]types.TypedFieldData
	fieldCount   uint8

	// Dynamic fields (only allocated if >16 fields needed)
	dynamicFields []types.TypedFieldData
}

// Reset clears the entry for reuse.
//
//go:inline
func (e *Entry) Reset() {
	e.Timestamp = 0
	e.Level = types.InfoLevel
	e.Component = ""
	e.Message = ""
	e.fieldCount = 0

	// Clear static fields metadata only (keys remain for reuse)
	for i := uint8(0); i < 16; i++ {
		e.staticFields[i].Value = nil
	}

	// Reset dynamic slice but keep capacity
	if e.dynamicFields != nil {
		e.dynamicFields = e.dynamicFields[:0]
	}
}

// AddField adds a field to the entry.
// Uses static storage for first 16 fields (zero allocation).
// Falls back to dynamic slice for additional fields.
//
//go:inline
func (e *Entry) AddField(key string, value interface{}) {
	if e.fieldCount < 16 {
		e.staticFields[e.fieldCount].Key = key
		e.staticFields[e.fieldCount].Value = value
		e.fieldCount++
		return
	}

	// Overflow to dynamic slice (rare case)
	if e.dynamicFields == nil {
		e.dynamicFields = make([]types.TypedFieldData, 0, 8)
	}
	e.dynamicFields = append(e.dynamicFields, types.TypedFieldData{
		Key:   key,
		Value: value,
	})
}

// AddString adds a string field.
//
//go:inline
func (e *Entry) AddString(key string, value string) {
	e.AddField(key, value)
}

// AddInt adds an integer field.
//
//go:inline
func (e *Entry) AddInt(key string, value int) {
	e.AddField(key, value)
}

// AddInt64 adds an int64 field.
//
//go:inline
func (e *Entry) AddInt64(key string, value int64) {
	e.AddField(key, value)
}

// AddFloat64 adds a float64 field.
//
//go:inline
func (e *Entry) AddFloat64(key string, value float64) {
	e.AddField(key, value)
}

// AddBool adds a boolean field.
//
//go:inline
func (e *Entry) AddBool(key string, value bool) {
	e.AddField(key, value)
}

// AddError adds an error field.
//
//go:inline
func (e *Entry) AddError(err error) {
	if err != nil {
		e.AddField("error", err.Error())
	}
}

// FieldCount returns the total number of fields.
//
//go:inline
func (e *Entry) FieldCount() int {
	return int(e.fieldCount) + len(e.dynamicFields)
}

// Fields returns all fields as a slice.
// Note: This allocates - use only when necessary (formatters, etc.)
func (e *Entry) Fields() []types.TypedFieldData {
	total := e.FieldCount()
	if total == 0 {
		return nil
	}

	result := make([]types.TypedFieldData, 0, total)

	// Append static fields
	for i := uint8(0); i < e.fieldCount; i++ {
		result = append(result, e.staticFields[i])
	}

	// Append dynamic fields
	if len(e.dynamicFields) > 0 {
		result = append(result, e.dynamicFields...)
	}

	return result
}

// ForEachField iterates over all fields without allocation.
// Stops iteration if the callback returns false.
//
//go:inline
func (e *Entry) ForEachField(fn func(key string, value interface{}) bool) {
	// Iterate static fields
	for i := uint8(0); i < e.fieldCount; i++ {
		if !fn(e.staticFields[i].Key, e.staticFields[i].Value) {
			return
		}
	}

	// Iterate dynamic fields
	for _, f := range e.dynamicFields {
		if !fn(f.Key, f.Value) {
			return
		}
	}
}

// ToLogEntry converts Entry to types.LogEntry for adapter compatibility.
// This is used when calling adapters that expect the legacy LogEntry type.
//
//go:inline
func (e *Entry) ToLogEntry(target *types.LogEntry) {
	target.TimestampUnix = e.Timestamp
	target.Level = e.Level
	target.Component = e.Component
	target.Message = e.Message

	// Copy fields to target. If the target's buffer is too small (e.g. a
	// freshly declared LogEntry with no backing array), allocate one that fits
	// so no fields are silently dropped.
	target.StaticFieldCount = int(e.fieldCount)
	if target.StaticFieldCount > cap(target.StaticFields) {
		target.StaticFields = make([]types.TypedFieldData, target.StaticFieldCount)
	} else {
		target.StaticFields = target.StaticFields[:target.StaticFieldCount]
	}
	for i := uint8(0); i < e.fieldCount; i++ {
		target.StaticFields[i] = e.staticFields[i]
	}

	// Copy dynamic fields if any
	if len(e.dynamicFields) > 0 {
		if target.Fields == nil {
			target.Fields = make([]types.TypedFieldData, 0, len(e.dynamicFields))
		}
		target.Fields = append(target.Fields[:0], e.dynamicFields...)
	}
}
