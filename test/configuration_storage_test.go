package gosync_test

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestInspectConfigurationFileDistinguishesMissingRegularAndUnsupportedEntries(t *testing.T) {
	temporaryDirectory := t.TempDir()
	missingPath := filepath.Join(temporaryDirectory, "missing", "config.json")
	status, err := inspectConfigurationFile(missingPath)
	if err != nil || status != ConfigurationFileMissing {
		t.Fatalf("inspect missing = %v, %v, want missing without error", status, err)
	}

	regularPath := filepath.Join(temporaryDirectory, "config.json")
	if err := os.WriteFile(regularPath, []byte("{}"), 0o600); err != nil {
		t.Fatalf("WriteFile(regular config) error = %v", err)
	}
	status, err = inspectConfigurationFile(regularPath)
	if err != nil || status != ConfigurationFileRegular {
		t.Fatalf("inspect regular = %v, %v, want regular without error", status, err)
	}

	directoryPath := filepath.Join(temporaryDirectory, "directory-config")
	if err := os.Mkdir(directoryPath, 0o700); err != nil {
		t.Fatalf("Mkdir(directory config) error = %v", err)
	}
	if _, err := inspectConfigurationFile(directoryPath); err == nil {
		t.Fatal("inspect directory error = nil, want unsupported-entry error")
	}

	symlinkPath := filepath.Join(temporaryDirectory, "symlink-config")
	if err := os.Symlink(regularPath, symlinkPath); err != nil {
		t.Skipf("symbolic links unavailable: %v", err)
	}
	if _, err := inspectConfigurationFile(symlinkPath); err == nil {
		t.Fatal("inspect symlink error = nil, want unsupported-entry error")
	}
}

func TestLoadConfigurationReturnsDistinctTypedOutcomes(t *testing.T) {
	temporaryDirectory := t.TempDir()
	validPath := filepath.Join(temporaryDirectory, "valid.json")
	validJSON := `{"schemaVersion":1,"synchronizationIntervalSeconds":60,"synchronizationMode":"bidirectional"}`
	if err := os.WriteFile(validPath, []byte(validJSON), 0o600); err != nil {
		t.Fatalf("WriteFile(valid config) error = %v", err)
	}
	invalidPath := filepath.Join(temporaryDirectory, "invalid.json")
	if err := os.WriteFile(invalidPath, []byte("{}"), 0o600); err != nil {
		t.Fatalf("WriteFile(invalid config) error = %v", err)
	}
	unsupportedPath := filepath.Join(temporaryDirectory, "unsupported")
	if err := os.Mkdir(unsupportedPath, 0o700); err != nil {
		t.Fatalf("Mkdir(unsupported config) error = %v", err)
	}

	testCases := []struct {
		name       string
		path       string
		wantStatus ConfigurationLoadStatus
		wantError  bool
	}{
		{name: "missing", path: filepath.Join(temporaryDirectory, "absent.json"), wantStatus: ConfigurationLoadMissing},
		{name: "valid", path: validPath, wantStatus: ConfigurationLoadValid},
		{name: "invalid", path: invalidPath, wantStatus: ConfigurationLoadInvalid, wantError: true},
		{name: "unsupported", path: unsupportedPath, wantStatus: ConfigurationLoadUnsupported, wantError: true},
		{name: "io failure", path: "bad\x00path", wantStatus: ConfigurationLoadIOFailure, wantError: true},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result, err := loadConfiguration(testCase.path)
			if result.Status != testCase.wantStatus {
				t.Fatalf("load status = %v, want %v", result.Status, testCase.wantStatus)
			}
			if (err != nil) != testCase.wantError {
				t.Fatalf("load error = %v, want error = %t", err, testCase.wantError)
			}
		})
	}
}

