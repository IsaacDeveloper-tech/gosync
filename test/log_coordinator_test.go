package gosync_test

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRotatingFileDestinationSurfacesEveryPersistentFailure(t *testing.T) {
	testCases := []struct {
		name           string
		prepare        func(t *testing.T, directory string, activeFilePath string)
		logDirectory   func(directory string) string
		formattedEntry string
	}{
		{
			name: "creation",
			prepare: func(t *testing.T, directory string, activeFilePath string) {
				blockedPath := filepath.Join(directory, "blocked")
				if err := os.WriteFile(blockedPath, []byte("file"), 0o600); err != nil {
					t.Fatalf("WriteFile(blocked path) error = %v", err)
				}
			},
			logDirectory: func(directory string) string {
				return filepath.Join(directory, "blocked", "logs")
			},
			formattedEntry: "creation failure\n",
		},
		{
			name: "opening",
			prepare: func(t *testing.T, directory string, activeFilePath string) {
				if err := os.Mkdir(activeFilePath, 0o700); err != nil {
					t.Fatalf("Mkdir(active path) error = %v", err)
				}
			},
			logDirectory:   func(directory string) string { return directory },
			formattedEntry: "opening failure\n",
		},
		{
			name: "append",
			prepare: func(t *testing.T, directory string, activeFilePath string) {
				if err := os.WriteFile(activeFilePath, []byte("existing"), 0o600); err != nil {
					t.Fatalf("WriteFile(active path) error = %v", err)
				}
				if err := os.Chmod(activeFilePath, 0o400); err != nil {
					t.Fatalf("Chmod(active path) error = %v", err)
				}
			},
			logDirectory:   func(directory string) string { return directory },
			formattedEntry: "append failure\n",
		},
		{
			name: "rotation",
			prepare: func(t *testing.T, directory string, activeFilePath string) {
				createFullActiveLogForTest(t, activeFilePath)
				createBlockingArchiveForTest(t, activeFilePath)
			},
			logDirectory:   func(directory string) string { return directory },
			formattedEntry: "rotation failure\n",
		},
		{
			name: "retention",
			prepare: func(t *testing.T, directory string, activeFilePath string) {
				createFullActiveLogForTest(t, activeFilePath)
				createBlockingArchiveForTest(t, activeFilePath)
			},
			logDirectory:   func(directory string) string { return directory },
			formattedEntry: "retention failure\n",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			directory := t.TempDir()
			logDirectory := testCase.logDirectory(directory)
			testCase.prepare(t, directory, filepath.Join(logDirectory, "gosync.log"))

			destination := newRotatingFileDestination(logDirectory)
			if err := destination.Write(testCase.formattedEntry); err == nil {
				t.Fatal("Write() error = nil, want persistent logging failure")
			}
		})
	}
}

func TestLoggingCoordinatorWritesOneSanitizedRecordToBothHealthyDestinations(t *testing.T) {
	var consoleOutput bytes.Buffer
	sanitizer := newLogSanitizer()
	sanitizer.RegisterProtectedValue("secret-token")
	coordinator := newLoggingCoordinatorForTest(t, LoggingCoordinatorOptions{
		ConsoleWriter: &consoleOutput,
		FileDirectory: t.TempDir(),
		Clock:         fixedLogClock,
		Sanitizer:     sanitizer,
	})

	if err := coordinator.Log(LogEntry{
		Severity: LogSeverityInfo,
		Event:    LogEventChangeSynchronized,
		Message:  "copied with secret-token",
		Context:  map[string]string{"path": "/sync/notes.txt"},
	}); err != nil {
		t.Fatalf("Log() error = %v", err)
	}

	fileOutput, err := os.ReadFile(coordinator.ActiveFilePath())
	if err != nil {
		t.Fatalf("ReadFile(active log) error = %v", err)
	}
	if consoleOutput.String() != string(fileOutput) {
		t.Fatalf("console output = %q, file output = %q, want equivalent records", consoleOutput.String(), fileOutput)
	}
	if strings.Contains(consoleOutput.String(), "secret-token") {
		t.Fatalf("console output = %q, contains protected value", consoleOutput.String())
	}
}

