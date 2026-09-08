package main

import "testing"

func TestClassifyAdditionsAndDeletions(t *testing.T) {
	testCases := []struct {
		name             string
		comparison       EntryComparison
		wantAction       SynchronizationActionKind
		wantRelativePath string
	}{
		{
			name: "addition from first directory",
			comparison: EntryComparison{
				RelativePath: "new-first.txt",
				First:        synchronizationEntry("new-first.txt", EntryKindFile),
			},
			wantAction:       SynchronizationActionCopyToSecond,
			wantRelativePath: "new-first.txt",
		},
		{
			name: "addition from second directory",
			comparison: EntryComparison{
				RelativePath: "new-second.txt",
				Second:       synchronizationEntry("new-second.txt", EntryKindFile),
			},
			wantAction:       SynchronizationActionCopyToFirst,
			wantRelativePath: "new-second.txt",
		},
		{
			name: "deletion from first directory",
			comparison: EntryComparison{
				RelativePath: "deleted-first.txt",
				First:        synchronizationEntry("deleted-first.txt", EntryKindFile),
				Confirmed:    synchronizationEntry("deleted-first.txt", EntryKindFile),
			},
			wantAction:       SynchronizationActionDeleteFromFirst,
			wantRelativePath: "deleted-first.txt",
		},
		{
			name: "deletion from second directory",
			comparison: EntryComparison{
				RelativePath: "deleted-second.txt",
				Second:       synchronizationEntry("deleted-second.txt", EntryKindFile),
				Confirmed:    synchronizationEntry("deleted-second.txt", EntryKindFile),
			},
			wantAction:       SynchronizationActionDeleteFromSecond,
			wantRelativePath: "deleted-second.txt",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			actions := classifyAdditionsAndDeletions([]EntryComparison{testCase.comparison})
			if len(actions) != 1 {
				t.Fatalf("classifyAdditionsAndDeletions() returned %d actions, want 1", len(actions))
			}
			if actions[0].Kind != testCase.wantAction {
				t.Fatalf("action kind = %v, want %v", actions[0].Kind, testCase.wantAction)
			}
			if actions[0].RelativePath != testCase.wantRelativePath {
				t.Fatalf("action path = %q, want %q", actions[0].RelativePath, testCase.wantRelativePath)
			}
		})
	}
}

func TestClassifyAdditionsAndDeletionsIgnoresUnchangedOrAbsentEntries(t *testing.T) {
	unchangedEntry := synchronizationEntry("unchanged.txt", EntryKindFile)
	comparisons := []EntryComparison{
		{
			RelativePath: "unchanged.txt",
			First:        unchangedEntry,
			Second:       synchronizationEntry("unchanged.txt", EntryKindFile),
			Confirmed:    synchronizationEntry("unchanged.txt", EntryKindFile),
		},
		{RelativePath: "absent.txt", Confirmed: synchronizationEntry("absent.txt", EntryKindFile)},
		{RelativePath: "not-yet-present.txt"},
	}

	actions := classifyAdditionsAndDeletions(comparisons)
	if len(actions) != 0 {
		t.Fatalf("classifyAdditionsAndDeletions() returned %d actions, want 0", len(actions))
	}
}

func TestClassifyAdditionsAndDeletionsPreservesComparisonOrder(t *testing.T) {
	comparisons := []EntryComparison{
		{
			RelativePath: "second.txt",
			Second:       synchronizationEntry("second.txt", EntryKindFile),
		},
		{
			RelativePath: "first.txt",
			First:        synchronizationEntry("first.txt", EntryKindFile),
		},
	}

	actions := classifyAdditionsAndDeletions(comparisons)
	if len(actions) != 2 {
		t.Fatalf("classifyAdditionsAndDeletions() returned %d actions, want 2", len(actions))
	}
	if actions[0].RelativePath != "second.txt" || actions[1].RelativePath != "first.txt" {
		t.Fatalf("action order = [%q, %q], want [second.txt, first.txt]", actions[0].RelativePath, actions[1].RelativePath)
	}
}

func synchronizationEntry(path string, kind EntryKind) *SynchronizationEntry {
	return &SynchronizationEntry{RelativePath: path, Kind: kind}
}
