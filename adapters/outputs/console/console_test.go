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

// @author Admilson B. F. Cossa
package console

import (
	"bytes"
	"strings"
	"testing"

	"github.com/go-gen-ecosystem/halolog/core"
	"github.com/go-gen-ecosystem/halolog/types"
)

// Compile-time proof the adapter satisfies the full contract.
var _ types.Adapter = (*Adapter)(nil)

func TestConsoleAdapter_WritesFormattedEntryEndToEnd(t *testing.T) {
	var buf bytes.Buffer
	logger := core.New().Component("test").Adapter(NewWithWriter(&buf, nil)).MustBuild()

	logger.Info("hello console")

	out := buf.String()
	if !strings.Contains(out, "hello console") {
		t.Fatalf("expected output to contain the message, got %q", out)
	}
	if !strings.HasSuffix(out, "\n") {
		t.Errorf("expected a trailing newline, got %q", out)
	}
}

func TestConsoleAdapter_Contract(t *testing.T) {
	var buf bytes.Buffer
	a := NewWithWriter(&buf, nil)

	if a.Name() != "console" {
		t.Errorf("Name() = %q, want console", a.Name())
	}
	if err := a.Health(); err != nil {
		t.Errorf("Health() = %v, want nil", err)
	}
	if err := a.Flush(); err != nil {
		t.Errorf("Flush() = %v, want nil", err)
	}
	if err := a.Write(nil); err != nil {
		t.Errorf("Write(nil) = %v, want nil (no-op)", err)
	}

	// After Close: writes are no-ops and Health reports unhealthy.
	if err := a.Close(); err != nil {
		t.Fatalf("Close() = %v", err)
	}
	if err := a.Health(); err == nil {
		t.Error("Health() after Close should report an error")
	}
	buf.Reset()
	if err := a.Write(&types.LogEntry{Message: "dropped"}); err != nil {
		t.Errorf("Write after Close = %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("Write after Close should be a no-op, wrote %q", buf.String())
	}
}

func TestConsoleAdapter_SetFormatterAndDefaults(t *testing.T) {
	// New() defaults to stdout — just assert it constructs a usable adapter.
	if New().Name() != "console" {
		t.Fatal("New() should produce a console adapter")
	}

	var buf bytes.Buffer
	a := NewWithWriter(&buf, nil)
	a.SetFormatter(nil) // ignored
	a.SetFormatter(types.NewDefaultConsoleFormatter())
	if err := a.Write(&types.LogEntry{Message: "after set formatter"}); err != nil {
		t.Fatalf("Write = %v", err)
	}
	if !strings.Contains(buf.String(), "after set formatter") {
		t.Errorf("expected formatted output, got %q", buf.String())
	}
}
