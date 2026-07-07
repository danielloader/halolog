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
	"sync"
)

/* =====================================================================
   ENTRY POOL & CONSTRUCTORS
   ===================================================================== */

// LogEntryPool is the shared sync.Pool used to recycle LogEntry values and
// keep the logging hot path allocation-free.
var LogEntryPool = sync.Pool{
	New: func() interface{} {
		return &LogEntry{
			Fields:            make([]TypedFieldData, 0, 8),
			Context:           make([]TypedFieldData, 0, 8),
			QuantumStore:      nil,
			UseQuantumStorage: false,
		}
	},
}

// NewLogEntry creates a new log entry
func NewLogEntry(level LogLevel, message string) *LogEntry {
	return &LogEntry{
		Timestamp:         globalEntryClock.now(),
		Level:             level,
		Message:           message,
		Fields:            make([]TypedFieldData, 0, 8),
		Context:           make([]TypedFieldData, 0, 8),
		QuantumStore:      nil,
		UseQuantumStorage: false,
	}
}

// NewLogEntryWithCaller creates a new log entry with caller information
func NewLogEntryWithCaller(level LogLevel, message string, skip int) *LogEntry {
	entry := LogEntryPool.Get().(*LogEntry)
	entry.Timestamp = globalEntryClock.now()
	entry.Level = level
	entry.Message = message
	entry.Component = ""
	entry.Fields = nil
	entry.Context = nil
	entry.Error = nil
	entry.QuantumStore = nil
	entry.UseQuantumStorage = false

	// Get caller info if GetHighPerformanceCallerInfo is available
	// Otherwise just leave File and Line empty
	entry.File = ""
	entry.Line = 0
	entry.Caller = nil

	return entry
}

// AcquireLogEntry gets a log entry from the pool (zero-allocation if pool is warm)
func AcquireLogEntry() *LogEntry {
	entry := LogEntryPool.Get().(*LogEntry)
	// Reset entry to clean state
	entry.Timestamp = globalEntryClock.now()
	entry.Level = 0
	entry.Message = ""
	entry.Component = ""
	entry.Fields = nil
	entry.Context = nil
	entry.Error = nil
	entry.ErrorMsg = ""
	entry.File = ""
	entry.Line = 0
	entry.Caller = nil
	entry.StaticFieldCount = 0
	entry.StaticContextCount = 0
	entry.QuantumStore = nil
	entry.UseQuantumStorage = false
	return entry
}

// ReleaseLogEntry returns a log entry to the pool for reuse
func ReleaseLogEntry(entry *LogEntry) {
	if entry != nil {
		// Clear references to help GC
		entry.Fields = nil
		entry.Context = nil
		entry.Error = nil
		entry.Caller = nil

		// Reset quantum store if present
		if entry.QuantumStore != nil {
			entry.QuantumStore.Reset()
			entry.QuantumStore = nil
		}
		entry.UseQuantumStorage = false

		LogEntryPool.Put(entry)
	}
}
