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

// typedField represents a key-value pair with typed storage locally in core.
type typedField struct {
	Key string
	Val types.FieldValue
}

// TypedFieldSet holds stack-allocated typed fields (no interface{} boxing).
// Size optimized for 2 fields - limits struct size for value copying.
type TypedFieldSet struct {
	n      uint8
	_      [7]byte // padding for alignment
	fields [2]typedField
}

// TypedFieldBuilder provides a typed fluent API that avoids interface{} boxing.
// Use this for maximum performance when field types are known at compile time.
//
// Usage:
//
//	logger.Typed().WithString("user", "alice").WithInt("count", 42).Info("message")
//
// This is faster than the generic With(key, value interface{}) because:
// - No interface{} boxing/unboxing
// - No type switch in serialization
// - Smaller field storage (no type word)
type TypedFieldBuilder struct {
	logger *Logger
	fields TypedFieldSet
	state  *perPState // only used for >4 fields
}

// Typed returns a typed field builder for maximum performance.
//
//go:inline
func (l *Logger) Typed() TypedFieldBuilder {
	return TypedFieldBuilder{logger: l}
}

// WithString adds a string field without interface{} boxing.
//
//go:inline
func (fb TypedFieldBuilder) WithString(key string, value string) TypedFieldBuilder {
	if fb.fields.n < 2 && fb.state == nil {
		fb.fields.fields[fb.fields.n] = typedField{Key: key, Val: types.StringValue(value)}
		fb.fields.n++
		return fb
	}
	return fb.withOverflow(key, types.StringValue(value))
}

// WithInt adds an int field without interface{} boxing.
//
//go:inline
func (fb TypedFieldBuilder) WithInt(key string, value int) TypedFieldBuilder {
	if fb.fields.n < 2 && fb.state == nil {
		fb.fields.fields[fb.fields.n] = typedField{Key: key, Val: types.IntValue(value)}
		fb.fields.n++
		return fb
	}
	return fb.withOverflow(key, types.IntValue(value))
}

// WithInt64 adds an int64 field without interface{} boxing.
//
//go:inline
func (fb TypedFieldBuilder) WithInt64(key string, value int64) TypedFieldBuilder {
	if fb.fields.n < 2 && fb.state == nil {
		fb.fields.fields[fb.fields.n] = typedField{Key: key, Val: types.Int64Value(value)}
		fb.fields.n++
		return fb
	}
	return fb.withOverflow(key, types.Int64Value(value))
}

// WithFloat64 adds a float64 field without interface{} boxing.
//
//go:inline
func (fb TypedFieldBuilder) WithFloat64(key string, value float64) TypedFieldBuilder {
	if fb.fields.n < 2 && fb.state == nil {
		fb.fields.fields[fb.fields.n] = typedField{Key: key, Val: types.Float64Value(value)}
		fb.fields.n++
		return fb
	}
	return fb.withOverflow(key, types.Float64Value(value))
}

// WithBool adds a bool field without interface{} boxing.
//
//go:inline
func (fb TypedFieldBuilder) WithBool(key string, value bool) TypedFieldBuilder {
	if fb.fields.n < 2 && fb.state == nil {
		fb.fields.fields[fb.fields.n] = typedField{Key: key, Val: types.BoolValue(value)}
		fb.fields.n++
		return fb
	}
	return fb.withOverflow(key, types.BoolValue(value))
}

// WithError adds an error field without interface{} boxing.
//
//go:inline
func (fb TypedFieldBuilder) WithError(err error) TypedFieldBuilder {
	if err == nil {
		return fb
	}
	if fb.fields.n < 2 && fb.state == nil {
		fb.fields.fields[fb.fields.n] = typedField{Key: "error", Val: types.ErrorValue(err)}
		fb.fields.n++
		return fb
	}
	return fb.withOverflow("error", types.ErrorValue(err))
}

// withOverflow handles >4 fields by using per-P pool.
func (fb TypedFieldBuilder) withOverflow(key string, val types.FieldValue) TypedFieldBuilder {
	if fb.state == nil {
		fb.state = globalPerPPool.get()
		// Copy existing typed fields to pool state
		for i := uint8(0); i < fb.fields.n; i++ {
			f := &fb.fields.fields[i]
			fb.state.entry.StaticFields[i] = types.TypedFieldData{
				Key: f.Key,
				Val: f.Val,
			}
		}
		fb.state.entry.StaticFieldCount = int(fb.fields.n)
	}

	entry := &fb.state.entry
	n := entry.StaticFieldCount
	if n < len(entry.StaticFields) {
		entry.StaticFields[n] = types.TypedFieldData{Key: key, Val: val}
		entry.StaticFieldCount = n + 1
	}
	return fb
}