func TestLoggingCoordinatorDisablesPersistentDestinationAfterFailure(t *testing.T) {
	temporaryDirectory := t.TempDir()
	blockedDirectory := filepath.Join(temporaryDirectory, "blocked")
	if err := os.WriteFile(blockedDirectory, []byte("file"), 0o600); err != nil {
		t.Fatalf("WriteFile(blocked directory) error = %v", err)
	}
	fileDestination := newRotatingFileDestination(filepath.Join(blockedDirectory, "logs"))
	var consoleOutput bytes.Buffer
	coordinator := newLoggingCoordinatorForTest(t, LoggingCoordinatorOptions{
		ConsoleWriter:   &consoleOutput,
		FileDestination: &fileDestination,
		Clock:           fixedLogClock,
	})

	if err := coordinator.Log(LogEntry{Severity: LogSeverityInfo, Event: LogEventSynchronizationStarted, Message: "first"}); err != nil {
		t.Fatalf("first Log() error = %v, want console fallback", err)
	}
	if coordinator.PersistentAvailable() {
		t.Fatal("persistent destination remains available after failure")
	}
	if err := coordinator.Log(LogEntry{Severity: LogSeverityInfo, Event: LogEventSynchronizationCompleted, Message: "second"}); err != nil {
		t.Fatalf("second Log() error = %v, want console-only continuation", err)
	}
	if !strings.Contains(consoleOutput.String(), "event=persistent_destination_failure") {
		t.Fatalf("console output = %q, want persistent failure record", consoleOutput.String())
	}
}

func TestLoggingCoordinatorRecordsConsoleFailureInFileAndContinues(t *testing.T) {
	consoleWriter := &failOnFirstWriteWriter{err: errors.New("console unavailable")}
	coordinator := newLoggingCoordinatorForTest(t, LoggingCoordinatorOptions{
		ConsoleWriter: consoleWriter,
		FileDirectory: t.TempDir(),
		Clock:         fixedLogClock,
	})

	if err := coordinator.Log(LogEntry{Severity: LogSeverityInfo, Event: LogEventSynchronizationStarted, Message: "first"}); err != nil {
		t.Fatalf("first Log() error = %v, want file fallback", err)
	}
	if err := coordinator.Log(LogEntry{Severity: LogSeverityInfo, Event: LogEventSynchronizationCompleted, Message: "second"}); err != nil {
		t.Fatalf("second Log() error = %v, want file-only continuation", err)
	}

	fileOutput, err := os.ReadFile(coordinator.ActiveFilePath())
	if err != nil {
		t.Fatalf("ReadFile(active log) error = %v", err)
	}
	if !strings.Contains(string(fileOutput), "event=console_destination_failure") || !strings.Contains(string(fileOutput), "message=\"first\"") || !strings.Contains(string(fileOutput), "message=\"second\"") {
		t.Fatalf("file output = %q, want failure and both records", fileOutput)
	}
	if coordinator.ConsoleAvailable() {
		t.Fatal("console destination remains available after failure")
	}
}

func TestLoggingCoordinatorReturnsFatalErrorWhenBothDestinationsFail(t *testing.T) {
	temporaryDirectory := t.TempDir()
	blockedDirectory := filepath.Join(temporaryDirectory, "blocked")
	if err := os.WriteFile(blockedDirectory, []byte("file"), 0o600); err != nil {
		t.Fatalf("WriteFile(blocked directory) error = %v", err)
	}
	fileDestination := newRotatingFileDestination(filepath.Join(blockedDirectory, "logs"))
	coordinator := newLoggingCoordinatorForTest(t, LoggingCoordinatorOptions{
		ConsoleWriter:   failOnFirstWriteWriter{err: errors.New("console unavailable")},
		FileDestination: &fileDestination,
		Clock:           fixedLogClock,
	})

	err := coordinator.Log(LogEntry{Severity: LogSeverityError, Event: LogEventOperationFailed, Message: "failed"})
	if !errors.Is(err, errLoggingUnavailable) {
		t.Fatalf("Log() error = %v, want logging unavailable error", err)
	}
	if coordinator.ConsoleAvailable() || coordinator.PersistentAvailable() {
		t.Fatal("a failed destination remains available")
	}
}

func TestInitializeLoggingCoordinatorEnablesHealthyDestinations(t *testing.T) {
	var consoleOutput bytes.Buffer
	coordinator, err := initializeLoggingCoordinator(RootPaths{
		First:  filepath.Join(t.TempDir(), "first"),
		Second: filepath.Join(t.TempDir(), "second"),
	}, LoggingCoordinatorOptions{
		ConsoleWriter: &consoleOutput,
		FileDirectory: t.TempDir(),
		Clock:         fixedLogClock,
	})
	if err != nil {
		t.Fatalf("InitializeLoggingCoordinator() error = %v", err)
	}
	if !coordinator.ConsoleAvailable() || !coordinator.PersistentAvailable() {
		t.Fatal("healthy initialization did not enable both destinations")
	}
}

