//go:build unix || darwin || linux
// +build unix darwin linux

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
	"fmt"
	"os"
	"syscall"
	"time"
)

type unixFileLock struct{}

func newPlatformLock() fileLockImpl {
	return &unixFileLock{}
}

func (u *unixFileLock) Lock(file *os.File, timeout time.Duration) error {
	// For Unix, we use a blocking flock
	// The timeout is handled at a higher level in the FileLock struct
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX); err != nil {
		return fmt.Errorf("flock failed: %w", err)
	}
	return nil
}

func (u *unixFileLock) TryLock(file *os.File) error {
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		if err == syscall.EWOULDBLOCK {
			return ErrLockHeld
		}
		return fmt.Errorf("flock failed: %w", err)
	}
	return nil
}

func (u *unixFileLock) Unlock(file *os.File) error {
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_UN); err != nil {
		return fmt.Errorf("unlock failed: %w", err)
	}
	return nil
}
