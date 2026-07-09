// @author Admilson B. F. Cossa

// Package json — direct-append fast path.
//
// These helpers let a logger encode a JSON log line by appending bytes straight
// into a caller-owned buffer, field by field, instead of capturing each field as
// a types.TypedFieldData and formatting the array afterwards. They reuse the
// exact key-prefix, value, timestamp, and level rendering that Formatter.Format
// uses, so the direct path is byte-for-byte identical to the capture+format path
// (guarded by TestDirectMatchesFormat). This is the path that lets pre-declared
// keys beat a re-escape-every-call encoder: the key is a single memcpy of its
// pre-escaped fragment.
package json

import (
	"github.com/go-gen-ecosystem/halolog/types"
)

// appendHeaderWith renders the line prefix `{"time":..,"level":..,"message":..`
// using the given header strategy. It is the single header renderer shared by
// Formatter.Format and the direct-append path, so the two stay byte-identical
// by construction.
func appendHeaderWith(dst []byte, ha headerAppender, tsUnixNanos int64, level types.LogLevel, msg string) []byte {
	dst = ha(dst, tsUnixNanos, level)
	dst = append(dst, `,"message":"`...)
	dst = appendJSONString(dst, msg)
	return append(dst, '"')
}

// AppendHeader writes the object open through the message field, with no trailing
// comma or closing brace: `{"time":"..","level":"..","message":".."`, at the
// default second precision. tsUnixNanos is the entry timestamp in unix
// nanoseconds as the logger clock reports it.
func AppendHeader(dst []byte, tsUnixNanos int64, level types.LogLevel, msg string) []byte {
	return appendHeaderWith(dst, appendHeaderSecond, tsUnixNanos, level, msg)
}

// AppendCloser writes the object close and trailing newline that terminate a line.
func AppendCloser(dst []byte) []byte {
	return append(dst, '}', '\n')
}

// AppendField appends `,"key":value` for a single field, byte-identical to how
// Formatter.Format renders the same field. A pre-declared key (kd non-nil) is a
// single memcpy of its pre-escaped fragment; otherwise key is escaped inline. The
// value is encoded from v by the same appendValue used by the capture path.
func AppendField(dst []byte, kd *types.FieldKey, key string, v types.FieldValue) []byte {
	dst = appendKeyPrefix(dst, kd, key)
	return appendValue(dst, &v, nil)
}

// The *Formatter methods below satisfy types.DirectFieldEncoder by delegating to
// the package helpers, so a core dispatcher can direct-encode a line that is
// byte-identical to Format's output (guarded by TestDirectMatchesFormat).
var _ types.DirectFieldEncoder = (*Formatter)(nil)

// AppendHeader implements types.DirectFieldEncoder using this formatter's
// configured timestamp precision.
func (f *Formatter) AppendHeader(dst []byte, tsUnixNanos int64, level types.LogLevel, msg string) []byte {
	return appendHeaderWith(dst, f.appendHeader, tsUnixNanos, level, msg)
}

// AppendField implements types.DirectFieldEncoder.
func (f *Formatter) AppendField(dst []byte, kd *types.FieldKey, key string, v types.FieldValue) []byte {
	return AppendField(dst, kd, key, v)
}

// AppendCloser implements types.DirectFieldEncoder.
func (f *Formatter) AppendCloser(dst []byte) []byte {
	return AppendCloser(dst)
}
