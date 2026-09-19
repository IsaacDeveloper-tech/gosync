package gosync_test

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestValidateConfigurationPathRejectsRootsContainingConfigLocation(t *testing.T) {
	temporaryDirectory := t.TempDir()
	roots := RootPaths{
		First:  filepath.Join(temporaryDirectory, "first"),
		Second: filepath.Join(temporaryDirectory, "second"),
	}

	for _, configPath := range []string{
		roots.First,
		filepath.Join(roots.First, "config.json"),
		roots.Second,
		filepath.Join(roots.Second, "nested", "config.json"),
	} {
		if err := validateConfigurationPath(configPath, roots); err == nil {
			t.Fatalf("ValidateConfigurationPath(%q) error = nil, want overlap rejection", configPath)
		}
	}
	if err := validateConfigurationPath(filepath.Join(temporaryDirectory, "outside", "config.json"), roots); err != nil {
		t.Fatalf("ValidateConfigurationPath(outside) error = %v, want nil", err)
	}
}

func TestWatchStartsConfigurationForMissingFileBeforeSynchronization(t *testing.T) {
	temporaryDirectory := t.TempDir()
	roots := createEmptySynchronizationRoots(t, temporaryDirectory)
	configurationStore := newConfigurationStore(filepath.Join(temporaryDirectory, "application", "config.json"))
	stateStore := newConfirmedStateStoreAt(filepath.Join(temporaryDirectory, "state"))
	stop := make(chan struct{})
	var received ConfigurationSnapshot
	var synchronizationCalls int
	var output bytes.Buffer

	err := runWatchCommand([]string{"watch", roots.First, roots.Second}, WatchCommandOptions{
		Input:              strings.NewReader("60\nbidirectional\nyes\n"),
		Output:             &output,
		Stop:               stop,
		StateStore:         &stateStore,
		ConfigurationStore: &configurationStore,
		Logging: LoggingCoordinatorOptions{
			ConsoleWriter: &output,
			FileDirectory: filepath.Join(temporaryDirectory, "logs"),
			Clock:         fixedLogClock,
		},
		SynchronizeWithConfiguration: func(_ RootPaths, configuration ConfigurationSnapshot) error {
			synchronizationCalls++
			received = configuration
			close(stop)
			return nil
		},
	})
	if err != nil {
		t.Fatalf("runWatchCommand() error = %v", err)
	}
	if synchronizationCalls != 1 || received.SynchronizationIntervalSeconds != 60 {
		t.Fatalf("synchronization calls = %d, configuration = %+v, want one call after confirmed configuration", synchronizationCalls, received)
	}
	if result, err := configurationStore.Load(); err != nil || result.Status != ConfigurationLoadValid {
		t.Fatalf("configuration load after watch = %+v, error = %v, want valid saved configuration", result, err)
	}
}

func TestWatchRejectsInvalidConfigurationBeforeSynchronization(t *testing.T) {
	temporaryDirectory := t.TempDir()
	roots := createEmptySynchronizationRoots(t, temporaryDirectory)
	configurationPath := filepath.Join(temporaryDirectory, "config.json")
	if err := os.WriteFile(configurationPath, []byte("{}"), 0o600); err != nil {
		t.Fatalf("WriteFile(invalid config) error = %v", err)
	}
	configurationStore := newConfigurationStore(configurationPath)
	stateStore := newConfirmedStateStoreAt(filepath.Join(temporaryDirectory, "state"))
	synchronizationCalls := 0
	err := runWatchCommand([]string{"watch", roots.First, roots.Second}, WatchCommandOptions{
		Input:              strings.NewReader("60\nbidirectional\nyes\n"),
		Output:             io.Discard,
		StateStore:         &stateStore,
		ConfigurationStore: &configurationStore,
		SynchronizeWithConfiguration: func(RootPaths, ConfigurationSnapshot) error {
			synchronizationCalls++
			return nil
		},
		Logging: LoggingCoordinatorOptions{FileDirectory: filepath.Join(temporaryDirectory, "logs")},
	})
	if err == nil || !strings.Contains(err.Error(), "invalid") {
		t.Fatalf("runWatchCommand() error = %v, want invalid configuration error", err)
	}
	if synchronizationCalls != 0 {
		t.Fatalf("synchronization calls = %d, want 0", synchronizationCalls)
	}
}

