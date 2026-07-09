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

// Package core — level-first Line API.
// Author: Admilson B. F. Cossa

package core

import (
	"github.com/go-gen-ecosystem/halolog/types"
)

// Line is a level-first fluent builder: the level is fixed when the line is
// opened, so a DISABLED level costs a single check — no state acquisition, no
// field encoding (the zerolog model). Line is one word; a nil state marks a
// disabled line and every method no-ops on it.
//
//	logger.InfoLine().Str(keyUser, "alice").WithInt("status", 200).Msg("handled")
//
// Field and dispatch mechanics are shared with the typed builder (addField /
// dispatchLine), so both APIs stay byte-identical and zero-allocation.
type Line struct {
	s *perPState
}

// line opens a level-first line, returning the disabled (no-op) Line when the
// level is filtered out.
//
//go:inline
func (l *Logger) line(level types.LogLevel) Line {
	if level < l.Level() {
		return Line{}
	}
	s := acquireState(l)
	s.owner = l
	s.lineLevel = level
	return Line{s: s}
}

// TraceLine opens a TRACE-level line.
func (l *Logger) TraceLine() Line { return l.line(types.TraceLevel) }

// DebugLine opens a DEBUG-level line.
func (l *Logger) DebugLine() Line { return l.line(types.DebugLevel) }

// InfoLine opens an INFO-level line.
func (l *Logger) InfoLine() Line { return l.line(types.InfoLevel) }

// WarnLine opens a WARN-level line.
func (l *Logger) WarnLine() Line { return l.line(types.WarnLevel) }

// ErrorLine opens an ERROR-level line.
func (l *Logger) ErrorLine() Line { return l.line(types.ErrorLevel) }

// add appends one field unless the line is disabled.
//
//go:inline
func (ln Line) add(kd *types.FieldKey, key string, val types.FieldValue) Line {
	if ln.s != nil {
		addField(ln.s, kd, key, val)
	}
	return ln
}

// Str adds a string field under a pre-declared key.
//
//go:inline
func (ln Line) Str(key *types.FieldKey, value string) Line {
	return ln.add(key, keyName(key), types.StringValue(value))
}

// Int adds an int field under a pre-declared key.
//
//go:inline
func (ln Line) Int(key *types.FieldKey, value int) Line {
	return ln.add(key, keyName(key), types.IntValue(value))
}

// Int64 adds an int64 field under a pre-declared key.
//
//go:inline
func (ln Line) Int64(key *types.FieldKey, value int64) Line {
	return ln.add(key, keyName(key), types.Int64Value(value))
}

// Float64 adds a float64 field under a pre-declared key.
//
//go:inline
func (ln Line) Float64(key *types.FieldKey, value float64) Line {
	return ln.add(key, keyName(key), types.Float64Value(value))
}

// Bool adds a bool field under a pre-declared key.
//
//go:inline
func (ln Line) Bool(key *types.FieldKey, value bool) Line {
	return ln.add(key, keyName(key), types.BoolValue(value))
}

// Err adds an error field under a pre-declared key. A nil error is a no-op.
//
//go:inline
func (ln Line) Err(key *types.FieldKey, err error) Line {
	if err == nil {
		return ln
	}
	return ln.add(key, keyName(key), types.ErrorValue(err))
}

// Any adds an arbitrary value under a pre-declared key. Prefer the typed
// methods on hot paths.
//
//go:inline
func (ln Line) Any(key *types.FieldKey, value interface{}) Line {
	return ln.add(key, keyName(key), types.AnyValue(value))
}

// WithString adds a string field under a plain string key.
//
//go:inline
func (ln Line) WithString(key, value string) Line {
	return ln.add(nil, key, types.StringValue(value))
}

// WithInt adds an int field under a plain string key.
//
//go:inline
func (ln Line) WithInt(key string, value int) Line {
	return ln.add(nil, key, types.IntValue(value))
}

// WithInt64 adds an int64 field under a plain string key.
//
//go:inline
func (ln Line) WithInt64(key string, value int64) Line {
	return ln.add(nil, key, types.Int64Value(value))
}

// WithFloat64 adds a float64 field under a plain string key.
//
//go:inline
func (ln Line) WithFloat64(key string, value float64) Line {
	return ln.add(nil, key, types.Float64Value(value))
}

// WithBool adds a bool field under a plain string key.
//
//go:inline
func (ln Line) WithBool(key string, value bool) Line {
	return ln.add(nil, key, types.BoolValue(value))
}

// WithError adds an error field under the "error" key. A nil error is a no-op.
//
//go:inline
func (ln Line) WithError(err error) Line {
	if err == nil {
		return ln
	}
	return ln.add(nil, "error", types.ErrorValue(err))
}

// Msg completes the line with a message and writes it. On a disabled line it is
// a no-op. The Line must not be used after Msg.
//
//go:inline
func (ln Line) Msg(msg string) {
	if ln.s == nil {
		return
	}
	dispatchLine(ln.s.owner, ln.s, ln.s.lineLevel, msg)
}

// Send completes the line with an empty message.
//
//go:inline
func (ln Line) Send() { ln.Msg("") }

// keyName returns the raw name of a pre-declared key (nil-safe).
//
//go:inline
func keyName(key *types.FieldKey) string {
	if key == nil {
		return ""
	}
	return key.Name
}
