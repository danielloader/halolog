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

// FieldBuilder provides a fluent API for building log entries with fields.
// This is a VALUE TYPE to avoid heap escape - returned by value, not pointer.
//
// Design:
//   - Get per-P pooled state on first field addition (lazy acquisition)
//   - Write fields directly into the pooled entry's static buffer (zero-copy)
//   - Dispatch reuses the pooled entry so passing it through the adapter's
//     WriteZero interface method does not force a per-call heap allocation
//
// Usage:
//
//	logger.With("key", "value").Info("message")
//	logger.With("k1", "v1").With("k2", 123).Info("message")
type FieldBuilder struct {
	logger *Logger
	state  *perPState // per-P pooled state (acquired on first field)
}

// WithField adds a field to the entry.
// Returns FieldBuilder by VALUE to avoid heap allocation.
// Gets per-P state on first call, writes directly to StaticBuffer.
//
//go:inline
func (fb FieldBuilder) WithField(key string, value interface{}) FieldBuilder {
	// Get per-P state on first field (lazy acquisition)
	if fb.state == nil {
		fb.state = globalPerPPool.get()
	}

	entry := &fb.state.entry
	n := entry.StaticFieldCount

	// Direct buffer population (zero-copy)
	if n < len(entry.StaticFields) {
		entry.StaticFields[n] = types.TypedFieldData{Key: key, Value: value}
		entry.StaticFieldCount = n + 1
	} else {
		// Overflow to dynamic fields (rare case: >64 fields)
		entry.Fields = append(entry.Fields, types.TypedFieldData{Key: key, Value: value})
	}

	return fb
}

// String adds a string field.
//
//go:inline
func (fb FieldBuilder) String(key string, value string) FieldBuilder {
	return fb.WithField(key, value)
}

// Int adds an integer field.
//
//go:inline
func (fb FieldBuilder) Int(key string, value int) FieldBuilder {
	return fb.WithField(key, value)
}

// Int64 adds an int64 field.
//
//go:inline
func (fb FieldBuilder) Int64(key string, value int64) FieldBuilder {
	return fb.WithField(key, value)
}

// Float64 adds a float64 field.
//
//go:inline
func (fb FieldBuilder) Float64(key string, value float64) FieldBuilder {
	return fb.WithField(key, value)
}

// Bool adds a boolean field.
//
//go:inline
func (fb FieldBuilder) Bool(key string, value bool) FieldBuilder {
	return fb.WithField(key, value)
}

// Err adds an error field.
//
//go:inline
func (fb FieldBuilder) Err(err error) FieldBuilder {
	if err != nil {
		return fb.WithField("error", err.Error())
	}
	return fb
}

// Info logs an info message with the accumulated fields.
//
//go:inline
func (fb FieldBuilder) Info(msg string) {
	if fb.state == nil {
		// No fields case - use direct path (sub-2ns)
		fb.logger.Info(msg)
		return
	}

	entry := &fb.state.entry

	// Dispatch through the pooled per-P entry. A single-field special case using
	// a stack-local MinimalFieldEntry was removed: passing its address through the
	// adapter's WriteZero interface method forced escape analysis to heap-allocate
	// it on every call (2 allocs/op), which was slower than reusing the pooled
	// entry the way the multi-field path already does (0 allocs/op).
	entry.Level = types.InfoLevel
	entry.Message = msg
	entry.Component = fb.logger.component
	entry.TimestampUnix = fb.logger.clock.GetNsecValue()
	entry.StaticFields = entry.StaticFields[:entry.StaticFieldCount]

	// Apply masking if configured
	if fb.logger.enableMasking && fb.logger.masker != nil {
		fb.logger.masker.Apply(entry)
	}

	// Direct adapter call
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

	// Update metrics if configured
	if fb.logger.metrics != nil {
		fb.logger.metrics.counts[types.InfoLevel].Add(1)
	}

	// Return state to pool
	globalPerPPool.put(fb.state)
}

// Debug logs a debug message with the accumulated fields.
//
//go:inline
func (fb FieldBuilder) Debug(msg string) {
	if fb.state == nil {
		fb.logger.Debug(msg)
		return
	}

	entry := &fb.state.entry

	entry.Level = types.DebugLevel
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
		fb.logger.metrics.counts[types.DebugLevel].Add(1)
	}

	globalPerPPool.put(fb.state)
}

// Warn logs a warning message with the accumulated fields.
//
//go:inline
func (fb FieldBuilder) Warn(msg string) {
	if fb.state == nil {
		fb.logger.Warn(msg)
		return
	}

	entry := &fb.state.entry

	entry.Level = types.WarnLevel
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
		fb.logger.metrics.counts[types.WarnLevel].Add(1)
	}

	globalPerPPool.put(fb.state)
}

// Error logs an error message with the accumulated fields.
//
//go:inline
func (fb FieldBuilder) Error(msg string) {
	if fb.state == nil {
		fb.logger.Error(msg)
		return
	}

	entry := &fb.state.entry

	entry.Level = types.ErrorLevel
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
		fb.logger.metrics.counts[types.ErrorLevel].Add(1)
	}

	globalPerPPool.put(fb.state)
}

// Trace logs a trace message with the accumulated fields.
//
//go:inline
func (fb FieldBuilder) Trace(msg string) {
	if fb.state == nil {
		fb.logger.Trace(msg)
		return
	}

	entry := &fb.state.entry
	entry.Level = types.TraceLevel
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

	globalPerPPool.put(fb.state)
}

// Fatal logs a fatal message with the accumulated fields.
//
//go:inline
func (fb FieldBuilder) Fatal(msg string) {
	if fb.state == nil {
		fb.logger.Fatal(msg)
		return
	}

	entry := &fb.state.entry
	entry.Level = types.FatalLevel
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

	globalPerPPool.put(fb.state)
}

// Panic logs a panic message with the accumulated fields.
//
//go:inline
func (fb FieldBuilder) Panic(msg string) {
	if fb.state == nil {
		fb.logger.Panic(msg)
		return
	}

	entry := &fb.state.entry
	entry.Level = types.PanicLevel
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

	globalPerPPool.put(fb.state)
}