func TestWatchUsesOneStartupConfigurationSnapshotUntilProcessEnds(t *testing.T) {
	temporaryDirectory := t.TempDir()
	roots := createEmptySynchronizationRoots(t, temporaryDirectory)
	configurationStore := newConfigurationStore(filepath.Join(temporaryDirectory, "config.json"))
	firstConfiguration := ConfigurationSnapshot{SchemaVersion: 1, SynchronizationIntervalSeconds: 1, SynchronizationMode: SynchronizationModeBidirectional}
	secondConfiguration := ConfigurationSnapshot{SchemaVersion: 1, SynchronizationIntervalSeconds: 2, SynchronizationMode: SynchronizationModeUnidirectional}
	if err := configurationStore.Save(firstConfiguration); err != nil {
		t.Fatalf("Save(first configuration) error = %v", err)
	}
	stateStore := newConfirmedStateStoreAt(filepath.Join(temporaryDirectory, "state"))
	stop := make(chan struct{})
	received := make([]ConfigurationSnapshot, 0, 1)
	err := runWatchCommand([]string{"watch", roots.First, roots.Second}, WatchCommandOptions{
		Output:             io.Discard,
		Stop:               stop,
		StateStore:         &stateStore,
		ConfigurationStore: &configurationStore,
		Logging:            LoggingCoordinatorOptions{FileDirectory: filepath.Join(temporaryDirectory, "logs")},
		Wait: func(time.Duration) {
			if err := configurationStore.Save(secondConfiguration); err != nil {
				t.Fatalf("Save(second configuration) error = %v", err)
			}
			close(stop)
		},
		SynchronizeWithConfiguration: func(_ RootPaths, configuration ConfigurationSnapshot) error {
			received = append(received, configuration)
			return nil
		},
	})
	if err != nil {
		t.Fatalf("runWatchCommand() error = %v", err)
	}
	if len(received) != 1 || received[0] != firstConfiguration {
		t.Fatalf("received configurations = %+v, want one startup snapshot %+v", received, firstConfiguration)
	}

	newStop := make(chan struct{})
	var nextConfiguration ConfigurationSnapshot
	err = runWatchCommand([]string{"watch", roots.First, roots.Second}, WatchCommandOptions{
		Output:             io.Discard,
		Stop:               newStop,
		StateStore:         &stateStore,
		ConfigurationStore: &configurationStore,
		Logging:            LoggingCoordinatorOptions{FileDirectory: filepath.Join(temporaryDirectory, "new-logs")},
		SynchronizeWithConfiguration: func(_ RootPaths, configuration ConfigurationSnapshot) error {
			nextConfiguration = configuration
			close(newStop)
			return nil
		},
	})
	if err != nil {
		t.Fatalf("second runWatchCommand() error = %v", err)
	}
	if nextConfiguration != secondConfiguration {
		t.Fatalf("next configuration = %+v, want updated snapshot %+v", nextConfiguration, secondConfiguration)
	}
}

