package gosync

import (
	"fmt"
	"sort"
	"time"
)

func verifyAndCommitSynchronization(roots RootPaths, store confirmedStateStore, result SynchronizationResult) error {
	if !result.Completed || result.Failure != nil {
		return errSynchronizationIncomplete
	}

	firstInventory, err := buildDirectoryInventory(roots.First)
	if err != nil {
		return fmt.Errorf("verify first directory: %w", err)
	}
	secondInventory, err := buildDirectoryInventory(roots.Second)
	if err != nil {
		return fmt.Errorf("verify second directory: %w", err)
	}
	if !directoryInventoriesHaveEquivalentContents(firstInventory, secondInventory) {
		return fmt.Errorf("synchronized directories have different contents")
	}

	entries := make([]SynchronizationEntry, 0, len(firstInventory))
	for _, entry := range firstInventory {
		entries = append(entries, entry)
	}
	sort.Slice(entries, func(firstIndex, secondIndex int) bool {
		return entries[firstIndex].RelativePath < entries[secondIndex].RelativePath
	})

	state := ConfirmedSynchronizationState{
		Version:     1,
		Roots:       roots,
		Entries:     entries,
		CompletedAt: time.Now().UTC(),
	}
	if err := store.save(state, result); err != nil {
		return fmt.Errorf("commit confirmed synchronization state: %w", err)
	}

	return nil
}
