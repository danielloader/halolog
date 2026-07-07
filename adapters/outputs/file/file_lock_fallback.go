//go:build !unix && !darwin && !linux && !windows
// +build !unix,!darwin,!linux,!windows

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

package file_lock_fallback

import (
	"fmt"
	"os"
	"time"

	"github.com/go-gen-ecosystem/halolog/utils"
)

// fallbackFileLock uses PID file checking (less robust but portable)
type fallbackFileLock struct{}

func newPlatformLock() fileLockImpl {
	return &fallbackFileLock{}
}

func (f *fallbackFileLock) Lock(file *os.File, timeout time.Duration) error {
	// For fallback, we implement a simple blocking lock
	// The timeout is handled at a higher level in the FileLock struct
	return f.TryLock(file)
}

func (f *fallbackFileLock) TryLock(file *os.File) error {
	// Check if PID file exists and is valid
	pidFile := file.Name() + ".pid"

	data, err := os.ReadFile(pidFile)
	if err == nil {
		// PID file exists - check if process is alive
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
		// If we found a valid PID, check if process exists
		if pid > 0 && processExists(pid) {
			return ErrLockHeld
		}
	}

	// Write our PID
	pid := utils.FormatString("%d\n", os.Getpid())
	if err := os.WriteFile(pidFile, []byte(pid), 0o600); err != nil {
		return fmt.Errorf("failed to write PID file: %w", err)
	}

	// Small delay to detect race conditions
	time.Sleep(10 * time.Millisecond)

	// Verify we still own the lock
	data, err = os.ReadFile(pidFile)
	if err != nil || string(data) != pid {
		return ErrLockHeld
	}

	return nil
}

func (f *fallbackFileLock) Unlock(file *os.File) error {
	pidFile := file.Name() + ".pid"
	os.Remove(pidFile)
	return nil
}
