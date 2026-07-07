// @author Admilson B. F. Cossa

package benchmarks

import (
	"io"
	"log/slog"
	"testing"
	"time"

	jsonfmt "github.com/go-gen-ecosystem/halolog/adapters/formatters/json"
	"github.com/go-gen-ecosystem/halolog/core"
	"github.com/go-gen-ecosystem/halolog/types"
	"github.com/rs/zerolog"
	"github.com/sirupsen/logrus"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// A realistic HTTP-request log record: 8 fields mixing strings, ints, and a bool.
// Every logger below serializes exactly these fields as JSON to io.Discard. The
// logger constructors (newHaloJSON/newZerolog/newZap/newSlog/newLogrus) are
// shared with comparison_bench_test.go.
const rwMsg = "request handled"

// Pre-declared keys for the HaloLog "Keyed" variant (declared once, as an app would).
var (
	kMethod   = &types.FieldKey{Name: "method", JSONFragment: jsonfmt.KeyFragment("method")}
	kPath     = &types.FieldKey{Name: "path", JSONFragment: jsonfmt.KeyFragment("path")}
	kStatus   = &types.FieldKey{Name: "status", JSONFragment: jsonfmt.KeyFragment("status")}
	kDuration = &types.FieldKey{Name: "duration_ms", JSONFragment: jsonfmt.KeyFragment("duration_ms")}
	kUser     = &types.FieldKey{Name: "user_id", JSONFragment: jsonfmt.KeyFragment("user_id")}
	kReqID    = &types.FieldKey{Name: "request_id", JSONFragment: jsonfmt.KeyFragment("request_id")}
	kBytes    = &types.FieldKey{Name: "bytes", JSONFragment: jsonfmt.KeyFragment("bytes")}
	kCached   = &types.FieldKey{Name: "cached", JSONFragment: jsonfmt.KeyFragment("cached")}
)

func haloWithField(l *core.Logger) {
	l.WithField("method", "GET").WithField("path", "/api/users").
		WithField("status", 200).WithField("duration_ms", 42).
		WithField("user_id", "alice").WithField("request_id", "abc-123").
		WithField("bytes", 1024).WithField("cached", true).Info(rwMsg)
}

func haloTyped(l *core.Logger) {
	l.Typed().WithString("method", "GET").WithString("path", "/api/users").
		WithInt("status", 200).WithInt("duration_ms", 42).
		WithString("user_id", "alice").WithString("request_id", "abc-123").
		WithInt("bytes", 1024).WithBool("cached", true).Info(rwMsg)
}

func haloKeyed(l *core.Logger) {
	l.Typed().Str(kMethod, "GET").Str(kPath, "/api/users").
		Int(kStatus, 200).Int(kDuration, 42).
		Str(kUser, "alice").Str(kReqID, "abc-123").
		Int(kBytes, 1024).Bool(kCached, true).Info(rwMsg)
}

func zerologRecord(l *zerolog.Logger) {
	l.Info().Str("method", "GET").Str("path", "/api/users").
		Int("status", 200).Int("duration_ms", 42).
		Str("user_id", "alice").Str("request_id", "abc-123").
		Int("bytes", 1024).Bool("cached", true).Msg(rwMsg)
}

func zapRecord(l *zap.Logger) {
	l.Info(rwMsg, zap.String("method", "GET"), zap.String("path", "/api/users"),
		zap.Int("status", 200), zap.Int("duration_ms", 42),
		zap.String("user_id", "alice"), zap.String("request_id", "abc-123"),
		zap.Int("bytes", 1024), zap.Bool("cached", true))
}

func slogRecord(l *slog.Logger) {
	l.Info(rwMsg, slog.String("method", "GET"), slog.String("path", "/api/users"),
		slog.Int("status", 200), slog.Int("duration_ms", 42),
		slog.String("user_id", "alice"), slog.String("request_id", "abc-123"),
		slog.Int("bytes", 1024), slog.Bool("cached", true))
}

func logrusRecord(l *logrus.Logger) {
	l.WithFields(logrus.Fields{
		"method": "GET", "path": "/api/users", "status": 200, "duration_ms": 42,
		"user_id": "alice", "request_id": "abc-123", "bytes": 1024, "cached": true,
	}).Info(rwMsg)
}

// Timestamp-neutralised competitor constructors: each drops its timestamp field
// so the logger serialises only level+message+fields (used by BenchmarkNoTimestamp).
func newZapNoTime() *zap.Logger {
	cfg := zap.NewProductionEncoderConfig()
	cfg.TimeKey = ""
	return zap.New(zapcore.NewCore(zapcore.NewJSONEncoder(cfg), zapcore.AddSync(io.Discard), zap.InfoLevel))
}

func newSlogNoTime() *slog.Logger {
	opts := &slog.HandlerOptions{Level: slog.LevelInfo, ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
		if a.Key == slog.TimeKey {
			return slog.Attr{}
		}
		return a
	}}
	return slog.New(slog.NewJSONHandler(io.Discard, opts))
}

