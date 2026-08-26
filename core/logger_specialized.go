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
	"github.com/go-gen-ecosystem/halolog/pool"
	"github.com/go-gen-ecosystem/halolog/types"
)

// This file contains specialized implementations of logging functions to eliminate branches.
// "Specialize EVERYTHING no setup" - User Optimization 3.3

// Tiny helper to avoid repetition in specialized functions.
// Note: We duplicate code here on purpose to ensure linear execution paths for the CPU.

// =============================================================================
// INFO SPECIALIZATIONS
// =============================================================================

func (l *Logger) infoNoMaskOne(l2 *Logger, msg string) {
	entry := pool.AcquireEntry()
	entry.Level = types.InfoLevel
	entry.Message = msg
	entry.Component = l.component
	entry.TimestampUnix = l.clock.GetNsecValue()

	_ = l.adapters[0].WriteZero(entry)
	if l.metrics != nil {
		l.metrics.counts[types.InfoLevel].Add(1)
	}
	pool.ReleaseEntry(entry)
}

func (l *Logger) infoNoMaskMulti(l2 *Logger, msg string) {
	entry := pool.AcquireEntry()
	entry.Level = types.InfoLevel
	entry.Message = msg
	entry.Component = l.component
	entry.TimestampUnix = l.clock.GetNsecValue()

	for _, a := range l.adapters {
		_ = a.WriteZero(entry)
	}
	if l.metrics != nil {
		l.metrics.counts[types.InfoLevel].Add(1)
	}
	pool.ReleaseEntry(entry)
}

func (l *Logger) infoMaskOne(l2 *Logger, msg string) {
	var entry types.LogEntry
	entry.Level = types.InfoLevel
	entry.Message = msg
	entry.Component = l.component
	entry.TimestampUnix = l.clock.GetNsecValue()
	entry.StaticFieldCount = 0

	l.masker.Apply(&entry)
	_ = l.adapters[0].WriteZero(&entry)
	if l.metrics != nil {
		l.metrics.counts[types.InfoLevel].Add(1)
	}
}

func (l *Logger) infoMaskMulti(l2 *Logger, msg string) {
	var entry types.LogEntry
	entry.Level = types.InfoLevel
	entry.Message = msg
	entry.Component = l.component
	entry.TimestampUnix = l.clock.GetNsecValue()
	entry.StaticFieldCount = 0

	l.masker.Apply(&entry)
	for _, a := range l.adapters {
		_ = a.WriteZero(&entry)
	}
	if l.metrics != nil {
		l.metrics.counts[types.InfoLevel].Add(1)
	}
}

// =============================================================================
// DEBUG SPECIALIZATIONS
// =============================================================================

func (l *Logger) debugNoMaskOne(l2 *Logger, msg string) {
	var entry types.LogEntry
	entry.Level = types.DebugLevel
	entry.Message = msg
	entry.Component = l.component
	entry.TimestampUnix = l.clock.GetNsecValue()
	entry.StaticFieldCount = 0

	_ = l.adapters[0].WriteZero(&entry)
	if l.metrics != nil {
		l.metrics.counts[types.DebugLevel].Add(1)
	}
}

func (l *Logger) debugNoMaskMulti(l2 *Logger, msg string) {
	var entry types.LogEntry
	entry.Level = types.DebugLevel
	entry.Message = msg
	entry.Component = l.component
	entry.TimestampUnix = l.clock.GetNsecValue()
	entry.StaticFieldCount = 0

	for _, a := range l.adapters {
		_ = a.WriteZero(&entry)
	}
	if l.metrics != nil {
		l.metrics.counts[types.DebugLevel].Add(1)
	}
}

func (l *Logger) debugMaskOne(l2 *Logger, msg string) {
	var entry types.LogEntry
	entry.Level = types.DebugLevel
	entry.Message = msg
	entry.Component = l.component
	entry.TimestampUnix = l.clock.GetNsecValue()
	entry.StaticFieldCount = 0

	l.masker.Apply(&entry)
	_ = l.adapters[0].WriteZero(&entry)
	if l.metrics != nil {
		l.metrics.counts[types.DebugLevel].Add(1)
	}
}

func (l *Logger) debugMaskMulti(l2 *Logger, msg string) {
	var entry types.LogEntry
	entry.Level = types.DebugLevel
	entry.Message = msg
	entry.Component = l.component
	entry.TimestampUnix = l.clock.GetNsecValue()
	entry.StaticFieldCount = 0

	l.masker.Apply(&entry)
	for _, a := range l.adapters {
		_ = a.WriteZero(&entry)
	}
	if l.metrics != nil {
		l.metrics.counts[types.DebugLevel].Add(1)
	}
}

