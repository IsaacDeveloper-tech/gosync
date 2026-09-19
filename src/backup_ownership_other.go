//go:build !windows

package gosync

import (
	"errors"
	"os"
	"syscall"
)

func lockBackupOwnershipFile(file *os.File) error {
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		if errors.Is(err, syscall.EWOULDBLOCK) || errors.Is(err, syscall.EAGAIN) {
			return errBackupDestinationInUse
		}
		return err
	}
	return nil
}

func unlockBackupOwnershipFile(file *os.File) error {
	return syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
}
