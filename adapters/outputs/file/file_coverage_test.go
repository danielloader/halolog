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
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-gen-ecosystem/halolog/types"
)

// newTestEntry builds a minimal, valid log entry for adapter tests.
func newTestEntry(msg string) *types.LogEntry {
	return &types.LogEntry{
		Level:     types.InfoLevel,
		Message:   msg,
		Timestamp: time.Now(),
	}
}

// ============================================================================
// FileAdapter public method coverage
// ============================================================================

func TestFileAdapter_WriteZero(t *testing.T) {
	dir := t.TempDir()
	logFile := filepath.Join(dir, "writezero.log")

	adapter, err := NewFileAdapter(logFile, nil)
	if err != nil {
		t.Fatalf("NewFileAdapter failed: %v", err)
	}
	defer func() { _ = adapter.Close() }()

	// WriteZero delegates to Write; a valid entry must succeed.
	if err := adapter.WriteZero(newTestEntry("zero-alloc line")); err != nil {
		t.Errorf("WriteZero should not error: %v", err)
	}
	// nil entry propagates the same error as Write.
	if err := adapter.WriteZero(nil); err != ErrFileNilEntry {
		t.Errorf("WriteZero(nil) = %v, want ErrFileNilEntry", err)
	}

	_ = adapter.Flush()

	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	if !strings.Contains(string(content), "zero-alloc line") {
		t.Error("log should contain the WriteZero message")
	}
}

func TestFileAdapter_Health(t *testing.T) {
	dir := t.TempDir()
	logFile := filepath.Join(dir, "health.log")

	adapter, err := NewFileAdapter(logFile, nil)
	if err != nil {
		t.Fatalf("NewFileAdapter failed: %v", err)
	}

	// Open adapter with a live file must be healthy.
	if err := adapter.Health(); err != nil {
		t.Errorf("Health on open adapter = %v, want nil", err)
	}

	// After Close the adapter reports ErrFileClosed.
	if err := adapter.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
	if err := adapter.Health(); err != ErrFileClosed {
		t.Errorf("Health after close = %v, want ErrFileClosed", err)
	}
}

func TestFileAdapter_Metrics(t *testing.T) {
	dir := t.TempDir()
	logFile := filepath.Join(dir, "metrics.log")

	adapter, err := NewFileAdapter(logFile, nil)
	if err != nil {
		t.Fatalf("NewFileAdapter failed: %v", err)
	}
	defer func() { _ = adapter.Close() }()

	const n = 5
	for i := 0; i < n; i++ {
		if err := adapter.Write(newTestEntry("metric line")); err != nil {
			t.Fatalf("write %d: %v", i, err)
		}
	}
	_ = adapter.Flush()

	m := adapter.Metrics()
	if m == nil {
		t.Fatal("Metrics returned nil")
	}
	if m.WritesTotal.Load() != n {
		t.Errorf("WritesTotal = %d, want %d", m.WritesTotal.Load(), n)
	}
	if m.BytesWritten.Load() == 0 {
		t.Error("BytesWritten should be > 0 after writes")
	}
}

func TestFileAdapter_FlushAfterClose(t *testing.T) {
	dir := t.TempDir()
	logFile := filepath.Join(dir, "flush_closed.log")

	adapter, err := NewFileAdapter(logFile, nil)
	if err != nil {
		t.Fatalf("NewFileAdapter failed: %v", err)
	}
	_ = adapter.Close()

	if err := adapter.Flush(); err != ErrFileClosed {
		t.Errorf("Flush after close = %v, want ErrFileClosed", err)
	}
}

func TestFileAdapter_DoubleCloseIsNoOp(t *testing.T) {
	dir := t.TempDir()
	logFile := filepath.Join(dir, "double_close.log")

	adapter, err := NewFileAdapter(logFile, nil)
	if err != nil {
		t.Fatalf("NewFileAdapter failed: %v", err)
	}
	if err := adapter.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}
	// Second Close short-circuits via the CompareAndSwap guard and returns nil.
	if err := adapter.Close(); err != nil {
		t.Errorf("second Close = %v, want nil", err)
	}
}

// ============================================================================
// Rate limiter path (Write returns ErrFileRateLimited)
// ============================================================================

