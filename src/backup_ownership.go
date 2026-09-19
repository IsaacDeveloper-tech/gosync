package gosync

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

var errBackupDestinationInUse = errors.New("backup destination is already in use")

type BackupDestinationOwnership struct {
	lockPath string
	file     *os.File
}

func acquireBackupDestinationOwnership(lockPath string) (BackupDestinationOwnership, error) {
	if lockPath == "" {
		return BackupDestinationOwnership{}, errors.New("backup destination lock path is required")
	}
	if err := os.MkdirAll(filepath.Dir(lockPath), 0o700); err != nil {
		return BackupDestinationOwnership{}, fmt.Errorf("create backup ownership directory: %w", err)
	}
	file, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return BackupDestinationOwnership{}, fmt.Errorf("open backup ownership lock: %w", err)
	}
	if err := lockBackupOwnershipFile(file); err != nil {
		_ = file.Close()
		if errors.Is(err, errBackupDestinationInUse) {
			return BackupDestinationOwnership{}, err
		}
		return BackupDestinationOwnership{}, fmt.Errorf("acquire backup destination ownership: %w", err)
	}
	return BackupDestinationOwnership{lockPath: lockPath, file: file}, nil
}

func (ownership *BackupDestinationOwnership) Release() error {
	if ownership == nil || ownership.file == nil {
		return nil
	}
	unlockErr := unlockBackupOwnershipFile(ownership.file)
	closeErr := ownership.file.Close()
	removeErr := os.Remove(ownership.lockPath)
	ownership.file = nil
	if removeErr != nil && !os.IsNotExist(removeErr) {
		removeErr = fmt.Errorf("remove backup ownership lock: %w", removeErr)
	}
	return errors.Join(unlockErr, closeErr, removeErr)
}
