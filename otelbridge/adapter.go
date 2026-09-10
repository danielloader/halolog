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

// Package otelbridge — OpenTelemetry Logs output adapter.
//
// Bind (bridge.go) correlates a log line by stamping trace_id and span_id
// onto it. Adapter goes the other way: it emits the whole entry into the
// OpenTelemetry Logs API, so the same line reaches an OTLP backend as a
// LogRecord while a console or file adapter keeps writing it to stderr.
// Register both and core.Logger fans out to each in turn:
//
//	prov := otelsdklog.NewLoggerProvider(otelsdklog.WithProcessor(proc))
//	logger := core.New().
//	    Adapters(console.New(), otelbridge.NewAdapter("my/service",
//	        otelbridge.WithLoggerProvider(prov))).
//	    MustBuild()
//
// Author: Admilson B. F. Cossa
package otelbridge

import (
	"context"
	"fmt"
	"strconv"

	"github.com/go-gen-ecosystem/halolog/types"
	otellog "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/trace"
)

// DefaultAdapterName is the adapter's registry name when WithName is not given.
const DefaultAdapterName = "otel"

// Adapter emits HaloLog entries as OpenTelemetry log records. It satisfies
// types.Adapter, so it composes with every other output through the logger's
// normal adapter fan-out.
//
// The zero-allocation hot path stops here by construction: a LogRecord is a
// structured object, not a byte slice, so each entry costs one Record plus its
// attribute slice. Put it alongside a console adapter, not in place of one, and
// keep the console adapter for the paths where the byte cost matters.
type Adapter struct {
	logger otellog.Logger
	name   string
}

type adapterConfig struct {
	provider  otellog.LoggerProvider
	name      string
	version   string
	schemaURL string
}

// Option configures an Adapter.
type Option func(*adapterConfig)

// WithLoggerProvider sets the provider the adapter draws its Logger from.
// Without it the adapter uses the global provider resolved at construction
// time, so install the provider before building the logger.
func WithLoggerProvider(provider otellog.LoggerProvider) Option {
	return func(c *adapterConfig) {
		if provider != nil {
			c.provider = provider
		}
	}
}

// WithName sets the adapter's registry name, used by AdapterManager lookups.
// Give each Adapter a distinct name when registering more than one.
func WithName(name string) Option {
	return func(c *adapterConfig) {
		if name != "" {
			c.name = name
		}
	}
}

// WithVersion records the instrumented package's version on the emitting scope.
func WithVersion(version string) Option {
	return func(c *adapterConfig) { c.version = version }
}

// WithSchemaURL records the schema URL of the emitting scope.
func WithSchemaURL(url string) Option {
	return func(c *adapterConfig) { c.schemaURL = url }
}

// NewAdapter returns an Adapter emitting through scopeName, which should be the
// import path of the package doing the logging (the OpenTelemetry instrumentation
// scope convention) — not this bridge's own path.
func NewAdapter(scopeName string, opts ...Option) *Adapter {
	cfg := adapterConfig{name: DefaultAdapterName}
	for _, opt := range opts {
		opt(&cfg)
	}
	if cfg.provider == nil {
		cfg.provider = global.GetLoggerProvider()
	}

	logOpts := make([]otellog.LoggerOption, 0, 2)
	if cfg.version != "" {
		logOpts = append(logOpts, otellog.WithInstrumentationVersion(cfg.version))
	}
	if cfg.schemaURL != "" {
		logOpts = append(logOpts, otellog.WithSchemaURL(cfg.schemaURL))
	}
	return &Adapter{logger: cfg.provider.Logger(scopeName, logOpts...), name: cfg.name}
}

// Name returns the adapter's registry name.
func (a *Adapter) Name() string { return a.name }

// Write emits entry as a log record.
func (a *Adapter) Write(entry *types.LogEntry) error {
	if entry == nil {
		return nil
	}
	ctx, rec, ok := a.build(entry)
	if !ok {
		return nil
	}
	a.logger.Emit(ctx, rec)
	return nil
}

// WriteZero emits entry as a log record. The Record path never renders bytes,
// so there is no cheaper variant to offer here — it is Write.
func (a *Adapter) WriteZero(entry *types.LogEntry) error { return a.Write(entry) }

