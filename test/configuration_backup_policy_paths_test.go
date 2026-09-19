package gosync_test

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildBackupPolicySnapshotUsesBackupConfigurationAndRootRoles(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	destination := filepath.Join(root, "destination")
	if err := os.Mkdir(source, 0o700); err != nil {
		t.Fatalf("Mkdir(source) error = %v", err)
	}
	if err := os.Mkdir(destination, 0o700); err != nil {
		t.Fatalf("Mkdir(destination) error = %v", err)
	}

	snapshot, err := buildBackupPolicySnapshot(
		RootPaths{First: source, Second: destination},
		ConfigurationSnapshot{
			SchemaVersion:                  2,
			SynchronizationIntervalSeconds: 45,
			SynchronizationMode:            SynchronizationModeBackup,
			BackupRetentionCount:           7,
		},
	)
	if err != nil {
		t.Fatalf("buildBackupPolicySnapshot() error = %v", err)
	}
	canonicalSource, err := filepath.EvalSymlinks(source)
	if err != nil {
		t.Fatalf("EvalSymlinks(source) error = %v", err)
	}
	canonicalDestination, err := filepath.EvalSymlinks(destination)
	if err != nil {
		t.Fatalf("EvalSymlinks(destination) error = %v", err)
	}
	if snapshot.Source != canonicalSource || snapshot.Destination != canonicalDestination {
		t.Fatalf("snapshot roots = %+v, want source %q and destination %q", snapshot, canonicalSource, canonicalDestination)
	}
	if snapshot.SynchronizationIntervalSeconds != 45 || snapshot.BackupRetentionCount != 7 {
		t.Fatalf("snapshot policy = %+v, want interval 45 and retention 7", snapshot)
	}

	for _, mode := range []SynchronizationMode{SynchronizationModeBidirectional, SynchronizationModeUnidirectional} {
		if _, err := buildBackupPolicySnapshot(
			RootPaths{First: source, Second: destination},
			ConfigurationSnapshot{SchemaVersion: 2, SynchronizationIntervalSeconds: 45, SynchronizationMode: mode},
		); err == nil {
			t.Fatalf("buildBackupPolicySnapshot(%q) error = nil, want mode rejection", mode)
		}
	}
}

func TestDeriveBackupRootPathsCanonicalizesAliasesAndKeepsSourceFirst(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	destination := filepath.Join(root, "destination")
	if err := os.Mkdir(source, 0o700); err != nil {
		t.Fatalf("Mkdir(source) error = %v", err)
	}
	if err := os.Mkdir(destination, 0o700); err != nil {
		t.Fatalf("Mkdir(destination) error = %v", err)
	}

	canonical, err := deriveBackupRootPaths(filepath.Join(root, "source", "."), filepath.Join(root, "nested", "..", "destination"))
	if err != nil {
		t.Fatalf("deriveBackupRootPaths() error = %v", err)
	}
	expectedSource, err := filepath.EvalSymlinks(source)
	if err != nil {
		t.Fatalf("EvalSymlinks(source) error = %v", err)
	}
	expectedDestination, err := filepath.EvalSymlinks(destination)
	if err != nil {
		t.Fatalf("EvalSymlinks(destination) error = %v", err)
	}
	if canonical.Source != expectedSource || canonical.Destination != expectedDestination {
		t.Fatalf("canonical roots = %+v, want source %q and destination %q", canonical, expectedSource, expectedDestination)
	}

	alias := filepath.Join(root, "source-alias")
	if err := os.Symlink(source, alias); err != nil {
		t.Skipf("symbolic links unavailable: %v", err)
	}
	canonical, err = deriveBackupRootPaths(alias, destination)
	if err != nil {
		t.Fatalf("deriveBackupRootPaths(symlink alias) error = %v", err)
	}
	if canonical.Source != source {
		t.Fatalf("canonical source = %q, want resolved source %q", canonical.Source, source)
	}
}

func TestValidateBackupSourceRejectsInvalidRootsBeforeDestinationAccess(t *testing.T) {
	root := t.TempDir()
	validSource := filepath.Join(root, "source")
	if err := os.Mkdir(validSource, 0o700); err != nil {
		t.Fatalf("Mkdir(valid source) error = %v", err)
	}
	if err := validateBackupSource(validSource); err != nil {
		t.Fatalf("validateBackupSource(valid source) error = %v", err)
	}

	fileSource := filepath.Join(root, "source-file")
	if err := os.WriteFile(fileSource, []byte("file"), 0o600); err != nil {
		t.Fatalf("WriteFile(file source) error = %v", err)
	}
	for _, source := range []string{filepath.Join(root, "missing"), fileSource, "bad\x00source"} {
		if err := validateBackupSource(source); err == nil {
			t.Fatalf("validateBackupSource(%q) error = nil, want rejection", source)
		}
	}

	destination := filepath.Join(root, "destination")
	if _, err := validateBackupRootPaths(filepath.Join(root, "missing-source"), destination, filepath.Join(root, "config.json")); err == nil {
		t.Fatal("validateBackupRootPaths(missing source) error = nil, want source validation failure")
	}
	if _, err := os.Stat(destination); !os.IsNotExist(err) {
		t.Fatalf("destination stat error = %v, want destination untouched", err)
	}
}

