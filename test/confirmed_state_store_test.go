package gosync_test

import (
	"reflect"
	"testing"
	"time"
)

func TestConfirmedStateStoreSavesAndLoadsCompletedState(t *testing.T) {
	store := newConfirmedStateStoreAt(t.TempDir())
	roots := RootPaths{First: "C:/first", Second: "C:/second"}
	state := ConfirmedSynchronizationState{
		Version: 1,
		Roots:   roots,
		Entries: []SynchronizationEntry{
			{RelativePath: "notes.txt", Kind: EntryKindFile},
			{RelativePath: "archive", Kind: EntryKindDirectory},
		},
		CompletedAt: time.Date(2026, time.September, 6, 12, 0, 0, 0, time.UTC),
	}

	if err := store.Save(state, SynchronizationResult{Completed: true}); err != nil {
		t.Fatalf("save() error = %v", err)
	}

	loadedState, found, err := store.Load(roots)
	if err != nil {
		t.Fatalf("load() error = %v", err)
	}
	if !found {
		t.Fatal("load() found = false, want true")
	}
	if !reflect.DeepEqual(loadedState, state) {
		t.Fatalf("load() = %+v, want %+v", loadedState, state)
	}
}

func TestConfirmedStateStorePreservesPreviousStateForIncompleteResult(t *testing.T) {
	store := newConfirmedStateStoreAt(t.TempDir())
	roots := RootPaths{First: "C:/first", Second: "C:/second"}
	confirmedState := ConfirmedSynchronizationState{
		Version: 1,
		Roots:   roots,
		Entries: []SynchronizationEntry{{RelativePath: "confirmed.txt", Kind: EntryKindFile}},
	}
	if err := store.Save(confirmedState, SynchronizationResult{Completed: true}); err != nil {
		t.Fatalf("save() error = %v", err)
	}

	incompleteState := ConfirmedSynchronizationState{
		Version: 1,
		Roots:   roots,
		Entries: []SynchronizationEntry{{RelativePath: "incomplete.txt", Kind: EntryKindFile}},
	}
	err := store.Save(incompleteState, SynchronizationResult{Failure: errSynchronizationIncomplete})
	if err == nil {
		t.Fatal("save() error = nil, want an error for an incomplete result")
	}

	loadedState, found, err := store.Load(roots)
	if err != nil {
		t.Fatalf("load() error = %v", err)
	}
	if !found {
		t.Fatal("load() found = false, want true")
	}
	if !reflect.DeepEqual(loadedState, confirmedState) {
		t.Fatalf("load() = %+v, want preserved state %+v", loadedState, confirmedState)
	}
}

func TestConfirmedStateStoreSeparatesRootPairs(t *testing.T) {
	store := newConfirmedStateStoreAt(t.TempDir())
	firstRoots := RootPaths{First: "C:/first", Second: "C:/second"}
	secondRoots := RootPaths{First: "C:/third", Second: "C:/fourth"}
	firstState := ConfirmedSynchronizationState{Version: 1, Roots: firstRoots}
	secondState := ConfirmedSynchronizationState{Version: 1, Roots: secondRoots}

	if err := store.Save(firstState, SynchronizationResult{Completed: true}); err != nil {
		t.Fatalf("save(firstState) error = %v", err)
	}
	if err := store.Save(secondState, SynchronizationResult{Completed: true}); err != nil {
		t.Fatalf("save(secondState) error = %v", err)
	}

	loadedFirstState, foundFirst, err := store.Load(firstRoots)
	if err != nil {
		t.Fatalf("load(firstRoots) error = %v", err)
	}
	if !foundFirst || !reflect.DeepEqual(loadedFirstState, firstState) {
		t.Fatalf("load(firstRoots) = %+v, found = %t, want %+v", loadedFirstState, foundFirst, firstState)
	}

	loadedSecondState, foundSecond, err := store.Load(secondRoots)
	if err != nil {
		t.Fatalf("load(secondRoots) error = %v", err)
	}
	if !foundSecond || !reflect.DeepEqual(loadedSecondState, secondState) {
		t.Fatalf("load(secondRoots) = %+v, found = %t, want %+v", loadedSecondState, foundSecond, secondState)
	}
}

func TestConfirmedStateStoreReportsMissingState(t *testing.T) {
	store := newConfirmedStateStoreAt(t.TempDir())

	_, found, err := store.Load(RootPaths{First: "C:/first", Second: "C:/second"})
	if err != nil {
		t.Fatalf("load() error = %v", err)
	}
	if found {
		t.Fatal("load() found = true, want false")
	}
}
