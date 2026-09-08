package gosync_test

import (
	"errors"
	"testing"
	"time"
)

func TestResolveFileContentConflictCopiesNewerFirstFileToSecond(t *testing.T) {
	comparison := EntryComparison{
		RelativePath: "notes.txt",
		First:        fileConflictEntry("notes.txt", "newer-first", time.Date(2026, time.September, 6, 12, 0, 0, 0, time.UTC)),
		Second:       fileConflictEntry("notes.txt", "older-second", time.Date(2026, time.September, 6, 11, 0, 0, 0, time.UTC)),
	}

	action, required, err := resolveFileContentConflict(comparison, nil)
	if err != nil {
		t.Fatalf("resolveFileContentConflict() error = %v", err)
	}
	if !required {
		t.Fatal("resolveFileContentConflict() required = false, want true")
	}
	if action.Kind != SynchronizationActionCopyToSecond {
		t.Fatalf("action kind = %v, want %v", action.Kind, SynchronizationActionCopyToSecond)
	}
	if action.RelativePath != comparison.RelativePath {
		t.Fatalf("action path = %q, want %q", action.RelativePath, comparison.RelativePath)
	}
}

func TestResolveFileContentConflictCopiesNewerSecondFileToFirst(t *testing.T) {
	comparison := EntryComparison{
		RelativePath: "notes.txt",
		First:        fileConflictEntry("notes.txt", "older-first", time.Date(2026, time.September, 6, 11, 0, 0, 0, time.UTC)),
		Second:       fileConflictEntry("notes.txt", "newer-second", time.Date(2026, time.September, 6, 12, 0, 0, 0, time.UTC)),
	}

	action, required, err := resolveFileContentConflict(comparison, nil)
	if err != nil {
		t.Fatalf("resolveFileContentConflict() error = %v", err)
	}
	if !required {
		t.Fatal("resolveFileContentConflict() required = false, want true")
	}
	if action.Kind != SynchronizationActionCopyToFirst {
		t.Fatalf("action kind = %v, want %v", action.Kind, SynchronizationActionCopyToFirst)
	}
}

func TestResolveFileContentConflictRequestsChoiceWhenModificationTimesMatch(t *testing.T) {
	modificationTime := time.Date(2026, time.September, 6, 12, 0, 0, 0, time.UTC)
	comparison := EntryComparison{
		RelativePath: "notes.txt",
		First:        fileConflictEntry("notes.txt", "first-content", modificationTime),
		Second:       fileConflictEntry("notes.txt", "second-content", modificationTime),
	}
	requestCalls := 0
	requestConflictSide := func() (SynchronizationSide, error) {
		requestCalls++
		return SynchronizationSideFirst, nil
	}

	action, required, err := resolveFileContentConflict(comparison, requestConflictSide)
	if err != nil {
		t.Fatalf("resolveFileContentConflict() error = %v", err)
	}
	if !required {
		t.Fatal("resolveFileContentConflict() required = false, want true")
	}
	if action.Kind != SynchronizationActionCopyToSecond {
		t.Fatalf("action kind = %v, want %v", action.Kind, SynchronizationActionCopyToSecond)
	}
	if requestCalls != 1 {
		t.Fatalf("conflict choice request calls = %d, want 1", requestCalls)
	}
}

func TestResolveFileContentConflictPropagatesChoiceError(t *testing.T) {
	modificationTime := time.Date(2026, time.September, 6, 12, 0, 0, 0, time.UTC)
	comparison := EntryComparison{
		RelativePath: "notes.txt",
		First:        fileConflictEntry("notes.txt", "first-content", modificationTime),
		Second:       fileConflictEntry("notes.txt", "second-content", modificationTime),
	}
	wantError := errors.New("user cancelled conflict resolution")

	_, _, err := resolveFileContentConflict(comparison, func() (SynchronizationSide, error) {
		return SynchronizationSideFirst, wantError
	})
	if !errors.Is(err, wantError) {
		t.Fatalf("resolveFileContentConflict() error = %v, want %v", err, wantError)
	}
}

func TestResolveFileContentConflictDoesNothingForMatchingContent(t *testing.T) {
	firstTime := time.Date(2026, time.September, 6, 12, 0, 0, 0, time.UTC)
	secondTime := firstTime.Add(time.Hour)
	comparison := EntryComparison{
		RelativePath: "notes.txt",
		First:        fileConflictEntry("notes.txt", "same-content", firstTime),
		Second:       fileConflictEntry("notes.txt", "same-content", secondTime),
	}

	action, required, err := resolveFileContentConflict(comparison, nil)
	if err != nil {
		t.Fatalf("resolveFileContentConflict() error = %v", err)
	}
	if required {
		t.Fatal("resolveFileContentConflict() required = true, want false")
	}
	if action != (SynchronizationAction{}) {
		t.Fatalf("action = %+v, want empty action", action)
	}
}

func fileConflictEntry(path, content string, modificationTime time.Time) *SynchronizationEntry {
	return &SynchronizationEntry{
		RelativePath:     path,
		Kind:             EntryKindFile,
		ContentDigest:    content,
		ModificationTime: modificationTime,
	}
}
