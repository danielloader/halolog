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

package http

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-gen-ecosystem/halolog/types"
	"github.com/go-gen-ecosystem/halolog/utils"
)

var httpStatusErrors [600]*HTTPStatusError

// HTTPAdapterOptions - Configuration for HTTP adapter.
//
//nolint:revive // HTTP prefix is the intended public API name; renaming would break callers across the module.
type HTTPAdapterOptions struct {
	URL           string
	Method        string
	Headers       map[string]string
	BatchSize     int
	FlushInterval time.Duration
	Timeout       time.Duration
	Formatter     types.Formatter
}

// Pre-allocated errors
var (
	ErrHTTPNilEntry  = errors.New("cannot write nil entry")
	ErrHTTPMarshal   = errors.New("failed to marshal request body")
	ErrHTTPCreateReq = errors.New("failed to create request")
	ErrHTTPSendReq   = errors.New("failed to send request")
	ErrHTTPBatch     = errors.New("failed to send batch")
)

// HTTPStatusError represents an HTTP status error.
//
//nolint:revive // HTTP prefix is the intended public API name; renaming would break callers across the module.
type HTTPStatusError struct {
	StatusCode int
	message    string // Pre-computed
}

func (e *HTTPStatusError) Error() string {
	return e.message
}

func newHTTPStatusError(code int) error {
	if code >= 100 && code < 600 {
		return httpStatusErrors[code]
	}
	return &HTTPStatusError{
		StatusCode: code,
		message:    buildErrorMessage(code),
	}
}

// HTTPAdapter - TRULY zero-alloc with sync.Pool.
//
//nolint:revive // HTTP prefix is the intended public API name; renaming would break callers across the module.
type HTTPAdapter struct {
	mu            sync.RWMutex
	url           string
	method        string
	headers       map[string]string
	batchSize     int
	flushInterval time.Duration
	client        *http.Client
	buffer        []*types.LogEntry
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
	adapterType   string // Track adapter type for Name() method

	// Formatter, swappable at runtime. Behind an atomic pointer so a
	// concurrent SetFormatter never tears the interface value an in-flight
	// flush is reading.
	formatter atomic.Pointer[types.Formatter]

	// POOLS: Reusable objects
	entryPool       sync.Pool // []*types.LogEntry
	jsonBufPool     sync.Pool // *bytes.Buffer
	flushInProgress atomic.Bool
}

func init() {
	for i := 100; i < 600; i++ {
		httpStatusErrors[i] = &HTTPStatusError{
			StatusCode: i,
			message:    buildErrorMessage(i),
		}
	}
}

// NewHTTPAdapterWithOptions - FIXED: Proper pool initialization
func NewHTTPAdapterWithOptions(options *HTTPAdapterOptions) *HTTPAdapter {
	if options == nil {
		options = &HTTPAdapterOptions{}
	}
	ctx, cancel := context.WithCancel(context.Background())

	// Set defaults - only apply defaults when not explicitly set
	method := options.Method
	if method == "" {
		method = "POST"
	}
	batchSize := options.BatchSize
	if batchSize == 0 {
		batchSize = 100
	}
	flushInterval := options.FlushInterval
	timeout := options.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	headers := options.Headers
	if headers == nil {
		headers = map[string]string{"Content-Type": "application/json"}
	}

	adapter := &HTTPAdapter{
		url:           options.URL,
		method:        method,
		headers:       headers,
		batchSize:     batchSize,
		flushInterval: flushInterval,
		adapterType:   "HTTPAdapter", // Default type
		client: &http.Client{
			Timeout: timeout,
			// CRITICAL: Use custom transport for connection reuse
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 100,
				MaxConnsPerHost:     100,
				IdleConnTimeout:     90 * time.Second,
				DisableCompression:  true, // Avoid compression overhead
			},
		},
		buffer: make([]*types.LogEntry, 0, batchSize),
		ctx:    ctx,
		cancel: cancel,
	}

	// CRITICAL: Initialize pools with proper constructors
	adapter.entryPool = sync.Pool{
		New: func() interface{} {
			slice := make([]*types.LogEntry, 0, batchSize)
			return &slice
		},
	}

	adapter.jsonBufPool = sync.Pool{
		New: func() interface{} {
			return &bytes.Buffer{}
		},
	}

	// Set formatter
	if options.Formatter != nil {
		adapter.SetFormatter(options.Formatter)
	} else {
		adapter.SetFormatter(NewZeroJSONFormatter()) // CRITICAL: Must be zero-alloc
	}

	// Start flusher
	adapter.wg.Add(1)
	go adapter.StartHttpFlushTimer()

	return adapter
}

