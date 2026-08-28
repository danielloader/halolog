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

// Package otelbridge — correlation behavior tests.
// @author Admilson B. F. Cossa
package otelbridge

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	jsonfmt "github.com/go-gen-ecosystem/halolog/adapters/formatters/json"
	"github.com/go-gen-ecosystem/halolog/adapters/outputs/console"
	"github.com/go-gen-ecosystem/halolog/core"
	"github.com/go-gen-ecosystem/halolog/types"
	"go.opentelemetry.io/otel/trace"
)

func newTestLogger(buf *bytes.Buffer) *core.Logger {
	return core.NewLogger(core.Config{
		Level:    types.InfoLevel,
		Adapters: []types.Adapter{console.NewWithWriter(buf, jsonfmt.NewJsonFormatter())},
	})
}

func TestBind_CorrelatesEveryLine(t *testing.T) {
	var buf bytes.Buffer
	base := newTestLogger(&buf)

	traceID, _ := trace.TraceIDFromHex("4bf92f3577b34da6a3ce929d0e0e4736")
	spanID, _ := trace.SpanIDFromHex("00f067aa0ba902b7")
	ctx := trace.ContextWithSpanContext(context.Background(), trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
	}))

	log := Bind(ctx, base)
	if log == base {
		t.Fatal("Bind with a valid span must derive a child")
	}
	log.Info("first")
	log.Typed().WithInt("status", 200).Info("second")

	for i, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatalf("line %d invalid JSON: %v", i, err)
		}
		if m["trace_id"] != "4bf92f3577b34da6a3ce929d0e0e4736" || m["span_id"] != "00f067aa0ba902b7" {
			t.Fatalf("line %d missing correlation: %v", i, m)
		}
		if m["trace_flags"] != "01" { // FlagsSampled, per the W3C two-hex-digit form
			t.Fatalf("line %d missing/wrong trace_flags: %v", i, m)
		}
	}
}

func TestBind_NoSpanReturnsSameLogger(t *testing.T) {
	var buf bytes.Buffer
	base := newTestLogger(&buf)
	if got := Bind(context.Background(), base); got != base {
		t.Fatal("Bind without a span context must return the logger unchanged")
	}
}

func TestBind_ZeroAllocPerLineAfterBinding(t *testing.T) {
	base := core.NewLogger(core.Config{
		Level:    types.InfoLevel,
		Adapters: []types.Adapter{console.NewWithWriter(discardWriter{}, jsonfmt.NewJsonFormatter())},
	})
	traceID, _ := trace.TraceIDFromHex("4bf92f3577b34da6a3ce929d0e0e4736")
	spanID, _ := trace.SpanIDFromHex("00f067aa0ba902b7")
	ctx := trace.ContextWithSpanContext(context.Background(), trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: traceID, SpanID: spanID, TraceFlags: trace.FlagsSampled,
	}))
	log := Bind(ctx, base)

	if allocs := testing.AllocsPerRun(1000, func() {
		log.Info("correlated hot path")
	}); allocs != 0 {
		t.Fatalf("bound correlated line must allocate 0 times/op, got %.2f", allocs)
	}
}

type discardWriter struct{}

func (discardWriter) Write(p []byte) (int, error) { return len(p), nil }
