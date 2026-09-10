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
	"math"
	"strconv"
	"time"

	"github.com/go-gen-ecosystem/halolog/types"
	otellog "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/trace"
)

// DefaultAdapterName is the adapter's registry name when WithName is not given.
const DefaultAdapterName = "otel"

// attrStackCap stages attributes in a stack array so building them adds no
// allocation of its own up to this width; beyond it the slice spills to the
// heap once. It is deliberately above log.Record's own 5-attribute inline
// capacity — the record starts allocating before this buffer does, so the
// staging buffer is never the first thing to cost an allocation.
const attrStackCap = 8

// Adapter emits HaloLog entries as OpenTelemetry log records. It satisfies
// types.Adapter, so it composes with every other output through the logger's
// normal adapter fan-out.
//
// Cost per emitted record, measured against a discarding logger by
// TestAdapter_AllocationBudgets (which holds these as ceilings):
//
//	up to 5 attributes                 0 allocs
//	6+ attributes (past Record inline) 1 alloc
//	9+ attributes (past the staging buffer) 2 allocs
//	correlated (Bind)                  2 allocs, for the span context
//
// A real SDK adds its own cost on top: these are the adapter's, not the
// export pipeline's. Severities the SDK drops cost nothing beyond the
// Enabled check.
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
	if ts := entryTime(entry); !ts.IsZero() {
		rec.SetTimestamp(ts)
	}
	rec.SetSeverity(sev)
	rec.SetSeverityText(entry.Level.String())
	rec.SetBody(otellog.StringValue(entry.Message))

	// One AddAttributes call, not one per attribute: past the record's inline
	// capacity each call grows the overflow slice again.
	var stack [attrStackCap]otellog.KeyValue
	rec.AddAttributes(a.attributes(stack[:0], entry)...)
	return ctx, rec, true
}

