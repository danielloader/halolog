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

// TypedFieldBuilder provides a typed fluent API that avoids interface{} boxing.
// Use it when field types are known at compile time.
//
// Usage:
//
//	logger.Typed().WithString("user", "alice").WithInt("count", 42).Info("message")
//
// Like FieldBuilder it is a VALUE TYPE, but it holds only two pointers (logger
// and the pooled per-P state), so chaining copies almost nothing. Fields are
// written straight into the pooled entry as boxing-free FieldValues, so the
// serializer can render them without an interface type switch. Contrast with the
// generic FieldBuilder.WithField, which stores an interface{} value.
type TypedFieldBuilder struct {
	logger *Logger
	state  *perPState // per-P pooled state (acquired on first field)
}

// Typed returns a typed field builder for boxing-free structured logging.
//
//go:inline
func (l *Logger) Typed() TypedFieldBuilder {
	return TypedFieldBuilder{logger: l}
}

// withTyped acquires the pooled state on the first field and writes the typed
// value directly into it. Reslicing the fixed static buffer and appending to the
// overflow slice allocate nothing, so the hot path stays zero-allocation.
//
//go:inline
func (fb TypedFieldBuilder) withTyped(key string, val types.FieldValue) TypedFieldBuilder {
	if fb.state == nil {
		fb.state = globalPerPPool.get()
	}
	entry := &fb.state.entry
	n := entry.StaticFieldCount
	if n < len(entry.StaticFields) {
		entry.StaticFields[n] = types.TypedFieldData{Key: key, Val: val}
		entry.StaticFieldCount = n + 1
	} else {
		entry.Fields = append(entry.Fields, types.TypedFieldData{Key: key, Val: val})
	}
	return fb
}

// WithString adds a string field without interface{} boxing.
//
//go:inline
func (fb TypedFieldBuilder) WithString(key string, value string) TypedFieldBuilder {
	return fb.withTyped(key, types.StringValue(value))
}

// WithInt adds an int field without interface{} boxing.
//
//go:inline
func (fb TypedFieldBuilder) WithInt(key string, value int) TypedFieldBuilder {
	return fb.withTyped(key, types.IntValue(value))
}

// WithInt64 adds an int64 field without interface{} boxing.
//
//go:inline
func (fb TypedFieldBuilder) WithInt64(key string, value int64) TypedFieldBuilder {
	return fb.withTyped(key, types.Int64Value(value))
}

// WithFloat64 adds a float64 field without interface{} boxing.
//
//go:inline
func (fb TypedFieldBuilder) WithFloat64(key string, value float64) TypedFieldBuilder {
	return fb.withTyped(key, types.Float64Value(value))
}

// WithBool adds a bool field without interface{} boxing.
//
//go:inline
func (fb TypedFieldBuilder) WithBool(key string, value bool) TypedFieldBuilder {
	return fb.withTyped(key, types.BoolValue(value))
}

// WithError adds an error field without interface{} boxing. A nil error is a
// no-op.
//
//go:inline
func (fb TypedFieldBuilder) WithError(err error) TypedFieldBuilder {
	if err == nil {
		return fb
	}
	return fb.withTyped("error", types.ErrorValue(err))
}

// withKeyed writes a field under a pre-declared key, carrying the key's
// pre-escaped fragment so the formatter emits it without escaping. Mirrors
// withTyped but stores the FieldKey descriptor.
//
//go:inline
func (fb TypedFieldBuilder) withKeyed(key *types.FieldKey, val types.FieldValue) TypedFieldBuilder {
	if fb.state == nil {
		fb.state = globalPerPPool.get()
	}
	entry := &fb.state.entry
	n := entry.StaticFieldCount
	if n < len(entry.StaticFields) {
		entry.StaticFields[n] = types.TypedFieldData{Key: key.Name, KeyDesc: key, Val: val}
		entry.StaticFieldCount = n + 1
	} else {
		entry.Fields = append(entry.Fields, types.TypedFieldData{Key: key.Name, KeyDesc: key, Val: val})
	}
	return fb
}