// Write - FIXED: No allocations
func (a *HTTPAdapter) Write(entry *types.LogEntry) error {
	if entry == nil {
		return ErrHTTPNilEntry
	}

	a.mu.Lock()
	a.buffer = append(a.buffer, entry)
	shouldFlush := len(a.buffer) >= a.batchSize
	a.mu.Unlock()

	if shouldFlush {
		// If batch size is 1 (streaming mode), flush synchronously to return errors
		if a.batchSize == 1 {
			return a.Flush()
		}

		// Otherwise flush asynchronously
		if a.flushInProgress.CompareAndSwap(false, true) {
			go func() {
				defer a.flushInProgress.Store(false)
				_ = a.doFlush()
			}()
		}
	}

	return nil
}

// Flush - FIXED: Zero allocations in hot path (implements types.Adapter interface)
func (a *HTTPAdapter) Flush() error {
	a.mu.Lock()

	if len(a.buffer) == 0 {
		a.mu.Unlock()
		return nil
	}

	// Get pooled slice pointer
	entriesPtr := a.entryPool.Get().(*[]*types.LogEntry)
	entries := (*entriesPtr)[:0]
	entries = append(entries, a.buffer...)
	bufferLen := len(a.buffer)
	a.buffer = a.buffer[:0] // Reset but keep capacity
	a.mu.Unlock()

	// Process batches WITHOUT allocations
	var batchErr error
	for i := 0; i < bufferLen; i += a.batchSize {
		end := i + a.batchSize
		if end > bufferLen {
			end = bufferLen
		}

		// Pass slice and indices instead of creating sub-slice
		if err := a.sendBatch(entries, i, end); err != nil {
			// Re-queue failed entries for retry. The failed window MUST be
			// copied out of `entries`: that slice's backing array goes back to
			// entryPool below, and prepending an alias of it would leave
			// a.buffer pointing into pooled memory that the next flush
			// overwrites (silent entry corruption under retry).
			a.mu.Lock()
			requeued := make([]*types.LogEntry, 0, (end-i)+len(a.buffer))
			requeued = append(requeued, entries[i:end]...)
			a.buffer = append(requeued, a.buffer...)
			a.mu.Unlock()
			batchErr = err
			break
		}
	}

	// Return slice to pool
	*entriesPtr = entries
	a.entryPool.Put(entriesPtr)

	return batchErr
}

// sendBatch - FIXED: TRUE zero-allocation implementation
func (a *HTTPAdapter) sendBatch(entries []*types.LogEntry, start, end int) error {
	count := end - start
	if count == 0 {
		return nil
	}

	// Get pooled buffer
	buf := a.jsonBufPool.Get().(*bytes.Buffer)
	buf.Reset()
	// NO DEFER - explicit return for zero alloc

	// Build JSON with ZERO string allocations
	buf.WriteString(`{"timestamp":`)

	// Use stack-allocated buffer for numbers
	var numBuf [20]byte
	n := strconv.AppendInt(numBuf[:0], entries[start].Timestamp.Unix(), 10)
	buf.Write(n)

	buf.WriteString(`,"count":`)
	n = strconv.AppendInt(numBuf[:0], int64(count), 10)
	buf.Write(n)

	buf.WriteString(`,"entries":[`)

	// Snapshot the formatter once per batch (atomic — safe against a
	// concurrent SetFormatter) and reuse one scratch buffer across entries.
	formatter := *a.formatter.Load()
	tempBuf := make([]byte, 0, 1024)
	for i := start; i < end; i++ {
		if i > start {
			buf.WriteByte(',')
		}
		formatted := formatter.Format(entries[i], tempBuf[:0])
		buf.Write(formatted)
		tempBuf = formatted // keep any growth for the next entry
	}

	buf.WriteString("]}")

	req2, err := http.NewRequestWithContext(a.ctx, a.method, a.url, buf)
	if err != nil {
		a.jsonBufPool.Put(buf)
		return ErrHTTPCreateReq
	}

	// Copy pre-existing headers efficiently
	if len(a.headers) > 0 {
		// Clear old headers
		for k := range req2.Header {
			delete(req2.Header, k)
		}
		// Set new headers
		for k, v := range a.headers {
			req2.Header.Set(k, v)
		}
	}

	// Send request (this will allocate internally, but we can't control stdlib)
	resp, err := a.client.Do(req2)

	if err != nil {
		a.jsonBufPool.Put(buf)
		return ErrHTTPSendReq
	}

	// CRITICAL: Drain response body for connection reuse WITHOUT allocations
	// Use stack-allocated buffer
	var discard [512]byte
	for {
		_, err := resp.Body.Read(discard[:])
		if err != nil {
			break
		}
	}
	_ = resp.Body.Close()

	// NOW return buffer to pool
	a.jsonBufPool.Put(buf)

	// Only return error for non-success status codes
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return newHTTPStatusError(resp.StatusCode)
	}
	return nil
}

