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

package discard

import (
	"github.com/go-gen-ecosystem/halolog/types"
)

// Adapter is a no-op adapter that discards all log entries
type Adapter struct{}

// New creates a new discard adapter
func New() *Adapter {
	return &Adapter{}
}

// Write implements the Adapter interface - discards all data
func (a *Adapter) Write(entry *types.LogEntry) error {
	return nil
}

// WriteZero implements the Adapter interface - discards all data with zero-copy
func (a *Adapter) WriteZero(entry *types.LogEntry) error {
	return nil
}

// Name returns the adapter name
func (a *Adapter) Name() string {
	return "discard"
}

// Flush implements the Adapter interface - no-op for discard
func (a *Adapter) Flush() error {
	return nil
}

// Close implements the Adapter interface - no-op for discard
func (a *Adapter) Close() error {
	return nil
}

// SetFormatter implements the Adapter interface - no-op for discard
func (a *Adapter) SetFormatter(formatter types.Formatter) {
	// No-op for discard adapter
}

// Health implements the Adapter interface - always healthy
func (a *Adapter) Health() error {
	return nil
}