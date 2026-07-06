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
	"sync/atomic"

	"github.com/go-gen-ecosystem/halolog/adapters/outputs/discard"
	"github.com/go-gen-ecosystem/halolog/cache"
	"github.com/go-gen-ecosystem/halolog/types"
)

// hotState contains frequently accessed data that fits in one cache line.
// All log dispatch happens through function pointers here.
type hotState struct {
	// Cached timestamp from clock
	cachedNanos uint64 // 8 bytes

	// Level threshold
	level uint64 // 8 bytes

	// Function pointers for level-based dispatch
	// Set at startup based on configuration - avoids method value escapes
	infoFunc  func(*Logger, string)
	debugFunc func(*Logger, string)
	warnFunc  func(*Logger, string)
	errorFunc func(*Logger, string)
	fatalFunc func(*Logger, string)
	panicFunc func(*Logger, string)
	traceFunc func(*Logger, string)

	_ [8]byte // Pad to 64 bytes
}

// Logger is the core logging engine.
// Uses the clean package's proven approach: function pointers in hot state,
// concrete adapter types for discard optimization, stack-allocated entries.
type Logger struct {
	// Hot state pointer (cache line 1)
	hot atomic.Pointer[hotState]
	_   [56]byte // Pad to 64 bytes

	// Cold state
	component      string
	clock          *cache.CachedClock
	adapters       []types.Adapter
	masker         types.PIIMasker
	enableMasking  bool
	discardAdapter *discard.Adapter // Concrete type for zero-allocation

	// Optional features (nil if not configured)
	sampler  types.Sampler
	features *featureManager
	metrics  *metricsCollector
}

// featureManager handles optional advanced features.
type featureManager struct {
	sampling    types.Sampler
	alerts      *alertManager
	aggregation *aggregationManager
}

// metricsCollector tracks logging metrics.
type metricsCollector struct {
	counts [8]atomic.Int64
}

type alertManager struct{}
type aggregationManager struct{}

// Config holds logger configuration.
type Config struct {
	Component string
	Level     types.LogLevel
	Adapters  []types.Adapter

	// Optional features
	EnableMasking  bool
	Masker         types.PIIMasker
	EnableSampling bool
	Sampler        types.Sampler
	EnableMetrics  bool

	// Advanced features
	EnableAlerts      bool
	EnableAggregation bool
}

// NewLogger creates a new logger with the given configuration.
func NewLogger(config Config) *Logger {
	l := &Logger{
		component:     config.Component,
		clock:         cache.GetGlobalCachedClock(),
		adapters:      config.Adapters,
		masker:        config.Masker,
		enableMasking: config.EnableMasking && config.Masker != nil,
	}

	// Detect discard adapter for zero-allocation optimization
	if len(config.Adapters) == 1 {
		if da, ok := config.Adapters[0].(*discard.Adapter); ok {
			l.discardAdapter = da
		}
	}

	// Set up optional features
	if config.EnableSampling && config.Sampler != nil {
		l.sampler = config.Sampler
	}

	if config.EnableMetrics {
		l.metrics = &metricsCollector{}
	}

	// Build hot state
	hot := &hotState{
		level:       uint64(config.Level),
		cachedNanos: uint64(l.clock.GetNsecValue()),
	}

	// Set up function pointers for optimized dispatch
	l.setupFunctionPointers(hot, config.Level)

	// Store hot state
	l.hot.Store(hot)

	return l
}

