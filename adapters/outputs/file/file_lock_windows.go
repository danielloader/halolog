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
// Author: Admilson B. F. Cossa

package file

import (
	"os"
	"syscall"
	"time"
	"unsafe"
)

const (
	LOCKFILE_EXCLUSIVE_LOCK   = 0x00000002
	LOCKFILE_FAIL_IMMEDIATELY = 0x00000001
)

var (
	kernel32         = syscall.NewLazyDLL("kernel32.dll")
	procLockFileEx   = kernel32.NewProc("LockFileEx")
	procUnlockFileEx = kernel32.NewProc("UnlockFileEx")
)

type windowsFileLock struct{}

func newPlatformLock() fileLockImpl {
	return &windowsFileLock{}
}

func (w *windowsFileLock) Lock(file *os.File, timeout time.Duration) error {
	// For Windows, we use a simple blocking lock
	// The timeout is handled at a higher level in the FileLock struct
	return lockFileEx(file.Fd(), LOCKFILE_EXCLUSIVE_LOCK, 0)
}

func (w *windowsFileLock) TryLock(file *os.File) error {
	err := lockFileEx(file.Fd(), LOCKFILE_EXCLUSIVE_LOCK|LOCKFILE_FAIL_IMMEDIATELY, 0)
	if err != nil {
		// On Windows, LockFileEx with LOCKFILE_FAIL_IMMEDIATELY returns ERROR_IO_PENDING
		// or ERROR_LOCK_VIOLATION when the lock cannot be acquired immediately
		// We treat any error as "lock held" for simplicity
		return ErrLockHeld
	}
	return nil
}

func (w *windowsFileLock) Unlock(file *os.File) error {
	return unlockFileEx(file.Fd())
}

// lockFileEx wraps Windows LockFileEx API
func lockFileEx(fd uintptr, flags uint32, reserved uint32) error {
	var overlapped syscall.Overlapped

	// Lock entire file (maxuint64 bytes)
	r1, _, err := procLockFileEx.Call(
		fd,
		uintptr(flags),
		uintptr(reserved),
		0xFFFFFFFF, // nNumberOfBytesToLockLow
		0xFFFFFFFF, // nNumberOfBytesToLockHigh
		uintptr(unsafe.Pointer(&overlapped)),
	)

	if r1 == 0 {
		return err
	}
	return nil
}

// unlockFileEx wraps Windows UnlockFileEx API
func unlockFileEx(fd uintptr) error {
	var overlapped syscall.Overlapped

	r1, _, err := procUnlockFileEx.Call(
		fd,
		0, // reserved
		0xFFFFFFFF,
		0xFFFFFFFF,
		uintptr(unsafe.Pointer(&overlapped)),
	)

	if r1 == 0 {
		return err
	}
	return nil
}
