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

// Package types coverage tests for adapters, registries, clocks, formatters,
// config accessors, and LogEntry Indexed-storage integration.
// @author Admilson B. F. Cossa

package types

import (
	"errors"
	"testing"
	"time"
)

func TestFuncAdapter(t *testing.T) {
	var captured struct {
		level LogLevel
		msg   string
	}
	adapter := &FuncAdapter{
		WriteFunc: func(e *LogEntry) error {
			// Copy fields inside the callback (pool recycles the entry).
			captured.level = e.Level
			captured.msg = e.Message
			return nil
		},
	}

	if adapter.Name() != "func_adapter" {
		t.Errorf("Name wrong: %q", adapter.Name())
	}

	e := NewLogEntry(ErrorLevel, "the-msg")
	if err := adapter.Write(e); err != nil {
		t.Errorf("Write returned error: %v", err)
	}
	if captured.level != ErrorLevel || captured.msg != "the-msg" {
		t.Errorf("Write did not invoke callback correctly: %+v", captured)
	}

	if err := adapter.WriteZero(e); err != nil {
		t.Errorf("WriteZero returned error: %v", err)
	}

	if err := adapter.Flush(); err != nil {
		t.Errorf("Flush should be nil, got %v", err)
	}
	if err := adapter.Close(); err != nil {
		t.Errorf("Close should be nil, got %v", err)
	}
	if err := adapter.Health(); err != nil {
		t.Errorf("Health should be nil, got %v", err)
	}
	// SetFormatter is a no-op but must be callable.
	adapter.SetFormatter(NewDefaultConsoleFormatter())
}

func TestFuncAdapter_PropagatesError(t *testing.T) {
	sentinel := errors.New("write failed")
	adapter := &FuncAdapter{
		WriteFunc: func(e *LogEntry) error { return sentinel },
	}
	e := NewLogEntry(InfoLevel, "x")
	if err := adapter.Write(e); !errors.Is(err, sentinel) {
		t.Errorf("Write should propagate error, got %v", err)
	}
	if err := adapter.WriteZero(e); !errors.Is(err, sentinel) {
		t.Errorf("WriteZero should propagate error, got %v", err)
	}
}

func TestAdapterRegistry(t *testing.T) {
	var reg AdapterRegistry

	if reg.HasAdapter(AdapterTypeConsole) {
		t.Error("empty registry should not have console")
	}

	reg.RegisterAdapter(AdapterTypeConsole)
	reg.RegisterAdapter(AdapterTypeFile)

	if !reg.HasAdapter(AdapterTypeConsole) {
		t.Error("console should be registered")
	}
	if !reg.HasAdapter(AdapterTypeFile) {
		t.Error("file should be registered")
	}
	if reg.HasAdapter(AdapterTypeHTTP) {
		t.Error("HTTP should not be registered")
	}

	// Registering again is idempotent.
	reg.RegisterAdapter(AdapterTypeConsole)
	if !reg.HasAdapter(AdapterTypeConsole) {
		t.Error("console should still be registered")
	}

	reg.UnregisterAdapter(AdapterTypeConsole)
	if reg.HasAdapter(AdapterTypeConsole) {
		t.Error("console should be unregistered")
	}
	if !reg.HasAdapter(AdapterTypeFile) {
		t.Error("file should remain registered after unregistering console")
	}
}

func TestDefaultConsoleFormatter(t *testing.T) {
	f := NewDefaultConsoleFormatter()
	if f.TimeFormat != "15:04:05" || !f.ShowLevel || !f.ShowTime {
		t.Errorf("default formatter defaults wrong: %+v", f)
	}
	if f.EstimatedSize() != 256 {
		t.Errorf("EstimatedSize wrong: %d", f.EstimatedSize())
	}
	f.Reset() // no-op, must not panic

	e := NewLogEntry(WarnLevel, "hello world")
	buf := make([]byte, 0, 64)
	out := f.Format(e, buf)
	got := string(out)
	if got != "[WARN] hello world\n" {
		t.Errorf("Format output wrong: %q", got)
	}
}

func TestDefaultConsoleFormatter_NilEntry(t *testing.T) {
	f := NewDefaultConsoleFormatter()
	buf := make([]byte, 0, 8)
	out := f.Format(nil, buf)
	if len(out) != 0 {
		t.Errorf("Format(nil) should return empty, got %q", string(out))
	}
}