func TestInitializeLoggingCoordinatorUsesConsoleOnlyWhenLogLocationOverlapsRoot(t *testing.T) {
	firstRoot := t.TempDir()
	var consoleOutput bytes.Buffer
	coordinator, err := initializeLoggingCoordinator(RootPaths{
		First:  firstRoot,
		Second: filepath.Join(t.TempDir(), "second"),
	}, LoggingCoordinatorOptions{
		ConsoleWriter: &consoleOutput,
		FileDirectory: filepath.Join(firstRoot, "logs"),
		Clock:         fixedLogClock,
	})
	if err != nil {
		t.Fatalf("InitializeLoggingCoordinator() error = %v", err)
	}
	if !coordinator.ConsoleAvailable() || coordinator.PersistentAvailable() {
		t.Fatal("overlap initialization did not enable console-only mode")
	}
	if !strings.Contains(consoleOutput.String(), "event=warning_raised") {
		t.Fatalf("console output = %q, want overlap warning", consoleOutput.String())
	}
}

func TestInitializeLoggingCoordinatorFailsWhenNoDestinationCanBeInitialized(t *testing.T) {
	temporaryDirectory := t.TempDir()
	blockedDirectory := filepath.Join(temporaryDirectory, "blocked")
	if err := os.WriteFile(blockedDirectory, []byte("file"), 0o600); err != nil {
		t.Fatalf("WriteFile(blocked directory) error = %v", err)
	}

	_, err := initializeLoggingCoordinator(RootPaths{
		First:  filepath.Join(temporaryDirectory, "first"),
		Second: filepath.Join(temporaryDirectory, "second"),
	}, LoggingCoordinatorOptions{
		FileDirectory: filepath.Join(blockedDirectory, "logs"),
		Clock:         fixedLogClock,
	})
	if !errors.Is(err, errLoggingUnavailable) {
		t.Fatalf("InitializeLoggingCoordinator() error = %v, want logging unavailable", err)
	}
}

func TestSynchronizeDirectoriesLogsLifecycleEvents(t *testing.T) {
	temporaryDirectory := t.TempDir()
	roots := createEmptySynchronizationRoots(t, temporaryDirectory)
	store := newConfirmedStateStoreAt(filepath.Join(temporaryDirectory, "state"))
	if err := store.Save(ConfirmedSynchronizationState{Version: 1, Roots: roots}, SynchronizationResult{Completed: true}); err != nil {
		t.Fatalf("Save(empty state) error = %v", err)
	}
	var consoleOutput bytes.Buffer
	coordinator := newLoggingCoordinatorForTest(t, LoggingCoordinatorOptions{
		ConsoleWriter: &consoleOutput,
		FileDirectory: filepath.Join(temporaryDirectory, "logs"),
		Clock:         fixedLogClock,
	})

	if err := synchronizeDirectoriesWithLogger(roots, store, bytes.NewBuffer(nil), &bytes.Buffer{}, coordinator); err != nil {
		t.Fatalf("synchronizeDirectoriesWithLogger() error = %v", err)
	}
	if !strings.Contains(consoleOutput.String(), "event=synchronization_started") || !strings.Contains(consoleOutput.String(), "event=synchronization_completed") {
		t.Fatalf("console output = %q, want lifecycle events", consoleOutput.String())
	}
}

func TestSynchronizationWithLoggerWritesEquivalentRecordsToConsoleAndFile(t *testing.T) {
	temporaryDirectory := t.TempDir()
	roots := createEmptySynchronizationRoots(t, temporaryDirectory)
	store := newConfirmedStateStoreAt(filepath.Join(temporaryDirectory, "state"))
	if err := store.Save(ConfirmedSynchronizationState{Version: 1, Roots: roots}, SynchronizationResult{Completed: true}); err != nil {
		t.Fatalf("Save(empty state) error = %v", err)
	}
	var consoleOutput bytes.Buffer
	coordinator := newLoggingCoordinatorForTest(t, LoggingCoordinatorOptions{
		ConsoleWriter: &consoleOutput,
		FileDirectory: filepath.Join(temporaryDirectory, "logs"),
		Clock:         fixedLogClock,
	})

	if err := synchronizeDirectoriesWithLogger(roots, store, bytes.NewBuffer(nil), &bytes.Buffer{}, coordinator); err != nil {
		t.Fatalf("synchronizeDirectoriesWithLogger() error = %v", err)
	}
	fileOutput, err := os.ReadFile(coordinator.ActiveFilePath())
	if err != nil {
		t.Fatalf("ReadFile(active log) error = %v", err)
	}
	if consoleOutput.String() != string(fileOutput) {
		t.Fatalf("console output = %q, file output = %q, want equivalent records", consoleOutput.String(), fileOutput)
	}
	if !strings.Contains(consoleOutput.String(), "event=synchronization_started") || !strings.Contains(consoleOutput.String(), "event=synchronization_completed") {
		t.Fatalf("synchronized output = %q, want lifecycle records", consoleOutput.String())
	}
}