func TestInspectBackupDestinationDistinguishesMissingDirectoryAndUnsupportedEntries(t *testing.T) {
	root := t.TempDir()
	missing := filepath.Join(root, "missing")
	status, err := inspectBackupDestination(missing)
	if err != nil || status != BackupDestinationMissing {
		t.Fatalf("inspect missing = %v, %v, want missing without error", status, err)
	}

	directory := filepath.Join(root, "directory")
	if err := os.Mkdir(directory, 0o700); err != nil {
		t.Fatalf("Mkdir(directory) error = %v", err)
	}
	status, err = inspectBackupDestination(directory)
	if err != nil || status != BackupDestinationDirectory {
		t.Fatalf("inspect directory = %v, %v, want directory without error", status, err)
	}

	file := filepath.Join(root, "file")
	if err := os.WriteFile(file, []byte("foreign"), 0o600); err != nil {
		t.Fatalf("WriteFile(file) error = %v", err)
	}
	if _, err := inspectBackupDestination(file); err == nil {
		t.Fatal("inspect file error = nil, want unsupported destination error")
	}

	link := filepath.Join(root, "link")
	if err := os.Symlink(directory, link); err != nil {
		t.Skipf("symbolic links unavailable: %v", err)
	}
	if _, err := inspectBackupDestination(link); err == nil {
		t.Fatal("inspect symlink error = nil, want unsupported destination error")
	}
}

func TestValidateBackupRootPathsRejectsEqualNestedAndConfigurationOverlap(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	destination := filepath.Join(root, "destination")
	if err := os.Mkdir(source, 0o700); err != nil {
		t.Fatalf("Mkdir(source) error = %v", err)
	}
	if err := os.Mkdir(destination, 0o700); err != nil {
		t.Fatalf("Mkdir(destination) error = %v", err)
	}
	validConfig := filepath.Join(root, "config", "config.json")

	if _, err := validateBackupRootPaths(source, destination, validConfig); err != nil {
		t.Fatalf("validateBackupRootPaths(valid) error = %v", err)
	}
	for name, testCase := range map[string]struct {
		source        string
		destination   string
		configuration string
	}{
		"equal roots": {
			source: source, destination: source, configuration: validConfig,
		},
		"destination nested in source": {
			source: source, destination: filepath.Join(source, "nested"), configuration: validConfig,
		},
		"source nested in destination": {
			source: filepath.Join(destination, "nested"), destination: destination, configuration: validConfig,
		},
		"configuration in source": {
			source: source, destination: destination, configuration: filepath.Join(source, "config.json"),
		},
		"configuration in destination": {
			source: source, destination: destination, configuration: filepath.Join(destination, "config.json"),
		},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := validateBackupRootPaths(testCase.source, testCase.destination, testCase.configuration); err == nil {
				t.Fatalf("validateBackupRootPaths(%+v) error = nil, want safety rejection", testCase)
			}
		})
	}
}

func TestBackupLogOverlapUsesConsoleOnlyAndPreservesFailureRules(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	destination := filepath.Join(root, "destination")
	if err := os.Mkdir(source, 0o700); err != nil {
		t.Fatalf("Mkdir(source) error = %v", err)
	}
	if err := os.Mkdir(destination, 0o700); err != nil {
		t.Fatalf("Mkdir(destination) error = %v", err)
	}
	roots := RootPaths{First: source, Second: destination}

	var console strings.Builder
	coordinator, err := initializeLoggingCoordinator(roots, LoggingCoordinatorOptions{
		ConsoleWriter: &console,
		FileDirectory: filepath.Join(source, "logs"),
	})
	if err != nil {
		t.Fatalf("initializeLoggingCoordinator(overlap) error = %v", err)
	}
	if !coordinator.ConsoleAvailable() || coordinator.PersistentAvailable() {
		t.Fatal("overlapping log coordinator did not switch to console-only mode")
	}
	if !strings.Contains(console.String(), "overlaps") {
		t.Fatalf("console warning = %q, want overlap warning", console.String())
	}

	if _, err := initializeLoggingCoordinator(roots, LoggingCoordinatorOptions{
		FileDirectory: filepath.Join(destination, "logs"),
	}); !errors.Is(err, errLoggingUnavailable) {
		t.Fatalf("initializeLoggingCoordinator(no destinations) error = %v, want logging unavailable", err)
	}

	if _, err := initializeLoggingCoordinator(roots, LoggingCoordinatorOptions{
		ConsoleWriter: io.Discard,
		FileDirectory: filepath.Join(root, "logs"),
	}); err != nil {
		t.Fatalf("initializeLoggingCoordinator(console and file) error = %v, want healthy destinations", err)
	}
}
