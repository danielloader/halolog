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

//go:build amd64 || arm64

// Package json provides JSON formatting for log entries
// Author: Admilson B. F. Cossa
package json

import (
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-gen-ecosystem/halolog/types"
)

const (
	timestampLen      = 32
	cacheLineSize     = 64
	smallIntCacheSize = 100
)

// FIXED: Atomic pointer to immutable cache
type timestampCache struct {
	lastUnix int64
	cached   [timestampLen]byte
}

var globalTsCachePtr atomic.Value // stores *timestampCache

func init() {
	// Initialize cache
	globalTsCachePtr.Store(&timestampCache{})

	// Pre-compute small ints
	for i := 0; i < smallIntCacheSize; i++ {
		if i < 10 {
			smallInts[i] = string([]byte{byte('0' + i)})
		} else {
			smallInts[i] = string([]byte{byte('0' + i/10), byte('0' + i%10)})
		}
	}

	// A byte can be written into a JSON string verbatim unless it is a control
	// character (< 0x20), a double quote, or a backslash.
	for c := 0; c < 256; c++ {
		jsonNoEscape[c] = c >= 0x20 && c != '"' && c != '\\'
	}

	empty := map[string][]byte{}
	keyFragCache.Store(&empty)
}

// jsonNoEscape[c] reports whether byte c needs no escaping in a JSON string.
// A single indexed load replaces three comparisons per byte on the hot path.
var jsonNoEscape [256]bool

// maxCachedKeyFragments bounds the transparent key-fragment cache so that a
// workload generating unbounded distinct keys cannot grow it without limit;
// beyond this size, keys are escaped per call instead of cached.
const maxCachedKeyFragments = 4096

// keyFragCache maps a raw field key to its pre-escaped `,"key":` fragment. It is
// read on the hot path with a lock-free atomic load and updated copy-on-write, so
// warm reads never take a lock and never allocate. keyFragMu serializes the rare
// copy-on-write inserts.
var (
	keyFragCache atomic.Pointer[map[string][]byte]
	keyFragMu    sync.Mutex
)

// KeyFragment returns the pre-escaped JSON member prefix for name — the bytes
// `,"<escaped-name>":`. Callers (e.g. the root Key constructor) cache the result
// on an immutable FieldKey so the hot path can emit the key with a single copy.
func KeyFragment(name string) []byte {
	buf := make([]byte, 0, len(name)+4)
	buf = append(buf, ',', '"')
	buf = appendJSONString(buf, name)
	return append(buf, '"', ':')
}

// cachedKeyFragment returns key's pre-escaped fragment from the shared cache,
// computing and inserting it on first use. The returned slice is immutable and
// safe to append to a destination buffer concurrently.
//
//go:inline
func cachedKeyFragment(key string) []byte {
	m := keyFragCache.Load()
	if frag, ok := (*m)[key]; ok {
		return frag
	}
	return insertKeyFragment(key)
}

//go:noinline
func insertKeyFragment(key string) []byte {
	keyFragMu.Lock()
	defer keyFragMu.Unlock()

	m := keyFragCache.Load()
	if frag, ok := (*m)[key]; ok { // re-check under lock
		return frag
	}
	frag := KeyFragment(key)
	if len(*m) >= maxCachedKeyFragments {
		return frag // cache is full: serve without caching
	}
	next := make(map[string][]byte, len(*m)+1)
	for k, v := range *m {
		next[k] = v
	}
	next[key] = frag
	keyFragCache.Store(&next)
	return frag
}

var smallInts [smallIntCacheSize]string

var levelCache = [7]string{
	`,"level":"TRACE"`,
	`,"level":"DEBUG"`,
	`,"level":"INFO"`,
	`,"level":"WARN"`,
	`,"level":"ERROR"`,
	`,"level":"FATAL"`,
	`,"level":"PANIC"`,
}

// Formatter is a zero-allocation formatter that renders log entries as JSON.
type Formatter struct {
	_ [64]byte // Padding
}

// NewJsonFormatter creates a new JSON formatter.
func NewJsonFormatter() *Formatter {
	return &Formatter{}
}

