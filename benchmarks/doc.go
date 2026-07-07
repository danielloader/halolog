// @author Admilson B. F. Cossa

// Package benchmarks holds fair, apples-to-apples comparison benchmarks between
// HaloLog and the mainstream Go structured loggers (zerolog, zap, slog, logrus).
//
// Every logger is configured to serialize a structured JSON record
// (timestamp + level + message + fields) and write it to io.Discard, so the
// numbers measure encoding + dispatch overhead on equal footing — not disk I/O
// and not a disabled-output fast path. This is a separate Go module (it pulls in
// the competitor libraries) and is NOT part of the shipped, dependency-free
// halolog module.
package benchmarks
