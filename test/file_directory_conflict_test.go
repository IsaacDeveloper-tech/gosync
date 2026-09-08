package gosync_test

import (
	"errors"
	"reflect"
	"testing"
)

func TestResolveFileDirectoryConflictListsContentsAndCopiesFirstFileToSecondDirectory(t *testing.T) {
	comparison := EntryComparison{
		RelativePath: "notes",
		First:        synchronizationEntry("notes", EntryKindFile),
		Second:       synchronizationEntry("notes", EntryKindDirectory),
	}
	providedContents := []string{"notes/z.txt", "notes/a.txt"}
	var receivedContents []string
	requestConflictSide := func(contents []string) (SynchronizationSide, error) {
		receivedContents = contents
		return SynchronizationSideFirst, nil
	}

	action, required, err := resolveFileDirectoryConflict(comparison, providedContents, requestConflictSide)
	if err != nil {
		t.Fatalf("resolveFileDirectoryConflict() error = %v", err)
	}
	if !required {
		t.Fatal("resolveFileDirectoryConflict() required = false, want true")
	}
	if action.Kind != SynchronizationActionCopyToSecond {
		t.Fatalf("action kind = %v, want %v", action.Kind, SynchronizationActionCopyToSecond)
	}
	if !reflect.DeepEqual(receivedContents, []string{"notes/a.txt", "notes/z.txt"}) {
		t.Fatalf("listed contents = %v, want [notes/a.txt notes/z.txt]", receivedContents)
	}
	if !reflect.DeepEqual(providedContents, []string{"notes/z.txt", "notes/a.txt"}) {
		t.Fatal("resolveFileDirectoryConflict() modified the provided contents")
	}
}

func TestResolveFileDirectoryConflictCopiesSecondDirectoryToFirstFile(t *testing.T) {
	comparison := EntryComparison{
		RelativePath: "archive",
		First:        synchronizationEntry("archive", EntryKindDirectory),
		Second:       synchronizationEntry("archive", EntryKindFile),
	}
	requestConflictSide := func(contents []string) (SynchronizationSide, error) {
		if len(contents) != 1 || contents[0] != "archive/notes.txt" {
			t.Fatalf("listed contents = %v, want [archive/notes.txt]", contents)
		}
		return SynchronizationSideSecond, nil
	}

	action, required, err := resolveFileDirectoryConflict(comparison, []string{"archive/notes.txt"}, requestConflictSide)
	if err != nil {
		t.Fatalf("resolveFileDirectoryConflict() error = %v", err)
	}
	if !required {
		t.Fatal("resolveFileDirectoryConflict() required = false, want true")
	}
	if action.Kind != SynchronizationActionCopyToFirst {
		t.Fatalf("action kind = %v, want %v", action.Kind, SynchronizationActionCopyToFirst)
	}
}

func TestResolveFileDirectoryConflictRequiresUserChoice(t *testing.T) {
	comparison := EntryComparison{
		RelativePath: "notes",
		First:        synchronizationEntry("notes", EntryKindFile),
		Second:       synchronizationEntry("notes", EntryKindDirectory),
	}

	action, required, err := resolveFileDirectoryConflict(comparison, []string{"notes/a.txt"}, nil)
	if err == nil {
		t.Fatal("resolveFileDirectoryConflict() error = nil, want a user-choice error")
	}
	if required {
		t.Fatal("resolveFileDirectoryConflict() required = true, want false on error")
	}
	if action != (SynchronizationAction{}) {
		t.Fatalf("action = %+v, want empty action", action)
	}
}

func TestResolveFileDirectoryConflictPropagatesChoiceError(t *testing.T) {
	comparison := EntryComparison{
		RelativePath: "notes",
		First:        synchronizationEntry("notes", EntryKindFile),
		Second:       synchronizationEntry("notes", EntryKindDirectory),
	}
	wantError := errors.New("user cancelled conflict resolution")

	_, _, err := resolveFileDirectoryConflict(comparison, nil, func([]string) (SynchronizationSide, error) {
		return SynchronizationSideFirst, wantError
	})
	if !errors.Is(err, wantError) {
		t.Fatalf("resolveFileDirectoryConflict() error = %v, want %v", err, wantError)
	}
}
