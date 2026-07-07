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
// Package file secure-default permission tests.
// Author: Admilson B. F. Cossa

package file

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestFileAdapter_SecureDefaultPermissions asserts that the shipped defaults are
// owner-only: log files are created 0600 and their parent directory 0700. Log
// data may contain PII/PHI even with masking, so world-readable defaults would
// contradict the module's GDPR/HIPAA/PCI positioning. Skipped on Windows where
// the Unix permission bits are not enforced by the OS.
func TestFileAdapter_SecureDefaultPermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix permission bits are not enforced on Windows")
	}

	tempDir := t.TempDir()
	// Nest one level so MkdirAll actually creates a directory we can inspect.
	logDir := filepath.Join(tempDir, "logs")
	logFile := filepath.Join(logDir, "secure.log")

	adapter, err := NewFileAdapter(logFile, nil)
	if err != nil {
		t.Fatalf("NewFileAdapter failed: %v", err)
	}
	defer func() { _ = adapter.Close() }()

	fi, err := os.Stat(logFile)
	if err != nil {
		t.Fatalf("stat log file: %v", err)
	}
	if got := fi.Mode().Perm(); got != 0o600 {
		t.Errorf("log file mode = %o, want 0600 (owner-only)", got)
	}

	di, err := os.Stat(logDir)
	if err != nil {
		t.Fatalf("stat log dir: %v", err)
	}
	if got := di.Mode().Perm(); got != 0o700 {
		t.Errorf("log dir mode = %o, want 0700 (owner-only)", got)
	}
}