// =============================================================================
// WARN SPECIALIZATIONS
// =============================================================================

func (l *Logger) warnNoMaskOne(l2 *Logger, msg string) {
	var entry types.LogEntry
	entry.Level = types.WarnLevel
	entry.Message = msg
	entry.Component = l.component
	entry.TimestampUnix = l.clock.GetNsecValue()
	entry.StaticFieldCount = 0

	_ = l.adapters[0].WriteZero(&entry)
	if l.metrics != nil {
		l.metrics.counts[types.WarnLevel].Add(1)
	}
}

func (l *Logger) warnNoMaskMulti(l2 *Logger, msg string) {
	var entry types.LogEntry
	entry.Level = types.WarnLevel
	entry.Message = msg
	entry.Component = l.component
	entry.TimestampUnix = l.clock.GetNsecValue()
	entry.StaticFieldCount = 0

	for _, a := range l.adapters {
		_ = a.WriteZero(&entry)
	}
	if l.metrics != nil {
		l.metrics.counts[types.WarnLevel].Add(1)
	}
}

func (l *Logger) warnMaskOne(l2 *Logger, msg string) {
	var entry types.LogEntry
	entry.Level = types.WarnLevel
	entry.Message = msg
	entry.Component = l.component
	entry.TimestampUnix = l.clock.GetNsecValue()
	entry.StaticFieldCount = 0

	l.masker.Apply(&entry)
	_ = l.adapters[0].WriteZero(&entry)
	if l.metrics != nil {
		l.metrics.counts[types.WarnLevel].Add(1)
	}
}

func (l *Logger) warnMaskMulti(l2 *Logger, msg string) {
	var entry types.LogEntry
	entry.Level = types.WarnLevel
	entry.Message = msg
	entry.Component = l.component
	entry.TimestampUnix = l.clock.GetNsecValue()
	entry.StaticFieldCount = 0

	l.masker.Apply(&entry)
	for _, a := range l.adapters {
		_ = a.WriteZero(&entry)
	}
	if l.metrics != nil {
		l.metrics.counts[types.WarnLevel].Add(1)
	}
}

// =============================================================================
// ERROR SPECIALIZATIONS
// =============================================================================

func (l *Logger) errorNoMaskOne(l2 *Logger, msg string) {
	var entry types.LogEntry
	entry.Level = types.ErrorLevel
	entry.Message = msg
	entry.Component = l.component
	entry.TimestampUnix = l.clock.GetNsecValue()
	entry.StaticFieldCount = 0

	_ = l.adapters[0].WriteZero(&entry)
	if l.metrics != nil {
		l.metrics.counts[types.ErrorLevel].Add(1)
	}
}

func (l *Logger) errorNoMaskMulti(l2 *Logger, msg string) {
	var entry types.LogEntry
	entry.Level = types.ErrorLevel
	entry.Message = msg
	entry.Component = l.component
	entry.TimestampUnix = l.clock.GetNsecValue()
	entry.StaticFieldCount = 0

	for _, a := range l.adapters {
		_ = a.WriteZero(&entry)
	}
	if l.metrics != nil {
		l.metrics.counts[types.ErrorLevel].Add(1)
	}
}

func (l *Logger) errorMaskOne(l2 *Logger, msg string) {
	var entry types.LogEntry
	entry.Level = types.ErrorLevel
	entry.Message = msg
	entry.Component = l.component
	entry.TimestampUnix = l.clock.GetNsecValue()
	entry.StaticFieldCount = 0

	l.masker.Apply(&entry)
	_ = l.adapters[0].WriteZero(&entry)
	if l.metrics != nil {
		l.metrics.counts[types.ErrorLevel].Add(1)
	}
}

func (l *Logger) errorMaskMulti(l2 *Logger, msg string) {
	var entry types.LogEntry
	entry.Level = types.ErrorLevel
	entry.Message = msg
	entry.Component = l.component
	entry.TimestampUnix = l.clock.GetNsecValue()
	entry.StaticFieldCount = 0

	l.masker.Apply(&entry)
	for _, a := range l.adapters {
		_ = a.WriteZero(&entry)
	}
	if l.metrics != nil {
		l.metrics.counts[types.ErrorLevel].Add(1)
	}
}