// Format appends the JSON encoding of entry to dst and returns the extended slice.
//
//go:noinline
func (f *Formatter) Format(entry *types.LogEntry, dst []byte) []byte {
	if entry == nil {
		return dst
	}

	dst = append(dst, '{')

	// Timestamp — the hot path sets TimestampUnix (unix nanos); the wall-clock
	// Timestamp is only a fallback for paths that populate it instead.
	dst = append(dst, `"time":"`...)
	dst = fastAppendTime(dst, entryUnixSeconds(entry))
	dst = append(dst, '"')

	// Level
	lvl := entry.Level
	if lvl > 6 {
		lvl = 6
	}
	dst = append(dst, levelCache[lvl]...)

	// Message (FIXED: With JSON escaping)
	dst = append(dst, `,"message":"`...)
	dst = appendJSONString(dst, entry.Message)
	dst = append(dst, '"')

	// Caller (FIXED: Cross-platform)
	if entry.Line >= 0 && entry.File != "" {
		dst = f.formatCaller(dst, entry)
	}

	// Fields — WithField stores into StaticFields (the fast path); the dynamic
	// Fields slice holds overflow and map-style fields. Both must be emitted.
	n := entry.StaticFieldCount
	if n > len(entry.StaticFields) {
		n = len(entry.StaticFields)
	}
	for i := 0; i < n; i++ {
		dst = appendField(dst, &entry.StaticFields[i])
	}
	for i := range entry.Fields {
		dst = appendField(dst, &entry.Fields[i])
	}

	return append(dst, '}', '\n')
}

// FIXED: Race-free cache with atomic pointer
//
//go:inline
func fastAppendTime(dst []byte, unixSec int64) []byte {
	cache := globalTsCachePtr.Load().(*timestampCache)

	if unixSec == cache.lastUnix {
		return append(dst, cache.cached[:25]...)
	}

	return slowAppendTime(dst, unixSec)
}

//go:noinline
func slowAppendTime(dst []byte, unixSec int64) []byte {
	t := time.Unix(unixSec, 0)
	var scratch [64]byte
	b := t.AppendFormat(scratch[:0], "2006-01-02T15:04:05Z07:00")

	if len(b) <= timestampLen {
		// Create new immutable cache
		newCache := &timestampCache{lastUnix: unixSec}
		copy(newCache.cached[:], b)

		// Atomic swap (race-free!)
		globalTsCachePtr.Store(newCache)
	}

	return append(dst, b...)
}

// FIXED: Cross-platform basename
func (f *Formatter) formatCaller(dst []byte, entry *types.LogEntry) []byte {
	dst = append(dst, `,"caller":"`...)

	file := entry.File
	for i := len(file) - 1; i >= 0; i-- {
		if file[i] == '/' || file[i] == '\\' { // FIXED: Both separators
			file = file[i+1:]
			break
		}
	}

	dst = append(dst, file...)
	dst = append(dst, ':')
	dst = appendInt(dst, int64(entry.Line))
	return append(dst, '"')
}

// entryUnixSeconds resolves the entry's timestamp to unix seconds, preferring
// the hot-path TimestampUnix (unix nanos) and falling back to the wall-clock
// Timestamp only when TimestampUnix was not populated.
func entryUnixSeconds(entry *types.LogEntry) int64 {
	if entry.TimestampUnix != 0 {
		return entry.TimestampUnix / int64(time.Second)
	}
	if !entry.Timestamp.IsZero() {
		return entry.Timestamp.Unix()
	}
	return 0
}

// appendField appends `,"key":value` for a single field, emitting the value with
// its correct JSON type (numbers and bools unquoted). It prefers the
// boxing-free typed Val; when Val is unset (KindUnknown, as produced by the
// interface-based WithField API) it falls back to the legacy Value interface{}.
func appendField(dst []byte, field *types.TypedFieldData) []byte {
	// Emit the `,"key":` prefix. A pre-declared FieldKey carries its own
	// pre-escaped fragment (no lookup); otherwise the shared copy-on-write cache
	// serves it (escaping once per distinct key). Both reduce to a single copy on
	// the warm path, replacing the per-call escape of every key byte.
	if kd := field.KeyDesc; kd != nil && len(kd.JSONFragment) > 0 {
		dst = append(dst, kd.JSONFragment...)
	} else {
		dst = append(dst, cachedKeyFragment(field.Key)...)
	}

	switch field.Val.Kind {
	case types.KindString:
		dst = append(dst, '"')
		dst = appendJSONString(dst, field.Val.String)
		return append(dst, '"')
	case types.KindInt, types.KindInt64:
		return strconv.AppendInt(dst, field.Val.Int64, 10)
	case types.KindFloat64:
		return strconv.AppendFloat(dst, field.Val.Float64, 'g', -1, 64)
	case types.KindBool:
		if field.Val.Int64 != 0 {
			return append(dst, "true"...)
		}
		return append(dst, "false"...)
	case types.KindError:
		dst = append(dst, '"')
		if field.Val.String != "" {
			dst = appendJSONString(dst, field.Val.String)
		} else if e, ok := field.Val.Any.(error); ok && e != nil {
			dst = appendJSONString(dst, e.Error())
		}
		return append(dst, '"')
	case types.KindAny:
		return appendAny(dst, field.Val.Any)
	default: // KindUnknown → legacy interface value
		return appendAny(dst, field.Value)
	}
}