// Flush is a no-op: buffering and export belong to the SDK's processor. Call
// ForceFlush on the LoggerProvider to drain pending records.
func (a *Adapter) Flush() error { return nil }

// Close is a no-op: the LoggerProvider owns the exporter's lifetime. Call
// Shutdown on it to flush and release the export pipeline.
func (a *Adapter) Close() error { return nil }

// SetFormatter is a no-op. Records carry structured attributes to the SDK,
// which serializes them; a text/JSON formatter has nothing to do here.
func (a *Adapter) SetFormatter(types.Formatter) {}

// Health reports the adapter healthy whenever it holds a Logger. Export
// failures surface through the SDK's error handler, not here.
func (a *Adapter) Health() error {
	if a.logger == nil {
		return types.ErrAdapterClosed
	}
	return nil
}

// build converts entry into a record plus the context carrying its span. The
// bool is false when the record should be dropped (the SDK is not sampling
// this severity), which spares the attribute conversion.
//
// Every value copied here is owned by the record before Emit returns: the
// caller's LogEntry goes straight back to the pool.
func (a *Adapter) build(entry *types.LogEntry) (context.Context, otellog.Record, bool) {
	sev := severityOf(entry.Level)
	ctx := a.spanContext(entry)

	var rec otellog.Record
	if !a.logger.Enabled(ctx, otellog.EnabledParameters{Severity: sev}) {
		return ctx, rec, false
	}

	// ObservedTimestamp is deliberately left unset: the SDK stamps it at Emit,
	// which is a truer observation time than HaloLog's cached clock (up to
	// ~10ms of skew at the default refresh interval).
	rec.SetTimestamp(entry.Timestamp)
	rec.SetSeverity(sev)
	rec.SetSeverityText(entry.Level.String())
	rec.SetBody(otellog.StringValue(entry.Message))
	a.addAttributes(&rec, entry)
	return ctx, rec, true
}

// spanContext rebuilds the span the entry was logged under. The adapter
// interface passes no context.Context, so the only trace information available
// is what Bind already stamped onto the entry as fields — re-parsing those hex
// strings is what puts a real TraceID on the record instead of a pair of
// attributes the backend cannot correlate on.
func (a *Adapter) spanContext(entry *types.LogEntry) context.Context {
	var (
		traceID trace.TraceID
		spanID  trace.SpanID
		flags   trace.TraceFlags
		found   bool
	)
	forEachField(entry, func(key string, f types.TypedFieldData) {
		hex, ok := stringOf(f)
		if !ok {
			return
		}
		switch key {
		case keyTraceID.Name:
			if id, err := trace.TraceIDFromHex(hex); err == nil {
				traceID, found = id, true
			}
		case keySpanID.Name:
			if id, err := trace.SpanIDFromHex(hex); err == nil {
				spanID = id
			}
		case keyTraceFlags.Name:
			if v, err := strconv.ParseUint(hex, 16, 8); err == nil {
				flags = trace.TraceFlags(v)
			}
		}
	})
	if !found {
		return context.Background()
	}
	return trace.ContextWithSpanContext(context.Background(), trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: flags,
	}))
}

// addAttributes copies the entry's fields, context, component, error, and
// source location onto the record. The correlation fields are skipped: they
// are already the record's TraceID and SpanID, and re-emitting them as
// attributes would have the backend index the same value twice.
func (a *Adapter) addAttributes(rec *otellog.Record, entry *types.LogEntry) {
	forEachField(entry, func(key string, f types.TypedFieldData) {
		switch key {
		case keyTraceID.Name, keySpanID.Name, keyTraceFlags.Name:
			return
		}
		rec.AddAttributes(otellog.KeyValue{Key: key, Value: logValue(f)})
	})
	for _, f := range entry.GetAllContext() {
		rec.AddAttributes(otellog.KeyValue{Key: fieldName(f), Value: logValue(f)})
	}
	if entry.Component != "" {
		rec.AddAttributes(otellog.String("component", entry.Component))
	}
	if msg := errorMessage(entry); msg != "" {
		rec.AddAttributes(otellog.String("error", msg))
	}
	if entry.File != "" {
		rec.AddAttributes(
			otellog.String("code.file.path", entry.File),
			otellog.Int("code.line.number", entry.Line),
		)
	}
}

