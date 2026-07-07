// @author Admilson B. F. Cossa

package benchmarks

import (
	"io"
	"log/slog"
	"testing"

	"github.com/go-gen-ecosystem/halolog/adapters/formatters/json"
	"github.com/go-gen-ecosystem/halolog/adapters/outputs/console"
	"github.com/go-gen-ecosystem/halolog/adapters/outputs/discard"
	"github.com/go-gen-ecosystem/halolog/core"
	"github.com/go-gen-ecosystem/halolog/types"
	"github.com/rs/zerolog"
	"github.com/sirupsen/logrus"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Every constructor below emits a structured JSON record (timestamp + level +
// message + fields) to io.Discard, so the benchmarks compare encoding + dispatch
// on equal footing.

func newHaloJSON() *core.Logger {
	return core.NewLogger(core.Config{
		Component: "bench",
		Level:     types.InfoLevel,
		Adapters:  []types.Adapter{console.NewWithWriter(io.Discard, json.NewJsonFormatter())},
	})
}

// newHaloDisabledOutput uses the no-op discard adapter, which short-circuits
// before any formatting happens. This is the "fast path" the historical
// LIBRARY_COMPARISON.md numbers measured — it does NOT serialize JSON, so it is
// only comparable to the other loggers' level-gated (disabled) path, never to
// their real JSON output. Included for transparency, not as a like-for-like win.
func newHaloDisabledOutput() *core.Logger {
	return core.NewLogger(core.Config{
		Component: "bench",
		Level:     types.InfoLevel,
		Adapters:  []types.Adapter{discard.New()},
	})
}

func newZerolog() zerolog.Logger {
	return zerolog.New(io.Discard).With().Timestamp().Logger()
}

func newZap() *zap.Logger {
	return zap.New(zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		zapcore.AddSync(io.Discard),
		zap.InfoLevel,
	))
}

func newSlog() *slog.Logger {
	return slog.New(slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelInfo}))
}

func newLogrus() *logrus.Logger {
	l := logrus.New()
	l.SetOutput(io.Discard)
	l.SetFormatter(&logrus.JSONFormatter{})
	l.SetLevel(logrus.InfoLevel)
	return l
}

const msg = "user login processed"

// BenchmarkInfo — a bare message, no fields.
func BenchmarkInfo(b *testing.B) {
	b.Run("HaloLog", func(b *testing.B) {
		l := newHaloJSON()
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			l.Info(msg)
		}
	})
	b.Run("Zerolog", func(b *testing.B) {
		l := newZerolog()
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			l.Info().Msg(msg)
		}
	})
	b.Run("Zap", func(b *testing.B) {
		l := newZap()
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			l.Info(msg)
		}
	})
	b.Run("Slog", func(b *testing.B) {
		l := newSlog()
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			l.Info(msg)
		}
	})
	b.Run("Logrus", func(b *testing.B) {
		l := newLogrus()
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			l.Info(msg)
		}
	})
	b.Run("HaloLog_DisabledOutput", func(b *testing.B) {
		l := newHaloDisabledOutput()
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			l.Info(msg)
		}
	})
}

// BenchmarkOneField — a single string field.
func BenchmarkOneField(b *testing.B) {
	b.Run("HaloLog_WithField", func(b *testing.B) {
		l := newHaloJSON()
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			l.WithField("key", "value").Info(msg)
		}
	})
	b.Run("HaloLog_Typed", func(b *testing.B) {
		l := newHaloJSON()
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			l.Typed().WithString("key", "value").Info(msg)
		}
	})
	b.Run("Zerolog", func(b *testing.B) {
		l := newZerolog()
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			l.Info().Str("key", "value").Msg(msg)
		}
	})
	b.Run("Zap", func(b *testing.B) {
		l := newZap()
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			l.Info(msg, zap.String("key", "value"))
		}
	})
	b.Run("Slog", func(b *testing.B) {
		l := newSlog()
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			l.Info(msg, slog.String("key", "value"))
		}
	})
	b.Run("Logrus", func(b *testing.B) {
		l := newLogrus()
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			l.WithField("key", "value").Info(msg)
		}
	})
}