func newLogrusNoTime() *logrus.Logger {
	l := logrus.New()
	l.SetOutput(io.Discard)
	l.SetFormatter(&logrus.JSONFormatter{DisableTimestamp: true})
	l.SetLevel(logrus.InfoLevel)
	return l
}

// BenchmarkDefaults — Table A. Every logger at its idiomatic default, each
// emitting a real timestamp. This is the realistic "out-of-the-box" comparison.
//
// FAIRNESS NOTE: HaloLog's default timestamp is a background-cached, second-
// granularity value (an atomic read + a per-second-cached formatted string),
// whereas zerolog/logrus call time.Now() and format RFC3339 per line, and
// zap/slog format a fresh sub-second timestamp per line. HaloLog therefore does
// LESS timestamp work here — this table slightly FLATTERS HaloLog. See
// BenchmarkNoTimestamp for the conservative counterpart and BenchmarkClockCost
// for the size of the difference.
func BenchmarkDefaults(b *testing.B) {
	b.Run("HaloLog_WithField", func(b *testing.B) { l := newHaloJSON(); benchLoop(b, func() { haloWithField(l) }) })
	b.Run("HaloLog_Typed", func(b *testing.B) { l := newHaloJSON(); benchLoop(b, func() { haloTyped(l) }) })
	b.Run("HaloLog_Keyed", func(b *testing.B) { l := newHaloJSON(); benchLoop(b, func() { haloKeyed(l) }) })
	b.Run("Zerolog", func(b *testing.B) { l := newZerolog(); benchLoop(b, func() { zerologRecord(&l) }) })
	b.Run("Zap", func(b *testing.B) { l := newZap(); benchLoop(b, func() { zapRecord(l) }) })
	b.Run("Slog", func(b *testing.B) { l := newSlog(); benchLoop(b, func() { slogRecord(l) }) })
	b.Run("Logrus", func(b *testing.B) { l := newLogrus(); benchLoop(b, func() { logrusRecord(l) }) })
}

// BenchmarkNoTimestamp — Table B. The competitors' timestamps are switched OFF
// (zerolog: no .Timestamp(); zap: TimeKey=""; slog: time attr dropped; logrus:
// DisableTimestamp) so they serialise only level+message+fields. HaloLog CANNOT
// drop its timestamp, so the HaloLog rows here still emit their cached time and
// therefore do strictly MORE work than the competitors — making this a
// CONSERVATIVE lower bound for HaloLog. The true position sits between Table A
// and Table B.
func BenchmarkNoTimestamp(b *testing.B) {
	b.Run("HaloLog_WithField", func(b *testing.B) { l := newHaloJSON(); benchLoop(b, func() { haloWithField(l) }) })
	b.Run("HaloLog_Keyed", func(b *testing.B) { l := newHaloJSON(); benchLoop(b, func() { haloKeyed(l) }) })
	b.Run("Zerolog_NoTS", func(b *testing.B) {
		l := zerolog.New(io.Discard)
		benchLoop(b, func() { zerologRecord(&l) })
	})
	b.Run("Zap_NoTS", func(b *testing.B) { l := newZapNoTime(); benchLoop(b, func() { zapRecord(l) }) })
	b.Run("Slog_NoTS", func(b *testing.B) { l := newSlogNoTime(); benchLoop(b, func() { slogRecord(l) }) })
	b.Run("Logrus_NoTS", func(b *testing.B) { l := newLogrusNoTime(); benchLoop(b, func() { logrusRecord(l) }) })
}

// Package-level sinks keep the compiler from eliminating the measured work.
var (
	sinkTime time.Time
	sinkBuf  = make([]byte, 0, 64)
)

// BenchmarkClockCost isolates the per-line timestamp cost that every competitor
// pays and HaloLog caches away: a live time.Now(), and time.Now() formatted the
// way zerolog/logrus (RFC3339) and zap/slog (sub-second) do it. The delta here
// is roughly how much of HaloLog's Table-A lead is the cached clock.
func BenchmarkClockCost(b *testing.B) {
	b.Run("TimeNow", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			sinkTime = time.Now()
		}
	})
	b.Run("TimeNow_RFC3339", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			sinkBuf = time.Now().AppendFormat(sinkBuf[:0], time.RFC3339)
		}
	})
	b.Run("TimeNow_RFC3339Nano", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			sinkBuf = time.Now().AppendFormat(sinkBuf[:0], time.RFC3339Nano)
		}
	})
}

// benchLoop is the shared measured loop: allocs reported, timer reset, N calls.
func benchLoop(b *testing.B, call func()) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		call()
	}
}
