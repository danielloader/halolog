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

package adapters

import (
	// 	core "github.com/go-gen-ecosystem/halolog/sndabox_"
	"time"

	"github.com/go-gen-ecosystem/halolog/types"
)

// RotationConfig configures file rotation
type RotationConfig struct {
	MaxSize    int64
	MaxBackups int
	MaxAge     time.Duration
}

// ConsoleAdapter provides console-based logging
type ConsoleAdapter struct {
	types.Adapter
}

// NewConsoleAdapterWithTarget creates a new console adapter with specified target
// func NewConsoleAdapterWithTarget(target string) *ConsoleAdapter {
// 	var writer io.Writer
// 	switch target {
// 	case "stderr":
// 		writer = os.Stderr
// 	default:
// 		writer = os.Stdout
// 	}
// 	return &ConsoleAdapter{
// 		Adapter: types.DefaultConsoleAdapter(writer, nil),
// 	}
// }

// FileAdapter represents a file adapter (placeholder for compilation)
type FileAdapter struct {
	types.Adapter
}

// NewFileAdapter creates a file adapter (placeholder for compilation)
func NewFileAdapter(path string, config *RotationConfig) (*FileAdapter, error) {
	return &FileAdapter{}, nil
}

// SetFormatter sets the formatter for the adapter
func (f *FileAdapter) SetFormatter(formatter interface{}) {}

// AdapterOption configures adapters (functional options pattern)
type AdapterOption func(types.Adapter) error

// PerfNullAdapter is a high-performance null adapter for benchmarking and testing
type PerfNullAdapter struct{}

// Write implements the Adapter interface - no-op for performance
func (p *PerfNullAdapter) Write(entry *types.LogEntry) error {
	return nil
}

// WriteZero implements the Adapter interface - no-op for performance
func (p *PerfNullAdapter) WriteZero(entry *types.LogEntry) error {
	return nil
}

// Name returns the adapter name
func (p *PerfNullAdapter) Name() string {
	return "perf_null"
}

// Close implements the Adapter interface - no-op
func (p *PerfNullAdapter) Close() error {
	return nil
}

// Flush implements the Adapter interface - no-op
func (p *PerfNullAdapter) Flush() error {
	return nil
}

// Health implements the Adapter interface - no-op
func (p *PerfNullAdapter) Health() error {
	return nil
}

// SetFormatter implements the Adapter interface - no-op
func (p *PerfNullAdapter) SetFormatter(formatter types.Formatter) {
	// No-op for null adapter
}
