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
// Package adapters provides output adapters
// Author: Admilson B. F. Cossa

package syslog

import (
	"testing"
)

func TestSyslogUnixAdapter_Basic(t *testing.T) {
	adapter := NewSyslogAdapter("test")
	if adapter == nil {
		t.Fatal("NewSyslogAdapter returned nil")
	}

	// Test basic properties
	if adapter.Name() == "" {
		t.Error("SyslogUnix adapter should have a name")
	}

	// Test that adapter can be closed
	err := adapter.Close()
	if err != nil {
		t.Errorf("Close should not error: %v", err)
	}

	// Test that adapter can be flushed
	err = adapter.Flush()
	if err != nil {
		t.Errorf("Flush should not error: %v", err)
	}
}