// setupFunctionPointers sets up function pointers based on configuration.
// This is the startup-time specialization - each level gets the exact function it needs.
func (l *Logger) setupFunctionPointers(hot *hotState, level types.LogLevel) {
	// Discard adapter optimization
	if l.discardAdapter != nil {
		// Use concrete discard adapter - zero interface dispatch
		if level <= types.TraceLevel {
			hot.traceFunc = l.realTraceDiscard
		} else {
			hot.traceFunc = noopLog
		}
		if level <= types.DebugLevel {
			hot.debugFunc = l.realDebugDiscard
		} else {
			hot.debugFunc = noopLog
		}
		if level <= types.InfoLevel {
			hot.infoFunc = l.realInfoDiscard
		} else {
			hot.infoFunc = noopLog
		}
		if level <= types.WarnLevel {
			hot.warnFunc = l.realWarnDiscard
		} else {
			hot.warnFunc = noopLog
		}
		if level <= types.ErrorLevel {
			hot.errorFunc = l.realErrorDiscard
		} else {
			hot.errorFunc = noopLog
		}
		hot.fatalFunc = l.realFatalDiscard
		hot.panicFunc = l.realPanicDiscard
		return
	}

	// Regular adapter path - Specialize EVERYTHING
	// We select the EXACT function needed to avoid "if masking" or "if loop" checks at runtime.

	// Pre-calculate flags
	isMasking := l.enableMasking
	isSingle := len(l.adapters) == 1

	// Setup Info
	if level <= types.InfoLevel {
		if isMasking {
			if isSingle {
				hot.infoFunc = l.infoMaskOne
			} else {
				hot.infoFunc = l.infoMaskMulti
			}
		} else {
			if isSingle {
				hot.infoFunc = l.infoNoMaskOne
			} else {
				hot.infoFunc = l.infoNoMaskMulti
			}
		}
	} else {
		hot.infoFunc = noopLog
	}

	// Setup Debug
	if level <= types.DebugLevel {
		if isMasking {
			if isSingle {
				hot.debugFunc = l.debugMaskOne
			} else {
				hot.debugFunc = l.debugMaskMulti
			}
		} else {
			if isSingle {
				hot.debugFunc = l.debugNoMaskOne
			} else {
				hot.debugFunc = l.debugNoMaskMulti
			}
		}
	} else {
		hot.debugFunc = noopLog
	}

	// Setup Warn
	if level <= types.WarnLevel {
		if isMasking {
			if isSingle {
				hot.warnFunc = l.warnMaskOne
			} else {
				hot.warnFunc = l.warnMaskMulti
			}
		} else {
			if isSingle {
				hot.warnFunc = l.warnNoMaskOne
			} else {
				hot.warnFunc = l.warnNoMaskMulti
			}
		}
	} else {
		hot.warnFunc = noopLog
	}

	// Setup Error
	if level <= types.ErrorLevel {
		if isMasking {
			if isSingle {
				hot.errorFunc = l.errorMaskOne
			} else {
				hot.errorFunc = l.errorMaskMulti
			}
		} else {
			if isSingle {
				hot.errorFunc = l.errorNoMaskOne
			} else {
				hot.errorFunc = l.errorNoMaskMulti
			}
		}
	} else {
		hot.errorFunc = noopLog
	}

	// Setup Trace (using generic realTrace for now, optimize if needed)
	if level <= types.TraceLevel {
		hot.traceFunc = l.realTrace
	} else {
		hot.traceFunc = noopLog
	}

	// Fatal/Panic (less critical)
	hot.fatalFunc = l.realFatal
	hot.panicFunc = l.realPanic
}

// noopLog is the no-op function for disabled levels.
//
//go:inline
func noopLog(_ *Logger, _ string) {}

// ===== HOT PATH METHODS =====

// Trace logs a trace message.
//
//go:inline
func (l *Logger) Trace(msg string) {
	hot := l.hot.Load()
	hot.traceFunc(l, msg)
}

// Debug logs a debug message.
//
//go:inline
func (l *Logger) Debug(msg string) {
	hot := l.hot.Load()
	hot.debugFunc(l, msg)
}

// Info logs an info message.
//
//go:inline
func (l *Logger) Info(msg string) {
	hot := l.hot.Load()
	hot.infoFunc(l, msg)
}

// Warn logs a warning message.
//
//go:inline
func (l *Logger) Warn(msg string) {
	hot := l.hot.Load()
	hot.warnFunc(l, msg)
}

// Error logs an error message.
//
//go:inline
func (l *Logger) Error(msg string) {
	hot := l.hot.Load()
	hot.errorFunc(l, msg)
}

// Fatal logs a fatal message.
//
//go:inline
func (l *Logger) Fatal(msg string) {
	hot := l.hot.Load()
	hot.fatalFunc(l, msg)
}

// Panic logs a panic message.
//
//go:inline
func (l *Logger) Panic(msg string) {
	hot := l.hot.Load()
	hot.panicFunc(l, msg)
}

// ===== DISCARD-OPTIMIZED FUNCTIONS (Concrete type dispatch) =====

//go:inline
func (l *Logger) realTraceDiscard(_ *Logger, msg string) {
	var entry types.LogEntry
	entry.Level = types.TraceLevel
	entry.Message = msg
	entry.Component = l.component
	entry.TimestampUnix = l.clock.GetNsecValue()
	entry.StaticFieldCount = 0
	l.discardAdapter.WriteZero(&entry)
}

//go:inline
func (l *Logger) realDebugDiscard(_ *Logger, msg string) {
	var entry types.LogEntry
	entry.Level = types.DebugLevel
	entry.Message = msg
	entry.Component = l.component
	entry.TimestampUnix = l.clock.GetNsecValue()
	entry.StaticFieldCount = 0
	l.discardAdapter.WriteZero(&entry)
}

//go:inline
func (l *Logger) realInfoDiscard(_ *Logger, msg string) {
	// Optimization: Discard adapter ignores entry, so pass nil to avoid allocation/zeroing
	l.discardAdapter.WriteZero(nil)
}

