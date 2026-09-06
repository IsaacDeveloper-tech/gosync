package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

var errSynchronizationIncomplete = errors.New("synchronization is not complete")

type ConfirmedSynchronizationState struct {
	Version     int                    `json:"version"`
	Roots       RootPaths              `json:"roots"`
	Entries     []SynchronizationEntry `json:"entries"`
	CompletedAt time.Time              `json:"completedAt"`
}

type confirmedStateStore struct {
	directory string
}

func newConfirmedStateStore() (confirmedStateStore, error) {
	userConfigDirectory, err := os.UserConfigDir()
	if err != nil {
		return confirmedStateStore{}, fmt.Errorf("find user configuration directory: %w", err)
	}

	return newConfirmedStateStoreAt(filepath.Join(userConfigDirectory, "gosync", "state")), nil
}

func newConfirmedStateStoreAt(directory string) confirmedStateStore {
	return confirmedStateStore{directory: directory}
}

func (store confirmedStateStore) save(state ConfirmedSynchronizationState, result SynchronizationResult) error {
	if !result.Completed || result.Failure != nil {
		return errSynchronizationIncomplete
	}

	if err := os.MkdirAll(store.directory, 0o700); err != nil {
		return fmt.Errorf("create state directory: %w", err)
	}

	stateData, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("encode confirmed state: %w", err)
	}

	temporaryFile, err := os.CreateTemp(store.directory, ".confirmed-state-*")
	if err != nil {
		return fmt.Errorf("create temporary state file: %w", err)
	}
	temporaryPath := temporaryFile.Name()
	defer os.Remove(temporaryPath)

	if err := temporaryFile.Chmod(0o600); err != nil {
		temporaryFile.Close()
		return fmt.Errorf("set temporary state file permissions: %w", err)
	}
	if _, err := temporaryFile.Write(stateData); err != nil {
		temporaryFile.Close()
		return fmt.Errorf("write confirmed state: %w", err)
	}
	if err := temporaryFile.Close(); err != nil {
		return fmt.Errorf("close temporary state file: %w", err)
	}

	if err := os.Rename(temporaryPath, store.stateFilePath(state.Roots)); err != nil {
		return fmt.Errorf("replace confirmed state: %w", err)
	}

	return nil
}

func (store confirmedStateStore) load(roots RootPaths) (ConfirmedSynchronizationState, bool, error) {
	stateData, err := os.ReadFile(store.stateFilePath(roots))
	if err != nil {
		if os.IsNotExist(err) {
			return ConfirmedSynchronizationState{}, false, nil
		}
		return ConfirmedSynchronizationState{}, false, fmt.Errorf("read confirmed state: %w", err)
	}

	var state ConfirmedSynchronizationState
	if err := json.Unmarshal(stateData, &state); err != nil {
		return ConfirmedSynchronizationState{}, false, fmt.Errorf("decode confirmed state: %w", err)
	}
	if state.Roots != roots {
		return ConfirmedSynchronizationState{}, false, fmt.Errorf("confirmed state roots do not match requested roots")
	}

	return state, true, nil
}

func (store confirmedStateStore) stateFilePath(roots RootPaths) string {
	rootPair := roots.First + "\x00" + roots.Second
	digest := sha256.Sum256([]byte(rootPair))
	return filepath.Join(store.directory, hex.EncodeToString(digest[:])+".json")
}
