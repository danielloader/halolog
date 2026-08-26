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

// Package console provides a console/stdout output adapter for HaloLog.
//
// @author Admilson B. F. Cossa
package console

import (
	"io"
	"os"
	"sync"
	"sync/atomic"

	"github.com/go-gen-ecosystem/halolog/types"
)

// Adapter writes formatted log entries to an io.Writer (os.Stdout by default).
// It is safe for concurrent use: a mutex serialises the underlying writes so
// entries from concurrent goroutines are never interleaved, while formatting
// happens OUTSIDE that lock into a pooled buffer, keeping the critical section
// to the single Write call. The formatter is held behind an atomic pointer so
// SetFormatter can never tear the interface value a concurrent writer is
// reading, and DirectEncoder is a lock-free load on the hot path.
type Adapter struct {
	mu        sync.Mutex // serialises writes only — never held while formatting
	w         io.Writer
	formatter atomic.Pointer[types.Formatter]
	bufPool   sync.Pool
	closed    atomic.Bool
}

// New returns a console adapter writing to os.Stdout with the default formatter.
func New() *Adapter { return NewWithWriter(os.Stdout, nil) }

// NewWithWriter returns a console adapter writing to w (defaults to os.Stdout
// when nil) using formatter (defaults to the standard console formatter).
func NewWithWriter(w io.Writer, formatter types.Formatter) *Adapter {
	if w == nil {
		w = os.Stdout
	}
	if formatter == nil {
		formatter = types.NewDefaultConsoleFormatter()
	}
	a := &Adapter{w: w}
	a.formatter.Store(&formatter)
	a.bufPool.New = func() interface{} {
		b := make([]byte, 0, 256)
		return &b
	}
	return a
}

// Name identifies the adapter.
func (a *Adapter) Name() string { return "console" }

// Write formats and writes a single entry, appending a newline if the formatter
// did not. A nil entry is a no-op; writing after Close is a no-op.
func (a *Adapter) Write(entry *types.LogEntry) error {
	if entry == nil || a.closed.Load() {
		return nil
	}

	// Format outside the lock: the entry belongs to the caller for the whole
	// call either way, and keeping serialization to the write alone means
	// concurrent loggers contend for nanoseconds, not for encoding time.
	formatter := *a.formatter.Load()
	bufPtr := a.bufPool.Get().(*[]byte)
	buf := (*bufPtr)[:0]
	buf = formatter.Format(entry, buf)
	if len(buf) == 0 || buf[len(buf)-1] != '\n' {
		buf = append(buf, '\n')
	}

	var err error
	a.mu.Lock()
	if !a.closed.Load() {
		_, err = a.w.Write(buf)
	}
	a.mu.Unlock()

	*bufPtr = buf
	a.bufPool.Put(bufPtr)
	return err
}

// WriteZero is the zero-allocation write path; for the console it is identical
// to Write (both take the pooled formatting buffer).
func (a *Adapter) WriteZero(entry *types.LogEntry) error { return a.Write(entry) }

// WriteRaw writes an already-formatted line (including its trailing newline)
// under the same mutex as Write, so raw and formatted lines never interleave.
// It implements types.RawWriter for the direct-append fast path.
func (a *Adapter) WriteRaw(line []byte) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.closed.Load() {
		return nil
	}
	_, err := a.w.Write(line)
	return err
}

// DirectEncoder implements types.DirectCapableAdapter: it exposes the current
// formatter when that formatter can direct-encode, else nil. The formatter is
// read with a single atomic load — no lock — so swapping to a non-capable
// formatter via SetFormatter still disables the fast path for subsequent
// lines, at zero cost to the ones in flight.
func (a *Adapter) DirectEncoder() types.DirectFieldEncoder {
	if enc, ok := (*a.formatter.Load()).(types.DirectFieldEncoder); ok {
		return enc
	}
	return nil
}

// Flush flushes the underlying writer if it supports flushing.
func (a *Adapter) Flush() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if f, ok := a.w.(interface{ Flush() error }); ok {
		return f.Flush()
	}
	return nil
}

// Close marks the adapter closed; subsequent writes are no-ops. The underlying
// writer (typically os.Stdout) is not closed.
func (a *Adapter) Close() error {
	a.closed.Store(true)
	return nil
}

// SetFormatter replaces the formatter used to render entries (nil is ignored).
// Safe to call concurrently with writes; lines already being formatted finish
// with the formatter they loaded.
func (a *Adapter) SetFormatter(formatter types.Formatter) {
	if formatter == nil {
		return
	}
	a.formatter.Store(&formatter)
}

// Health reports adapter health; a closed adapter is unhealthy.
func (a *Adapter) Health() error {
	if a.closed.Load() {
		return io.ErrClosedPipe
	}
	return nil
}
