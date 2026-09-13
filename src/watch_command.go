package gosync

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"
)

type WatchCommandOptions struct {
	Input       io.Reader
	Output      io.Writer
	Interval    time.Duration
	Stop        <-chan struct{}
	StateStore  *ConfirmedStateStore
	Synchronize func(RootPaths) error
	Logger      *LoggingCoordinator
	Logging     LoggingCoordinatorOptions
}

func runWatchCommand(arguments []string, options WatchCommandOptions) error {
	roots, err := parseWatchCommand(arguments)
	if err != nil {
		return err
	}

	if options.Synchronize != nil {
		return runWatchLoop(func() error {
			return options.Synchronize(roots)
		}, WatchLoopOptions{Interval: options.Interval, Stop: options.Stop})
	}

	input := options.Input
	if input == nil {
		input = os.Stdin
	}
	output := options.Output
	if output == nil {
		output = os.Stdout
	}
	store := options.StateStore
	if store == nil {
		defaultStore, err := newConfirmedStateStore()
		if err != nil {
			return err
		}
		store = &defaultStore
	}
	logger := options.Logger
	if logger == nil {
		loggingOptions := options.Logging
		if loggingOptions.ConsoleWriter == nil {
			loggingOptions.ConsoleWriter = output
		}
		logger, err = initializeLoggingCoordinator(roots, loggingOptions)
		if err != nil {
			return err
		}
	}

	return runWatchLoop(func() error {
		return synchronizeDirectoriesWithLogger(roots, *store, input, output, logger)
	}, WatchLoopOptions{Interval: options.Interval, Stop: options.Stop})
}

func synchronizeDirectories(roots RootPaths, store confirmedStateStore, input io.Reader, output io.Writer) error {
	return synchronizeDirectoriesWithLogger(roots, store, input, output, nil)
}

func synchronizeDirectoriesWithLogger(
	roots RootPaths,
	store confirmedStateStore,
	input io.Reader,
	output io.Writer,
	logger *LoggingCoordinator,
) error {
	if logger != nil {
		if err := logger.Log(LogEntry{
			Severity: LogSeverityInfo,
			Event:    LogEventSynchronizationStarted,
			Message:  "synchronization started",
			Context: map[string]string{
				"first_root":  roots.First,
				"second_root": roots.Second,
			},
		}); err != nil {
			return err
		}
	}

	synchronizationError := performSynchronization(roots, store, input, output, logger)
	if synchronizationError != nil {
		if logger == nil {
			return synchronizationError
		}
		if err := logger.Log(LogEntry{
			Severity: LogSeverityError,
			Event:    LogEventOperationFailed,
			Message:  "synchronization operation failed",
			Context:  map[string]string{"error": synchronizationError.Error()},
		}); err != nil {
			return err
		}
		if err := logger.Log(LogEntry{
			Severity: LogSeverityError,
			Event:    LogEventSynchronizationCompleted,
			Message:  "synchronization completed with failure",
			Context:  map[string]string{"error": synchronizationError.Error()},
		}); err != nil {
			return err
		}
		return synchronizationError
	}

	if logger != nil {
		if err := logger.Log(LogEntry{
			Severity: LogSeverityInfo,
			Event:    LogEventSynchronizationCompleted,
			Message:  "synchronization completed successfully",
		}); err != nil {
			return err
		}
	}
	return nil
}

func performSynchronization(
	roots RootPaths,
	store confirmedStateStore,
	input io.Reader,
	output io.Writer,
	logger *LoggingCoordinator,
) error {
	if err := scanRootsForUnsupportedEntries(roots); err != nil {
		return err
	}
	if err := ensureRootDirectories(roots); err != nil {
		return err
	}

	firstInventory, err := buildDirectoryInventory(roots.First)
	if err != nil {
		return err
	}
	secondInventory, err := buildDirectoryInventory(roots.Second)
	if err != nil {
		return err
	}
	confirmedState, confirmedStateFound, err := store.load(roots)
	if err != nil {
		return err
	}

	comparisons := compareDirectoryInventories(firstInventory, secondInventory)
	attachConfirmedEntries(comparisons, confirmedState, confirmedStateFound)
	if logger != nil {
		if err := logSynchronizationConflicts(logger, comparisons); err != nil {
			return err
		}
	}

	var plan SynchronizationPlan
	if !confirmedStateFound {
		recovery, err := recoverFromMissingConfirmedState(
			false,
			firstInventory,
			secondInventory,
			func() (SynchronizationSide, error) {
				return requestSynchronizationSide(input, output, "Choose the directory that should prevail (1 or 2):", nil)
			},
		)
		if err != nil {
			return err
		}
		if recovery.RequiresAuthoritativeSide {
			if logger != nil {
				if err := logger.Log(LogEntry{
					Severity: LogSeverityWarn,
					Event:    LogEventWarningRaised,
					Message:  "confirmed synchronization state is unavailable; selected side will prevail",
				}); err != nil {
					return err
				}
			}
			plan = generateAuthoritativeSynchronizationPlan(comparisons, recovery.AuthoritativeSide)
		} else {
			plan, err = generateSynchronizationPlan(comparisons, nil, nil, nil)
			if err != nil {
				return err
			}
		}
	} else {
		plan, err = generateSynchronizationPlan(
			comparisons,
			directoryContentsByPath(firstInventory, secondInventory),
			func(comparison EntryComparison) (SynchronizationSide, error) {
				return requestSynchronizationSide(input, output, "Choose the file version that should prevail (1 or 2):", nil)
			},
			func(comparison EntryComparison, contents []string) (SynchronizationSide, error) {
				return requestSynchronizationSide(input, output, "Choose the item that should prevail (1 or 2):", contents)
			},
		)
		if err != nil {
			return err
		}
	}

	result := executeSynchronizationPlan(roots, plan, SynchronizationExecutionOptions{
		Notify: func(message string) {
			fmt.Fprintln(output, message)
		},
		Logger: logger,
	})
	if !result.Completed {
		return result.Failure
	}

	return verifyAndCommitSynchronization(roots, store, result)
}

