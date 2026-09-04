package main

import (
	"errors"
	"testing"
)

func TestSynchronizationEntryRepresentsFileAndDirectory(t *testing.T) {
	fileEntry := SynchronizationEntry{
		RelativePath: "notes.txt",
		Kind:         EntryKindFile,
	}
	directoryEntry := SynchronizationEntry{
		RelativePath: "archive",
		Kind:         EntryKindDirectory,
	}

	if fileEntry.Kind != EntryKindFile {
		t.Fatalf("file entry kind = %v, want %v", fileEntry.Kind, EntryKindFile)
	}
	if directoryEntry.Kind != EntryKindDirectory {
		t.Fatalf("directory entry kind = %v, want %v", directoryEntry.Kind, EntryKindDirectory)
	}
}

func TestEntryComparisonRepresentsBothDirectoriesAndConfirmedState(t *testing.T) {
	firstEntry := SynchronizationEntry{RelativePath: "notes.txt", Kind: EntryKindFile}
	secondEntry := SynchronizationEntry{RelativePath: "notes.txt", Kind: EntryKindFile}
	confirmedEntry := SynchronizationEntry{RelativePath: "notes.txt", Kind: EntryKindFile}

	comparison := EntryComparison{
		RelativePath: "notes.txt",
		First:        &firstEntry,
		Second:       &secondEntry,
		Confirmed:    &confirmedEntry,
	}

	if comparison.First == nil || comparison.Second == nil || comparison.Confirmed == nil {
		t.Fatal("comparison must represent entries in both directories and confirmed state")
	}
}

func TestSynchronizationActionRepresentsCopyAndDeletionDirections(t *testing.T) {
	testCases := []struct {
		name string
		kind SynchronizationActionKind
	}{
		{name: "copy to first directory", kind: SynchronizationActionCopyToFirst},
		{name: "copy to second directory", kind: SynchronizationActionCopyToSecond},
		{name: "delete from first directory", kind: SynchronizationActionDeleteFromFirst},
		{name: "delete from second directory", kind: SynchronizationActionDeleteFromSecond},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			action := SynchronizationAction{
				Kind:         testCase.kind,
				RelativePath: "notes.txt",
			}

			if action.Kind != testCase.kind {
				t.Fatalf("action kind = %v, want %v", action.Kind, testCase.kind)
			}
		})
	}
}

func TestSynchronizationResultRepresentsSuccessAndFailure(t *testing.T) {
	failure := errors.New("write failed")

	successResult := SynchronizationResult{Completed: true}
	failureResult := SynchronizationResult{Failure: failure}

	if !successResult.Completed || successResult.Failure != nil {
		t.Fatal("successful result must be completed without a failure")
	}
	if failureResult.Completed || !errors.Is(failureResult.Failure, failure) {
		t.Fatal("failed result must retain its failure and remain incomplete")
	}
}
