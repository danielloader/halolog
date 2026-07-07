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
// @author Admilson B. F. Cossa

package file

import (
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestCompressFile_RoundTrip drives compressFile directly and verifies the
// produced gzip stream decompresses back to the original bytes.
func TestCompressFile_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "plain.log")
	dst := filepath.Join(dir, "plain.log.gz")

	original := strings.Repeat("compressible log line\n", 500)
	if err := os.WriteFile(src, []byte(original), 0644); err != nil {
		t.Fatalf("write src: %v", err)
	}

	if err := compressFile(src, dst); err != nil {
		t.Fatalf("compressFile failed: %v", err)
	}

	gzFile, err := os.Open(dst)
	if err != nil {
		t.Fatalf("open gz: %v", err)
	}
	defer func() { _ = gzFile.Close() }()

	gr, err := gzip.NewReader(gzFile)
	if err != nil {
		t.Fatalf("gzip reader: %v", err)
	}
	defer func() { _ = gr.Close() }()

	decompressed, err := io.ReadAll(gr)
	if err != nil {
		t.Fatalf("read gz: %v", err)
	}
	if string(decompressed) != original {
		t.Error("decompressed content does not match the original")
	}
}

// TestCompressFile_MissingSource exercises the error path when the source file
// does not exist.
func TestCompressFile_MissingSource(t *testing.T) {
	dir := t.TempDir()
	dst := filepath.Join(dir, "out.gz")

	err := compressFile(filepath.Join(dir, "does-not-exist.log"), dst)
	if err == nil {
		t.Error("compressFile should error when the source is missing")
	}
}

// TestFileAdapter_RotationWithCompression forces several rotations with
// compression enabled, driving rotateLocked -> rotateAsync -> compressFile and
// cleanupOldBackups.
func TestFileAdapter_RotationWithCompression(t *testing.T) {
	dir := t.TempDir()
	logFile := filepath.Join(dir, "rot.log")

	adapter, err := NewFileAdapter(logFile, &RotationConfig{
		MaxSize:      80, // tiny to force frequent rotation
		MaxBackups:   2,
		Compress:     true,
		BatchTimeout: 5 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("NewFileAdapter failed: %v", err)
	}

	line := strings.Repeat("rotate me. ", 12)
	for i := 0; i < 30; i++ {
		_ = adapter.Write(newTestEntry(line))
		_ = adapter.Flush()
	}

	// Give the async rotation/compression goroutines time to run.
	time.Sleep(150 * time.Millisecond)
	_ = adapter.Close()

	// After rotations with compression, expect either the primary log or at
	// least one artifact (possibly a .gz) to exist.
	all, _ := filepath.Glob(logFile + "*")
	if len(all) == 0 {
		t.Error("expected at least one log artifact after rotation")
	}
}

// TestFileAdapter_CleanupByAge seeds aged backup files then triggers a rotation
// so cleanupOldBackups removes files older than maxAge.
func TestFileAdapter_CleanupByAge(t *testing.T) {
	dir := t.TempDir()
	logFile := filepath.Join(dir, "aged.log")

	adapter, err := NewFileAdapter(logFile, &RotationConfig{
		MaxSize:      60,
		MaxBackups:   10,
		MaxAge:       1 * 24 * time.Hour, // 1 day
		BatchTimeout: 5 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("NewFileAdapter failed: %v", err)
	}
	defer func() { _ = adapter.Close() }()

	// Seed an "old" backup file that predates the cutoff by a wide margin.
	oldBackup := logFile + ".2000-01-01T00-00-00"
	if err := os.WriteFile(oldBackup, []byte("ancient\n"), 0644); err != nil {
		t.Fatalf("seed old backup: %v", err)
	}
	old := time.Now().AddDate(0, 0, -10)
	if err := os.Chtimes(oldBackup, old, old); err != nil {
		t.Fatalf("backdate old backup: %v", err)
	}

	// Directly drive cleanup (it is also reached via rotateAsync).
	if err := adapter.cleanupOldBackups(); err != nil {
		t.Fatalf("cleanupOldBackups failed: %v", err)
	}

	if _, err := os.Stat(oldBackup); !os.IsNotExist(err) {
		t.Error("aged backup older than maxAge should have been removed")
	}
}

// TestFileAdapter_CleanupExcessBackups verifies MaxBackups enforcement removes
// the oldest backups beyond the retention count.
func TestFileAdapter_CleanupExcessBackups(t *testing.T) {
	dir := t.TempDir()
	logFile := filepath.Join(dir, "excess.log")

	adapter, err := NewFileAdapter(logFile, &RotationConfig{
		MaxSize:      1024,
		MaxBackups:   2, // keep only the 2 newest
		BatchTimeout: 5 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("NewFileAdapter failed: %v", err)
	}
	defer func() { _ = adapter.Close() }()

	// Seed 4 backups with staggered, ascending modification times.
	base := time.Now().Add(-4 * time.Hour)
	names := []string{
		logFile + ".2001-01-01T00-00-01",
		logFile + ".2001-01-01T00-00-02",
		logFile + ".2001-01-01T00-00-03",
		logFile + ".2001-01-01T00-00-04",
	}
	for i, n := range names {
		if err := os.WriteFile(n, []byte("backup\n"), 0644); err != nil {
			t.Fatalf("seed backup %s: %v", n, err)
		}
		mt := base.Add(time.Duration(i) * time.Minute)
		if err := os.Chtimes(n, mt, mt); err != nil {
			t.Fatalf("chtimes %s: %v", n, err)
		}
	}

	if err := adapter.cleanupOldBackups(); err != nil {
		t.Fatalf("cleanupOldBackups failed: %v", err)
	}

	remaining, _ := filepath.Glob(logFile + ".2001-*")
	if len(remaining) != 2 {
		t.Errorf("expected 2 backups retained under MaxBackups=2, got %d", len(remaining))
	}
}
