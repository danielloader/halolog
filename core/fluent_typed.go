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
	jsonfmt "github.com/go-gen-ecosystem/halolog/adapters/formatters/json"
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

// acquire fetches the pooled per-P state for a new line and selects the line's
// encoding mode: when the logger is direct-eligible, the adapter's current
// direct encoder is captured once here, and every field of this line is encoded
// straight to bytes instead of being captured as a struct. Querying per line
// (not per logger) means a formatter swap safely flips subsequent lines back to
// the capture path.
//
//go:inline
func (fb TypedFieldBuilder) acquire() TypedFieldBuilder {
	fb.state = acquireState(fb.logger)
	return fb
}

// acquireState fetches the pooled per-P state for a new line and selects its
// encoding mode: the adapter's current direct encoder is asserted to the
// CONCRETE JSON formatter so every subsequent per-field call is statically
// dispatched (jsonfmt package functions — no interface calls on the hot loop).
// Querying per line means a runtime formatter swap safely flips later lines
// back to the capture path; a non-JSON encoder also keeps the capture path.
//
//go:inline
func acquireState(l *Logger) *perPState {
	s := globalPerPPool.get()
	if da := l.directAdapter; da != nil {
		s.directJSON, _ = da.DirectEncoder().(*jsonfmt.Formatter)
	}
	return s
}

// addField writes one field into the line, shared by the typed builder and the
// level-first Line API (kd nil ⇒ plain string key). On the direct fast path the
// field is encoded to JSON bytes immediately via a static package call
// (zerolog-style, no struct capture); otherwise it is captured as a
// TypedFieldData for the formatter. Both paths allocate nothing.
//
//go:inline
func addField(s *perPState, kd *types.FieldKey, key string, val types.FieldValue) {
	if s.directJSON != nil {
		s.directFields = jsonfmt.AppendField(s.directFields, kd, key, val)
		return
	}
	entry := &s.entry
	n := entry.StaticFieldCount
	if n < len(entry.StaticFields) {
		entry.StaticFields[n] = types.TypedFieldData{Key: key, KeyDesc: kd, Val: val}
		entry.StaticFieldCount = n + 1
	} else {
		entry.Fields = append(entry.Fields, types.TypedFieldData{Key: key, KeyDesc: kd, Val: val})
	}
}

// withTyped acquires the pooled state on the first field and adds a
// string-keyed typed value.
//
//go:inline
func (fb TypedFieldBuilder) withTyped(key string, val types.FieldValue) TypedFieldBuilder {
	if fb.state == nil {
		fb = fb.acquire()
	}
	addField(fb.state, nil, key, val)
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

// WithAny adds an arbitrary value under a plain string key. Prefer the typed
// setters on hot paths — Any values box through an interface.
//
//go:inline
func (fb TypedFieldBuilder) WithAny(key string, value interface{}) TypedFieldBuilder {
	return fb.withTyped(key, types.AnyValue(value))
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
		fb = fb.acquire()
	}
	// Fastest path in the logger: on the direct route the pre-escaped `,"key":`
	// fragment is a single memcpy — no key escaping, no struct capture.
	addField(fb.state, key, key.Name, val)
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
	dispatchLine(fb.logger, fb.state, level, msg)
}

// dispatchLine renders the accumulated line and returns the state to the pool.
// It is shared by the typed builder and the level-first Line API.
func dispatchLine(l *Logger, s *perPState, level types.LogLevel, msg string) {
	if level < l.Level() {
		globalPerPPool.put(s)
		return
	}

	// Direct fast path: fields are already encoded bytes; assemble
	// header + fields + closer in the pooled line buffer and hand the finished
	// line to the raw writer. Every call here is statically dispatched on the
	// concrete JSON formatter. Eligibility (see NewLogger) guarantees masking
	// and sampling are off, so no transform is skipped.
	if f := s.directJSON; f != nil {
		line := f.AppendHeader(s.lineBuf[:0], l.clock.GetNsecValue(), level, msg)
		line = append(line, s.directFields...)
		line = jsonfmt.AppendCloser(line)
		s.lineBuf = line // retain growth for reuse
		_ = l.rawWriter.WriteRaw(line)
		if l.metrics != nil {
			l.metrics.counts[level].Add(1)
		}
		globalPerPPool.put(s)
		return
	}

	fb := TypedFieldBuilder{logger: l, state: s}
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
