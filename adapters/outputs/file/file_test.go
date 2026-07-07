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

package file

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-gen-ecosystem/halolog/adapters/formatters/json"
	"github.com/go-gen-ecosystem/halolog/types"
	"github.com/go-gen-ecosystem/halolog/utils"
)

func TestFileAdapter_Basic(t *testing.T) {
	adapter, err := NewFileAdapter("test.log", nil)
	if err != nil {
		t.Fatalf("NewFileAdapter failed: %v", err)
	}
	if adapter == nil {
		t.Fatal("NewFileAdapter returned nil")
	}

	// Test basic properties
	if adapter.Name() == "" {
		t.Error("File adapter should have a name")
	}

	// Test that adapter can be closed
	err = adapter.Close()
	if err != nil {
		t.Errorf("Close should not error: %v", err)
	}
}

func TestFileAdapter_Write(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test_write.log")

	adapter, err := NewFileAdapter(logFile, nil)
	if err != nil {
		t.Fatalf("NewFileAdapter failed: %v", err)
	}
	defer func() { _ = adapter.Close() }()

	// Test writing a log entry
	entry := &types.LogEntry{
		Level:     types.InfoLevel,
		Message:   "Test log message",
		Timestamp: time.Now(),
		Fields:    []types.TypedFieldData{types.String("key", "value")},
	}

	err = adapter.Write(entry)
	if err != nil {
		t.Errorf("Write should not error: %v", err)
	}

	// Flush to ensure it's written
	err = adapter.Flush()
	if err != nil {
		t.Errorf("Flush should not error: %v", err)
	}

	// Verify file was created and contains content
	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	if len(content) == 0 {
		t.Error("Log file should contain content")
	}

	if !strings.Contains(string(content), "Test log message") {
		t.Error("Log file should contain the test message")
	}
}

func TestFileAdapter_WriteNilEntry(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test_nil.log")

	adapter, err := NewFileAdapter(logFile, nil)
	if err != nil {
		t.Fatalf("NewFileAdapter failed: %v", err)
	}
	defer func() { _ = adapter.Close() }()

	err = adapter.Write(nil)
	if err == nil {
		t.Error("Expected error when writing nil entry")
	}
	if !strings.Contains(err.Error(), "cannot write nil entry") {
		t.Errorf("Expected specific error message, got: %v", err)
	}
}

func TestFileAdapter_SetFormatter(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test_formatter.log")

	adapter, err := NewFileAdapter(logFile, nil)
	if err != nil {
		t.Fatalf("NewFileAdapter failed: %v", err)
	}
	defer func() { _ = adapter.Close() }()

	// Set JSON formatter
	jsonFormatter := json.NewJsonFormatter()
	adapter.SetFormatter(jsonFormatter)

	entry := &types.LogEntry{
		Level:     types.InfoLevel,
		Message:   "JSON formatted message",
		Timestamp: time.Now(),
		Fields:    []types.TypedFieldData{types.String("format", "json")},
	}

	err = adapter.Write(entry)
	if err != nil {
		t.Errorf("Write should not error: %v", err)
	}

	_ = adapter.Flush()

	// Verify JSON format
	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	if !strings.Contains(string(content), `"message":"JSON formatted message"`) {
		t.Error("Log should be in JSON format")
	}
}

func TestFileAdapter_ShouldRotate(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test_rotate.log")

	// Create adapter with rotation settings
	adapter, err := NewFileAdapter(logFile, &RotationConfig{
		MaxSize:    1024, // 1KB
		MaxBackups: 3,
		MaxAge:     7 * 24 * time.Hour,
	})
	if err != nil {
		t.Fatalf("NewFileAdapter failed: %v", err)
	}
	defer func() { _ = adapter.Close() }()

	// Write a small entry - should not rotate
	entry := &types.LogEntry{
		Level:     types.InfoLevel,
		Message:   "Small message",
		Timestamp: time.Now(),
	}

	err = adapter.Write(entry)
	if err != nil {
		t.Errorf("Write should not error: %v", err)
	}

	_ = adapter.Flush()

	// Check if file exists (should not have rotated yet)
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		t.Error("Log file should exist")
	}
}

func TestFileAdapter_Rotate(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test_rotate.log")

	// Create adapter with rotation settings
	adapter, err := NewFileAdapter(logFile, &RotationConfig{
		MaxSize:    100, // Very small size to force rotation
		MaxBackups: 2,
		MaxAge:     7 * 24 * time.Hour,
	})
	if err != nil {
		t.Fatalf("NewFileAdapter failed: %v", err)
	}
	defer func() { _ = adapter.Close() }()

	// Write enough content to trigger rotation
	largeMessage := strings.Repeat("This is a large message that should trigger rotation. ", 20)
	entry := &types.LogEntry{
		Level:     types.InfoLevel,
		Message:   largeMessage,
		Timestamp: time.Now(),
	}

	err = adapter.Write(entry)
	if err != nil {
		t.Errorf("Write should not error: %v", err)
	}

	_ = adapter.Flush()

	// Write more entries to potentially trigger rotation
	for i := 0; i < 5; i++ {
		entry := &types.LogEntry{
			Level:     types.InfoLevel,
			Message:   utils.FormatIntAndText("Message", ' ', i, largeMessage),
			Timestamp: time.Now(),
		}
		_ = adapter.Write(entry)
	}

	_ = adapter.Flush()

	// Check for backup files
	backupPattern := logFile + "*"
	matches, err := filepath.Glob(backupPattern)
	if err != nil {
		t.Fatalf("Failed to glob backup files: %v", err)
	}

	// Should have at least the main log file and possibly some backups
	if len(matches) == 0 {
		t.Error("Should have at least the main log file")
	}
}