func attachConfirmedEntries(comparisons []EntryComparison, state ConfirmedSynchronizationState, stateFound bool) {
	if !stateFound {
		return
	}

	confirmedEntries := make(map[string]SynchronizationEntry, len(state.Entries))
	for _, entry := range state.Entries {
		confirmedEntries[entry.RelativePath] = entry
	}
	for index := range comparisons {
		entry, found := confirmedEntries[comparisons[index].RelativePath]
		if found {
			comparisons[index].Confirmed = &entry
		}
	}
}

func generateAuthoritativeSynchronizationPlan(comparisons []EntryComparison, authoritativeSide SynchronizationSide) SynchronizationPlan {
	actions := make([]SynchronizationAction, 0)
	for _, comparison := range comparisons {
		var authoritativeEntry, otherEntry *SynchronizationEntry
		if authoritativeSide == SynchronizationSideFirst {
			authoritativeEntry = comparison.First
			otherEntry = comparison.Second
		} else {
			authoritativeEntry = comparison.Second
			otherEntry = comparison.First
		}

		if authoritativeEntry == nil && otherEntry != nil {
			actions = append(actions, deleteActionForSide(comparison.RelativePath, oppositeSynchronizationSide(authoritativeSide)))
			continue
		}
		if authoritativeEntry != nil && (otherEntry == nil || !synchronizationEntriesHaveEquivalentContents(authoritativeEntry, otherEntry)) {
			actions = append(actions, copyActionForSide(comparison.RelativePath, authoritativeSide))
		}
	}

	return SynchronizationPlan{Actions: actions}
}

func synchronizationEntriesHaveEquivalentContents(firstEntry, secondEntry *SynchronizationEntry) bool {
	if firstEntry == nil || secondEntry == nil || firstEntry.Kind != secondEntry.Kind {
		return false
	}
	return firstEntry.Kind == EntryKindDirectory || firstEntry.ContentDigest == secondEntry.ContentDigest
}

func copyActionForSide(relativePath string, sourceSide SynchronizationSide) SynchronizationAction {
	if sourceSide == SynchronizationSideFirst {
		return SynchronizationAction{Kind: SynchronizationActionCopyToSecond, RelativePath: relativePath}
	}
	return SynchronizationAction{Kind: SynchronizationActionCopyToFirst, RelativePath: relativePath}
}

func deleteActionForSide(relativePath string, side SynchronizationSide) SynchronizationAction {
	if side == SynchronizationSideFirst {
		return SynchronizationAction{Kind: SynchronizationActionDeleteFromFirst, RelativePath: relativePath}
	}
	return SynchronizationAction{Kind: SynchronizationActionDeleteFromSecond, RelativePath: relativePath}
}

func directoryContentsByPath(firstInventory, secondInventory DirectoryInventory) map[string][]string {
	contents := make(map[string][]string)
	for _, inventory := range []DirectoryInventory{firstInventory, secondInventory} {
		for path, entry := range inventory {
			if entry.Kind != EntryKindDirectory {
				continue
			}
			for childPath := range inventory {
				if childPath != path && filepath.Dir(childPath) == path {
					contents[path] = append(contents[path], childPath)
				}
			}
		}
	}
	for path := range contents {
		sort.Strings(contents[path])
	}
	return contents
}

func requestSynchronizationSide(input io.Reader, output io.Writer, prompt string, directoryContents []string) (SynchronizationSide, error) {
	if input == nil || output == nil {
		return SynchronizationSideFirst, errors.New("synchronization decision streams are required")
	}
	fmt.Fprintln(output, prompt)
	if len(directoryContents) > 0 {
		fmt.Fprintln(output, "Directory contents:")
		for _, content := range directoryContents {
			fmt.Fprintln(output, content)
		}
	}

	var choice int
	if _, err := fmt.Fscan(input, &choice); err != nil {
		return SynchronizationSideFirst, fmt.Errorf("read synchronization decision: %w", err)
	}
	if choice != 1 && choice != 2 {
		return SynchronizationSideFirst, fmt.Errorf("synchronization decision must be 1 or 2")
	}
	if choice == 1 {
		return SynchronizationSideFirst, nil
	}
	return SynchronizationSideSecond, nil
}

func oppositeSynchronizationSide(side SynchronizationSide) SynchronizationSide {
	if side == SynchronizationSideFirst {
		return SynchronizationSideSecond
	}
	return SynchronizationSideFirst
}
