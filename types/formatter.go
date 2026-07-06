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

// Formatter represents a log entry formatter interface (OPTIMIZED for zero-allocation)
// The dst parameter is a reusable buffer to avoid allocations.
// Formatters should append to dst and return the result.
type Formatter interface {
	// Format formats a log entry into the provided buffer
	// entry: pointer to log entry (avoids copy)
	// dst: destination buffer (reusable, avoids allocation)
	// returns: formatted bytes (may be dst or a new buffer if dst was too small)
	Format(entry *LogEntry, dst []byte) []byte

	// EstimatedSize returns the estimated size for buffer pre-allocation
	EstimatedSize() int

	// Reset resets any internal state (for pooling)
	Reset()
}

// DefaultConsoleFormatter provides simple console formatting
type DefaultConsoleFormatter struct {
	TimeFormat string
	ShowLevel  bool
	ShowTime   bool
}

// NewDefaultConsoleFormatter creates a new default console formatter
func NewDefaultConsoleFormatter() *DefaultConsoleFormatter {
	return &DefaultConsoleFormatter{
		TimeFormat: "15:04:05",
		ShowLevel:  true,
		ShowTime:   true,
	}
}

// Format formats a log entry into bytes (zero-allocation)
func (f *DefaultConsoleFormatter) Format(entry *LogEntry, dst []byte) []byte {
	if entry == nil {
		return dst[:0]
	}

	// Reset dst
	dst = dst[:0]

	// Simple format: [LEVEL] message
	dst = append(dst, '[')
	dst = append(dst, entry.Level.String()...)
	dst = append(dst, ']')
	dst = append(dst, ' ')
	dst = append(dst, entry.Message...)
	dst = append(dst, '\n')

	return dst
}

// EstimatedSize returns estimated buffer size
func (f *DefaultConsoleFormatter) EstimatedSize() int {
	return 256
}

// Reset implements Formatter interface
func (f *DefaultConsoleFormatter) Reset() {
	// Stateless formatter - no-op
}