func TestConfigurationStoreWritesClosedCompleteCandidateAndPreservesActiveOnCandidateFailure(t *testing.T) {
	temporaryDirectory := t.TempDir()
	configPath := filepath.Join(temporaryDirectory, "config.json")
	store := newConfigurationStore(configPath)
	configuration := ConfigurationSnapshot{SchemaVersion: 1, SynchronizationIntervalSeconds: 60, SynchronizationMode: SynchronizationModeBidirectional}

	candidatePath, err := store.WriteCandidate(configuration)
	if err != nil {
		t.Fatalf("WriteCandidate() error = %v", err)
	}
	defer os.Remove(candidatePath)
	candidateContents, err := os.ReadFile(candidatePath)
	if err != nil {
		t.Fatalf("ReadFile(candidate) error = %v", err)
	}
	if string(candidateContents) != `{"schemaVersion":1,"synchronizationIntervalSeconds":60,"synchronizationMode":"bidirectional"}` {
		t.Fatalf("candidate = %q, want complete exact JSON", candidateContents)
	}

	if err := os.WriteFile(configPath, []byte("previous complete config"), 0o600); err != nil {
		t.Fatalf("WriteFile(previous config) error = %v", err)
	}
	blockedStore := newConfigurationStore(filepath.Join(configPath, "child", "config.json"))
	if _, err := blockedStore.WriteCandidate(configuration); err == nil {
		t.Fatal("WriteCandidate(blocked path) error = nil, want candidate creation error")
	}
	activeContents, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("ReadFile(previous config) error = %v", err)
	}
	if string(activeContents) != "previous complete config" {
		t.Fatalf("active config = %q, want previous complete config", activeContents)
	}
}

func TestConfigurationStoreReplacesConfigAtomicallyWithCompleteDocument(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	store := newConfigurationStore(configPath)
	first := ConfigurationSnapshot{SchemaVersion: 1, SynchronizationIntervalSeconds: 60, SynchronizationMode: SynchronizationModeBidirectional}
	second := ConfigurationSnapshot{SchemaVersion: 1, SynchronizationIntervalSeconds: 120, SynchronizationMode: SynchronizationModeUnidirectional}
	if err := store.Save(first); err != nil {
		t.Fatalf("Save(first) error = %v", err)
	}
	if err := store.Save(second); err != nil {
		t.Fatalf("Save(second) error = %v", err)
	}
	contents, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("ReadFile(config) error = %v", err)
	}
	want := `{"schemaVersion":1,"synchronizationIntervalSeconds":120,"synchronizationMode":"unidirectional"}`
	if string(contents) != want {
		t.Fatalf("config = %q, want complete new document %q", contents, want)
	}
}

func TestConfigurationStorePropagatesStorageFailures(t *testing.T) {
	temporaryDirectory := t.TempDir()
	blockedPath := filepath.Join(temporaryDirectory, "blocked")
	if err := os.WriteFile(blockedPath, []byte("file"), 0o600); err != nil {
		t.Fatalf("WriteFile(blocked path) error = %v", err)
	}
	store := newConfigurationStore(filepath.Join(blockedPath, "config.json"))
	configuration := ConfigurationSnapshot{SchemaVersion: 1, SynchronizationIntervalSeconds: 60, SynchronizationMode: SynchronizationModeBidirectional}
	if err := store.Save(configuration); err == nil {
		t.Fatal("Save(blocked path) error = nil, want storage failure")
	}
	if _, err := newConfigurationStore("bad\x00config").Load(); err == nil {
		t.Fatal("Load(invalid path) error = nil, want storage failure")
	}
}

func TestConfigurationOwnershipRejectsConcurrentOwnerAndReleasesAfterCompletion(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	owner, err := acquireConfigurationOwnership(configPath)
	if err != nil {
		t.Fatalf("AcquireConfigurationOwnership(first) error = %v", err)
	}
	defer owner.Release()
	if _, err := acquireConfigurationOwnership(configPath); !errors.Is(err, errConfigurationInUse) {
		t.Fatalf("AcquireConfigurationOwnership(second) error = %v, want in-use error", err)
	}
	if err := owner.Release(); err != nil {
		t.Fatalf("Release(first) error = %v", err)
	}
	secondOwner, err := acquireConfigurationOwnership(configPath)
	if err != nil {
		t.Fatalf("AcquireConfigurationOwnership(after release) error = %v", err)
	}
	if err := secondOwner.Release(); err != nil {
		t.Fatalf("Release(second) error = %v", err)
	}
}

func TestConfigurationServiceRunsConfirmedConfigureFlow(t *testing.T) {
	store := newConfigurationStore(filepath.Join(t.TempDir(), "config.json"))
	var output strings.Builder
	service := newConfigurationService(ConfigurationServiceOptions{
		Store:  store,
		Input:  strings.NewReader("60\nbidirectional\nyes\n"),
		Output: &output,
	})
	configuration, err := service.Configure()
	if err != nil {
		t.Fatalf("Configure() error = %v", err)
	}
	if configuration.SynchronizationIntervalSeconds != 60 || configuration.SynchronizationMode != SynchronizationModeBidirectional {
		t.Fatalf("configuration = %+v, want confirmed configuration", configuration)
	}
	loaded, err := store.Load()
	if err != nil || loaded.Status != ConfigurationLoadValid {
		t.Fatalf("stored configuration = %+v, error = %v, want valid", loaded, err)
	}
}