// entryTime resolves the entry's timestamp the way the JSON formatter's
// entryUnixNanos does — the hot path writes TimestampUnix and may leave the
// wall-clock Timestamp zero, so preferring the other order dates records to
// the zero time. A zero result means the entry carried no timestamp at all
// and the SDK should stamp its own.
func entryTime(entry *types.LogEntry) time.Time {
	if entry.TimestampUnix != 0 {
		return time.Unix(0, entry.TimestampUnix)
	}
	return entry.Timestamp
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

// attributes appends the entry's fields, context, component, error, and source
// location to dst. The correlation fields are skipped: they are already the
// record's TraceID and SpanID, and re-emitting them as attributes would have
// the backend index the same value twice.
//
// Neither GetAllFields nor GetAllContext is used — both build a merged slice
// per call, which is an allocation this path can avoid by walking the three
// storage forms directly.
func (a *Adapter) attributes(dst []otellog.KeyValue, entry *types.LogEntry) []otellog.KeyValue {
	forEachField(entry, func(key string, f types.TypedFieldData) {
		switch key {
		case keyTraceID.Name, keySpanID.Name, keyTraceFlags.Name:
			return
		}
		dst = append(dst, otellog.KeyValue{Key: key, Value: logValue(f)})
	})
	for i := 0; i < entry.StaticContextCount && i < len(entry.StaticContext); i++ {
		f := entry.StaticContext[i]
		dst = append(dst, otellog.KeyValue{Key: fieldName(f), Value: logValue(f)})
	}
	for _, f := range entry.Context {
		dst = append(dst, otellog.KeyValue{Key: fieldName(f), Value: logValue(f)})
	}
	if entry.Component != "" {
		dst = append(dst, otellog.String("component", entry.Component))
	}
	if msg := errorMessage(entry); msg != "" {
		dst = append(dst, otellog.String("error", msg))
	}
	if entry.File != "" {
		dst = append(dst,
			otellog.String("code.file.path", entry.File),
			otellog.Int("code.line.number", entry.Line),
		)
	}
	return dst
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
	// Iterate, not GetAll: the snapshot helper appends into a fresh slice on
	// every call. Indexed values are always strings.
	if entry.IndexedStore != nil {
		entry.IndexedStore.Iterate(func(key, value string) {
			fn(key, types.TypedFieldData{Key: key, Val: types.StringValue(value)})
		})
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

// logValue converts a HaloLog field to its OpenTelemetry equivalent.
//
// A non-nil Value alongside typed storage means something rewrote the field
// after the builder set it, and masking is the only thing in the pipeline that
// does. The rewrite wins: preferring the typed original here would export the
// value the masker had just replaced, which on this boundary means shipping
// the PII off the host.
func logValue(f types.TypedFieldData) otellog.Value {
	if f.Val.Kind != types.KindUnknown && f.Value != nil {
		return anyValue(f.Value)
	}
	switch f.Val.Kind {
	case types.KindString, types.KindError:
		return otellog.StringValue(f.Val.String)
	case types.KindInt, types.KindInt64:
		return otellog.Int64Value(f.Val.Int64)
	case types.KindFloat64:
		return otellog.Float64Value(f.Val.Float64)
	case types.KindBool:
		return otellog.BoolValue(f.Val.Int64 != 0)
	case types.KindAny:
		return anyValue(f.Val.Any)
	case types.KindUnknown:
		return anyValue(f.Value)
	default:
		return anyValue(f.Value)
	}
}

// anyValue converts an interface-boxed value. Every fixed-width integer and
// float narrows to the two widths the log API models (int64, float64), which
// is lossless for all of them except uint64 above math.MaxInt64 — see
// uintValue. Unhandled types render through fmt.Sprint, matching what the JSON
// formatter does with the same value.
func anyValue(v interface{}) otellog.Value {
	switch t := v.(type) {
	case nil:
		return otellog.Value{}
	case string:
		return otellog.StringValue(t)
	case bool:
		return otellog.BoolValue(t)
	case int:
		return otellog.Int64Value(int64(t))
	case int8:
		return otellog.Int64Value(int64(t))
	case int16:
		return otellog.Int64Value(int64(t))
	case int32:
		return otellog.Int64Value(int64(t))
	case int64:
		return otellog.Int64Value(t)
	case uint:
		return uintValue(uint64(t))
	case uint8:
		return otellog.Int64Value(int64(t))
	case uint16:
		return otellog.Int64Value(int64(t))
	case uint32:
		return otellog.Int64Value(int64(t))
	case uint64:
		return uintValue(t)
	case uintptr:
		return uintValue(uint64(t))
	case float32:
		return otellog.Float64Value(float64(t))
	case float64:
		return otellog.Float64Value(t)
	case []byte:
		return bytesValue(t)
	case error:
		return otellog.StringValue(t.Error())
	default:
		return otellog.StringValue(fmt.Sprint(t))
	}
}

// uintValue keeps unsigned values exact. The log API has no unsigned integer
// kind, so anything above math.MaxInt64 would wrap to a negative number if it
// were cast; those emit as an exact decimal string instead. The type of an
// attribute therefore depends on its value near the top of the uint64 range —
// deliberate, and the alternative is silently wrong numbers.
func uintValue(v uint64) otellog.Value {
	if v > math.MaxInt64 {
		return otellog.StringValue(strconv.FormatUint(v, 10))
	}
	return otellog.Int64Value(int64(v))
}

// bytesValue copies the payload. log.BytesValue keeps a pointer to the
// caller's array rather than copying it, so without this a caller reusing its
// buffer after the logging call would rewrite an already-emitted record — and
// records outlive the call, both in the SDK's batch queue and in this
// adapter's contract that nothing survives into the pooled LogEntry. One
// allocation per []byte attribute is the price of that guarantee.
func bytesValue(v []byte) otellog.Value {
	if v == nil {
		return otellog.Value{}
	}
	owned := make([]byte, len(v))
	copy(owned, v)
	return otellog.BytesValue(owned)
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
