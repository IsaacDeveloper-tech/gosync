//go:build windows

package gosync

import (
	"errors"
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

const (
	backupLockFileExclusive = 0x00000002
	backupLockFileImmediate = 0x00000001
	backupLockBytesLow      = 1
	backupUnlockBytesLow    = 1
)

var (
	backupKernel32     = syscall.NewLazyDLL("kernel32.dll")
	backupLockFileEx   = backupKernel32.NewProc("LockFileEx")
	backupUnlockFileEx = backupKernel32.NewProc("UnlockFileEx")
)

func lockBackupOwnershipFile(file *os.File) error {
	var overlapped syscall.Overlapped
	result, _, callErr := backupLockFileEx.Call(
		file.Fd(),
		backupLockFileExclusive|backupLockFileImmediate,
		0,
		backupLockBytesLow,
		0,
		uintptr(unsafe.Pointer(&overlapped)),
	)
	if result == 0 {
		if errors.Is(callErr, syscall.Errno(33)) || errors.Is(callErr, syscall.Errno(32)) {
			return errBackupDestinationInUse
		}
		return fmt.Errorf("lock backup ownership file: %w", callErr)
	}
	return nil
}

func unlockBackupOwnershipFile(file *os.File) error {
	var overlapped syscall.Overlapped
	result, _, callErr := backupUnlockFileEx.Call(
		file.Fd(),
		0,
		backupUnlockBytesLow,
		0,
		uintptr(unsafe.Pointer(&overlapped)),
	)
	if result == 0 {
		return fmt.Errorf("unlock backup ownership file: %w", callErr)
	}
	return nil
}