// Str adds a string field under a pre-declared key (fastest path — the key is
// never escaped at log time). Pair with a package-level key from Key(...).
//
//go:inline
func (fb TypedFieldBuilder) Str(key *types.FieldKey, value string) TypedFieldBuilder {
	return fb.withKeyed(key, types.StringValue(value))
}

// Int adds an int field under a pre-declared key.
//
//go:inline
func (fb TypedFieldBuilder) Int(key *types.FieldKey, value int) TypedFieldBuilder {
	return fb.withKeyed(key, types.IntValue(value))
}

// Int64 adds an int64 field under a pre-declared key.
//
//go:inline
func (fb TypedFieldBuilder) Int64(key *types.FieldKey, value int64) TypedFieldBuilder {
	return fb.withKeyed(key, types.Int64Value(value))
}

// Float64 adds a float64 field under a pre-declared key.
//
//go:inline
func (fb TypedFieldBuilder) Float64(key *types.FieldKey, value float64) TypedFieldBuilder {
	return fb.withKeyed(key, types.Float64Value(value))
}

// Bool adds a bool field under a pre-declared key.
//
//go:inline
func (fb TypedFieldBuilder) Bool(key *types.FieldKey, value bool) TypedFieldBuilder {
	return fb.withKeyed(key, types.BoolValue(value))
}

// Err adds an error field under a pre-declared key. A nil error is a no-op.
//
//go:inline
func (fb TypedFieldBuilder) Err(key *types.FieldKey, err error) TypedFieldBuilder {
	if err == nil {
		return fb
	}
	return fb.withKeyed(key, types.ErrorValue(err))
}

// Any adds an arbitrary value under a pre-declared key. Prefer the typed methods
// on the hot path; Any is a convenience for values whose type is not known ahead
// of time.
//
//go:inline
func (fb TypedFieldBuilder) Any(key *types.FieldKey, value interface{}) TypedFieldBuilder {
	return fb.withKeyed(key, types.AnyValue(value))
}

// Info logs an info message with the accumulated typed fields.
//
//go:inline
func (fb TypedFieldBuilder) Info(msg string) {
	if fb.state == nil {
		fb.logger.Info(msg)
		return
	}
	fb.dispatch(types.InfoLevel, msg)
}

// Debug logs a debug message with the accumulated typed fields.
//
//go:inline
func (fb TypedFieldBuilder) Debug(msg string) {
	if fb.state == nil {
		fb.logger.Debug(msg)
		return
	}
	fb.dispatch(types.DebugLevel, msg)
}

// Warn logs a warning message with the accumulated typed fields.
//
//go:inline
func (fb TypedFieldBuilder) Warn(msg string) {
	if fb.state == nil {
		fb.logger.Warn(msg)
		return
	}
	fb.dispatch(types.WarnLevel, msg)
}

// Error logs an error message with the accumulated typed fields.
//
//go:inline
func (fb TypedFieldBuilder) Error(msg string) {
	if fb.state == nil {
		fb.logger.Error(msg)
		return
	}
	fb.dispatch(types.ErrorLevel, msg)
}

// dispatch renders the accumulated fields through the pooled per-P entry and
// returns the state to the pool. Passing the pooled (already heap-resident)
// entry to the adapter's WriteZero interface method keeps dispatch
// zero-allocation.
//
//go:inline
func (fb TypedFieldBuilder) dispatch(level types.LogLevel, msg string) {
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

	switch {
	case fb.logger.discardAdapter != nil:
		_ = fb.logger.discardAdapter.WriteZero(nil)
	case len(fb.logger.adapters) == 1:
		_ = fb.logger.adapters[0].WriteZero(entry)
	default:
		for _, a := range fb.logger.adapters {
			_ = a.WriteZero(entry)
		}
	}

	if fb.logger.metrics != nil {
		fb.logger.metrics.counts[level].Add(1)
	}

	globalPerPPool.put(fb.state)
}
