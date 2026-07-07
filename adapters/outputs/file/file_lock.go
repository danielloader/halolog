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
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/go-gen-ecosystem/halolog/utils"
)

var (
	// ErrLockTimeout is returned when the lock cannot be acquired within the timeout.
	ErrLockTimeout = errors.New("failed to acquire lock: timeout")
	// ErrLockHeld is returned when the lock is currently held by another process.
	ErrLockHeld = errors.New("lock held by another process")
)

// fileLockImpl is the platform-specific implementation
type fileLockImpl interface {
	Lock(file *os.File, timeout time.Duration) error
	Unlock(file *os.File) error
	TryLock(file *os.File) error
}

// FileLock provides cross-platform file locking.
//
//nolint:revive // exported name intentionally kept for a stable public API; renaming to Lock would break importers
type FileLock struct {
	path     string
	lockFile *os.File
	impl     fileLockImpl
	acquired bool
	pidFile  string // For stale lock detection
}

// NewFileLock creates a new file lock
func NewFileLock(path string) *FileLock {
	return &FileLock{
		path:    path,
		pidFile: path + ".pid",
		impl:    newPlatformLock(),
	}
}

// Lock acquires the lock with timeout
func (fl *FileLock) Lock(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for {
		err := fl.TryLock()
		if err == nil {
			return nil
		}

		if !errors.Is(err, ErrLockHeld) {
			return err
		}

		// Check for stale lock
		if fl.isLockStale() {
			if breakErr := fl.breakStaleLock(); breakErr == nil {
				continue
			}
		}

		if time.Now().After(deadline) {
			return ErrLockTimeout
		}

		time.Sleep(100 * time.Millisecond)
	}
}

// TryLock attempts to acquire the lock without blocking
func (fl *FileLock) TryLock() error {
	// Open or create lock file
	lockFile, err := os.OpenFile(fl.path+".lock", os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("failed to open lock file: %w", err)
	}

	// Try platform-specific lock
	if err := fl.impl.TryLock(lockFile); err != nil {
		_ = lockFile.Close()
		return err
	}

	// Write PID for stale lock detection
	if err := fl.writePID(lockFile); err != nil {
		_ = fl.impl.Unlock(lockFile)
		_ = lockFile.Close()
		return fmt.Errorf("failed to write PID: %w", err)
	}

	fl.lockFile = lockFile
	fl.acquired = true
	return nil
}

// processExists checks if a process with the given PID exists
func processExists(pid int) bool {
	// Try to find the process
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// Try to send signal 0 (no-op signal)
	// This may not work on all platforms, but it's the best cross-platform approach
	err = process.Signal(os.Signal(nil))
	return err == nil
}

// Unlock releases the lock
func (fl *FileLock) Unlock() error {
	if !fl.acquired || fl.lockFile == nil {
		return nil
	}

	// Release platform-specific lock
	if err := fl.impl.Unlock(fl.lockFile); err != nil {
		return err
	}

	// Close and remove lock file (removal is best-effort cleanup)
	_ = fl.lockFile.Close()
	_ = os.Remove(fl.path + ".lock")
	_ = os.Remove(fl.pidFile)

	fl.lockFile = nil
	fl.acquired = false
	return nil
}

// writePID writes current process ID to lock file
func (fl *FileLock) writePID(file *os.File) error {
	pid := utils.FormatIntWithPrefix("", os.Getpid()) + "\n"
	if err := file.Truncate(0); err != nil {
		return err
	}
	if _, err := file.Seek(0, 0); err != nil {
		return err
	}
	if _, err := file.WriteString(pid); err != nil {
		return err
	}
	return file.Sync()
}

// isLockStale checks if the lock is held by a dead process
func (fl *FileLock) isLockStale() bool {
	data, err := os.ReadFile(fl.path + ".lock")
	if err != nil {
		return false
	}

	var pid int
	dataStr := string(data)
	// Simple integer parsing without fmt.Sscanf
	for i := 0; i < len(dataStr); i++ {
		if dataStr[i] >= '0' && dataStr[i] <= '9' {
			pid = pid*10 + int(dataStr[i]-'0')
		} else {
			break
		}
	}
	// If no valid digits found, consider it stale
	if pid == 0 && len(dataStr) > 0 && (dataStr[0] < '0' || dataStr[0] > '9') {
		return true // Invalid PID file = stale
	}

	// Check if process exists
	return !processExists(pid)
}

// breakStaleLock forcibly removes a stale lock
func (fl *FileLock) breakStaleLock() error {
	if err := os.Remove(fl.path + ".lock"); err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := os.Remove(fl.pidFile); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
