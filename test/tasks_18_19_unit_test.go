package gosync_test

import (
	"path/filepath"
	"testing"
	"time"
)

func TestConfirmedStateStoreReplacesCompletedStateForSameRootPair(t *testing.T) {
	store := newConfirmedStateStoreAt(t.TempDir())
	roots := RootPaths{First: "first", Second: "second"}
	firstState := ConfirmedSynchronizationState{
		Version: 1,
		Roots:   roots,
		Entries: []SynchronizationEntry{{RelativePath: "first.txt", Kind: EntryKindFile}},
	}
	secondState := ConfirmedSynchronizationState{
		Version: 1,
		Roots:   roots,
		Entries: []SynchronizationEntry{{RelativePath: "second.txt", Kind: EntryKindFile}},
	}

	if err := store.Save(firstState, SynchronizationResult{Completed: true}); err != nil {
		t.Fatalf("save(firstState) error = %v", err)
	}
	if err := store.Save(secondState, SynchronizationResult{Completed: true}); err != nil {
		t.Fatalf("save(secondState) error = %v", err)
	}

	loadedState, found, err := store.Load(roots)
	if err != nil {
		t.Fatalf("load() error = %v", err)
	}
	if !found || len(loadedState.Entries) != 1 || loadedState.Entries[0].RelativePath != "second.txt" {
		t.Fatalf("loaded state = %+v, found = %t, want second.txt state", loadedState, found)
	}
}

func TestRunWatchCommandDoesNotSynchronizeWhenArgumentsAreInvalid(t *testing.T) {
	callCount := 0
	_, err := parseWatchCommand([]string{"watch", filepath.Join(t.TempDir(), "first")})
	if err == nil {
		t.Fatal("parseWatchCommand() error = nil, want invalid-argument error")
	}

	commandErr := runWatchCommand([]string{"watch"}, WatchCommandOptions{
		Synchronize: func(RootPaths) error {
			callCount++
			return nil
		},
	})
	if commandErr == nil {
		t.Fatal("runWatchCommand() error = nil, want invalid-argument error")
	}
	if callCount != 0 {
		t.Fatalf("synchronization calls = %d, want 0", callCount)
	}
}

func TestResolveFileContentConflictRejectsInvalidChoice(t *testing.T) {
	modificationTime := time.Date(2026, time.September, 6, 12, 0, 0, 0, time.UTC)
	comparison := EntryComparison{
		RelativePath: "notes.txt",
		First:        fileConflictEntry("notes.txt", "first", modificationTime),
		Second:       fileConflictEntry("notes.txt", "second", modificationTime),
	}

	_, _, err := resolveFileContentConflict(comparison, func() (SynchronizationSide, error) {
		return SynchronizationSide(99), nil
	})
	if err == nil {
		t.Fatal("resolveFileContentConflict() error = nil, want invalid-choice error")
	}
}

func TestResolveFileDirectoryConflictRejectsInvalidChoice(t *testing.T) {
	comparison := EntryComparison{
		RelativePath: "notes",
		First:        synchronizationEntry("notes", EntryKindFile),
		Second:       synchronizationEntry("notes", EntryKindDirectory),
	}

	_, _, err := resolveFileDirectoryConflict(comparison, nil, func([]string) (SynchronizationSide, error) {
		return SynchronizationSide(99), nil
	})
	if err == nil {
		t.Fatal("resolveFileDirectoryConflict() error = nil, want invalid-choice error")
	}
}
