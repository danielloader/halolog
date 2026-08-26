// @author Admilson B. F. Cossa

// Package json — line-header engine: timestamp precision and the fused header
// cache.
//
// The header of every line is `{"time":"<ts>","level":"<LVL>"`. Rendering it
// naively costs several appends plus a timestamp format per line. This file
// applies two patterns:
//
//   - Strategy: the formatter's TimePrecision selects a headerAppender from a
//     table ONCE at construction — the hot path carries zero precision
//     branching.
//   - Fused cache (second precision, the default): the entire header is cached
//     per (second, level) behind lock-free atomic pointers, so on the steady
//     state the header is a single memcpy. Sub-second precisions cache the
//     per-second RFC3339 prefix and timezone suffix and render only the
//     fractional digits per line.
//
// ACCURACY NOTE (disclosed, not hidden): the logger's default clock is a
// background-cached clock refreshed every 10ms, so displayed sub-second digits
// carry up to ~10ms of wall-clock skew. That is sufficient for in-service
// ordering and trace correlation; workloads needing tighter accuracy can run a
// finer cache.NewCachedClock interval.
package json

import (
	"sync/atomic"
	"time"

	"github.com/go-gen-ecosystem/halolog/types"
)

// TimePrecision selects the fractional-second resolution of the "time" field.
type TimePrecision uint8

// Supported "time" field resolutions.
const (
	// PrecisionSecond renders RFC3339 whole seconds (the default; fastest —
	// the whole header is served from the fused per-second cache).
	PrecisionSecond TimePrecision = iota
	// PrecisionMilli renders RFC3339 with 3 fractional digits.
	PrecisionMilli
	// PrecisionMicro renders RFC3339 with 6 fractional digits.
	PrecisionMicro
	// PrecisionNano renders RFC3339 with 9 fractional digits.
	PrecisionNano
	precisionCount
)

// headerAppender renders `{"time":"<ts>","level":"<LVL>"` for one line.
type headerAppender func(dst []byte, unixNanos int64, level types.LogLevel) []byte

// headerAppenders maps each precision to its rendering strategy (selected once
// at formatter construction).
var headerAppenders = [precisionCount]headerAppender{
	PrecisionSecond: appendHeaderSecond,
	PrecisionMilli:  subSecondAppender(3, int64(time.Millisecond)),
	PrecisionMicro:  subSecondAppender(6, int64(time.Microsecond)),
	PrecisionNano:   subSecondAppender(9, 1),
}

// clampLevel bounds a level to the levelCache index range.
func clampLevel(level types.LogLevel) types.LogLevel {
	if level > 6 {
		return 6
	}
	return level
}

// ---- Fused header cache (second precision) ----

// fusedHeader is an immutable snapshot of one (second, level) header.
type fusedHeader struct {
	sec int64
	n   uint8
	buf [64]byte
}

// fusedHeaders holds one lock-free slot per level. Entries are immutable and
// swapped atomically, so readers never see partial writes.
var fusedHeaders [7]atomic.Pointer[fusedHeader]

// splitUnixNanos floor-divides a unix-nanosecond timestamp into whole seconds
// and a non-negative fractional remainder. Go's integer division truncates
// toward zero, which for pre-epoch (negative) timestamps yields a negative
// remainder — and negative fractional digits render as garbage bytes.
func splitUnixNanos(unixNanos int64) (sec, frac int64) {
	sec = unixNanos / int64(time.Second)
	frac = unixNanos % int64(time.Second)
	if frac < 0 {
		sec--
		frac += int64(time.Second)
	}
	return sec, frac
}

// appendHeaderSecond emits the fused header: a single memcpy on the steady
// state, rebuilt lazily when the second rolls over.
func appendHeaderSecond(dst []byte, unixNanos int64, level types.LogLevel) []byte {
	lvl := clampLevel(level)
	sec, _ := splitUnixNanos(unixNanos)
	if h := fusedHeaders[lvl].Load(); h != nil && h.sec == sec {
		return append(dst, h.buf[:h.n]...)
	}
	return buildFusedHeader(dst, sec, lvl)
}

//go:noinline
func buildFusedHeader(dst []byte, sec int64, lvl types.LogLevel) []byte {
	h := &fusedHeader{sec: sec}
	b := append(h.buf[:0], `{"time":"`...)
	b = time.Unix(sec, 0).AppendFormat(b, time.RFC3339)
	b = append(b, '"')
	b = append(b, levelCache[lvl]...)
	h.n = uint8(len(b)) // max 9+25+1+16 = 51, fits both buf and uint8
	fusedHeaders[lvl].Store(h)
	return append(dst, h.buf[:h.n]...)
}

// ---- Sub-second rendering (split prefix/timezone cache) ----

// rfc3339TzStart is where the timezone suffix begins in an RFC3339 timestamp:
// "2006-01-02T15:04:05" is always exactly 19 bytes.
const rfc3339TzStart = 19

// tsParts is an immutable per-second snapshot of the formatted timestamp; the
// fractional digits are inserted between buf[:19] and the timezone buf[19:n].
// Caching n (instead of assuming a fixed width) keeps "Z"-suffixed UTC
// timestamps exactly as correct as offset-suffixed ones.
type tsParts struct {
	sec int64
	n   uint8
	buf [40]byte
}

var tsPartsPtr atomic.Pointer[tsParts]

//go:noinline
func buildTsParts(sec int64) *tsParts {
	p := &tsParts{sec: sec}
	b := time.Unix(sec, 0).AppendFormat(p.buf[:0], time.RFC3339)
	p.n = uint8(len(b))
	tsPartsPtr.Store(p)
	return p
}

// subSecondAppender builds the rendering strategy for one fractional width.
// digits/div are captured once at table construction — no per-line branching.
func subSecondAppender(digits int, div int64) headerAppender {
	return func(dst []byte, unixNanos int64, level types.LogLevel) []byte {
		sec, nanos := splitUnixNanos(unixNanos)
		frac := nanos / div
		p := tsPartsPtr.Load()
		if p == nil || p.sec != sec {
			p = buildTsParts(sec)
		}
		dst = append(dst, `{"time":"`...)
		dst = append(dst, p.buf[:rfc3339TzStart]...)
		dst = append(dst, '.')
		dst = appendFracDigits(dst, frac, digits)
		dst = append(dst, p.buf[rfc3339TzStart:p.n]...)
		dst = append(dst, '"')
		return append(dst, levelCache[clampLevel(level)]...)
	}
}

// appendFracDigits writes frac as exactly `digits` zero-padded digits.
func appendFracDigits(dst []byte, frac int64, digits int) []byte {
	var scratch [9]byte
	for i := digits - 1; i >= 0; i-- {
		scratch[i] = byte('0' + frac%10)
		frac /= 10
	}
	return append(dst, scratch[:digits]...)
}

// entryUnixNanos resolves the entry's timestamp to unix nanoseconds, preferring
// the hot-path TimestampUnix and falling back to the wall-clock Timestamp.
func entryUnixNanos(entry *types.LogEntry) int64 {
	if entry.TimestampUnix != 0 {
		return entry.TimestampUnix
	}
	if !entry.Timestamp.IsZero() {
		return entry.Timestamp.UnixNano()
	}
	return 0
}
