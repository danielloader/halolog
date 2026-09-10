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

// Package otelbridge — draining the export pipeline.
//
// Logger.Fatal writes its line, calls Logger.Flush, and exits from inside the
// logging call. If the adapter's Flush does not reach the provider, the last
// line a program ever writes dies in a batch queue with no caller left to
// drain it.
// @author Admilson B. F. Cossa
package otelbridge

import (
	"bytes"
	"context"
	"sync/atomic"
	"testing"

	jsonfmt "github.com/go-gen-ecosystem/halolog/adapters/formatters/json"
	"github.com/go-gen-ecosystem/halolog/adapters/outputs/console"
	"github.com/go-gen-ecosystem/halolog/core"
	"github.com/go-gen-ecosystem/halolog/types"
	otellog "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/embedded"
	sdklog "go.opentelemetry.io/otel/sdk/log"
)

// flushCountingProvider records ForceFlush calls without an SDK.
type flushCountingProvider struct {
	embedded.LoggerProvider
	rec     *recorder
	flushes atomic.Int64
	err     error
}

func (p *flushCountingProvider) Logger(name string, opts ...otellog.LoggerOption) otellog.Logger {
	return p.rec.Logger(name, opts...)
}

func (p *flushCountingProvider) ForceFlush(context.Context) error {
	p.flushes.Add(1)
	return p.err
}

func TestAdapter_FlushAndCloseDrainTheProvider(t *testing.T) {
	p := &flushCountingProvider{rec: &recorder{}}
	a := NewAdapter("halolog/otelbridge_test", WithLoggerProvider(p))

	if err := a.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}
	if got := p.flushes.Load(); got != 1 {
		t.Fatalf("Flush reached the provider %d times, want 1", got)
	}

	if err := a.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if got := p.flushes.Load(); got != 2 {
		t.Fatalf("Close must drain too; provider saw %d flushes, want 2", got)
	}
}

// A provider with no ForceFlush is not an error; there is simply nothing to
// drain.
func TestAdapter_FlushIsQuietWithoutForceFlush(t *testing.T) {
	a := NewAdapter("halolog/otelbridge_test", WithLoggerProvider(&recorder{}))
	if err := a.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}
	if err := a.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func TestAdapter_FlushReportsProviderFailure(t *testing.T) {
	p := &flushCountingProvider{rec: &recorder{}, err: context.DeadlineExceeded}
	a := NewAdapter("halolog/otelbridge_test", WithLoggerProvider(p))

	if err := a.Flush(); err == nil {
		t.Fatal("a failing ForceFlush must surface through Flush")
	}
}

// The case that matters: a batching processor, a Fatal line, and no caller
// left to drain anything afterwards.
func TestSDK_FatalRecordSurvivesTheExit(t *testing.T) {
	exp := &memExporter{}
	provider := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewBatchProcessor(exp)),
	)
	t.Cleanup(func() {
		if err := provider.Shutdown(context.Background()); err != nil {
			t.Errorf("provider shutdown: %v", err)
		}
	})

	var buf bytes.Buffer
	var exited atomic.Int64
	logger := core.NewLogger(core.Config{
		Level:    types.InfoLevel,
		ExitFunc: func(int) { exited.Add(1) },
		Adapters: []types.Adapter{
			console.NewWithWriter(&buf, jsonfmt.NewJsonFormatter()),
			NewAdapter("github.com/acme/checkout", WithLoggerProvider(provider)),
		},
	})

	logger.Fatal("the last line this program writes")

	if exited.Load() != 1 {
		t.Fatalf("Fatal must go through the exit hook once, got %d", exited.Load())
	}
	if !bytes.Contains(buf.Bytes(), []byte("the last line this program writes")) {
		t.Fatalf("console adapter missed the fatal line: %s", buf.String())
	}

	// No ForceFlush here on purpose: Logger.Flush inside Fatal is the only
	// drain a real program gets before os.Exit.
	rec := exp.only(t)
	if got := rec.Body().AsString(); got != "the last line this program writes" {
		t.Fatalf("exported body = %q", got)
	}
	if got := rec.Severity(); got != otellog.SeverityFatal1 {
		t.Fatalf("exported severity = %v, want %v", got, otellog.SeverityFatal1)
	}
}