// Info logs an info message with typed fields.
//
//go:inline
func (fb TypedFieldBuilder) Info(msg string) {
	if fb.fields.n == 0 && fb.state == nil {
		fb.logger.Info(msg)
		return
	}

	// Typed fast path (1-4 fields, no pool)
	if fb.state == nil {
		fb.dispatchTyped(types.InfoLevel, msg)
		return
	}

	// Pool path (>4 fields)
	fb.dispatchPooled(types.InfoLevel, msg)
}

// Debug logs a debug message with typed fields.
//
//go:inline
func (fb TypedFieldBuilder) Debug(msg string) {
	if fb.fields.n == 0 && fb.state == nil {
		fb.logger.Debug(msg)
		return
	}
	if fb.state == nil {
		fb.dispatchTyped(types.DebugLevel, msg)
		return
	}
	fb.dispatchPooled(types.DebugLevel, msg)
}

// Warn logs a warning message with typed fields.
//
//go:inline
func (fb TypedFieldBuilder) Warn(msg string) {
	if fb.fields.n == 0 && fb.state == nil {
		fb.logger.Warn(msg)
		return
	}
	if fb.state == nil {
		fb.dispatchTyped(types.WarnLevel, msg)
		return
	}
	fb.dispatchPooled(types.WarnLevel, msg)
}

// Error logs an error message with typed fields.
//
//go:inline
func (fb TypedFieldBuilder) Error(msg string) {
	if fb.fields.n == 0 && fb.state == nil {
		fb.logger.Error(msg)
		return
	}
	if fb.state == nil {
		fb.dispatchTyped(types.ErrorLevel, msg)
		return
	}
	fb.dispatchPooled(types.ErrorLevel, msg)
}

// dispatchTyped handles the typed inline-field path (1-4 fields held on the
// builder value, no pool acquired while chaining).
//
//go:inline
func (fb TypedFieldBuilder) dispatchTyped(level types.LogLevel, msg string) {
	// Check level
	if level < fb.logger.Level() {
		return
	}

	// Optimization: Discard adapter ignores entry, so skip construction entirely
	if fb.logger.discardAdapter != nil {
		_ = fb.logger.discardAdapter.WriteZero(nil)
		return
	}

	// Dispatch through a pooled per-P entry. A stack-local entry cannot survive
	// being passed to the adapter's WriteZero interface method without escaping to
	// the heap (one allocation per call); the pooled entry is already heap-resident
	// and reused, so dispatch stays zero-allocation.
	state := globalPerPPool.get()
	entry := &state.entry
	entry.Level = level
	entry.Message = msg
	entry.Component = fb.logger.component
	entry.TimestampUnix = fb.logger.clock.GetNsecValue()

	n := int(fb.fields.n)
	for i := 0; i < n; i++ {
		f := &fb.fields.fields[i]
		entry.StaticFields[i] = types.TypedFieldData{Key: f.Key, Val: f.Val}
	}
	entry.StaticFieldCount = n
	entry.StaticFields = entry.StaticFields[:n]

	if fb.logger.enableMasking && fb.logger.masker != nil {
		fb.logger.masker.Apply(entry)
	}

	if len(fb.logger.adapters) == 1 {
		_ = fb.logger.adapters[0].WriteZero(entry)
	} else {
		for _, a := range fb.logger.adapters {
			_ = a.WriteZero(entry)
		}
	}

	if fb.logger.metrics != nil {
		fb.logger.metrics.counts[level].Add(1)
	}

	globalPerPPool.put(state)
}

// dispatchPooled handles the pool path (>4 fields).
//
//go:inline
func (fb TypedFieldBuilder) dispatchPooled(level types.LogLevel, msg string) {
	if level < fb.logger.Level() {
		globalPerPPool.put(fb.state)
		return
	}

	entry := &fb.state.entry
	entry.Level = level
	entry.Message = msg
	entry.Component = fb.logger.component
	entry.TimestampUnix = fb.logger.clock.GetNsecValue()
	entry.StaticFields = entry.StaticFields[:entry.StaticFieldCount]

	if fb.logger.enableMasking && fb.logger.masker != nil {
		fb.logger.masker.Apply(entry)
	}

	if len(fb.logger.adapters) == 1 {
		_ = fb.logger.adapters[0].WriteZero(entry)
	} else {
		for _, a := range fb.logger.adapters {
			_ = a.WriteZero(entry)
		}
	}

	if fb.logger.metrics != nil {
		fb.logger.metrics.counts[level].Add(1)
	}

	globalPerPPool.put(fb.state)
}