// BenchmarkTenFields — ten integer fields.
func BenchmarkTenFields(b *testing.B) {
	b.Run("HaloLog_WithField", func(b *testing.B) {
		l := newHaloJSON()
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			l.WithField("k1", 1).WithField("k2", 2).WithField("k3", 3).WithField("k4", 4).WithField("k5", 5).
				WithField("k6", 6).WithField("k7", 7).WithField("k8", 8).WithField("k9", 9).WithField("k10", 10).
				Info(msg)
		}
	})
	b.Run("HaloLog_Typed", func(b *testing.B) {
		l := newHaloJSON()
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			l.Typed().WithInt("k1", 1).WithInt("k2", 2).WithInt("k3", 3).WithInt("k4", 4).WithInt("k5", 5).
				WithInt("k6", 6).WithInt("k7", 7).WithInt("k8", 8).WithInt("k9", 9).WithInt("k10", 10).
				Info(msg)
		}
	})
	b.Run("Zerolog", func(b *testing.B) {
		l := newZerolog()
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			l.Info().
				Int("k1", 1).Int("k2", 2).Int("k3", 3).Int("k4", 4).Int("k5", 5).
				Int("k6", 6).Int("k7", 7).Int("k8", 8).Int("k9", 9).Int("k10", 10).
				Msg(msg)
		}
	})
	b.Run("Zap", func(b *testing.B) {
		l := newZap()
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			l.Info(msg,
				zap.Int("k1", 1), zap.Int("k2", 2), zap.Int("k3", 3), zap.Int("k4", 4), zap.Int("k5", 5),
				zap.Int("k6", 6), zap.Int("k7", 7), zap.Int("k8", 8), zap.Int("k9", 9), zap.Int("k10", 10),
			)
		}
	})
	b.Run("Slog", func(b *testing.B) {
		l := newSlog()
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			l.Info(msg,
				slog.Int("k1", 1), slog.Int("k2", 2), slog.Int("k3", 3), slog.Int("k4", 4), slog.Int("k5", 5),
				slog.Int("k6", 6), slog.Int("k7", 7), slog.Int("k8", 8), slog.Int("k9", 9), slog.Int("k10", 10),
			)
		}
	})
	b.Run("Logrus", func(b *testing.B) {
		l := newLogrus()
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			l.WithFields(logrus.Fields{
				"k1": 1, "k2": 2, "k3": 3, "k4": 4, "k5": 5,
				"k6": 6, "k7": 7, "k8": 8, "k9": 9, "k10": 10,
			}).Info(msg)
		}
	})
}

// BenchmarkTwentyFields — twenty integer fields (a heavy-field stress case).
func BenchmarkTwentyFields(b *testing.B) {
	b.Run("HaloLog_WithField", func(b *testing.B) {
		l := newHaloJSON()
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			fb := l.WithField("k0", 0)
			for j := 1; j < 20; j++ {
				fb = fb.WithField("k", j)
			}
			fb.Info(msg)
		}
	})
	b.Run("HaloLog_Typed", func(b *testing.B) {
		l := newHaloJSON()
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			fb := l.Typed().WithInt("k0", 0)
			for j := 1; j < 20; j++ {
				fb = fb.WithInt("k", j)
			}
			fb.Info(msg)
		}
	})
	b.Run("Zerolog", func(b *testing.B) {
		l := newZerolog()
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			e := l.Info()
			for j := 0; j < 20; j++ {
				e = e.Int("k", j)
			}
			e.Msg(msg)
		}
	})
	b.Run("Zap", func(b *testing.B) {
		l := newZap()
		fs := make([]zap.Field, 20)
		for j := 0; j < 20; j++ {
			fs[j] = zap.Int("k", j)
		}
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			l.Info(msg, fs...)
		}
	})
}