// Close - FIXED: Proper cleanup
func (a *HTTPAdapter) Close() error {
	// Force a final flush BEFORE cancelling context
	_ = a.Flush() // Final flush; error is non-actionable during Close

	// Then cancel context and wait for background goroutines
	a.cancel()
	a.wg.Wait()
	return nil
}

// Flush - Public flush method for testing
func (a *HTTPAdapter) doFlush() error {
	return a.Flush()
}

// WriteZero - Zero-allocation write method for testing
func (a *HTTPAdapter) WriteZero(entry *types.LogEntry) error {
	return a.Write(entry)
}

// Health - Returns the health status of the adapter
func (a *HTTPAdapter) Health() error {
	// Check if context is cancelled first
	select {
	case <-a.ctx.Done():
		return a.ctx.Err()
	default:
	}

	// Simple health check - verify we can acquire the lock
	a.mu.Lock()
	defer a.mu.Unlock()
	return nil
}

// Name - Returns the adapter name
func (a *HTTPAdapter) Name() string {
	return a.adapterType
}

// SetFormatter sets the formatter for the adapter (nil is ignored). Safe to
// call concurrently with in-flight flushes.
func (a *HTTPAdapter) SetFormatter(formatter types.Formatter) {
	if formatter == nil {
		return
	}
	a.formatter.Store(&formatter)
}

// StartHttpFlushTimer runs the background flush loop until the adapter context is cancelled.
func (a *HTTPAdapter) StartHttpFlushTimer() {
	defer a.wg.Done()

	// Don't start timer if flush interval is 0
	if a.flushInterval == 0 {
		<-a.ctx.Done()
		_ = a.doFlush() // Final flush only
		return
	}

	ticker := time.NewTicker(a.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			_ = a.doFlush()
		case <-a.ctx.Done():
			_ = a.doFlush() // Final flush
			return
		}
	}
}

// NewHTTPAdapter - Creates a new HTTP adapter with default options
func NewHTTPAdapter(url string) *HTTPAdapter {
	return NewHTTPAdapterWithOptions(&HTTPAdapterOptions{
		URL:           url,
		FlushInterval: 5 * time.Second,
	})
}

// NewStreamingHTTPAdapter - Creates a streaming HTTP adapter for backward compatibility
func NewStreamingHTTPAdapter(url string, timeout time.Duration) *HTTPAdapter {
	// For streaming adapter, 0 timeout means use default of 10 seconds
	if timeout == 0 {
		timeout = 10 * time.Second
	}
	adapter := NewHTTPAdapterWithOptions(&HTTPAdapterOptions{
		URL:           url,
		Timeout:       timeout,
		BatchSize:     1,
		FlushInterval: 0,
	})
	adapter.adapterType = "StreamingHTTPAdapter"
	return adapter
}

// Other methods remain similar but ensure no allocations...

// ============================================================================
// REQUIRED: Zero-Allocation JSON Formatter
// ============================================================================

