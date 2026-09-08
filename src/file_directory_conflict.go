package gosync

import (
	"fmt"
	"sort"
)

func resolveFileDirectoryConflict(
	comparison EntryComparison,
	directoryContents []string,
	requestConflictSide func([]string) (SynchronizationSide, error),
) (SynchronizationAction, bool, error) {
	if comparison.First == nil || comparison.Second == nil {
		return SynchronizationAction{}, false, fmt.Errorf("file-directory conflict requires both entries")
	}
	if comparison.First.Kind == comparison.Second.Kind {
		return SynchronizationAction{}, false, fmt.Errorf("file-directory conflict requires one file and one directory")
	}
	if comparison.First.Kind != EntryKindFile && comparison.First.Kind != EntryKindDirectory {
		return SynchronizationAction{}, false, fmt.Errorf("unsupported first entry kind %d", comparison.First.Kind)
	}
	if comparison.Second.Kind != EntryKindFile && comparison.Second.Kind != EntryKindDirectory {
		return SynchronizationAction{}, false, fmt.Errorf("unsupported second entry kind %d", comparison.Second.Kind)
	}
	if requestConflictSide == nil {
		return SynchronizationAction{}, false, fmt.Errorf("file-directory conflict decision is required")
	}

	listedContents := append([]string(nil), directoryContents...)
	sort.Strings(listedContents)
	conflictSide, err := requestConflictSide(listedContents)
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
