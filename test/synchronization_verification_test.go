package gosync_test

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestVerifyAndCommitSynchronizationStoresEqualContents(t *testing.T) {
	temporaryDirectory := t.TempDir()
	firstRoot := filepath.Join(temporaryDirectory, "first")
	secondRoot := filepath.Join(temporaryDirectory, "second")
	if err := os.MkdirAll(firstRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll(firstRoot) error = %v", err)
	}
	if err := os.MkdirAll(secondRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll(secondRoot) error = %v", err)
	}
	firstFile := filepath.Join(firstRoot, "notes.txt")
	secondFile := filepath.Join(secondRoot, "notes.txt")
	if err := os.WriteFile(firstFile, []byte("same"), 0o600); err != nil {
		t.Fatalf("WriteFile(firstFile) error = %v", err)
	}
	if err := os.WriteFile(secondFile, []byte("same"), 0o644); err != nil {
		t.Fatalf("WriteFile(secondFile) error = %v", err)
	}
	if err := os.Chtimes(firstFile, time.Unix(100, 0), time.Unix(100, 0)); err != nil {
		t.Fatalf("Chtimes(firstFile) error = %v", err)
	}
	if err := os.Chtimes(secondFile, time.Unix(200, 0), time.Unix(200, 0)); err != nil {
		t.Fatalf("Chtimes(secondFile) error = %v", err)
	}

	roots := RootPaths{First: firstRoot, Second: secondRoot}
	store := newConfirmedStateStoreAt(filepath.Join(temporaryDirectory, "state"))
	if err := verifyAndCommitSynchronization(roots, store, SynchronizationResult{Completed: true}); err != nil {
		t.Fatalf("verifyAndCommitSynchronization() error = %v", err)
	}

	state, found, err := store.Load(roots)
	if err != nil {
		t.Fatalf("load() error = %v", err)
	}
	if !found {
		t.Fatal("load() found = false, want true")
	}
	if len(state.Entries) != 1 || state.Entries[0].RelativePath != "notes.txt" {
		t.Fatalf("state entries = %+v, want notes.txt entry", state.Entries)
	}
}

func TestVerifyAndCommitSynchronizationDoesNotReplaceStateWhenContentsDiffer(t *testing.T) {
	temporaryDirectory := t.TempDir()
	firstRoot := filepath.Join(temporaryDirectory, "first")
	secondRoot := filepath.Join(temporaryDirectory, "second")
	if err := os.MkdirAll(firstRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll(firstRoot) error = %v", err)
	}
	if err := os.MkdirAll(secondRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll(secondRoot) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(firstRoot, "notes.txt"), []byte("first"), 0o644); err != nil {
		t.Fatalf("WriteFile(firstFile) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(secondRoot, "notes.txt"), []byte("second"), 0o644); err != nil {
		t.Fatalf("WriteFile(secondFile) error = %v", err)
	}

	roots := RootPaths{First: firstRoot, Second: secondRoot}
	store := newConfirmedStateStoreAt(filepath.Join(temporaryDirectory, "state"))
	previousState := ConfirmedSynchronizationState{Version: 1, Roots: roots, Entries: []SynchronizationEntry{{RelativePath: "previous.txt", Kind: EntryKindFile}}}
	if err := store.Save(previousState, SynchronizationResult{Completed: true}); err != nil {
		t.Fatalf("save() error = %v", err)
	}

	if err := verifyAndCommitSynchronization(roots, store, SynchronizationResult{Completed: true}); err == nil {
		t.Fatal("verifyAndCommitSynchronization() error = nil, want unequal-content error")
	}
	state, found, err := store.Load(roots)
	if err != nil {
		t.Fatalf("load() error = %v", err)
	}
	if !found || !reflect.DeepEqual(state, previousState) {
		t.Fatalf("loaded state = %+v, found = %t, want previous state %+v", state, found, previousState)
	}
}

func TestVerifyAndCommitSynchronizationRejectsIncompleteResult(t *testing.T) {
	store := newConfirmedStateStoreAt(t.TempDir())
	roots := RootPaths{First: "first", Second: "second"}

	err := verifyAndCommitSynchronization(roots, store, SynchronizationResult{})
	if err == nil {
		t.Fatal("verifyAndCommitSynchronization() error = nil, want incomplete-result error")
	}
	if _, found, loadErr := store.Load(roots); loadErr != nil || found {
		t.Fatalf("state after incomplete result: found = %t, error = %v, want no state", found, loadErr)
	}
}