func TestSynchronizeDirectoriesLogsChangesForCopiesAndDeletions(t *testing.T) {
	temporaryDirectory := t.TempDir()
	roots := createEmptySynchronizationRoots(t, temporaryDirectory)
	firstFilePath := filepath.Join(roots.First, "notes.txt")
	if err := os.WriteFile(firstFilePath, []byte("notes"), 0o600); err != nil {
		t.Fatalf("WriteFile(first file) error = %v", err)
	}
	store := newConfirmedStateStoreAt(filepath.Join(temporaryDirectory, "state"))
	if err := store.Save(ConfirmedSynchronizationState{Version: 1, Roots: roots}, SynchronizationResult{Completed: true}); err != nil {
		t.Fatalf("Save(empty state) error = %v", err)
	}
	var consoleOutput bytes.Buffer
	coordinator := newLoggingCoordinatorForTest(t, LoggingCoordinatorOptions{
		ConsoleWriter: &consoleOutput,
		FileDirectory: filepath.Join(temporaryDirectory, "logs"),
		Clock:         fixedLogClock,
	})

	if err := synchronizeDirectoriesWithLogger(roots, store, bytes.NewBuffer(nil), &bytes.Buffer{}, coordinator); err != nil {
		t.Fatalf("copy synchronization error = %v", err)
	}
	if err := os.Remove(firstFilePath); err != nil {
		t.Fatalf("Remove(first file) error = %v", err)
	}
	if err := synchronizeDirectoriesWithLogger(roots, store, bytes.NewBuffer(nil), &bytes.Buffer{}, coordinator); err != nil {
		t.Fatalf("deletion synchronization error = %v", err)
	}
	if countOccurrences(consoleOutput.String(), "event=change_synchronized") != 2 {
		t.Fatalf("change event count = %d, want copy and deletion events", countOccurrences(consoleOutput.String(), "event=change_synchronized"))
	}
	if !strings.Contains(consoleOutput.String(), "context.path=\"notes.txt\"") {
		t.Fatalf("console output = %q, want affected path context", consoleOutput.String())
	}
}

