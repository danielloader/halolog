//go:build amd64 || arm64
// +build amd64 arm64

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
// Package adapters provides output adapters
// Author: Admilson B. F. Cossa

package text

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/go-gen-ecosystem/halolog/types"
)

func TestNewOptimizedTextFormatter(t *testing.T) {
	f := NewTextFormatter()
	if f == nil {
		t.Fatal("Expected NewTextFormatter to return non-nil")
	}
}

func TestFormat_NilEntry(t *testing.T) {
	f := NewTextFormatter()
	dst := make([]byte, 0, 1024)

	// Should return empty dst (length 0)
	res := f.Format(nil, dst)
	if len(res) != 0 {
		t.Errorf("Expected 0 bytes for nil entry, got %d", len(res))
	}
}

func TestFormat_Levels(t *testing.T) {
	f := NewTextFormatter()
	dst := make([]byte, 0, 1024)

	// We test all switch cases to ensure 100% branch coverage
	tests := []struct {
		name     string
		level    types.LogLevel
		expected string
	}{
		{"Info", types.InfoLevel, "INFO  - "},
		{"Error", types.ErrorLevel, "ERROR - "},
		{"Debug", types.DebugLevel, "DEBUG - "},
		{"Warn", types.WarnLevel, "WARN  - "},
		{"Trace", types.TraceLevel, "TRACE - "},
		{"Fatal", types.FatalLevel, "FATAL - "},
		{"Unknown", types.LogLevel(99), "UNK   - "}, // Default case
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := &types.LogEntry{
				Level:   tt.level,
				Message: "test",
			}

			res := f.Format(entry, dst[:0])
			str := string(res)

			if !strings.Contains(str, tt.expected) {
				t.Errorf("Level %s: expected to contain %q, got %q", tt.name, tt.expected, str)
			}

			// Verify timestamp structure [YYYY...
			if !strings.HasPrefix(str, "[") {
				t.Error("Expected output to start with timestamp bracket [")
			}
		})
	}
}

func TestFormat_CallerAndLines(t *testing.T) {
	f := NewTextFormatter()
	dst := make([]byte, 0, 1024)

	tests := []struct {
		name        string
		file        string
		line        int
		wantCaller  string
		description string
	}{
		{
			name:       "Deep Path Small Line",
			file:       "/home/user/project/main.go",
			line:       42,
			wantCaller: " [main.go:42]", // Cached int path (<1000)
		},
		{
			name:       "Root File Large Line",
			file:       "server.go",
			line:       1500,
			wantCaller: " [server.go:1500]", // Strconv path (>=1000)
		},
		{
			name:       "Empty File",
			file:       "",
			line:       10,
			wantCaller: "", // Should not append caller info
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := &types.LogEntry{
				Level:   types.InfoLevel,
				Message: "msg",
				File:    tt.file,
				Line:    tt.line,
			}

			res := f.Format(entry, dst[:0])
			str := string(res)

			if tt.wantCaller != "" {
				if !strings.Contains(str, tt.wantCaller) {
					t.Errorf("Expected caller %q, got output %q", tt.wantCaller, str)
				}
			} else {
				// If empty file, verify no brackets or colons related to caller are added
				// Note: Timestamp has brackets, so we check for the specific caller format
				if strings.Contains(str, "go:") {
					t.Error("Expected no caller info, but found file-like string")
				}
			}
		})
	}
}

func TestFormat_MessageAppend(t *testing.T) {
	f := NewTextFormatter()
	dst := make([]byte, 0, 1024)

	msg := "Hello World"
	entry := &types.LogEntry{
		Level:   types.InfoLevel,
		Message: msg,
	}

	res := f.Format(entry, dst)
	if !bytes.Contains(res, []byte(msg)) {
		t.Error("Result should contain the message")
	}
	if !bytes.HasSuffix(res, []byte("\n")) {
		t.Error("Result should end with newline")
	}
}

// TestBackgroundTicker_Coverage ensures the init() goroutine loop is covered.
// This is strictly for coverage tools to register the inside of the generic ticker loop.
func TestBackgroundTicker_Coverage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping ticker coverage in short mode")
	}

	// The init() function started the ticker.
	// We sleep slightly longer than 1 second to ensure the ticker fires
	// and the code inside the 'range ticker.C' executes at least once.
	time.Sleep(1100 * time.Millisecond)

	// While we can't easily assert the internal state changed without exposing it,
	// simply running this test allows the coverage tool to mark the lines as executed.

	// Verify formatting still works after a tick
	f := NewTextFormatter()
	entry := &types.LogEntry{Level: types.InfoLevel}
	res := f.Format(entry, make([]byte, 0, 100))

	if len(res) == 0 {
		t.Error("Formatter stopped working after ticker update")
	}
}

// Benchmark to verify the <12ns claim (Optional, but good practice in formatter tests)
func BenchmarkFormat_HotPath(b *testing.B) {
	f := NewTextFormatter()
	entry := &types.LogEntry{
		Level:   types.InfoLevel,
		Message: "benchmark",
	}
	dst := make([]byte, 0, 1024)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		dst = f.Format(entry, dst[:0])
	}
}