//go:inline
func (l *Logger) realWarnDiscard(_ *Logger, msg string) {
	var entry types.LogEntry
	entry.Level = types.WarnLevel
	entry.Message = msg
	entry.Component = l.component
	entry.TimestampUnix = l.clock.GetNsecValue()
	entry.StaticFieldCount = 0
	l.discardAdapter.WriteZero(&entry)
}

//go:inline
func (l *Logger) realErrorDiscard(_ *Logger, msg string) {
	var entry types.LogEntry
	entry.Level = types.ErrorLevel
	entry.Message = msg
	entry.Component = l.component
	entry.TimestampUnix = l.clock.GetNsecValue()
	entry.StaticFieldCount = 0
	l.discardAdapter.WriteZero(&entry)
}

//go:inline
func (l *Logger) realFatalDiscard(_ *Logger, msg string) {
	var entry types.LogEntry
	entry.Level = types.FatalLevel
	entry.Message = msg
	entry.Component = l.component
	entry.TimestampUnix = l.clock.GetNsecValue()
	entry.StaticFieldCount = 0
	l.discardAdapter.WriteZero(&entry)
}

//go:inline
func (l *Logger) realPanicDiscard(_ *Logger, msg string) {
	var entry types.LogEntry
	entry.Level = types.PanicLevel
	entry.Message = msg
	entry.Component = l.component
	entry.TimestampUnix = l.clock.GetNsecValue()
	entry.StaticFieldCount = 0
	l.discardAdapter.WriteZero(&entry)
}

// ===== REGULAR ADAPTER FUNCTIONS =====

//go:inline
func (l *Logger) realTrace(_ *Logger, msg string) {
	var entry types.LogEntry
	entry.Level = types.TraceLevel
	entry.Message = msg
	entry.Component = l.component
	entry.TimestampUnix = l.clock.GetNsecValue()
	entry.StaticFieldCount = 0

	if l.enableMasking {
		l.masker.Apply(&entry)
	}

	if len(l.adapters) == 1 {
		l.adapters[0].WriteZero(&entry)
	} else {
		for _, a := range l.adapters {
			a.WriteZero(&entry)
		}
	}
}

//go:inline
func (l *Logger) realDebug(_ *Logger, msg string) {
	var entry types.LogEntry
	entry.Level = types.DebugLevel
	entry.Message = msg
	entry.Component = l.component
	entry.TimestampUnix = l.clock.GetNsecValue()
	entry.StaticFieldCount = 0

	if l.enableMasking {
		l.masker.Apply(&entry)
	}

	if len(l.adapters) == 1 {
		l.adapters[0].WriteZero(&entry)
	} else {
		for _, a := range l.adapters {
			a.WriteZero(&entry)
		}
	}
}

//go:inline
func (l *Logger) realInfo(_ *Logger, msg string) {
	var entry types.LogEntry
	entry.Level = types.InfoLevel
	entry.Message = msg
	entry.Component = l.component
	entry.TimestampUnix = l.clock.GetNsecValue()
	entry.StaticFieldCount = 0

	if l.enableMasking {
		l.masker.Apply(&entry)
	}

	if len(l.adapters) == 1 {
		l.adapters[0].WriteZero(&entry)
	} else {
		for _, a := range l.adapters {
			a.WriteZero(&entry)
		}
	}

	if l.metrics != nil {
		l.metrics.counts[types.InfoLevel].Add(1)
	}
}

//go:inline
func (l *Logger) realWarn(_ *Logger, msg string) {
	var entry types.LogEntry
	entry.Level = types.WarnLevel
	entry.Message = msg
	entry.Component = l.component
	entry.TimestampUnix = l.clock.GetNsecValue()
	entry.StaticFieldCount = 0

	if l.enableMasking {
		l.masker.Apply(&entry)
	}

	if len(l.adapters) == 1 {
		l.adapters[0].WriteZero(&entry)
	} else {
		for _, a := range l.adapters {
			a.WriteZero(&entry)
		}
	}

	if l.metrics != nil {
		l.metrics.counts[types.WarnLevel].Add(1)
	}
}

//go:inline
func (l *Logger) realError(_ *Logger, msg string) {
	var entry types.LogEntry
	entry.Level = types.ErrorLevel
	entry.Message = msg
	entry.Component = l.component
	entry.TimestampUnix = l.clock.GetNsecValue()
	entry.StaticFieldCount = 0

	if l.enableMasking {
		l.masker.Apply(&entry)
	}

	if len(l.adapters) == 1 {
		l.adapters[0].WriteZero(&entry)
	} else {
		for _, a := range l.adapters {
			a.WriteZero(&entry)
		}
	}
}