func TestWatchRoutesConfiguredModeToSynchronizationCallback(t *testing.T) {
	temporaryDirectory := t.TempDir()
	roots := createEmptySynchronizationRoots(t, temporaryDirectory)
	configurationStore := newConfigurationStore(filepath.Join(temporaryDirectory, "config.json"))
	configuration := ConfigurationSnapshot{SchemaVersion: 1, SynchronizationIntervalSeconds: 1, SynchronizationMode: SynchronizationModeUnidirectional}
	if err := configurationStore.Save(configuration); err != nil {
		t.Fatalf("Save(configuration) error = %v", err)
	}
	stateStore := newConfirmedStateStoreAt(filepath.Join(temporaryDirectory, "state"))
	stop := make(chan struct{})
	var received ConfigurationSnapshot
	err := runWatchCommand([]string{"watch", roots.First, roots.Second}, WatchCommandOptions{
		Output:             io.Discard,
		Stop:               stop,
		StateStore:         &stateStore,
		ConfigurationStore: &configurationStore,
		Logging:            LoggingCoordinatorOptions{FileDirectory: filepath.Join(temporaryDirectory, "logs")},
		SynchronizeWithConfiguration: func(_ RootPaths, selected ConfigurationSnapshot) error {
			received = selected
			close(stop)
			return nil
		},
	})
	if err != nil {
		t.Fatalf("runWatchCommand() error = %v", err)
	}
	if received != configuration {
		t.Fatalf("selected configuration = %+v, want %v", received, configuration)
	}
}

func TestWatchSchedulerWaitsAfterCompletionWithoutOverlap(t *testing.T) {
	stop := make(chan struct{})
	waits := make([]time.Duration, 0, 2)
	active := false
	maxActive := 0
	calls := 0
	err := runWatchLoop(func() error {
		if active {
			t.Fatal("synchronization cycles overlapped")
		}
		active = true
		calls++
		if calls == 3 {
			close(stop)
		}
		if active {
			maxActive = 1
		}
		active = false
		return nil
	}, WatchLoopOptions{
		Interval: time.Second,
		Stop:     stop,
		Wait: func(interval time.Duration) {
			waits = append(waits, interval)
		},
	})
	if err != nil {
		t.Fatalf("runWatchLoop() error = %v", err)
	}
	if calls != 3 || maxActive != 1 || len(waits) != 2 || !reflect.DeepEqual(waits, []time.Duration{time.Second, time.Second}) {
		t.Fatalf("calls = %d, waits = %v, maxActive = %d, want immediate plus two delayed non-overlapping cycles", calls, waits, maxActive)
	}
}

func TestValidateUnidirectionalRootsRequiresExistingSourceAndSafeRoots(t *testing.T) {
	temporaryDirectory := t.TempDir()
	firstRoot := filepath.Join(temporaryDirectory, "first")
	secondRoot := filepath.Join(temporaryDirectory, "second")
	if err := os.Mkdir(firstRoot, 0o700); err != nil {
		t.Fatalf("Mkdir(first root) error = %v", err)
	}
	if err := validateUnidirectionalRoots(RootPaths{First: firstRoot, Second: secondRoot}); err != nil {
		t.Fatalf("ValidateUnidirectionalRoots(existing source) error = %v", err)
	}
	if err := validateUnidirectionalRoots(RootPaths{First: filepath.Join(temporaryDirectory, "missing"), Second: secondRoot}); err == nil {
		t.Fatal("ValidateUnidirectionalRoots(missing source) error = nil, want source error")
	}
	if err := validateUnidirectionalRoots(RootPaths{First: firstRoot, Second: firstRoot}); err == nil {
		t.Fatal("ValidateUnidirectionalRoots(equal roots) error = nil, want root safety error")
	}
	if err := validateUnidirectionalRoots(RootPaths{First: firstRoot, Second: filepath.Join(firstRoot, "nested")}); err == nil {
		t.Fatal("ValidateUnidirectionalRoots(nested roots) error = nil, want root safety error")
	}
}

func TestSynchronizeUnidirectionalRejectsMissingSourceWithoutChangingDestination(t *testing.T) {
	temporaryDirectory := t.TempDir()
	missingSource := filepath.Join(temporaryDirectory, "missing-source")
	destination := filepath.Join(temporaryDirectory, "destination")
	if err := os.Mkdir(destination, 0o700); err != nil {
		t.Fatalf("Mkdir(destination) error = %v", err)
	}
	destinationFile := filepath.Join(destination, "keep.txt")
	if err := os.WriteFile(destinationFile, []byte("keep"), 0o600); err != nil {
		t.Fatalf("WriteFile(destination file) error = %v", err)
	}
	store := newConfirmedStateStoreAt(filepath.Join(temporaryDirectory, "state"))
	err := synchronizeUnidirectional(RootPaths{First: missingSource, Second: destination}, store, SynchronizationExecutionOptions{})
	if err == nil {
		t.Fatal("SynchronizeUnidirectional() error = nil, want missing-source error")
	}
	assertFileContent(t, destinationFile, "keep")
}