// forEachField walks the entry's static, dynamic, and indexed field storage in
// the order a formatter would, without the merged slice GetAllFields allocates.
func forEachField(entry *types.LogEntry, fn func(key string, f types.TypedFieldData)) {
	for i := 0; i < entry.StaticFieldCount && i < len(entry.StaticFields); i++ {
		f := entry.StaticFields[i]
		fn(fieldName(f), f)
	}
	for _, f := range entry.Fields {
		fn(fieldName(f), f)
	}
	if entry.IndexedStore != nil {
		for _, f := range entry.IndexedStore.GetAll() {
			fn(fieldName(f), f)
		}
	}
}

// stringOf reads a field as a string from whichever of the two storage forms
// holds it, so correlation works whether the caller used a typed setter or the
// interface-boxed WithField path.
func stringOf(f types.TypedFieldData) (string, bool) {
	if f.Val.Kind == types.KindString {
		return f.Val.String, true
	}
	if f.Val.Kind == types.KindUnknown {
		if s, ok := f.Value.(string); ok {
			return s, true
		}
	}
	return "", false
}

// fieldName resolves a field's key from either the plain string or the
// pre-declared descriptor the keyed builders carry.
func fieldName(f types.TypedFieldData) string {
	if f.Key != "" {
		return f.Key
	}
	if f.KeyDesc != nil {
		return f.KeyDesc.Name
	}
	return ""
}

// errorMessage prefers the pre-rendered string so the adapter never calls
// Error() on the hot path when the entry already did.
func errorMessage(entry *types.LogEntry) string {
	if entry.ErrorMsg != "" {
		return entry.ErrorMsg
	}
	if entry.Error != nil {
		return entry.Error.Error()
	}
	return ""
}

// logValue converts a HaloLog field to its OpenTelemetry equivalent. The
// typed storage wins when set; KindUnknown means the field arrived through the
// interface-boxed path (WithField, type inference, masking) and its value
// lives in Value instead.
func logValue(f types.TypedFieldData) otellog.Value {
	v := f.Val
	switch v.Kind {
	case types.KindString, types.KindError:
		return otellog.StringValue(v.String)
	case types.KindInt, types.KindInt64:
		return otellog.Int64Value(v.Int64)
	case types.KindFloat64:
		return otellog.Float64Value(v.Float64)
	case types.KindBool:
		return otellog.BoolValue(v.Int64 != 0)
	case types.KindAny:
		return anyValue(v.Any)
	case types.KindUnknown:
		return anyValue(f.Value)
	default:
		return anyValue(f.Value)
	}
}

func anyValue(v interface{}) otellog.Value {
	switch t := v.(type) {
	case nil:
		return otellog.Value{}
	case string:
		return otellog.StringValue(t)
	case bool:
		return otellog.BoolValue(t)
	case int:
		return otellog.IntValue(t)
	case int64:
		return otellog.Int64Value(t)
	case float64:
		return otellog.Float64Value(t)
	case []byte:
		return otellog.BytesValue(t)
	case error:
		return otellog.StringValue(t.Error())
	default:
		return otellog.StringValue(fmt.Sprint(t))
	}
}

// severityOf maps HaloLog levels onto the OpenTelemetry severity scale. Panic
// sits one step above Fatal: both are terminal, but the distinction survives
// the round trip to the backend.
func severityOf(level types.LogLevel) otellog.Severity {
	switch level {
	case types.TraceLevel:
		return otellog.SeverityTrace1
	case types.DebugLevel:
		return otellog.SeverityDebug1
	case types.InfoLevel:
		return otellog.SeverityInfo1
	case types.WarnLevel:
		return otellog.SeverityWarn1
	case types.ErrorLevel:
		return otellog.SeverityError1
	case types.FatalLevel:
		return otellog.SeverityFatal1
	case types.PanicLevel:
		return otellog.SeverityFatal2
	default:
		return otellog.SeverityUndefined
	}
}
