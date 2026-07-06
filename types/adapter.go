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
	"errors"
	"io"
	"sync/atomic"
)

type AdapterType uint32

const (
	AdapterTypeConsole AdapterType = 1 << iota
	AdapterTypeFile
	AdapterTypeHTTP
	AdapterTypeSyslog
	AdapterTypeCustom
)

// Adapter represents a log output adapter interface
type Adapter interface {
	Name() string
	Write(entry *LogEntry) error
	WriteZero(entry *LogEntry) error
	Flush() error
	Close() error
	SetFormatter(formatter Formatter)
	Health() error
}

// AsyncAdapter represents an asynchronous adapter interface
type AsyncAdapter interface {
	Adapter
	SetBufferSize(size int)
	SetFlushInterval(interval int64)
	Flush() error
}

// BatchAdapter represents a batch-writing adapter interface
type BatchAdapter interface {
	Adapter
	SetBatchSize(size int)
	SetBatchTimeout(timeout int64)
	FlushBatch() error
}

// FormattedAdapter represents an adapter with formatting capabilities
type FormattedAdapter interface {
	Adapter
	GetFormatter() Formatter
}

// WriterAdapter represents an adapter that writes to an io.Writer
type WriterAdapter interface {
	Adapter
	SetWriter(writer io.Writer) error
	GetWriter() io.Writer
}

// Output represents a log output destination
type Output interface {
	Write(entry *LogEntry) error
	Close() error
}

// AdapterRegistry tracks registered adapter types using bitflags for O(1) lookup
// This prevents duplicate registrations without map allocations
type AdapterRegistry struct {
	flags atomic.Uint32 // Bitflags for quick type checking
}

// HasAdapter checks if adapter type is registered (O(1), zero-allocation)
func (r *AdapterRegistry) HasAdapter(adapterType AdapterType) bool {
	return (r.flags.Load() & uint32(adapterType)) != 0
}

// RegisterAdapter marks adapter type as registered (O(1), zero-allocation)
func (r *AdapterRegistry) RegisterAdapter(adapterType AdapterType) {
	for {
		old := r.flags.Load()
		new := old | uint32(adapterType)
		if r.flags.CompareAndSwap(old, new) {
			return
		}
	}
}

// Common adapter errors
var (
	ErrAdapterWriteFailed       = errors.New("adapter write failed")
	ErrAdapterFlushFailed       = errors.New("adapter flush failed")
	ErrAdapterCloseFailed       = errors.New("adapter close failed")
	ErrAdapterHealthCheckFailed = errors.New("adapter health check failed")
	ErrAdapterClosed            = errors.New("adapter is closed")
)

// UnregisterAdapter removes adapter type registration (O(1), zero-allocation)
func (r *AdapterRegistry) UnregisterAdapter(adapterType AdapterType) {
	for {
		old := r.flags.Load()
		new := old &^ uint32(adapterType) // Clear bit
		if r.flags.CompareAndSwap(old, new) {
			return
		}
	}
}

// WriteFunc is a function type for writing log entries
type WriteFunc func(entry *LogEntry) error

// FuncAdapter adapts a WriteFunc to the Adapter interface
type FuncAdapter struct {
	WriteFunc WriteFunc
}

func (a *FuncAdapter) Name() string                    { return "func_adapter" }
func (a *FuncAdapter) Write(entry *LogEntry) error     { return a.WriteFunc(entry) }
func (a *FuncAdapter) WriteZero(entry *LogEntry) error { return a.WriteFunc(entry) }
func (a *FuncAdapter) Flush() error                    { return nil }
func (a *FuncAdapter) Close() error                    { return nil }
func (a *FuncAdapter) SetFormatter(f Formatter)        {}
func (a *FuncAdapter) Health() error                   { return nil }
