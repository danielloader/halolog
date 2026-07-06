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

package json

import (
	"fmt"
	"testing"
	"time"

	"github.com/go-gen-ecosystem/halolog/types"
)

func TestDemoFormatterOutput(t *testing.T) {
	formatter := NewJsonFormatter()

	// Test different scenarios to show what the formatter produces
	tests := []struct {
		name  string
		entry *types.LogEntry
	}{
		{
			name: "Simple Info",
			entry: &types.LogEntry{
				Timestamp:     time.Unix(1700000000, 0),
				TimestampUnix: 1700000000,
				Level:         types.InfoLevel,
				Message:       "hello world",
			},
		},
		{
			name: "With File Info",
			entry: &types.LogEntry{
				Timestamp:     time.Unix(1700000000, 0),
				TimestampUnix: 1700000000,
				Level:         types.InfoLevel,
				Message:       "processing request",
				File:          "handler.go",
				Line:          42,
			},
		},
		{
			name: "With Fields",
			entry: &types.LogEntry{
				Timestamp:     time.Unix(1700000000, 0),
				TimestampUnix: 1700000000,
				Level:         types.InfoLevel,
				Message:       "user login",
				File:          "auth.go",
				Line:          156,
				Fields: []types.TypedFieldData{
					{Key: "user_id", Value: "12345"},
					{Key: "ip_address", Value: "192.168.1.1"},
					{Key: "user_agent", Value: "Mozilla/5.0"},
				},
			},
		},
		{
			name: "Error with Details",
			entry: &types.LogEntry{
				Timestamp:     time.Unix(1700000000, 0),
				TimestampUnix: 1700000000,
				Level:         types.ErrorLevel,
				Message:       "database connection failed",
				File:          "db.go",
				Line:          89,
				ErrorMsg:      "connection timeout after 3 retries",
				Fields: []types.TypedFieldData{
					{Key: "error_type", Value: "timeout"},
					{Key: "retry_count", Value: "3"},
					{Key: "max_retries", Value: "5"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dst := make([]byte, 0, 512)
			result := formatter.Format(tt.entry, dst)
			fmt.Printf("\n=== %s ===\n", tt.name)
			fmt.Printf("Input: Level=%s, Message=%q\n", tt.entry.Level, tt.entry.Message)
			fmt.Printf("Output: %s\n", string(result))
			fmt.Printf("Length: %d bytes\n", len(result))
		})
	}
}

func BenchmarkDetailedAnalysis(b *testing.B) {
	formatter := NewJsonFormatter()

	// Create a realistic log entry
	entry := &types.LogEntry{
		Timestamp:     time.Unix(1700000000, 0),
		TimestampUnix: 1700000000,
		Level:         types.InfoLevel,
		Message:       "API request processed successfully",
		File:          "api.go",
		Line:          234,
		Fields: []types.TypedFieldData{
			{Key: "method", Value: "GET"},
			{Key: "path", Value: "/api/users"},
			{Key: "status_code", Value: "200"},
			{Key: "duration_ms", Value: "45.2"},
			{Key: "user_id", Value: "67890"},
		},
	}

	dst := make([]byte, 0, 512)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		result := formatter.Format(entry, dst)
		dst = result[:0]
	}

	// Show sample output after benchmark
	if b.N > 0 {
		result := formatter.Format(entry, dst)
		fmt.Printf("\nSample output: %s\n", string(result))
		fmt.Printf("Length: %d bytes\n", len(result))
	}
}