func TestConfigurationServiceRepairsInvalidJSONAfterConfirmation(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(configPath, []byte("invalid json"), 0o600); err != nil {
		t.Fatalf("WriteFile(invalid config) error = %v", err)
	}
	service := newConfigurationService(ConfigurationServiceOptions{
		Store:  newConfigurationStore(configPath),
		Input:  strings.NewReader("120\nunidirectional\nyes\n"),
		Output: io.Discard,
	})
	if _, err := service.Configure(); err != nil {
		t.Fatalf("Configure(repair) error = %v", err)
	}
	loaded, err := loadConfiguration(configPath)
	if err != nil || loaded.Status != ConfigurationLoadValid || loaded.Configuration.SynchronizationMode != SynchronizationModeUnidirectional {
		t.Fatalf("repaired configuration = %+v, error = %v, want valid unidirectional config", loaded, err)
	}
}

func TestConfigurationServiceCancellationPreservesExistingConfiguration(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	store := newConfigurationStore(configPath)
	previous := ConfigurationSnapshot{SchemaVersion: 1, SynchronizationIntervalSeconds: 60, SynchronizationMode: SynchronizationModeBidirectional}
	if err := store.Save(previous); err != nil {
		t.Fatalf("Save(previous) error = %v", err)
	}
	previousContents, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("ReadFile(previous) error = %v", err)
	}
	service := newConfigurationService(ConfigurationServiceOptions{
		Store:  store,
		Input:  strings.NewReader("120\nunidirectional\nno\n"),
		Output: io.Discard,
	})
	if _, err := service.Configure(); !errors.Is(err, errConfigurationCancelled) {
		t.Fatalf("Configure(rejected) error = %v, want cancellation", err)
	}
	currentContents, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("ReadFile(current) error = %v", err)
	}
	if string(currentContents) != string(previousContents) {
		t.Fatalf("configuration changed after cancellation: %q -> %q", previousContents, currentContents)
	}
}

func TestConfigurationServiceLogsLifecycleWithoutInteractiveData(t *testing.T) {
	testCases := []struct {
		name      string
		input     string
		wantEvent string
	}{
		{name: "success", input: "60\nbidirectional\nyes\n", wantEvent: "event=configuration_succeeded"},
		{name: "cancelled", input: "60\nbidirectional\nno\n", wantEvent: "event=configuration_cancelled"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			var logOutput strings.Builder
			logger := newLoggingCoordinatorForConfigurationTest(t, &logOutput)
			service := newConfigurationService(ConfigurationServiceOptions{
				Store:  newConfigurationStore(filepath.Join(t.TempDir(), "config.json")),
				Input:  strings.NewReader(testCase.input),
				Output: io.Discard,
				Logger: logger,
			})
			_, _ = service.Configure()
			if !strings.Contains(logOutput.String(), "event=configuration_started") || !strings.Contains(logOutput.String(), testCase.wantEvent) {
				t.Fatalf("log output = %q, want lifecycle events", logOutput.String())
			}
			if strings.Contains(logOutput.String(), "bidirectional") || strings.Contains(logOutput.String(), "60") || strings.Contains(logOutput.String(), "schemaVersion") {
				t.Fatalf("log output = %q, contains interactive or JSON data", logOutput.String())
			}
		})
	}

	var failureLog strings.Builder
	failureLogger := newLoggingCoordinatorForConfigurationTest(t, &failureLog)
	failureService := newConfigurationService(ConfigurationServiceOptions{
		Store:  newConfigurationStore("bad\x00config"),
		Input:  strings.NewReader("60\nbidirectional\nyes\n"),
		Output: io.Discard,
		Logger: failureLogger,
	})
	if _, err := failureService.Configure(); err == nil {
		t.Fatal("Configure(failure) error = nil, want storage failure")
	}
	if !strings.Contains(failureLog.String(), "event=configuration_failed") {
		t.Fatalf("failure log = %q, want configuration failure event", failureLog.String())
	}
}

func newLoggingCoordinatorForConfigurationTest(t *testing.T, output io.Writer) *LoggingCoordinator {
	t.Helper()
	coordinator, err := newLoggingCoordinator(LoggingCoordinatorOptions{
		ConsoleWriter: output,
		FileDirectory: t.TempDir(),
		Clock: func() time.Time {
			return time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)
		},
	})
	if err != nil {
		t.Fatalf("NewLoggingCoordinator() error = %v", err)
	}
	return coordinator
}
