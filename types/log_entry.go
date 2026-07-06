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

import "time"

// MinimalLogEntry is a lightweight entry for optimized logging without fields.
// Size: ~80 bytes - fits on stack for sub-10ns performance.
// Use for logger.Info("msg"), logger.Error("msg") etc. with zero fields.
type MinimalLogEntry struct {
	TimestampUnix int64    // Unix nanos
	Level         LogLevel // 8 bytes
	Component     string   // 16 bytes (string header)
	Message       string   // 16 bytes (string header)
}

type LogEntry struct {
	// Core metadata
	Timestamp     time.Time // Wall clock time
	TimestampUnix int64     // Unix nanos for sorting
	Level         LogLevel
	Component     string
	Message       string

	// Error handling
	Error    error  // Error interface
	ErrorMsg string // Error message string

	// Source location
	File   string
	Line   int
	Caller interface{}
	// Preformatted access
	// Ex: "User.go:113"
	LocationBuf [32]byte // preformatted fixed-size location
	LocationLen uint8    // actual used length in LocationBuf
	BaseName    string   // Base name of the file (without path)

	// Field storage
	Fields  []TypedFieldData // Dynamic slice for typical usage
	Context []TypedFieldData // Context fields

	// Pre-allocated static storage
	// Pre-allocated static storage
	StaticFields       []TypedFieldData           // Slice backed by external buffer
	StaticContext      []TypedFieldData           // Context fields slice
	StaticFieldCount   int                        // Counter for static fields
	StaticContextCount int                        // Counter for static context fields
	QuantumStore       *EnhancedQuantumFieldStore // Quantum storage for high-frequency fields
	UseQuantumStorage  bool                       // Flag to enable quantum storage

	// Performance optimization flags
	hasFieldOverflow   bool // Flag when static storage overflows to dynamic
	hasContextOverflow bool // Flag when static context overflows to dynamic
	isQuantumEnabled   bool // Flag for quantum storage mode

	// Memory pool optimization
	PoolID int // Identifier for memory pool tracking

	// internal bookkeeping or Internal flags for pool management
	metadata uint64

	// Scratch buffer for formatting
	OutputBuffer [1024]byte
}

// Reset clears the log entry for reuse in the pool
// Reset clears the log entry for reuse in the pool
//
//go:inline
func (e *LogEntry) Reset() {
	// Only reset what's absolutely necessary
	// Timestamp, Level, Message will be overwritten on next use

	// Reset static field count
	e.StaticFieldCount = 0
	e.StaticContextCount = 0

	// Reset dynamic fields slice but keep capacity
	if e.Fields != nil {
		e.Fields = e.Fields[:0]
	}

	// Reset dynamic context slice but keep capacity
	if e.Context != nil {
		e.Context = e.Context[:0]
	}

	// Reset flags
	e.hasFieldOverflow = false
	e.hasContextOverflow = false
	e.Error = nil
	e.ErrorMsg = ""
}

// LogEntryData represents the data structure for log entries
type LogEntryData struct {
	Level      LogLevel
	Message    string
	Timestamp  time.Time
	Fields     []TypedField
	Context    []TypedField
	Caller     string
	StackTrace string
}

// LogLevel represents the severity level of a log entry
type LogLevel uint64

// Log level constants
const (
	TraceLevel LogLevel = iota
	DebugLevel
	InfoLevel
	WarnLevel
	ErrorLevel
	FatalLevel
	PanicLevel
)

// String returns the string representation of the log level
func (l LogLevel) String() string {
	switch l {
	case TraceLevel:
		return "TRACE"
	case DebugLevel:
		return "DEBUG"
	case InfoLevel:
		return "INFO"
	case WarnLevel:
		return "WARN"
	case ErrorLevel:
		return "ERROR"
	case FatalLevel:
		return "FATAL"
	case PanicLevel:
		return "PANIC"
	default:
		return "UNKNOWN"
	}
}

// Priority returns the priority value for sorting
func (l LogLevel) Priority() int {
	return int(l)
}

// IsDefined checks if the log level is valid
func (l LogLevel) IsDefined() bool {
	return l >= TraceLevel && l <= PanicLevel
}

// IsValid checks if the log level is valid (alias for IsDefined)
func (l LogLevel) IsValid() bool {
	return l.IsDefined()
}

// SetTimestamp sets the timestamp for the log entry
func (e *LogEntry) SetTimestamp(timestamp time.Time) {
	e.Timestamp = timestamp
	e.TimestampUnix = timestamp.UnixNano()
}

// SetLevel sets the log level for the log entry
func (e *LogEntry) SetLevel(level LogLevel) {
	e.Level = level
}
