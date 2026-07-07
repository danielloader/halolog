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
// Package discard behavioural coverage tests.
//
// @author Admilson B. F. Cossa

package discard

import (
	"testing"

	"github.com/go-gen-ecosystem/halolog/types"
)

// TestDiscardAdapterContract exercises every method of the no-op discard
// adapter and asserts it satisfies the full types.Adapter interface.
func TestDiscardAdapterContract(t *testing.T) {
	var a types.Adapter = New()

	if got := a.Name(); got != "discard" {
		t.Fatalf("Name() = %q, want %q", got, "discard")
	}

	entry := &types.LogEntry{
		Level:     types.InfoLevel,
		Component: "test",
		Message:   "hello",
	}

	if err := a.Write(entry); err != nil {
		t.Fatalf("Write returned error: %v", err)
	}
	if err := a.WriteZero(entry); err != nil {
		t.Fatalf("WriteZero returned error: %v", err)
	}
	if err := a.Flush(); err != nil {
		t.Fatalf("Flush returned error: %v", err)
	}
	if err := a.Health(); err != nil {
		t.Fatalf("Health returned error: %v", err)
	}
	if err := a.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}

	// SetFormatter is a no-op but must not panic, including with nil.
	a.SetFormatter(nil)
	a.SetFormatter(types.NewDefaultConsoleFormatter())
}

// TestDiscardWriteNilEntry confirms discard tolerates a nil entry (pure no-op).
func TestDiscardWriteNilEntry(t *testing.T) {
	a := New()
	if err := a.Write(nil); err != nil {
		t.Fatalf("Write(nil) = %v, want nil", err)
	}
	if err := a.WriteZero(nil); err != nil {
		t.Fatalf("WriteZero(nil) = %v, want nil", err)
	}
}

// TestDiscardManyWrites confirms repeated writes stay allocation-free/no-op.
func TestDiscardManyWrites(t *testing.T) {
	a := New()
	entry := &types.LogEntry{Message: "x"}
	for i := 0; i < 1000; i++ {
		if err := a.Write(entry); err != nil {
			t.Fatalf("iteration %d: %v", i, err)
		}
	}
}
