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

package core

import (
	"testing"

	"github.com/go-gen-ecosystem/halolog/types"
)

func TestBuilder_Basic(t *testing.T) {
	logger := New().
		Component("test-service").
		Debug().
		Discard().
		MustBuild()

	if logger.Component() != "test-service" {
		t.Errorf("Expected component 'test-service', got '%s'", logger.Component())
	}

	if logger.Level() != types.DebugLevel {
		t.Errorf("Expected DebugLevel, got %v", logger.Level())
	}
}

func TestBuilder_WithMetrics(t *testing.T) {
	logger := New().
		Component("metrics-test").
		Discard().
		Metrics().
		MustBuild()

	if logger.metrics == nil {
		t.Fatal("Expected metrics to be enabled")
	}

	logger.Info("test message")
}

func TestBuilder_DefaultAdapter(t *testing.T) {
	// No adapter specified - should default to discard
	logger := New().
		Component("default-test").
		MustBuild()

	if logger.discardAdapter == nil {
		t.Fatal("Expected discard adapter to be set by default")
	}
}

func TestBuilder_MultipleAdapters(t *testing.T) {
	adapter1 := &testAdapter{name: "adapter1"}
	adapter2 := &testAdapter{name: "adapter2"}

	logger := New().
		Component("multi-adapter").
		Adapters(adapter1, adapter2).
		MustBuild()

	if len(logger.adapters) != 2 {
		t.Errorf("Expected 2 adapters, got %d", len(logger.adapters))
	}
}

type testAdapter struct {
	name string
}

func (a *testAdapter) Name() string                          { return a.name }
func (a *testAdapter) Write(entry *types.LogEntry) error     { return nil }
func (a *testAdapter) WriteZero(entry *types.LogEntry) error { return nil }
func (a *testAdapter) Flush() error                          { return nil }
func (a *testAdapter) Close() error                          { return nil }
func (a *testAdapter) SetFormatter(f types.Formatter)        {}
func (a *testAdapter) Health() error                         { return nil }
