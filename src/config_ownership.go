package gosync

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

var ErrConfigurationInUse = errors.New("configuration is already being changed")

type ConfigurationOwnership struct {
	lockPath string
	lockFile *os.File
}

func acquireConfigurationOwnership(configurationPath string) (ConfigurationOwnership, error) {
	configurationDirectory := filepath.Dir(configurationPath)
	if err := os.MkdirAll(configurationDirectory, 0o700); err != nil {
		return ConfigurationOwnership{}, fmt.Errorf("create configuration ownership directory: %w", err)
	}
	lockPath := configurationPath + ".lock"
	lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		if os.IsExist(err) {
			return ConfigurationOwnership{}, ErrConfigurationInUse
		}
		return ConfigurationOwnership{}, fmt.Errorf("acquire configuration ownership: %w", err)
	}
	return ConfigurationOwnership{lockPath: lockPath, lockFile: lockFile}, nil
}

func (ownership *ConfigurationOwnership) Release() error {
	if ownership == nil || ownership.lockFile == nil {
		return nil
	}
	closeErr := ownership.lockFile.Close()
	removeErr := os.Remove(ownership.lockPath)
	ownership.lockFile = nil
	if closeErr != nil {
		return fmt.Errorf("close configuration ownership: %w", closeErr)
	}
	if removeErr != nil && !os.IsNotExist(removeErr) {
		return fmt.Errorf("release configuration ownership: %w", removeErr)
	}
	return nil
}