func TestSynchronizeDirectoriesLogsFileAndDirectoryConflicts(t *testing.T) {
	temporaryDirectory := t.TempDir()
	roots := createEmptySynchronizationRoots(t, temporaryDirectory)
	if err := os.WriteFile(filepath.Join(roots.First, "file.txt"), []byte("first"), 0o600); err != nil {
		t.Fatalf("WriteFile(first conflict file) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(roots.Second, "file.txt"), []byte("second"), 0o600); err != nil {
		t.Fatalf("WriteFile(second conflict file) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(roots.First, "item"), []byte("file"), 0o600); err != nil {
		t.Fatalf("WriteFile(file-directory file) error = %v", err)
	}
	if err := os.Mkdir(filepath.Join(roots.Second, "item"), 0o700); err != nil {
		t.Fatalf("Mkdir(file-directory directory) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(roots.Second, "item", "child.txt"), []byte("child"), 0o600); err != nil {
		t.Fatalf("WriteFile(directory child) error = %v", err)
	}
	modificationTime := time.Date(2026, time.September, 13, 12, 0, 0, 0, time.UTC)
	for _, path := range []string{filepath.Join(roots.First, "file.txt"), filepath.Join(roots.Second, "file.txt")} {
		if err := os.Chtimes(path, modificationTime, modificationTime); err != nil {
			t.Fatalf("Chtimes(%q) error = %v", path, err)
		}
	}
	store := newConfirmedStateStoreAt(filepath.Join(temporaryDirectory, "state"))
	if err := store.Save(ConfirmedSynchronizationState{Version: 1, Roots: roots}, SynchronizationResult{Completed: true}); err != nil {
		t.Fatalf("Save(conflict state) error = %v", err)
	}
	var consoleOutput bytes.Buffer
	coordinator := newLoggingCoordinatorForTest(t, LoggingCoordinatorOptions{
		ConsoleWriter: &consoleOutput,
		FileDirectory: filepath.Join(temporaryDirectory, "logs"),
		Clock:         fixedLogClock,
	})

	if err := synchronizeDirectoriesWithLogger(roots, store, bytes.NewBufferString("1\n2\n"), &bytes.Buffer{}, coordinator); err != nil {
		t.Fatalf("conflict synchronization error = %v", err)
	}
	if countOccurrences(consoleOutput.String(), "event=conflict_detected") != 2 {
		t.Fatalf("conflict event count = %d, want file and directory conflicts", countOccurrences(consoleOutput.String(), "event=conflict_detected"))
	}
}

func TestRetryLockedFileLogsRetryAndWarningEvents(t *testing.T) {
	var consoleOutput bytes.Buffer
	coordinator := newLoggingCoordinatorForTest(t, LoggingCoordinatorOptions{
		ConsoleWriter: &consoleOutput,
		FileDirectory: t.TempDir(),
		Clock:         fixedLogClock,
	})
	attempts := 0
	err := retryLockedFile("notes.txt", func() error {
		attempts++
		if attempts == 1 {
			return errors.New("locked")
		}
		return nil
	}, SynchronizationExecutionOptions{
		IsLockedError: func(err error) bool { return err.Error() == "locked" },
		Sleep:         func() {},
		Logger:        coordinator,
	})
	if err != nil {
		t.Fatalf("retryLockedFile() error = %v", err)
	}
	if !strings.Contains(consoleOutput.String(), "event=retry_started") || !strings.Contains(consoleOutput.String(), "event=retry_completed") || !strings.Contains(consoleOutput.String(), "event=warning_raised") {
		t.Fatalf("console output = %q, want retry and warning events", consoleOutput.String())
	}
}

func TestSynchronizeDirectoriesLogsSanitizedFailureWithoutChangingReturnedError(t *testing.T) {
	temporaryDirectory := t.TempDir()
	roots := RootPaths{
		First:  filepath.Join(temporaryDirectory, "first"),
		Second: filepath.Join(temporaryDirectory, "top-secret"),
	}
	if err := os.MkdirAll(roots.First, 0o700); err != nil {
		t.Fatalf("MkdirAll(first root) error = %v", err)
	}
	if err := os.WriteFile(roots.Second, []byte("root is a file"), 0o600); err != nil {
		t.Fatalf("WriteFile(second root) error = %v", err)
	}
	store := newConfirmedStateStoreAt(filepath.Join(temporaryDirectory, "state"))
	var consoleOutput bytes.Buffer
	sanitizer := newLogSanitizer()
	sanitizer.RegisterProtectedValue("top-secret")
	coordinator := newLoggingCoordinatorForTest(t, LoggingCoordinatorOptions{
		ConsoleWriter: &consoleOutput,
		FileDirectory: filepath.Join(temporaryDirectory, "logs"),
		Clock:         fixedLogClock,
		Sanitizer:     sanitizer,
	})

	err := synchronizeDirectoriesWithLogger(roots, store, bytes.NewBuffer(nil), &bytes.Buffer{}, coordinator)
	if err == nil || !strings.Contains(err.Error(), "top-secret") {
		t.Fatalf("returned error = %v, want original user-facing path context", err)
	}
	if strings.Contains(consoleOutput.String(), "top-secret") {
		t.Fatalf("console output = %q, contains protected value", consoleOutput.String())
	}
	if !strings.Contains(consoleOutput.String(), "event=operation_failed") {
		t.Fatalf("console output = %q, want sanitized operation failure", consoleOutput.String())
	}
}

func TestLoggingCoordinatorRotationFailureFallsBackToConsole(t *testing.T) {
	temporaryDirectory := t.TempDir()
	activeFilePath := filepath.Join(temporaryDirectory, "gosync.log")
	createFullActiveLogForTest(t, activeFilePath)
	createBlockingArchiveForTest(t, activeFilePath)
	var consoleOutput bytes.Buffer
	fileDestination := newRotatingFileDestination(temporaryDirectory)
	coordinator := newLoggingCoordinatorForTest(t, LoggingCoordinatorOptions{
		ConsoleWriter:   &consoleOutput,
		FileDestination: &fileDestination,
		Clock:           fixedLogClock,
	})

	if err := coordinator.Log(LogEntry{Severity: LogSeverityInfo, Event: LogEventSynchronizationStarted, Message: "after rotation failure"}); err != nil {
		t.Fatalf("Log() error = %v, want console fallback", err)
	}
	if coordinator.PersistentAvailable() {
		t.Fatal("persistent destination remains available after rotation failure")
	}
	if !strings.Contains(consoleOutput.String(), "after rotation failure") || !strings.Contains(consoleOutput.String(), "event=persistent_destination_failure") {
		t.Fatalf("console output = %q, want triggering record and failure report", consoleOutput.String())
	}
}

func TestProtectedDataNeverReachesEitherLoggingDestination(t *testing.T) {
	var consoleOutput bytes.Buffer
	sanitizer := newLogSanitizer()
	sanitizer.RegisterProtectedValue("credential-value")
	coordinator := newLoggingCoordinatorForTest(t, LoggingCoordinatorOptions{
		ConsoleWriter: &consoleOutput,
		FileDirectory: t.TempDir(),
		Clock:         fixedLogClock,
		Sanitizer:     sanitizer,
	})
	entry := LogEntry{
		Severity: LogSeverityError,
		Event:    LogEventOperationFailed,
		Message:  "copy failed with credential-value",
		Context: map[string]string{
			"path":          "/safe/credential-value/file.txt",
			"error":         "permission denied: credential-value",
			"file_contents": "raw file body must not appear",
		},
	}
	if err := coordinator.Log(entry); err != nil {
		t.Fatalf("Log() error = %v", err)
	}
	fileOutput, err := os.ReadFile(coordinator.ActiveFilePath())
	if err != nil {
		t.Fatalf("ReadFile(active log) error = %v", err)
	}
	for destinationName, output := range map[string]string{"console": consoleOutput.String(), "file": string(fileOutput)} {
		if strings.Contains(output, "credential-value") || strings.Contains(output, "raw file body must not appear") {
			t.Fatalf("%s output = %q, contains protected data", destinationName, output)
		}
		if !strings.Contains(output, "/safe/") || !strings.Contains(output, "permission denied") {
			t.Fatalf("%s output = %q, lost safe diagnostic context", destinationName, output)
		}
	}
}

func newLoggingCoordinatorForTest(t *testing.T, options LoggingCoordinatorOptions) *LoggingCoordinator {
	t.Helper()
	coordinator, err := newLoggingCoordinator(options)
	if err != nil {
		t.Fatalf("NewLoggingCoordinator() error = %v", err)
	}
	return coordinator
}

func fixedLogClock() time.Time {
	return time.Date(2026, time.September, 13, 14, 15, 16, 0, time.UTC)
}

func createEmptySynchronizationRoots(t *testing.T, temporaryDirectory string) RootPaths {
	t.Helper()
	roots := RootPaths{
		First:  filepath.Join(temporaryDirectory, "first"),
		Second: filepath.Join(temporaryDirectory, "second"),
	}
	for _, root := range []string{roots.First, roots.Second} {
		if err := os.MkdirAll(root, 0o700); err != nil {
			t.Fatalf("MkdirAll(%q) error = %v", root, err)
		}
	}
	return roots
}

func createFullActiveLogForTest(t *testing.T, activeFilePath string) {
	t.Helper()
	contents := make([]byte, 10_000_000-1)
	for index := range contents {
		contents[index] = 'x'
	}
	if err := os.WriteFile(activeFilePath, contents, 0o600); err != nil {
		t.Fatalf("WriteFile(full active log) error = %v", err)
	}
}

func createBlockingArchiveForTest(t *testing.T, activeFilePath string) {
	t.Helper()
	archivePath := activeFilePath + ".4"
	if err := os.Mkdir(archivePath, 0o700); err != nil {
		t.Fatalf("Mkdir(blocking archive) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(archivePath, "keep"), []byte("keep"), 0o600); err != nil {
		t.Fatalf("WriteFile(blocking archive child) error = %v", err)
	}
}

func countOccurrences(value, substring string) int {
	return strings.Count(value, substring)
}

type failOnFirstWriteWriter struct {
	err    error
	writes int
}

func (writer failOnFirstWriteWriter) Write([]byte) (int, error) {
	return 0, writer.err
}

var _ io.Writer = (*failOnFirstWriteWriter)(nil)
