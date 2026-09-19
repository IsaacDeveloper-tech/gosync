package gosync

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type ConfigurationStore struct {
	path string
}

func newConfigurationStore(path string) ConfigurationStore {
	return ConfigurationStore{path: path}
}

func (store ConfigurationStore) Path() string {
	return store.path
}

func (store ConfigurationStore) Load() (ConfigurationLoadResult, error) {
	return loadConfiguration(store.path)
}

func (store ConfigurationStore) WriteCandidate(configuration ConfigurationSnapshot) (string, error) {
	encodedConfiguration, err := encodeConfiguration(configuration)
	if err != nil {
		return "", fmt.Errorf("encode configuration candidate: %w", err)
	}
	configurationDirectory := filepath.Dir(store.path)
	if err := os.MkdirAll(configurationDirectory, 0o700); err != nil {
		return "", fmt.Errorf("create configuration directory: %w", err)
	}
	candidateFile, err := os.CreateTemp(configurationDirectory, ".config-*")
	if err != nil {
		return "", fmt.Errorf("create configuration candidate: %w", err)
	}
	candidatePath := candidateFile.Name()
	removeCandidate := true
	defer func() {
		if removeCandidate {
			_ = os.Remove(candidatePath)
		}
	}()

	if err := candidateFile.Chmod(0o600); err != nil {
		_ = candidateFile.Close()
		return "", fmt.Errorf("set configuration candidate permissions: %w", err)
	}
	if _, err := io.WriteString(candidateFile, encodedConfiguration); err != nil {
		_ = candidateFile.Close()
		return "", fmt.Errorf("write configuration candidate: %w", err)
	}
	if err := candidateFile.Close(); err != nil {
		return "", fmt.Errorf("close configuration candidate: %w", err)
	}
	removeCandidate = false
	return candidatePath, nil
}

func (store ConfigurationStore) ReplaceCandidate(candidatePath string) error {
	if filepath.Clean(filepath.Dir(candidatePath)) != filepath.Clean(filepath.Dir(store.path)) {
		return fmt.Errorf("configuration candidate must be beside active configuration")
	}
	if err := os.Rename(candidatePath, store.path); err != nil {
		if _, statErr := os.Stat(store.path); statErr != nil {
			return fmt.Errorf("replace configuration file: %w", err)
		}
		if removeErr := os.Remove(store.path); removeErr != nil {
			return fmt.Errorf("replace existing configuration file: %w", removeErr)
		}
		if renameErr := os.Rename(candidatePath, store.path); renameErr != nil {
			return fmt.Errorf("replace configuration file after removing previous document: %w", renameErr)
		}
	}
	return nil
}

func (store ConfigurationStore) Save(configuration ConfigurationSnapshot) error {
	candidatePath, err := store.WriteCandidate(configuration)
	if err != nil {
		return err
	}
	defer os.Remove(candidatePath)
	if err := store.ReplaceCandidate(candidatePath); err != nil {
		return err
	}
	return nil
}
