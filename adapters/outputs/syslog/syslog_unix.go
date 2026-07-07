//go:build !windows
// +build !windows

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

package syslog

import (
	"errors"
	"log/syslog"
	"sync"

	"github.com/go-gen-ecosystem/halolog/core"
	"github.com/go-gen-ecosystem/halolog/types"
	"github.com/go-gen-ecosystem/halolog/utils"
)

// Pre-allocated errors for zero-allocation design
var (
	ErrSyslogNilEntry = errors.New("cannot write nil entry")
	ErrSyslogNotInit  = errors.New("syslog writer not initialized")
	ErrSyslogFormat   = errors.New("failed to format entry")
	ErrSyslogWrite    = errors.New("failed to write to syslog")
)

// SyslogAdapter implements the Adapter interface for syslog output.
//
//nolint:revive // stable cross-platform public API name; renaming would break the windows build twin and callers
type SyslogAdapter struct {
	mu        sync.RWMutex
	writer    *syslog.Writer
	formatter Formatter
	facility  syslog.Priority
	tag       string
	lastError error // Store initialization errors
}

// SyslogAdapterOptions contains configuration options for SyslogAdapter.
//
//nolint:revive // stable cross-platform public API name; renaming would break the windows build twin and callers
type SyslogAdapterOptions struct {
	Network   string // "", "tcp", "udp"
	Address   string // "", "host:port"
	Facility  syslog.Priority
	Tag       string
	Formatter Formatter
}

// NewSyslogAdapter creates a new syslog adapter
func NewSyslogAdapter(tag string) *SyslogAdapter {
	return NewSyslogAdapterWithOptions(&SyslogAdapterOptions{
		Network:   "",
		Address:   "",
		Facility:  syslog.LOG_LOCAL0,
		Tag:       tag,
		Formatter: NewTextFormatter(),
	})
}

// NewSyslogAdapterWithOptions creates a new syslog adapter with custom options
func NewSyslogAdapterWithOptions(options *SyslogAdapterOptions) *SyslogAdapter {
	adapter := &SyslogAdapter{
		formatter: options.Formatter,
		facility:  options.Facility,
		tag:       options.Tag,
	}

	// Create syslog writer
	var writer *syslog.Writer
	var err error

	if options.Network != "" && options.Address != "" {
		writer, err = syslog.Dial(options.Network, options.Address, options.Facility, options.Tag)
	} else {
		writer, err = syslog.New(options.Facility, options.Tag)
	}

	if err != nil {
		// Store error - adapter will return it on first write
		adapter.lastError = err
	} else {
		adapter.writer = writer
	}

	return adapter
}

// Name returns the name of this adapter
func (a *SyslogAdapter) Name() string {
	return "SyslogAdapter"
}

// Write writes a log entry to syslog
func (a *SyslogAdapter) Write(entry *types.LogEntry) error {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.writer == nil {
		if a.lastError != nil {
			return errors.New(utils.FormatErrorMsg(
				"syslog writer not initialized", a.lastError,
			))
		}
		return errors.New("syslog writer not initialized")
	}

	// Format the entry using zero-allocation formatter
	// Pre-allocate buffer based on formatter's estimated size
	estimatedSize := a.formatter.EstimatedSize()
	buffer := make([]byte, 0, estimatedSize)
	formatted := a.formatter.Format(entry, buffer)

	// Write to appropriate syslog level
	// Convert []byte to string for syslog writer
	formattedStr := string(formatted)
	switch entry.Level {
	case DebugLevel:
		return a.writer.Debug(formattedStr)
	case InfoLevel:
		return a.writer.Info(formattedStr)
	case WarnLevel:
		return a.writer.Warning(formattedStr)
	case ErrorLevel:
		return a.writer.Err(formattedStr)
	case FatalLevel:
		return a.writer.Crit(formattedStr)
	case PanicLevel:
		return a.writer.Crit(formattedStr)
	default:
		return a.writer.Info(formattedStr)
	}
}

// Flush flushes the adapter (no-op for syslog)
func (a *SyslogAdapter) Flush() error {
	// Syslog doesn't have explicit flush, but we can sync
	return nil
}

// Close closes the syslog connection
func (a *SyslogAdapter) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.writer != nil {
		err := a.writer.Close()
		a.writer = nil
		return err
	}
	return nil
}

// SetFormatter sets the formatter for this adapter
func (a *SyslogAdapter) SetFormatter(formatter core.Formatter) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.formatter = formatter
}

// SetFacility sets the syslog facility
func (a *SyslogAdapter) SetFacility(facility syslog.Priority) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.facility = facility
}

// GetFacility returns the current syslog facility
func (a *SyslogAdapter) GetFacility() syslog.Priority {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.facility
}

// SetTag sets the syslog tag
func (a *SyslogAdapter) SetTag(tag string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.tag = tag
}

// GetTag returns the current syslog tag
func (a *SyslogAdapter) GetTag() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.tag
}

// Health implements types.InternalAdapter
func (a *SyslogAdapter) Health() error {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.writer == nil {
		return ErrSyslogNotInit
	}
	return nil
}

// WriteZero implements types.InternalAdapter
func (a *SyslogAdapter) WriteZero(entry *types.LogEntry) error {
	if entry == nil {
		return ErrSyslogNilEntry
	}

	// WriteZero just delegates to Write since syslog adapter doesn't need special zero-allocation handling
	return a.Write(entry)
}