func TestSynchronizeUnidirectionalCreatesMissingDestinationAfterSourceValidation(t *testing.T) {
	temporaryDirectory := t.TempDir()
	source := filepath.Join(temporaryDirectory, "source")
	destination := filepath.Join(temporaryDirectory, "destination")
	if err := os.Mkdir(source, 0o700); err != nil {
		t.Fatalf("Mkdir(source) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(source, "notes.txt"), []byte("notes"), 0o600); err != nil {
		t.Fatalf("WriteFile(source file) error = %v", err)
	}
	store := newConfirmedStateStoreAt(filepath.Join(temporaryDirectory, "state"))
	if err := synchronizeUnidirectional(RootPaths{First: source, Second: destination}, store, SynchronizationExecutionOptions{}); err != nil {
		t.Fatalf("SynchronizeUnidirectional() error = %v", err)
	}
	assertFileContent(t, filepath.Join(destination, "notes.txt"), "notes")
}

func TestGenerateUnidirectionalPlanCopiesSourceDifferencesOnly(t *testing.T) {
	comparisons := []EntryComparison{
		{RelativePath: "source-only.txt", First: testSynchronizationEntry("source-only.txt", EntryKindFile, "source")},
		{RelativePath: "different.txt", First: testSynchronizationEntry("different.txt", EntryKindFile, "new"), Second: testSynchronizationEntry("different.txt", EntryKindFile, "old")},
		{RelativePath: "same.txt", First: testSynchronizationEntry("same.txt", EntryKindFile, "same"), Second: testSynchronizationEntry("same.txt", EntryKindFile, "same")},
	}
	plan, err := generateUnidirectionalSynchronizationPlan(comparisons)
	if err != nil {
		t.Fatalf("GenerateUnidirectionalSynchronizationPlan() error = %v", err)
	}
	if len(plan.Actions) != 2 {
		t.Fatalf("actions = %+v, want source-only and differing copies", plan.Actions)
	}
	for _, action := range plan.Actions {
		if action.Kind != SynchronizationActionCopyToSecond {
			t.Fatalf("action = %+v, want destination copy only", action)
		}
	}
}

func TestGenerateUnidirectionalPlanDeletesDestinationOnlyEntriesWithoutChangingSource(t *testing.T) {
	source := testSynchronizationEntry("source.txt", EntryKindFile, "source")
	destinationOnlyFile := testSynchronizationEntry("old.txt", EntryKindFile, "old")
	destinationOnlyDirectory := testSynchronizationEntry("old-dir", EntryKindDirectory, "")
	comparisons := []EntryComparison{
		{RelativePath: "old.txt", Second: destinationOnlyFile},
		{RelativePath: "old-dir", Second: destinationOnlyDirectory},
		{RelativePath: "source.txt", First: source, Second: source},
	}
	plan, err := generateUnidirectionalSynchronizationPlan(comparisons)
	if err != nil {
		t.Fatalf("GenerateUnidirectionalSynchronizationPlan() error = %v", err)
	}
	if len(plan.Actions) != 2 {
		t.Fatalf("actions = %+v, want two destination deletions", plan.Actions)
	}
	for _, action := range plan.Actions {
		if action.Kind != SynchronizationActionDeleteFromSecond {
			t.Fatalf("action = %+v, want destination deletion only", action)
		}
	}
	if source.ContentDigest != "source" {
		t.Fatalf("source entry = %+v, was changed while planning", source)
	}
}

func testSynchronizationEntry(path string, kind EntryKind, digest string) *SynchronizationEntry {
	return &SynchronizationEntry{RelativePath: path, Kind: kind, ContentDigest: digest}
}

var _ = errors.Is
