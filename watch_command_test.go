package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestRunWatchCommandParsesRootsAndRunsSynchronization(t *testing.T) {
	temporaryDirectory := t.TempDir()
	firstRoot := filepath.Join(temporaryDirectory, "first")
	secondRoot := filepath.Join(temporaryDirectory, "second")
	if err := os.MkdirAll(firstRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll(firstRoot) error = %v", err)
	}
	if err := os.MkdirAll(secondRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll(secondRoot) error = %v", err)
	}
	stop := make(chan struct{})
	var receivedRoots RootPaths
	callCount := 0

	err := runWatchCommand([]string{"watch", firstRoot, secondRoot}, WatchCommandOptions{
		Output: bytes.NewBuffer(nil),
		Stop:   stop,
		Synchronize: func(roots RootPaths) error {
			callCount++
			receivedRoots = roots
			close(stop)
			return nil
		},
	})
	if err != nil {
		t.Fatalf("runWatchCommand() error = %v", err)
	}
	if callCount != 1 {
		t.Fatalf("synchronization calls = %d, want 1", callCount)
	}
	if receivedRoots.First != firstRoot || receivedRoots.Second != secondRoot {
		t.Fatalf("received roots = %+v, want first %q and second %q", receivedRoots, firstRoot, secondRoot)
	}
}

func TestSynchronizeDirectoriesRunsTheCompleteFlowAndCommitsState(t *testing.T) {
	temporaryDirectory := t.TempDir()
	firstRoot := filepath.Join(temporaryDirectory, "first")
	secondRoot := filepath.Join(temporaryDirectory, "second")
	if err := os.MkdirAll(firstRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll(firstRoot) error = %v", err)
	}
	if err := os.MkdirAll(secondRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll(secondRoot) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(firstRoot, "notes.txt"), []byte("same"), 0o644); err != nil {
		t.Fatalf("WriteFile(firstRoot) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(secondRoot, "notes.txt"), []byte("same"), 0o600); err != nil {
		t.Fatalf("WriteFile(secondRoot) error = %v", err)
	}

	roots := RootPaths{First: firstRoot, Second: secondRoot}
	store := newConfirmedStateStoreAt(filepath.Join(temporaryDirectory, "state"))
	if err := synchronizeDirectories(roots, store, bytes.NewBufferString(""), bytes.NewBuffer(nil)); err != nil {
		t.Fatalf("synchronizeDirectories() error = %v", err)
	}
	if _, found, err := store.load(roots); err != nil || !found {
		t.Fatalf("committed state: found = %t, error = %v, want state", found, err)
	}
}
