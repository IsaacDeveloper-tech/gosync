package gosync

import (
	"fmt"
	"os"
)

func validateUnidirectionalRoots(roots RootPaths) error {
	normalizedRoots, err := validateRootPaths(roots.First, roots.Second)
	if err != nil {
		return err
	}
	sourceInformation, err := os.Stat(normalizedRoots.First)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("unidirectional source directory does not exist: %s", normalizedRoots.First)
		}
		return fmt.Errorf("inspect unidirectional source directory: %w", err)
	}
	if !sourceInformation.IsDir() {
		return fmt.Errorf("unidirectional source is not a directory: %s", normalizedRoots.First)
	}
	return nil
}

func generateUnidirectionalSynchronizationPlan(comparisons []EntryComparison) (SynchronizationPlan, error) {
	actions := make([]SynchronizationAction, 0)
	for _, comparison := range comparisons {
		switch {
		case comparison.First == nil && comparison.Second != nil:
			actions = append(actions, SynchronizationAction{
				Kind:         SynchronizationActionDeleteFromSecond,
				RelativePath: comparison.RelativePath,
			})
		case comparison.First != nil && comparison.Second == nil:
			actions = append(actions, SynchronizationAction{
				Kind:         SynchronizationActionCopyToSecond,
				RelativePath: comparison.RelativePath,
			})
		case comparison.First != nil && comparison.Second != nil:
			if comparison.First.Kind != comparison.Second.Kind ||
				(comparison.First.Kind == EntryKindFile && comparison.First.ContentDigest != comparison.Second.ContentDigest) {
				actions = append(actions, SynchronizationAction{
					Kind:         SynchronizationActionCopyToSecond,
					RelativePath: comparison.RelativePath,
				})
			}
		}
	}
	return SynchronizationPlan{Actions: actions}, nil
}

func synchronizeUnidirectional(roots RootPaths, store confirmedStateStore, options SynchronizationExecutionOptions) error {
	if err := validateUnidirectionalRoots(roots); err != nil {
		return err
	}
	normalizedRoots, err := validateRootPaths(roots.First, roots.Second)
	if err != nil {
		return err
	}
	if err := scanRootForUnsupportedEntries(normalizedRoots.First); err != nil {
		return err
	}
	if _, err := os.Stat(normalizedRoots.Second); err == nil {
		if err := scanRootForUnsupportedEntries(normalizedRoots.Second); err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("inspect unidirectional destination: %w", err)
	}
	if err := os.MkdirAll(normalizedRoots.Second, 0o755); err != nil {
		return fmt.Errorf("create unidirectional destination: %w", err)
	}

	sourceInventory, err := buildDirectoryInventory(normalizedRoots.First)
	if err != nil {
		return err
	}
	destinationInventory, err := buildDirectoryInventory(normalizedRoots.Second)
	if err != nil {
		return err
	}
	plan, err := generateUnidirectionalSynchronizationPlan(compareDirectoryInventories(sourceInventory, destinationInventory))
	if err != nil {
		return err
	}
	result := executeSynchronizationPlan(normalizedRoots, plan, options)
	if !result.Completed {
		return result.Failure
	}
	if err := verifyAndCommitSynchronization(normalizedRoots, store, result); err != nil {
		return err
	}
	return nil
}
