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

// Package otelbridge — the adapter's allocation contract.
//
// This export boundary is not a zero-allocation path and cannot be: Emit takes
// a Record by value through an interface, so the record escapes on every line.
// What it can have is a fixed, measured cost per shape. Each case below is a
// ceiling, not a target — raise one only with a reason.
// @author Admilson B. F. Cossa
package otelbridge

import (
	"context"
	"strconv"
	"testing"

	"github.com/go-gen-ecosystem/halolog/types"
	otellog "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/embedded"
)

// nullLogger discards records, so a measurement is the adapter's own cost and
// not an SDK's.
type nullLogger struct{ embedded.Logger }

func (nullLogger) Emit(context.Context, otellog.Record)                    {}
func (nullLogger) Enabled(context.Context, otellog.EnabledParameters) bool { return true }

type nullProvider struct{ embedded.LoggerProvider }

func (nullProvider) Logger(string, ...otellog.LoggerOption) otellog.Logger { return nullLogger{} }

func benchAdapter() *Adapter {
	return NewAdapter("halolog/otelbridge_bench", WithLoggerProvider(nullProvider{}))
}

// entryWith builds an entry carrying n plain fields, the shape the attribute
// staging buffer and log.Record's 5-slot inline array both key off.
func entryWith(n int) *types.LogEntry {
	entry := &types.LogEntry{
		Level:         types.InfoLevel,
		Message:       "measured",
		TimestampUnix: 1757535481000000000,
	}
	for i := range n {
		entry.Fields = append(entry.Fields, types.TypedFieldData{
			Key: "field" + strconv.Itoa(i),
			Val: types.IntValue(i),
		})
	}
	return entry
}

func correlatedEntry() *types.LogEntry {
	entry := entryWith(2)
	entry.Fields = append(entry.Fields,
		types.TypedFieldData{Key: keyTraceID.Name, Val: types.StringValue(testTraceHex)},
		types.TypedFieldData{Key: keySpanID.Name, Val: types.StringValue(testSpanHex)},
		types.TypedFieldData{Key: keyTraceFlags.Name, Val: types.StringValue("01")},
	)
	return entry
}

func contextEntry() *types.LogEntry {
	entry := entryWith(2)
	entry.Context = []types.TypedFieldData{
		{Key: "tenant", Val: types.StringValue("acme")},
		{Key: "region", Val: types.StringValue("eu-west-1")},
	}
	return entry
}

func indexedEntry() *types.LogEntry {
	entry := entryWith(2)
	entry.EnableIndexedStorage(8)
	entry.AddIndexedField("route", "/v1/quotes")
	entry.AddIndexedField("method", "POST")
	return entry
}

// allocBudget is the per-case ceiling in allocations per emitted record.
var allocBudget = []struct {
	name   string
	entry  func() *types.LogEntry
	budget float64
}{
	{"no fields", func() *types.LogEntry { return entryWith(0) }, 1},
	{"5 fields (record inline capacity)", func() *types.LogEntry { return entryWith(5) }, 1},
	{"6 fields (spills past inline)", func() *types.LogEntry { return entryWith(6) }, 2},
	{"9 fields (spills past the staging buffer)", func() *types.LogEntry { return entryWith(9) }, 3},
	{"context fields", contextEntry, 1},
	{"indexed fields", indexedEntry, 1},
	{"correlated", correlatedEntry, 2},
}

func TestAdapter_AllocationBudgets(t *testing.T) {
	a := benchAdapter()
	for _, tc := range allocBudget {
		t.Run(tc.name, func(t *testing.T) {
			entry := tc.entry()
			got := testing.AllocsPerRun(200, func() { _ = a.Write(entry) })
			if got > tc.budget {
				t.Fatalf("%.1f allocs/op exceeds the budget of %.0f", got, tc.budget)
			}
			t.Logf("%.1f allocs/op (budget %.0f)", got, tc.budget)
		})
	}
}

// A dropped severity must not pay for any per-entry work — including the
// correlation walk, which is the shape a Bind-ed request logger produces and
// the one an uncorrelated entry cannot exercise.
func TestAdapter_DisabledSeverityCostsNothing(t *testing.T) {
	a := NewAdapter("halolog/otelbridge_bench", WithLoggerProvider(&recorder{disableAll: true}))

	for _, tc := range []struct {
		name  string
		entry *types.LogEntry
	}{
		{"uncorrelated", entryWith(9)},
		{"correlated", correlatedEntry()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := testing.AllocsPerRun(200, func() { _ = a.Write(tc.entry) }); got != 0 {
				t.Fatalf("%.1f allocs/op for a record the SDK drops, want 0", got)
			}
		})
	}
}

func BenchmarkAdapterWrite(b *testing.B) {
	a := benchAdapter()
	for _, tc := range allocBudget {
		b.Run(tc.name, func(b *testing.B) {
			entry := tc.entry()
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				_ = a.Write(entry)
			}
		})
	}
}

// Parallel emission, which is how a server actually logs: interleaved requests
// each with their own span. It is also the evidence against sharing any
// single-entry span cache across goroutines.
func BenchmarkAdapterWriteParallel(b *testing.B) {
	a := benchAdapter()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		entry := correlatedEntry()
		for pb.Next() {
			_ = a.Write(entry)
		}
	})
}
