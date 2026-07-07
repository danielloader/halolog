//go:build windows
// +build windows

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
	"testing"
	"time"
)

// TestWindowsFileLock_BlockingLockUnlock drives the Windows platform lock's
// blocking Lock method (LockFileEx without FAIL_IMMEDIATELY) and Unlock.
func TestWindowsFileLock_BlockingLockUnlock(t *testing.T) {
	dir := t.TempDir()
	lockPath := filepath.Join(dir, "win.lock")

	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		t.Fatalf("open lock file: %v", err)
	}
	defer func() { _ = f.Close() }()

	impl := newPlatformLock()

	// Blocking exclusive lock on an uncontended file must succeed.
	if err := impl.Lock(f, 2*time.Second); err != nil {
		t.Fatalf("blocking Lock failed: %v", err)
	}

	// Unlock the range we just locked.
	if err := impl.Unlock(f); err != nil {
		t.Errorf("Unlock failed: %v", err)
	}
}

// TestWindowsFileLock_TryLockContended verifies TryLock reports ErrLockHeld
// when the byte range is already locked by another handle.
func TestWindowsFileLock_TryLockContended(t *testing.T) {
	dir := t.TempDir()
	lockPath := filepath.Join(dir, "win_try.lock")

	holder, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		t.Fatalf("open holder: %v", err)
	}
	defer func() { _ = holder.Close() }()

	impl := newPlatformLock()
	if err := impl.TryLock(holder); err != nil {
		t.Fatalf("first TryLock should succeed: %v", err)
	}
	defer func() { _ = impl.Unlock(holder) }()

	// A second independent handle to the same file must fail to acquire.
	contender, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		t.Fatalf("open contender: %v", err)
	}
	defer func() { _ = contender.Close() }()

	if err := impl.TryLock(contender); err != ErrLockHeld {
		t.Errorf("contended TryLock = %v, want ErrLockHeld", err)
	}
}
