// @author Admilson B. F. Cossa

package benchmarks

import (
	"io"
	"testing"

	"github.com/go-gen-ecosystem/halolog/adapters/formatters/json"
	"github.com/go-gen-ecosystem/halolog/adapters/outputs/console"
	"github.com/go-gen-ecosystem/halolog/core"
	"github.com/go-gen-ecosystem/halolog/types"
	phuslu "github.com/phuslu/log"
	"github.com/rs/zerolog"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// BenchmarkContext — request-scoped logging: five fields bound once to a
// child/contextual logger, then each line adds one call-site field. This is
// the dominant shape of service logging (per-request loggers), and the
// scenario where context memoization pays: HaloLog and phuslu emit the bound
// context as pre-encoded bytes; zerolog copies its context bytes per event;
// zap clones an encoder with pre-encoded fields.
func BenchmarkContext(b *testing.B) {
	b.Run("HaloLog", func(b *testing.B) {
		root := core.NewLogger(core.Config{
			Level:    types.InfoLevel,
			Adapters: []types.Adapter{console.NewWithWriter(io.Discard, json.NewJsonFormatter())},
		})
		l := root.With().
			WithString("svc", "auth").
			WithString("region", "eu-west-1").
			WithString("pod", "auth-7f9c").
			WithInt("shard", 7).
			WithString("ver", "1.4.2").
			Logger()
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			l.Typed().WithInt("status", 200).Info(msg)
		}
	})
	b.Run("Zerolog", func(b *testing.B) {
		l := zerolog.New(io.Discard).With().Timestamp().
			Str("svc", "auth").
			Str("region", "eu-west-1").
			Str("pod", "auth-7f9c").
			Int("shard", 7).
			Str("ver", "1.4.2").
			Logger()
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			l.Info().Int("status", 200).Msg(msg)
		}
	})
	b.Run("Zap", func(b *testing.B) {
		l := zap.New(zapcore.NewCore(
			zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
			zapcore.AddSync(io.Discard),
			zap.InfoLevel,
		)).With(
			zap.String("svc", "auth"),
			zap.String("region", "eu-west-1"),
			zap.String("pod", "auth-7f9c"),
			zap.Int("shard", 7),
			zap.String("ver", "1.4.2"),
		)
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			l.Info(msg, zap.Int("status", 200))
		}
	})
	b.Run("Phuslu", func(b *testing.B) {
		l := &phuslu.Logger{
			Level:   phuslu.InfoLevel,
			Writer:  phuslu.IOWriter{Writer: io.Discard},
			Context: phuslu.NewContext(nil).Str("svc", "auth").Str("region", "eu-west-1").Str("pod", "auth-7f9c").Int("shard", 7).Str("ver", "1.4.2").Value(),
		}
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			l.Info().Int("status", 200).Msg(msg)
		}
	})
}