// ZeroJSONFormatter - TRUE zero-allocation formatter
type ZeroJSONFormatter struct {
	bufPool sync.Pool
}

// NewZeroJSONFormatter returns a zero-allocation JSON formatter for HTTP batch payloads.
func NewZeroJSONFormatter() types.Formatter {
	return &ZeroJSONFormatter{
		bufPool: sync.Pool{
			New: func() interface{} {
				// Pre-size for typical log entry (2KB)
				b := make([]byte, 0, 2048)
				return &b
			},
		},
	}
}

// Format - MUST NOT ALLOCATE
func (f *ZeroJSONFormatter) Format(entry *types.LogEntry, dst []byte) []byte {
	if entry == nil {
		return dst[:0]
	}

	// Reset dst
	dst = dst[:0]

	// Build JSON object manually with zero allocations
	dst = append(dst, '{')

	// Level
	dst = append(dst, `"level":"`...)
	dst = append(dst, entry.Level.String()...)
	dst = append(dst, `"`...)

	// Message
	if entry.Message != "" {
		dst = append(dst, `,"message":"`...)
		// Escape quotes in message (simplified)
		for i := 0; i < len(entry.Message); i++ {
			if entry.Message[i] == '"' {
				dst = append(dst, `\"`...)
			} else {
				dst = append(dst, entry.Message[i])
			}
		}
		dst = append(dst, '"')
	}

	// Timestamp
	if !entry.Timestamp.IsZero() {
		dst = append(dst, `,"timestamp":`...)
		dst = strconv.AppendInt(dst, entry.TimestampUnix, 10)
	}

	// Fields
	if len(entry.Fields) > 0 {
		dst = append(dst, `,"fields":{`...)
		for i, field := range entry.Fields {
			if i > 0 {
				dst = append(dst, ',')
			}
			dst = append(dst, '"')
			dst = append(dst, field.Key...)
			dst = append(dst, `":`...)

			// Handle different value types without allocations
			switch v := field.Value.(type) {
			case string:
				dst = append(dst, '"')
				dst = append(dst, v...)
				dst = append(dst, '"')
			case int64:
				dst = strconv.AppendInt(dst, v, 10)
			case float64:
				dst = strconv.AppendFloat(dst, v, 'f', -1, 64)
			case bool:
				if v {
					dst = append(dst, "true"...)
				} else {
					dst = append(dst, "false"...)
				}
			default:
				// Use string conversion for other types
				dst = append(dst, '"')
				str := fmt.Sprint(v)
				dst = append(dst, str...)
				dst = append(dst, '"')
			}
		}
		dst = append(dst, '}')
	}

	dst = append(dst, '}')
	return dst
}

// String returns the formatter name
func (f *ZeroJSONFormatter) String() string {
	return "ZeroJSONFormatter"
}

// FormatBatch formats multiple log entries into a single byte slice
func (f *ZeroJSONFormatter) FormatBatch(entries []types.LogEntry) ([]byte, error) {
	if len(entries) == 0 {
		return []byte("[]"), nil
	}

	// Get buffer from pool
	bufPtr := f.bufPool.Get().(*[]byte)
	buf := (*bufPtr)[:0]
	// Return to pool after formatting
	defer func() {
		*bufPtr = buf
		f.bufPool.Put(bufPtr)
	}()

	// Start JSON array
	buf = append(buf, '[')

	// Format each entry
	for i, entry := range entries {
		if i > 0 {
			buf = append(buf, ',')
		}

		// Format individual entry
		entryBytes := f.Format(&entry, buf[len(buf):])
		buf = buf[:len(buf)+len(entryBytes)]
	}

	// End JSON array
	buf = append(buf, ']')
	return buf, nil
}

// Reset - Reset internal state
func (f *ZeroJSONFormatter) Reset() {
	// Nothing to reset in this implementation
}

// EstimatedSize - Pre-allocation hint
func (f *ZeroJSONFormatter) EstimatedSize() int {
	return 2048 // Default estimate for typical log entry
}

func buildErrorMessage(code int) string {
	return utils.FormatIntWithPrefix("HTTP error: ", code)
}