//go:inline
func (l *Logger) realFatal(_ *Logger, msg string) {
	var entry types.LogEntry
	entry.Level = types.FatalLevel
	entry.Message = msg
	entry.Component = l.component
	entry.TimestampUnix = l.clock.GetNsecValue()
	entry.StaticFieldCount = 0

	if l.enableMasking {
		l.masker.Apply(&entry)
	}

	if len(l.adapters) == 1 {
		l.adapters[0].WriteZero(&entry)
	} else {
		for _, a := range l.adapters {
			a.WriteZero(&entry)
		}
	}
}

//go:inline
func (l *Logger) realPanic(_ *Logger, msg string) {
	var entry types.LogEntry
	entry.Level = types.PanicLevel
	entry.Message = msg
	entry.Component = l.component
	entry.TimestampUnix = l.clock.GetNsecValue()
	entry.StaticFieldCount = 0

	if l.enableMasking {
		l.masker.Apply(&entry)
	}

	if len(l.adapters) == 1 {
		l.adapters[0].WriteZero(&entry)
	} else {
		for _, a := range l.adapters {
			a.WriteZero(&entry)
		}
	}
}

// With returns a fluent builder for adding fields.
// Returns by VALUE to avoid heap escape.
// Gets per-P state lazily on first field addition.
//
//go:inline
func (l *Logger) WithField(key string, value interface{}) FieldBuilder {
	state := globalPerPPool.get()
	state.entry.StaticFields[0] = types.TypedFieldData{Key: key, Value: value}
	state.entry.StaticFieldCount = 1

	return FieldBuilder{
		logger: l,
		state:  state,
	}
}

// WithError returns a fluent builder with an error field.
// Returns by VALUE to avoid heap escape.
//
//go:inline
func (l *Logger) WithError(err error) FieldBuilder {
	if err == nil {
		return FieldBuilder{logger: l}
	}

	state := globalPerPPool.get()
	state.entry.StaticFields[0] = types.TypedFieldData{Key: "error", Value: err.Error()}
	state.entry.StaticFieldCount = 1

	return FieldBuilder{
		logger: l,
		state:  state,
	}
}

// Component returns the logger's component name.
func (l *Logger) Component() string {
	return l.component
}

// Level returns the logger's minimum log level.
func (l *Logger) Level() types.LogLevel {
	hot := l.hot.Load()
	return types.LogLevel(hot.level)
}

// Flush flushes all adapters.
func (l *Logger) Flush() error {
	for _, a := range l.adapters {
		if err := a.Flush(); err != nil {
			return err
		}
	}
	return nil
}

// Close closes all adapters and releases resources.
func (l *Logger) Close() error {
	for _, a := range l.adapters {
		if err := a.Close(); err != nil {
			return err
		}
	}
	return nil
}

// writeMinimalEntry writes a MinimalFieldEntry to adapters.
// This is the optimized path for 1-4 fields that avoids allocating the full types.LogEntry.
//
//go:inline
func (l *Logger) writeMinimalEntry(entry *MinimalFieldEntry) {
	// For discard adapter, optimized path (no conversion needed)
	// For discard adapter, optimized path (no conversion needed)
	if l.discardAdapter != nil {
		// Discard adapter WriteZero is a no-op, so we can skip conversion and pass nil
		l.discardAdapter.WriteZero(nil)
		return
	}

	// For regular adapters, convert MinimalFieldEntry to types.LogEntry
	// Note: This creates types.LogEntry on stack here, but the function is inlined
	// and the compiler can see the entry doesn't escape.
	var logEntry types.LogEntry
	logEntry.Level = entry.Level
	logEntry.Message = entry.Message
	logEntry.Component = entry.Component
	logEntry.TimestampUnix = entry.TimestampUnix
	logEntry.StaticFieldCount = entry.StaticFieldCount

	// Zero-copy: point LogEntry slice to MinimalFieldEntry's stack buffer
	// This is safe because logEntry doesn't escape this function (it is passed to WriteZero which takes *LogEntry).
	// But we must assume WriteZero doesn't store the pointer.
	// However, TypedFieldData contains pointers (String, Any).

	// Since LogEntry no longer has StaticBuffer, we MUST point StaticFields to something.
	// Pointing to entry.StaticFields[:] works.
	logEntry.StaticFields = entry.StaticFields[:entry.StaticFieldCount]

	// Apply masking if enabled
	if l.enableMasking && l.masker != nil {
		l.masker.Apply(&logEntry)
	}

	// Dispatch to adapters
	if len(l.adapters) == 1 {
		l.adapters[0].WriteZero(&logEntry)
	} else {
		for _, a := range l.adapters {
			a.WriteZero(&logEntry)
		}
	}

	// Update metrics if configured
	if l.metrics != nil {
		l.metrics.counts[entry.Level].Add(1)
	}
}