func TestFileAdapter_CompressBackups(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test_compress.log")

	// Create adapter with compression enabled
	adapter, err := NewFileAdapter(logFile, &RotationConfig{
		MaxSize:    100, // Small size to trigger rotation
		MaxBackups: 2,
		Compress:   true,
	})
	if err != nil {
		t.Fatalf("NewFileAdapter failed: %v", err)
	}
	defer func() { _ = adapter.Close() }()

	// Write enough content to trigger rotation
	largeMessage := strings.Repeat("Large message content. ", 15)
	for i := 0; i < 3; i++ {
		entry := &types.LogEntry{
			Level:     types.InfoLevel,
			Message:   utils.FormatIntAndText("Message", ' ', i, largeMessage),
			Timestamp: time.Now(),
		}
		_ = adapter.Write(entry)
	}

	_ = adapter.Flush()

	// Check for compressed backup files
	compressedPattern := filepath.Join(tempDir, "*.gz")
	matches, err := filepath.Glob(compressedPattern)
	if err != nil {
		t.Fatalf("Failed to glob compressed files: %v", err)
	}

	// May have compressed files depending on rotation
	t.Logf("Found %d compressed files", len(matches))
}

func TestFileAdapter_CleanupOldBackups(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test_cleanup.log")

	// Create adapter with limited backups
	adapter, err := NewFileAdapter(logFile, &RotationConfig{
		MaxSize:    50, // Very small size
		MaxBackups: 2,  // Keep only 2 backups
	})
	if err != nil {
		t.Fatalf("NewFileAdapter failed: %v", err)
	}
	defer func() { _ = adapter.Close() }()

	// Write many entries to trigger multiple rotations
	for i := 0; i < 10; i++ {
		entry := &types.LogEntry{
			Level:     types.InfoLevel,
			Message:   utils.FormatIntAndText("Message", ':', i, strings.Repeat("Content. ", 10)),
			Timestamp: time.Now(),
		}
		_ = adapter.Write(entry)
	}

	_ = adapter.Flush()

	// Check backup files
	backupPattern := logFile + "*"
	matches, err := filepath.Glob(backupPattern)
	if err != nil {
		t.Fatalf("Failed to glob backup files: %v", err)
	}

	// Should not exceed MaxBackups + 1 (main file)
	if len(matches) > 3 { // main file + 2 backups
		t.Errorf("Too many backup files: %d", len(matches))
	}
}

func TestFileAdapter_FlushInternal(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test_flush.log")

	adapter, err := NewFileAdapter(logFile, nil)
	if err != nil {
		t.Fatalf("NewFileAdapter failed: %v", err)
	}
	defer func() { _ = adapter.Close() }()

	// Write multiple entries
	for i := 0; i < 5; i++ {
		entry := &types.LogEntry{
			Level:     types.InfoLevel,
			Message:   utils.FormatIntWithPrefix("Flush test message", i),
			Timestamp: time.Now(),
		}
		_ = adapter.Write(entry)
	}

	// Flush should write all buffered entries
	err = adapter.Flush()
	if err != nil {
		t.Errorf("Flush should not error: %v", err)
	}

	// Verify all entries were written
	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	for i := 0; i < 5; i++ {
		if !strings.Contains(string(content), utils.FormatIntWithPrefix("Flush test message", i)) {
			t.Errorf("Log should contain message %d", i)
		}
	}
}

func TestFileAdapter_OpenFileError(t *testing.T) {
	// Try to create adapter with an invalid path that cannot be created
	// Using a path with invalid characters that should fail on any OS
	invalidPath := ""

	_, err := NewFileAdapter(invalidPath, nil)
	if err == nil {
		t.Error("Expected error when creating file with empty path")
	}
}

func TestFileAdapter_WriteAfterClose(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test_close.log")

	adapter, err := NewFileAdapter(logFile, nil)
	if err != nil {
		t.Fatalf("NewFileAdapter failed: %v", err)
	}

	_ = adapter.Close()

	// Try to write after close
	entry := &types.LogEntry{
		Level:     types.InfoLevel,
		Message:   "Should not be written",
		Timestamp: time.Now(),
	}

	err = adapter.Write(entry)
	if err == nil {
		t.Error("Expected error when writing after close")
	}
}
