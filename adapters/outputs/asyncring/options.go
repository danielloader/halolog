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

// @author Admilson B. F. Cossa

package asyncring

import (
	"errors"
	"io"
	"time"

	"github.com/go-gen-ecosystem/halolog/types"
)

// OnFull selects what a producer does when the ring has no free slot.
type OnFull int

const (
	// Drop discards the record and increments the adapter's dropped counter. This
	// keeps the producer wait-free under overload — the default.
	Drop OnFull = iota
	// Block makes the producer spin until a slot frees (or the adapter closes).
	// It bounds memory and never loses records, at the cost of caller latency
	// under sustained overload.
	Block
)

// Default option values. They are documented so callers can reason about the
// memory/latency trade-off; nothing is hardcoded on the hot path.
const (
	defaultCapacity      = 1024
	defaultBatchSize     = 256
	defaultFlushInterval = time.Millisecond
)

var (
	// ErrNoWriter is returned when Options.Writer is nil.
	ErrNoWriter = errors.New("asyncring: Writer is required")
	// ErrNoFormatter is returned when Options.Formatter is nil.
	ErrNoFormatter = errors.New("asyncring: Formatter is required")
)

// Options configures a RingAdapter.
type Options struct {
	// Writer receives the serialized, newline-terminated records from the
	// background goroutine. Required. It is only ever called from that single
	// goroutine, so it need not be safe for concurrent use.
	Writer io.Writer

	// Formatter serializes each record. Required.
	Formatter types.Formatter

	// Capacity is the number of buffered records; it is rounded up to a power of
	// two. Memory use is roughly Capacity * per-slot size. Zero uses
	// defaultCapacity.
	Capacity int

	// OnFull is the overflow policy. The zero value is Drop.
	OnFull OnFull

	// BatchSize caps how many records the writer serializes before flushing to
	// Writer. Zero uses defaultBatchSize.
	BatchSize int

	// FlushInterval bounds how long a record may wait before the idle writer
	// flushes it. Zero uses defaultFlushInterval.
	FlushInterval time.Duration
}

// withDefaults returns a copy of o with zero fields replaced by defaults, after
// validating the required fields.
func (o Options) withDefaults() (Options, error) {
	if o.Writer == nil {
		return o, ErrNoWriter
	}
	if o.Formatter == nil {
		return o, ErrNoFormatter
	}
	if o.Capacity <= 0 {
		o.Capacity = defaultCapacity
	}
	if o.BatchSize <= 0 {
		o.BatchSize = defaultBatchSize
	}
	if o.FlushInterval <= 0 {
		o.FlushInterval = defaultFlushInterval
	}
	return o, nil
}