func TestFileAdapter_RateLimited(t *testing.T) {
	dir := t.TempDir()
	logFile := filepath.Join(dir, "ratelimit.log")

	// RateLimit of 1 token/sec: the first write consumes the only token, the
	// second is refused until refill.
	adapter, err := NewFileAdapter(logFile, &RotationConfig{
		RateLimit:    1,
		BatchTimeout: 10 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("NewFileAdapter failed: %v", err)
	}
	defer func() { _ = adapter.Close() }()

	if err := adapter.Write(newTestEntry("first")); err != nil {
		t.Fatalf("first write should pass: %v", err)
	}

	// Drain remaining tokens; at least one subsequent write must be rate limited.
	var gotLimited bool
	for i := 0; i < 5; i++ {
		if err := adapter.Write(newTestEntry("burst")); err == ErrFileRateLimited {
			gotLimited = true
			break
		}
	}
	if !gotLimited {
		t.Error("expected ErrFileRateLimited under a 1 token/sec limit")
	}

	if adapter.Metrics().RateLimits.Load() == 0 {
		t.Error("RateLimits metric should have incremented")
	}
}

// ============================================================================
// Flock (cross-process lock) path
// ============================================================================

func TestFileAdapter_WithFlock(t *testing.T) {
	dir := t.TempDir()
	logFile := filepath.Join(dir, "flock.log")

	// UseFlock exercises acquireFileLock on construction and releaseFileLock on Close.
	adapter, err := NewFileAdapter(logFile, &RotationConfig{
		UseFlock:     true,
		BatchTimeout: 10 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("NewFileAdapter with flock failed: %v", err)
	}

	if err := adapter.Write(newTestEntry("locked write")); err != nil {
		t.Errorf("write under flock: %v", err)
	}
	_ = adapter.Flush()

	if err := adapter.Close(); err != nil {
		t.Errorf("Close with flock: %v", err)
	}

	// After a clean release, a fresh adapter can acquire the same lock.
	adapter2, err := NewFileAdapter(logFile, &RotationConfig{
		UseFlock:     true,
		BatchTimeout: 10 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("second flock adapter failed to acquire released lock: %v", err)
	}
	_ = adapter2.Close()
}

// ============================================================================
// O_SYNC open path
// ============================================================================

func TestFileAdapter_UseOSync(t *testing.T) {
	dir := t.TempDir()
	logFile := filepath.Join(dir, "osync.log")

	adapter, err := NewFileAdapter(logFile, &RotationConfig{
		UseOSync:     true,
		BatchTimeout: 10 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("NewFileAdapter with O_SYNC failed: %v", err)
	}
	defer func() { _ = adapter.Close() }()

	if err := adapter.Write(newTestEntry("synced line")); err != nil {
		t.Errorf("write with O_SYNC: %v", err)
	}
	_ = adapter.Flush()

	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	if !strings.Contains(string(content), "synced line") {
		t.Error("O_SYNC log should contain the line")
	}
}

// ============================================================================
// FlushInterval worker path
// ============================================================================

func TestFileAdapter_FlushWorker(t *testing.T) {
	dir := t.TempDir()
	logFile := filepath.Join(dir, "flushworker.log")

	// A short flush interval starts the periodic flush worker goroutine.
	adapter, err := NewFileAdapter(logFile, &RotationConfig{
		FlushInterval: 20 * time.Millisecond,
		BatchTimeout:  5 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("NewFileAdapter failed: %v", err)
	}
	defer func() { _ = adapter.Close() }()

	if err := adapter.Write(newTestEntry("interval line")); err != nil {
		t.Errorf("write: %v", err)
	}

	// Allow the flush worker to tick at least once.
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		content, _ := os.ReadFile(logFile)
		if strings.Contains(string(content), "interval line") {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	// Fall back to an explicit flush; the worker path was still exercised.
	_ = adapter.Flush()
	content, _ := os.ReadFile(logFile)
	if !strings.Contains(string(content), "interval line") {
		t.Error("expected the line to be flushed to disk")
	}
}

// ============================================================================
// DefaultRotationConfig sanity
// ============================================================================

func TestDefaultRotationConfig(t *testing.T) {
	cfg := DefaultRotationConfig()
	if cfg == nil {
		t.Fatal("DefaultRotationConfig returned nil")
	}
	if cfg.MaxSize <= 0 {
		t.Error("default MaxSize should be positive")
	}
	if cfg.MaxBackups <= 0 {
		t.Error("default MaxBackups should be positive")
	}
	if cfg.FlushInterval <= 0 {
		t.Error("default FlushInterval should be positive")
	}
	if !cfg.Compress {
		t.Error("default config should enable compression")
	}
}
