// @author Admilson B. F. Cossa

// Package types — optional capability interfaces for the direct-append fast path.
//
// The fast path lets the core builder encode a log line straight to bytes,
// skipping per-field struct capture, when — and only when — the configuration
// provably produces identical output: a single adapter that accepts raw
// pre-formatted lines, whose current formatter can encode fields directly.
// Everything else (multiple adapters, masking, text/console formatters, custom
// formatters) uses the normal capture+Format path. These are deliberately tiny
// interfaces (ISP): adapters and formatters opt in independently.
package types

// RawWriter is implemented by output adapters that can write a fully formatted
// log line as-is. The line includes its trailing newline. Implementations must
// be safe for concurrent use and must not retain the slice after returning.
type RawWriter interface {
	WriteRaw(line []byte) error
}

// DirectFieldEncoder is implemented by formatters that can render a log line
// incrementally, byte-identical to their Format output. AppendHeader opens the
// line (object start through the message value), AppendField emits one
// `,"key":value` member (kd, when non-nil, carries a pre-escaped key fragment),
// and AppendCloser terminates the line including the trailing newline.
type DirectFieldEncoder interface {
	AppendHeader(dst []byte, tsUnixNanos int64, level LogLevel, msg string) []byte
	AppendField(dst []byte, kd *FieldKey, key string, v FieldValue) []byte
	AppendCloser(dst []byte) []byte
}

// DirectCapableAdapter is implemented by adapters that can expose a
// DirectFieldEncoder for their CURRENT formatter. It returns nil when the
// configured formatter cannot direct-encode (e.g. a text formatter), which
// makes a later SetFormatter swap automatically disable the fast path.
type DirectCapableAdapter interface {
	DirectEncoder() DirectFieldEncoder
}
