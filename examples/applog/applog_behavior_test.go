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

// Package applog — HANDWRITTEN behavior tests for the generated facade (the
// generated files carry their own frozen-fragments test). Proves the facade
// logs correct JSON end-to-end, composes with bound contexts, and keeps the
// hot path at zero allocations.
// @author Admilson B. F. Cossa

package applog

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	jsonfmt "github.com/go-gen-ecosystem/halolog/adapters/formatters/json"
	"github.com/go-gen-ecosystem/halolog/adapters/outputs/console"
	"github.com/go-gen-ecosystem/halolog/core"
	"github.com/go-gen-ecosystem/halolog/types"
)

func newFacade(buf *bytes.Buffer) Logger {
	return Wrap(core.NewLogger(core.Config{
		Level:    types.InfoLevel,
		Adapters: []types.Adapter{console.NewWithWriter(buf, jsonfmt.NewJsonFormatter())},
	}))
}

func TestFacade_TypedLineEndToEnd(t *testing.T) {
	var buf bytes.Buffer
	log := newFacade(&buf)

	log.Info().
		UserID("alice").
		Tenant("acme").
		Status(200).
		DurationMS(42).
		Ratio(2.5).
		CacheHit(true).
		ErrField(errors.New("soft")).
		Msg("handled")

	var m map[string]any
	line := strings.TrimSpace(buf.String())
	if err := json.Unmarshal([]byte(line), &m); err != nil {
		t.Fatalf("facade emitted invalid JSON: %v — %q", err, line)
	}
	want := map[string]any{
		"user_id": "alice", "tenant": "acme", "status": float64(200),
		"duration_ms": float64(42), "ratio": 2.5, "cache_hit": true, "err": "soft",
	}
	for k, v := range want {
		if m[k] != v {
			t.Errorf("field %s = %v, want %v", k, m[k], v)
		}
	}
}

func TestFacade_ContextBindsOnce(t *testing.T) {
	var buf bytes.Buffer
	log := newFacade(&buf).With().Tenant("acme").UserID("alice").Logger()

	log.Info().Status(200).Msg("first")
	log.Warn().Status(500).Msg("second")

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
	for i, ln := range lines {
		var m map[string]any
		if err := json.Unmarshal([]byte(ln), &m); err != nil {
			t.Fatalf("line %d invalid: %v", i, err)
		}
		if m["tenant"] != "acme" || m["user_id"] != "alice" {
			t.Errorf("line %d lost bound context: %v", i, m)
		}
	}
}

func TestFacade_DisabledLevelIsFree(t *testing.T) {
	var buf bytes.Buffer
	log := newFacade(&buf) // Info level: Debug is filtered

	log.Debug().UserID("alice").Status(1).Msg("hidden")
	if buf.Len() != 0 {
		t.Fatalf("disabled level leaked output: %q", buf.String())
	}
}

func TestFacade_ZeroAllocHotPath(t *testing.T) {
	log := Wrap(core.NewLogger(core.Config{
		Level:    types.InfoLevel,
		Adapters: []types.Adapter{console.NewWithWriter(devNull{}, jsonfmt.NewJsonFormatter())},
	}))
	if allocs := testing.AllocsPerRun(1000, func() {
		log.Info().UserID("alice").Status(200).Msg("hot")
	}); allocs != 0 {
		t.Fatalf("facade hot path must allocate 0 times/op, got %.2f", allocs)
	}
}

type devNull struct{}

func (devNull) Write(p []byte) (int, error) { return len(p), nil }
