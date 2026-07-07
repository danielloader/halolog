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
//
// Package adapters behavioural coverage tests.
//
// @author Admilson B. F. Cossa

package adapters

import (
	"testing"
	"time"

	"github.com/go-gen-ecosystem/halolog/types"
)

// TestPerfNullAdapterContract exercises every method of the high-performance
// null adapter and confirms it satisfies the full types.Adapter interface.
func TestPerfNullAdapterContract(t *testing.T) {
	var a types.Adapter = &PerfNullAdapter{}

	if got := a.Name(); got != "perf_null" {
		t.Fatalf("Name() = %q, want %q", got, "perf_null")
	}

	entry := &types.LogEntry{
		Level:     types.WarnLevel,
		Component: "svc",
		Message:   "benchmark entry",
	}

	if err := a.Write(entry); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if err := a.WriteZero(entry); err != nil {
		t.Fatalf("WriteZero: %v", err)
	}
	if err := a.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}
	if err := a.Health(); err != nil {
		t.Fatalf("Health: %v", err)
	}
	if err := a.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// SetFormatter is a no-op; must accept nil and a real formatter safely.
	a.SetFormatter(nil)
	a.SetFormatter(types.NewDefaultConsoleFormatter())
}

// TestPerfNullAdapterNilEntry confirms the null adapter tolerates nil entries.
func TestPerfNullAdapterNilEntry(t *testing.T) {
	a := &PerfNullAdapter{}
	if err := a.Write(nil); err != nil {
		t.Fatalf("Write(nil): %v", err)
	}
	if err := a.WriteZero(nil); err != nil {
		t.Fatalf("WriteZero(nil): %v", err)
	}
}

// TestNewFileAdapter validates the placeholder file-adapter constructor and its
// SetFormatter no-op. It returns a usable value with a nil error.
func TestNewFileAdapter(t *testing.T) {
	cfg := &RotationConfig{
		MaxSize:    1024,
		MaxBackups: 3,
		MaxAge:     24 * time.Hour,
	}

	fa, err := NewFileAdapter("/var/log/app.log", cfg)
	if err != nil {
		t.Fatalf("NewFileAdapter returned error: %v", err)
	}
	if fa == nil {
		t.Fatal("NewFileAdapter returned nil adapter")
	}

	// SetFormatter is a no-op placeholder; must not panic.
	fa.SetFormatter(nil)
	fa.SetFormatter(types.NewDefaultConsoleFormatter())
}

// TestNewFileAdapterNilConfig confirms the constructor tolerates a nil config.
func TestNewFileAdapterNilConfig(t *testing.T) {
	fa, err := NewFileAdapter("app.log", nil)
	if err != nil {
		t.Fatalf("NewFileAdapter(nil cfg): %v", err)
	}
	if fa == nil {
		t.Fatal("NewFileAdapter(nil cfg) returned nil")
	}
}

// TestRotationConfigFields confirms the RotationConfig struct carries values.
func TestRotationConfigFields(t *testing.T) {
	cfg := RotationConfig{
		MaxSize:    2048,
		MaxBackups: 5,
		MaxAge:     7 * 24 * time.Hour,
	}
	if cfg.MaxSize != 2048 || cfg.MaxBackups != 5 || cfg.MaxAge != 7*24*time.Hour {
		t.Fatalf("unexpected RotationConfig values: %+v", cfg)
	}
}

// TestAdapterOptionType exercises the AdapterOption functional-option type by
// constructing an option that mutates observable state when applied.
func TestAdapterOptionType(t *testing.T) {
	applied := false
	var opt AdapterOption = func(a types.Adapter) error {
		applied = true
		if a == nil {
			t.Error("option received nil adapter")
		}
		return nil
	}

	if err := opt(&PerfNullAdapter{}); err != nil {
		t.Fatalf("option returned error: %v", err)
	}
	if !applied {
		t.Fatal("AdapterOption was not applied")
	}
}

// TestConsoleAndFileAdapterEmbedding confirms the exported struct wrappers
// embed the types.Adapter interface and are constructible as zero values.
func TestConsoleAndFileAdapterEmbedding(t *testing.T) {
	// ConsoleAdapter embeds types.Adapter; zero value has a nil embedded
	// interface but must be constructible for downstream wiring.
	var c ConsoleAdapter
	_ = c

	var f FileAdapter
	// The FileAdapter placeholder SetFormatter is safe on a zero value.
	f.SetFormatter(nil)
}
