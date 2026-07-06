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

func BenchmarkDemoOutput(b *testing.B) {
	formatter := NewJsonFormatter()

	// Create a realistic log entry for performance analysis
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

	// Show output before benchmark
	result := formatter.Format(entry, dst)
	fmt.Printf("Sample Output (%d bytes):\n", len(result))
	fmt.Printf("%s\n\n", string(result))

	// Performance analysis
	fmt.Printf("Performance Breakdown:\n")
	fmt.Printf("- Message length: %d bytes\n", len(entry.Message))
	fmt.Printf("- Fields count: %d\n", len(entry.Fields))
	fmt.Printf("- Total output: %d bytes\n", len(result))
	fmt.Printf("- Throughput: ~%.1f MB/s (based on 37.25 ns/op)\n\n", float64(len(result))/37.25*1000)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		result := formatter.Format(entry, dst)
		dst = result[:0]
	}
}