func TestCachedClock(t *testing.T) {
	cc := NewCachedClock(time.Hour)
	if cc == nil {
		t.Fatal("NewCachedClock returned nil")
	}
	now := cc.Now()
	if now.IsZero() {
		t.Error("CachedClock.Now returned zero time")
	}
	// With a long interval, a second call returns the cached (equal) value.
	again := cc.Now()
	if !again.Equal(now) {
		t.Errorf("cached clock should return same time within interval: %v vs %v", now, again)
	}
}

func TestCachedClock_Refresh(t *testing.T) {
	// Zero interval forces a refresh on every call.
	cc := NewCachedClock(0)
	first := cc.Now()
	second := cc.Now()
	if second.Before(first) {
		t.Errorf("refreshed clock went backwards: %v -> %v", first, second)
	}
}

func TestUnixTimestamp_ToTime(t *testing.T) {
	nanos := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC).UnixNano()
	ts := UnixTimestamp(nanos)
	got := ts.ToTime()
	if got.UnixNano() != nanos {
		t.Errorf("UnixTimestamp.ToTime roundtrip wrong: %d vs %d", got.UnixNano(), nanos)
	}
}

func TestOutputStringers(t *testing.T) {
	co := &ConsoleOutput{}
	if co.String() != "console" {
		t.Errorf("ConsoleOutput.String wrong: %q", co.String())
	}
	if err := co.Write(LogEntry{}); err != nil {
		t.Errorf("ConsoleOutput.Write should be nil: %v", err)
	}
	if err := co.Close(); err != nil {
		t.Errorf("ConsoleOutput.Close should be nil: %v", err)
	}

	fo := &FileOutput{Path: "/var/log/app.log"}
	if fo.String() != "file:/var/log/app.log" {
		t.Errorf("FileOutput.String wrong: %q", fo.String())
	}
	if err := fo.Write(LogEntry{}); err != nil {
		t.Errorf("FileOutput.Write should be nil: %v", err)
	}
	if err := fo.Close(); err != nil {
		t.Errorf("FileOutput.Close should be nil: %v", err)
	}

	no := &NetworkOutput{Endpoint: "tcp://host:1234"}
	if no.String() != "network:tcp://host:1234" {
		t.Errorf("NetworkOutput.String wrong: %q", no.String())
	}
	if err := no.Write(LogEntry{}); err != nil {
		t.Errorf("NetworkOutput.Write should be nil: %v", err)
	}
	if err := no.Close(); err != nil {
		t.Errorf("NetworkOutput.Close should be nil: %v", err)
	}
}

func TestLoggerConfig_Accessors(t *testing.T) {
	cfg := &LoggerConfig{
		Level:               WarnLevel,
		SamplingRate:        0.25,
		BufferSize:          4096,
		FastPathEnabled:     true,
		MaxFieldsPerLog:     16,
		AutoMaskEnabled:     true,
		MetricsEnabled:      true,
		FlushInterval:       2 * time.Second,
		HealthCheckInterval: 5 * time.Second,
		MaskingRules:        []MaskingRule{{Pattern: "p", Replace: "r", Type: "exact"}},
	}

	if cfg.GetLevel() != WarnLevel {
		t.Errorf("GetLevel wrong: %v", cfg.GetLevel())
	}
	if cfg.GetSamplingRate() != 0.25 {
		t.Errorf("GetSamplingRate wrong: %v", cfg.GetSamplingRate())
	}
	if cfg.GetBufferSize() != 4096 {
		t.Errorf("GetBufferSize wrong: %d", cfg.GetBufferSize())
	}
	if !cfg.IsFastPathEnabled() {
		t.Errorf("IsFastPathEnabled wrong")
	}
	if cfg.GetStaticFieldCount() != 16 {
		t.Errorf("GetStaticFieldCount wrong: %d", cfg.GetStaticFieldCount())
	}
	if !cfg.IsAutoMaskEnabled() {
		t.Errorf("IsAutoMaskEnabled wrong")
	}
	if !cfg.IsMetricsEnabled() {
		t.Errorf("IsMetricsEnabled wrong")
	}
	if cfg.GetFlushInterval() != int64(2*time.Second) {
		t.Errorf("GetFlushInterval wrong: %d", cfg.GetFlushInterval())
	}
	if cfg.GetHealthCheckInterval() != int64(5*time.Second) {
		t.Errorf("GetHealthCheckInterval wrong: %d", cfg.GetHealthCheckInterval())
	}
	if len(cfg.GetMaskingRules()) != 1 {
		t.Errorf("GetMaskingRules wrong: %v", cfg.GetMaskingRules())
	}

	// Constant-return accessors.
	if cfg.GetOutputs() != nil {
		t.Errorf("GetOutputs should be nil")
	}
	if cfg.GetHooks() != nil {
		t.Errorf("GetHooks should be nil")
	}
	if cfg.GetBatchSize() != 100 {
		t.Errorf("GetBatchSize default wrong: %d", cfg.GetBatchSize())
	}
	if cfg.GetBatchTimeout() != 1000 {
		t.Errorf("GetBatchTimeout default wrong: %d", cfg.GetBatchTimeout())
	}
	if cfg.GetFieldPoolSize() != 1024 {
		t.Errorf("GetFieldPoolSize default wrong: %d", cfg.GetFieldPoolSize())
	}
}

