package gosync_test

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateUnidirectionalPlanResolvesContentAndTypeConflictsToSource(t *testing.T) {
	comparisons := []EntryComparison{
		{
			RelativePath: "content.txt",
			First:        testSynchronizationEntry("content.txt", EntryKindFile, "source-content"),
			Second:       testSynchronizationEntry("content.txt", EntryKindFile, "destination-content"),
		},
		{
			RelativePath: "type-conflict",
			First:        testSynchronizationEntry("type-conflict", EntryKindDirectory, ""),
			Second:       testSynchronizationEntry("type-conflict", EntryKindFile, "destination-file"),
		},
	}

	plan, err := generateUnidirectionalSynchronizationPlan(comparisons)
	if err != nil {
		t.Fatalf("GenerateUnidirectionalSynchronizationPlan() error = %v", err)
	}
	if len(plan.Actions) != 2 {
		t.Fatalf("actions = %+v, want two source-authoritative replacements", plan.Actions)
	}
	for _, action := range plan.Actions {
		if action.Kind != SynchronizationActionCopyToSecond {
			t.Fatalf("action = %+v, want copy to destination without a decision", action)
		}
	}
}

func TestUnidirectionalPreflightRejectsUnsupportedEntriesBeforeDestinationMutation(t *testing.T) {
	temporaryDirectory := t.TempDir()
	source := filepath.Join(temporaryDirectory, "source")
	destination := filepath.Join(temporaryDirectory, "destination")
	if err := os.MkdirAll(source, 0o700); err != nil {
		t.Fatalf("MkdirAll(source) error = %v", err)
	}
	if err := os.MkdirAll(destination, 0o700); err != nil {
		t.Fatalf("MkdirAll(destination) error = %v", err)
	}
	keepPath := filepath.Join(destination, "keep.txt")
	if err := os.WriteFile(keepPath, []byte("keep"), 0o600); err != nil {
		t.Fatalf("WriteFile(destination keep) error = %v", err)
	}
	symlinkPath := filepath.Join(source, "unsupported-link")
	if err := os.Symlink(keepPath, symlinkPath); err != nil {
		t.Skipf("symbolic links unavailable: %v", err)
	}
	store := newConfirmedStateStoreAt(filepath.Join(temporaryDirectory, "state"))
	if err := synchronizeUnidirectional(RootPaths{First: source, Second: destination}, store, SynchronizationExecutionOptions{}); err == nil {
		t.Fatal("SynchronizeUnidirectional() error = nil, want unsupported-entry error")
	}
	assertFileContent(t, keepPath, "keep")
}

func TestUnidirectionalFilesystemErrorDoesNotModifySourceOrConfirmedState(t *testing.T) {
	temporaryDirectory := t.TempDir()
	source := filepath.Join(temporaryDirectory, "source")
	destination := filepath.Join(temporaryDirectory, "destination-file")
	if err := os.Mkdir(source, 0o700); err != nil {
		t.Fatalf("Mkdir(source) error = %v", err)
	}
	sourcePath := filepath.Join(source, "source.txt")
	if err := os.WriteFile(sourcePath, []byte("source"), 0o600); err != nil {
		t.Fatalf("WriteFile(source file) error = %v", err)
	}
	if err := os.WriteFile(destination, []byte("destination is a file"), 0o600); err != nil {
		t.Fatalf("WriteFile(destination file) error = %v", err)
	}
	store := newConfirmedStateStoreAt(filepath.Join(temporaryDirectory, "state"))
	roots := RootPaths{First: source, Second: destination}
	previous := ConfirmedSynchronizationState{Version: 1, Roots: roots}
	if err := store.Save(previous, SynchronizationResult{Completed: true}); err != nil {
		t.Fatalf("Save(previous state) error = %v", err)
	}
	if err := synchronizeUnidirectional(roots, store, SynchronizationExecutionOptions{}); err == nil {
		t.Fatal("SynchronizeUnidirectional() error = nil, want destination filesystem error")
	}
	assertFileContent(t, sourcePath, "source")
	loaded, found, err := store.Load(roots)
	if err != nil || !found || len(loaded.Entries) != len(previous.Entries) {
		t.Fatalf("confirmed state after failure = %+v, found = %t, error = %v, want previous state", loaded, found, err)
	}
}