// appendAny renders an interface{} value as a correctly-typed JSON value.
// Common scalar types stay allocation-free; only genuinely unknown types fall
// back to fmt (a rare path).
func appendAny(dst []byte, v interface{}) []byte {
	switch x := v.(type) {
	case nil:
		return append(dst, "null"...)
	case string:
		dst = append(dst, '"')
		dst = appendJSONString(dst, x)
		return append(dst, '"')
	case bool:
		if x {
			return append(dst, "true"...)
		}
		return append(dst, "false"...)
	case int:
		return strconv.AppendInt(dst, int64(x), 10)
	case int8:
		return strconv.AppendInt(dst, int64(x), 10)
	case int16:
		return strconv.AppendInt(dst, int64(x), 10)
	case int32:
		return strconv.AppendInt(dst, int64(x), 10)
	case int64:
		return strconv.AppendInt(dst, x, 10)
	case uint:
		return strconv.AppendUint(dst, uint64(x), 10)
	case uint8:
		return strconv.AppendUint(dst, uint64(x), 10)
	case uint16:
		return strconv.AppendUint(dst, uint64(x), 10)
	case uint32:
		return strconv.AppendUint(dst, uint64(x), 10)
	case uint64:
		return strconv.AppendUint(dst, x, 10)
	case float32:
		return strconv.AppendFloat(dst, float64(x), 'g', -1, 32)
	case float64:
		return strconv.AppendFloat(dst, x, 'g', -1, 64)
	case error:
		dst = append(dst, '"')
		dst = appendJSONString(dst, x.Error())
		return append(dst, '"')
	default:
		dst = append(dst, '"')
		dst = appendJSONString(dst, fmt.Sprintf("%v", x))
		return append(dst, '"')
	}
}

// appendJSONString appends s to dst as a JSON string body (no surrounding
// quotes), escaping only where required. The common case — a string with no
// control characters, quotes or backslashes — is a single table-driven scan
// followed by one bulk append, and allocates nothing.
//
//go:inline
func appendJSONString(dst []byte, s string) []byte {
	for i := 0; i < len(s); i++ {
		if !jsonNoEscape[s[i]] {
			return appendJSONEscaped(dst, s, i)
		}
	}
	return append(dst, s...)
}

//go:noinline
func appendJSONEscaped(dst []byte, s string, start int) []byte {
	// Pre-allocate with extra capacity to avoid reallocations
	remaining := s[start:]
	if cap(dst)-len(dst) < len(remaining)*3 {
		newDst := make([]byte, len(dst), len(dst)+len(remaining)*3)
		copy(newDst, dst)
		dst = newDst
	}

	// Copy safe prefix
	dst = append(dst, s[:start]...)

	// High-performance escaping using jump table approach
	// Order by frequency: \\, \", \n, \t, \r, then control chars
	for i := 0; i < len(remaining); i++ {
		c := remaining[i]
		switch {
		case c == '\\':
			dst = append(dst, '\\', '\\')
		case c == '"':
			dst = append(dst, '\\', '"')
		case c == '\n':
			dst = append(dst, '\\', 'n')
		case c == '\t':
			dst = append(dst, '\\', 't')
		case c == '\r':
			dst = append(dst, '\\', 'r')
		case c < 0x20:
			dst = append(dst, '\\', 'u', '0', '0', hexDigit(c>>4), hexDigit(c&0xF))
		default:
			dst = append(dst, c)
		}
	}
	return dst
}

func hexDigit(n byte) byte {
	if n < 10 {
		return '0' + n
	}
	return 'a' + n - 10
}

//go:inline
func appendInt(dst []byte, i int64) []byte {
	if i >= 0 && i < int64(smallIntCacheSize) {
		return append(dst, smallInts[i]...)
	}
	return strconv.AppendInt(dst, i, 10)
}

// EstimatedSize returns an estimated buffer size for pre-allocation
func (f *Formatter) EstimatedSize() int {
	return 4096 // Default 4KB buffer size
}

// Reset resets any internal state (for pooling)
func (f *Formatter) Reset() {
	// Stateless formatter - no-op
}
