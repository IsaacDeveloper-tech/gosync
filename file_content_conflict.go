package main

import "fmt"

func resolveFileContentConflict(
	comparison EntryComparison,
	requestConflictSide func() (SynchronizationSide, error),
) (SynchronizationAction, bool, error) {
	if comparison.First == nil || comparison.Second == nil {
		return SynchronizationAction{}, false, fmt.Errorf("file content conflict requires both entries")
	}
	if comparison.First.Kind != EntryKindFile || comparison.Second.Kind != EntryKindFile {
		return SynchronizationAction{}, false, fmt.Errorf("file content conflict requires file entries")
	}
	if comparison.First.ContentDigest == comparison.Second.ContentDigest {
		return SynchronizationAction{}, false, nil
	}

	switch {
	case comparison.First.ModificationTime.After(comparison.Second.ModificationTime):
		return SynchronizationAction{
			Kind:         SynchronizationActionCopyToSecond,
			RelativePath: comparison.RelativePath,
		}, true, nil
	case comparison.Second.ModificationTime.After(comparison.First.ModificationTime):
		return SynchronizationAction{
			Kind:         SynchronizationActionCopyToFirst,
			RelativePath: comparison.RelativePath,
		}, true, nil
	}

	if requestConflictSide == nil {
		return SynchronizationAction{}, false, fmt.Errorf("file conflict decision is required")
	}
	conflictSide, err := requestConflictSide()
	if err != nil {
		return SynchronizationAction{}, false, err
	}
	switch conflictSide {
	case SynchronizationSideFirst:
		return SynchronizationAction{
			Kind:         SynchronizationActionCopyToSecond,
			RelativePath: comparison.RelativePath,
		}, true, nil
	case SynchronizationSideSecond:
		return SynchronizationAction{
			Kind:         SynchronizationActionCopyToFirst,
			RelativePath: comparison.RelativePath,
		}, true, nil
	default:
		return SynchronizationAction{}, false, fmt.Errorf("unknown conflict side %d", conflictSide)
	}
}
