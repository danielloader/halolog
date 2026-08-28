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

// Package otelbridge correlates HaloLog lines with OpenTelemetry traces.
//
// Bind derives a child logger whose trace_id and span_id are bound once —
// encoded to their final bytes at bind time (see core.Context) — so every
// line in the request pays a single memcpy for its correlation fields, not a
// per-line hex encode. This is a separate Go module: the core logger stays
// free of OpenTelemetry dependencies; only programs importing otelbridge
// pull go.opentelemetry.io/otel/trace.
//
//	func handle(w http.ResponseWriter, r *http.Request) {
//	    log := otelbridge.Bind(r.Context(), baseLogger)
//	    log.Info("handling")            // …,"trace_id":"…","span_id":"…"
//	}
//
// Author: Admilson B. F. Cossa
package otelbridge

import (
	"context"

	"github.com/go-gen-ecosystem/halolog"
	"github.com/go-gen-ecosystem/halolog/core"
	"go.opentelemetry.io/otel/trace"
)

// Field keys follow the OpenTelemetry log correlation convention
// (trace_id, span_id, trace_flags).
var (
	keyTraceID    = halolog.Key("trace_id")
	keySpanID     = halolog.Key("span_id")
	keyTraceFlags = halolog.Key("trace_flags")
)

// Bind returns a child of l carrying the trace_id, span_id, and trace_flags
// (the W3C sampled bit, as two hex digits) of the span in ctx, hex-encoded
// once at bind time. When ctx carries no valid span context it returns l
// unchanged, so it is always safe to call.
func Bind(ctx context.Context, l *core.Logger) *core.Logger {
	sc := trace.SpanContextFromContext(ctx)
	if !sc.IsValid() {
		return l
	}
	return l.With().
		Str(keyTraceID, sc.TraceID().String()).
		Str(keySpanID, sc.SpanID().String()).
		Str(keyTraceFlags, sc.TraceFlags().String()).
		Logger()
}