func TestUnidirectionalRetryUsesExistingLockedFileBehavior(t *testing.T) {
	lockedError := errors.New("destination locked")
	attempts := 0
	notifications := make([]string, 0, 2)
	err := retryLockedFile("destination.txt", func() error {
		attempts++
		if attempts == 1 {
			return lockedError
		}
		return nil
	}, SynchronizationExecutionOptions{
		IsLockedError: func(err error) bool { return errors.Is(err, lockedError) },
		Sleep:         func() {},
		Notify:        func(message string) { notifications = append(notifications, message) },
	})
	if err != nil || attempts != 2 || len(notifications) != 2 {
		t.Fatalf("retryLockedFile() error = %v, attempts = %d, notifications = %v, want retry completion", err, attempts, notifications)
	}
}

func TestUnidirectionalSuccessVerifiesExactMirrorAndCommitsConfirmedState(t *testing.T) {
	temporaryDirectory := t.TempDir()
	source := filepath.Join(temporaryDirectory, "source")
	destination := filepath.Join(temporaryDirectory, "destination")
	if err := os.MkdirAll(filepath.Join(source, "nested"), 0o700); err != nil {
		t.Fatalf("MkdirAll(source nested) error = %v", err)
	}
	if err := os.MkdirAll(destination, 0o700); err != nil {
		t.Fatalf("MkdirAll(destination) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(source, "nested", "source.txt"), []byte("source"), 0o600); err != nil {
		t.Fatalf("WriteFile(source) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(destination, "old.txt"), []byte("old"), 0o600); err != nil {
		t.Fatalf("WriteFile(destination old) error = %v", err)
	}
	store := newConfirmedStateStoreAt(filepath.Join(temporaryDirectory, "state"))
	roots := RootPaths{First: source, Second: destination}
	if err := synchronizeUnidirectional(roots, store, SynchronizationExecutionOptions{}); err != nil {
		t.Fatalf("SynchronizeUnidirectional() error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(destination, "old.txt")); !os.IsNotExist(err) {
		t.Fatalf("destination old file error = %v, want deleted", err)
	}
	assertFileContent(t, filepath.Join(destination, "nested", "source.txt"), "source")
	loaded, found, err := store.Load(roots)
	if err != nil || !found || len(loaded.Entries) == 0 {
		t.Fatalf("confirmed state = %+v, found = %t, error = %v, want committed mirror state", loaded, found, err)
	}
}

func TestConfiguredWatchRunsUnidirectionalSynchronizationEndToEnd(t *testing.T) {
	temporaryDirectory := t.TempDir()
	source := filepath.Join(temporaryDirectory, "source")
	destination := filepath.Join(temporaryDirectory, "destination")
	if err := os.Mkdir(source, 0o700); err != nil {
		t.Fatalf("Mkdir(source) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(source, "notes.txt"), []byte("notes"), 0o600); err != nil {
		t.Fatalf("WriteFile(source file) error = %v", err)
	}
	roots := RootPaths{First: source, Second: destination}
	configurationStore := newConfigurationStore(filepath.Join(temporaryDirectory, "config.json"))
	if err := configurationStore.Save(ConfigurationSnapshot{SchemaVersion: 1, SynchronizationIntervalSeconds: 1, SynchronizationMode: SynchronizationModeUnidirectional}); err != nil {
		t.Fatalf("Save(configuration) error = %v", err)
	}
	stateStore := newConfirmedStateStoreAt(filepath.Join(temporaryDirectory, "state"))
	stop := make(chan struct{})
	close(stop)
	if err := runWatchCommand([]string{"watch", roots.First, roots.Second}, WatchCommandOptions{
		Output:             io.Discard,
		Stop:               stop,
		StateStore:         &stateStore,
		ConfigurationStore: &configurationStore,
		Logging:            LoggingCoordinatorOptions{FileDirectory: filepath.Join(temporaryDirectory, "logs")},
	}); err != nil {
		t.Fatalf("runWatchCommand(unidirectional) error = %v", err)
	}
	assertFileContent(t, filepath.Join(destination, "notes.txt"), "notes")
}

func TestConfigureCommandIntegrationRejectsArgumentsAndSavesValidConfiguration(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	store := newConfigurationStore(configPath)
	if err := runConfigureCommand([]string{"configure", "extra"}, ConfigureCommandOptions{Store: &store, Input: strings.NewReader("60\nbidirectional\nyes\n"), Output: io.Discard}); err == nil {
		t.Fatal("RunConfigureCommand(extra argument) error = nil, want usage error")
	}
	if _, err := os.Stat(configPath); !os.IsNotExist(err) {
		t.Fatalf("config after invalid command error = %v, want no file", err)
	}
	if err := runConfigureCommand([]string{"configure"}, ConfigureCommandOptions{Store: &store, Input: strings.NewReader("0\n60\nBACKUP\n3\nyes\n"), Output: io.Discard}); err != nil {
		t.Fatalf("RunConfigureCommand(valid flow) error = %v", err)
	}
	loaded, err := store.Load()
	if err != nil || loaded.Status != ConfigurationLoadValid {
		t.Fatalf("configured file = %+v, error = %v, want valid", loaded, err)
	}
}

func TestWatchStartupIntegrationRejectsConfigurationRootOverlap(t *testing.T) {
	temporaryDirectory := t.TempDir()
	source := filepath.Join(temporaryDirectory, "source")
	if err := os.Mkdir(source, 0o700); err != nil {
		t.Fatalf("Mkdir(source) error = %v", err)
	}
	configurationStore := newConfigurationStore(filepath.Join(source, "config.json"))
	stateStore := newConfirmedStateStoreAt(filepath.Join(temporaryDirectory, "state"))
	if err := runWatchCommand([]string{"watch", source, filepath.Join(temporaryDirectory, "destination")}, WatchCommandOptions{
		Output:             io.Discard,
		StateStore:         &stateStore,
		ConfigurationStore: &configurationStore,
		SynchronizeWithConfiguration: func(RootPaths, ConfigurationSnapshot) error {
			t.Fatal("synchronization ran despite configuration-root overlap")
			return nil
		},
	}); err == nil {
		t.Fatal("runWatchCommand(overlap) error = nil, want root guard error")
	}
}

func TestUnidirectionalMirrorPreservesSourceOnChangesAndDeletions(t *testing.T) {
	temporaryDirectory := t.TempDir()
	source := filepath.Join(temporaryDirectory, "source")
	destination := filepath.Join(temporaryDirectory, "destination")
	if err := os.MkdirAll(source, 0o700); err != nil {
		t.Fatalf("MkdirAll(source) error = %v", err)
	}
	if err := os.MkdirAll(destination, 0o700); err != nil {
		t.Fatalf("MkdirAll(destination) error = %v", err)
	}
	sourcePath := filepath.Join(source, "same.txt")
	if err := os.WriteFile(sourcePath, []byte("source version"), 0o600); err != nil {
		t.Fatalf("WriteFile(source) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(destination, "same.txt"), []byte("destination version"), 0o600); err != nil {
		t.Fatalf("WriteFile(destination differing) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(destination, "destination-only.txt"), []byte("remove"), 0o600); err != nil {
		t.Fatalf("WriteFile(destination-only) error = %v", err)
	}
	sourceBefore, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatalf("ReadFile(source before) error = %v", err)
	}
	store := newConfirmedStateStoreAt(filepath.Join(temporaryDirectory, "state"))
	if err := synchronizeUnidirectional(RootPaths{First: source, Second: destination}, store, SynchronizationExecutionOptions{}); err != nil {
		t.Fatalf("SynchronizeUnidirectional() error = %v", err)
	}
	sourceAfter, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatalf("ReadFile(source after) error = %v", err)
	}
	if !bytes.Equal(sourceBefore, sourceAfter) {
		t.Fatalf("source changed from %q to %q", sourceBefore, sourceAfter)
	}
	assertFileContent(t, filepath.Join(destination, "same.txt"), "source version")
	if _, err := os.Stat(filepath.Join(destination, "destination-only.txt")); !os.IsNotExist(err) {
		t.Fatalf("destination-only path error = %v, want removed", err)
	}
}
