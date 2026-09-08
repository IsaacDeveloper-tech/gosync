package main

import (
	"errors"
	"reflect"
	"testing"
	"time"
)

func TestGenerateSynchronizationPlanIncludesChangesAndAutomaticConflictResolution(t *testing.T) {
	comparisons := []EntryComparison{
		{
			RelativePath: "new-first.txt",
			First:        synchronizationEntry("new-first.txt", EntryKindFile),
		},
		{
			RelativePath: "new-second.txt",
			Second:       synchronizationEntry("new-second.txt", EntryKindFile),
		},
		{
			RelativePath: "deleted-first.txt",
			First:        synchronizationEntry("deleted-first.txt", EntryKindFile),
			Confirmed:    synchronizationEntry("deleted-first.txt", EntryKindFile),
		},
		{
			RelativePath: "deleted-second.txt",
			Second:       synchronizationEntry("deleted-second.txt", EntryKindFile),
			Confirmed:    synchronizationEntry("deleted-second.txt", EntryKindFile),
		},
		{
			RelativePath: "newer-first.txt",
			First:        fileConflictEntry("newer-first.txt", "newer", time.Date(2026, time.September, 6, 12, 0, 0, 0, time.UTC)),
			Second:       fileConflictEntry("newer-first.txt", "older", time.Date(2026, time.September, 6, 11, 0, 0, 0, time.UTC)),
		},
	}

	plan, err := generateSynchronizationPlan(comparisons, nil, nil, nil)
	if err != nil {
		t.Fatalf("generateSynchronizationPlan() error = %v", err)
	}

	wantActions := []SynchronizationAction{
		{Kind: SynchronizationActionCopyToSecond, RelativePath: "new-first.txt"},
		{Kind: SynchronizationActionCopyToFirst, RelativePath: "new-second.txt"},
		{Kind: SynchronizationActionDeleteFromFirst, RelativePath: "deleted-first.txt"},
		{Kind: SynchronizationActionDeleteFromSecond, RelativePath: "deleted-second.txt"},
		{Kind: SynchronizationActionCopyToSecond, RelativePath: "newer-first.txt"},
	}
	if !reflect.DeepEqual(plan.Actions, wantActions) {
		t.Fatalf("plan actions = %+v, want %+v", plan.Actions, wantActions)
	}
}

func TestGenerateSynchronizationPlanIncludesUserDecisionsBeforeExecution(t *testing.T) {
	modificationTime := time.Date(2026, time.September, 6, 12, 0, 0, 0, time.UTC)
	comparisons := []EntryComparison{
		{
			RelativePath: "equal-time.txt",
			First:        fileConflictEntry("equal-time.txt", "first", modificationTime),
			Second:       fileConflictEntry("equal-time.txt", "second", modificationTime),
		},
		{
			RelativePath: "file-directory",
			First:        synchronizationEntry("file-directory", EntryKindFile),
			Second:       synchronizationEntry("file-directory", EntryKindDirectory),
		},
	}
	fileConflictRequests := 0
	fileDirectoryConflictRequests := 0
	var receivedDirectoryContents []string

	plan, err := generateSynchronizationPlan(
		comparisons,
		map[string][]string{"file-directory": {"file-directory/item.txt"}},
		func(comparison EntryComparison) (SynchronizationSide, error) {
			fileConflictRequests++
			if comparison.RelativePath != "equal-time.txt" {
				t.Fatalf("file conflict path = %q, want equal-time.txt", comparison.RelativePath)
			}
			return SynchronizationSideFirst, nil
		},
		func(comparison EntryComparison, contents []string) (SynchronizationSide, error) {
			fileDirectoryConflictRequests++
			if comparison.RelativePath != "file-directory" {
				t.Fatalf("file-directory conflict path = %q, want file-directory", comparison.RelativePath)
			}
			receivedDirectoryContents = contents
			return SynchronizationSideSecond, nil
		},
	)
	if err != nil {
		t.Fatalf("generateSynchronizationPlan() error = %v", err)
	}
	if fileConflictRequests != 1 || fileDirectoryConflictRequests != 1 {
		t.Fatalf("decision requests = file %d, file-directory %d, want 1 and 1", fileConflictRequests, fileDirectoryConflictRequests)
	}
	if !reflect.DeepEqual(receivedDirectoryContents, []string{"file-directory/item.txt"}) {
		t.Fatalf("directory contents = %v, want [file-directory/item.txt]", receivedDirectoryContents)
	}
	wantActions := []SynchronizationAction{
		{Kind: SynchronizationActionCopyToSecond, RelativePath: "equal-time.txt"},
		{Kind: SynchronizationActionCopyToFirst, RelativePath: "file-directory"},
	}
	if !reflect.DeepEqual(plan.Actions, wantActions) {
		t.Fatalf("plan actions = %+v, want %+v", plan.Actions, wantActions)
	}
}

func TestGenerateSynchronizationPlanReturnsErrorWithoutPartialPlan(t *testing.T) {
	comparisons := []EntryComparison{
		{
			RelativePath: "new.txt",
			First:        synchronizationEntry("new.txt", EntryKindFile),
		},
		{
			RelativePath: "equal-time.txt",
			First:        fileConflictEntry("equal-time.txt", "first", time.Time{}),
			Second:       fileConflictEntry("equal-time.txt", "second", time.Time{}),
		},
	}
	wantError := errors.New("user cancelled conflict resolution")

	plan, err := generateSynchronizationPlan(
		comparisons,
		nil,
		func(EntryComparison) (SynchronizationSide, error) {
			return SynchronizationSideFirst, wantError
		},
		nil,
	)
	if !errors.Is(err, wantError) {
		t.Fatalf("generateSynchronizationPlan() error = %v, want %v", err, wantError)
	}
	if len(plan.Actions) != 0 {
		t.Fatalf("plan actions = %+v, want no partial actions", plan.Actions)
	}
}
