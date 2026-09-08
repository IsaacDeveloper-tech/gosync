//go:build windows

package gosync

import (
	"errors"
	"syscall"
)

const (
	errorLockViolation    syscall.Errno = 33
	errorSharingViolation syscall.Errno = 32
)

func isLockedFileError(err error) bool {
	var errno syscall.Errno
	if !errors.As(err, &errno) {
		return false
	}

	return errno == errorLockViolation || errno == errorSharingViolation
}