func TestLogEntry_IndexedIntegration_EnableAndInspect(t *testing.T) {
	e := NewLogEntry(InfoLevel, "x")

	// Enabling Indexed storage installs a store.
	e.EnableIndexedStorage(32)
	if !e.IsUsingIndexedStorage() {
		t.Fatal("IsUsingIndexedStorage should be true after enable")
	}

	// NOTE: e.AddIndexedField(...) is intentionally NOT exercised here.
	// LogEntry.EnableIndexedStorage constructs the store with a nil
	// FieldDictionary, so AddIndexedField -> Set -> getOrCreateFieldID falls
	// back to hashString(key), producing enormous field IDs and an
	// effectively unbounded growChunk loop (hang/OOM). See the returned bug
	// report. We instead drive the store directly via a bounded dictionary
	// in TestLogEntry_IndexedIntegration_DictionaryBacked.

	// GetIndexedField on an empty store returns not-found without hanging.
	if _, ok := e.GetIndexedField("nope"); ok {
		t.Errorf("GetIndexedField missing should be false")
	}

	// GetAllFields on an entry with an empty Indexed store is safe.
	if all := e.GetAllFields(); all == nil {
		t.Errorf("GetAllFields should return a (possibly empty) slice")
	}

	// MergeIndexedFieldsIntoTyped with an empty store is a no-op.
	before := len(e.Fields)
	e.MergeIndexedFieldsIntoTyped()
	if len(e.Fields) != before {
		t.Errorf("MergeIndexedFieldsIntoTyped on empty store should not change Fields")
	}
}

func TestLogEntry_IndexedIntegration_DictionaryBacked(t *testing.T) {
	// Exercise the Indexed-store integration on a LogEntry by injecting a
	// bounded dictionary-backed store, avoiding the nil-dictionary hash-ID hang.
	e := NewLogEntry(InfoLevel, "x")
	dict := newTestDictionary()
	e.IndexedStore = NewIndexedFieldStoreWithDict(16, dict)
	e.UseIndexedStorage = true

	e.IndexedStore.Set("qk", "qv")

	if v, ok := e.GetIndexedField("qk"); !ok || v != "qv" {
		t.Errorf("Indexed field roundtrip wrong: %v %v", v, ok)
	}

	// GetAllFields includes Indexed fields.
	all := e.GetAllFields()
	found := false
	for _, f := range all {
		if f.Key == "qk" {
			found = true
		}
	}
	if !found {
		t.Errorf("GetAllFields should include Indexed field 'qk': %+v", all)
	}

	// MergeIndexedFieldsIntoTyped copies Indexed fields into Fields.
	before := len(e.Fields)
	e.MergeIndexedFieldsIntoTyped()
	if len(e.Fields) <= before {
		t.Errorf("MergeIndexedFieldsIntoTyped should append fields (before=%d after=%d)", before, len(e.Fields))
	}
}

func TestLogEntry_IndexedNotEnabled(t *testing.T) {
	e := NewLogEntry(InfoLevel, "x")
	if e.IsUsingIndexedStorage() {
		t.Error("fresh entry should not use Indexed storage")
	}
	if _, ok := e.GetIndexedField("k"); ok {
		t.Error("GetIndexedField with no store should be false")
	}
	// MergeIndexedFieldsIntoTyped is a no-op when no store present.
	e.MergeIndexedFieldsIntoTyped()
}
