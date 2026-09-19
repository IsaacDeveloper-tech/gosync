package gosync_test

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBackupOwnershipAcrossProcesses(t *testing.T) {
	if os.Getenv("GOSYNC_BACKUP_LOCK_HELPER") == "1" {
		lockPath := os.Getenv("GOSYNC_BACKUP_LOCK_PATH")
		signalPath := os.Getenv("GOSYNC_BACKUP_LOCK_SIGNAL")
		releasePath := os.Getenv("GOSYNC_BACKUP_LOCK_RELEASE")
		owner, err := acquireBackupDestinationOwnership(lockPath)
		if err != nil {
			os.Exit(2)
		}
		if err := os.WriteFile(signalPath, []byte("locked"), 0o600); err != nil {
			_ = owner.Release()
			os.Exit(3)
		}
		for {
			if _, err := os.Stat(releasePath); err == nil {
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
		if err := owner.Release(); err != nil {
			os.Exit(4)
		}
		return
	}

	root := t.TempDir()
	lockPath := filepath.Join(root, "backup.lock")
	signalPath := filepath.Join(root, "signal")
	releasePath := filepath.Join(root, "release")
	command := exec.Command(os.Args[0], "-test.run=TestBackupOwnershipAcrossProcesses", "-test.v")
	command.Env = append(os.Environ(),
		"GOSYNC_BACKUP_LOCK_HELPER=1",
		"GOSYNC_BACKUP_LOCK_PATH="+lockPath,
		"GOSYNC_BACKUP_LOCK_SIGNAL="+signalPath,
		"GOSYNC_BACKUP_LOCK_RELEASE="+releasePath,
	)
	if err := command.Start(); err != nil {
		t.Fatalf("start ownership helper error = %v", err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		if _, err := os.Stat(signalPath); err == nil {
			break
		}
		if time.Now().After(deadline) {
			_ = command.Process.Kill()
			t.Fatal("ownership helper did not acquire lock")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if _, err := acquireBackupDestinationOwnership(lockPath); err == nil {
		t.Fatal("second process ownership acquisition error = nil, want conflict")
	}
	if err := os.WriteFile(releasePath, []byte("release"), 0o600); err != nil {
		t.Fatalf("WriteFile(release) error = %v", err)
	}
	if err := command.Wait(); err != nil {
		t.Fatalf("ownership helper error = %v", err)
	}
	owner, err := acquireBackupDestinationOwnership(lockPath)
	if err != nil {
		t.Fatalf("ownership acquisition after helper exit error = %v", err)
	}
	if err := owner.Release(); err != nil {
		t.Fatalf("Release(after helper) error = %v", err)
	}
}

func TestBackupCycleCreatesOneConfirmedZIPAndPreservesSource(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	destination := filepath.Join(root, "destination")
	applicationData := filepath.Join(root, "application-data")
	if err := os.Mkdir(source, 0o700); err != nil {
		t.Fatalf("Mkdir(source) error = %v", err)
	}
	if err := os.Mkdir(applicationData, 0o700); err != nil {
		t.Fatalf("Mkdir(application data) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(source, "file.txt"), []byte("content"), 0o600); err != nil {
		t.Fatalf("WriteFile(source) error = %v", err)
	}
	sourceBefore := snapshotTreeBytes(t, source)
	policy, err := buildBackupPolicySnapshot(RootPaths{First: source, Second: destination}, ConfigurationSnapshot{SchemaVersion: 2, SynchronizationIntervalSeconds: 1, SynchronizationMode: SynchronizationModeBackup, BackupRetentionCount: 3})
	if err != nil {
		t.Fatalf("buildBackupPolicySnapshot() error = %v", err)
	}
	var logOutput strings.Builder
	logger := newLoggingCoordinatorForConfigurationTest(t, &logOutput)
	if err := runBackupCycle(policy, BackupCycleOptions{
		ApplicationDataDirectory: applicationData,
		Logger:                   logger,
		Clock:                    func() time.Time { return time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC) },
		ArchiveID:                func() (string, error) { return "cycle-1", nil },
	}); err != nil {
		t.Fatalf("runBackupCycle() error = %v", err)
	}
	entries, err := os.ReadDir(destination)
	if err != nil {
		t.Fatalf("ReadDir(destination) error = %v", err)
	}
	if len(entries) != 1 || filepath.Ext(entries[0].Name()) != ".zip" {
		t.Fatalf("destination entries = %+v, want exactly one ZIP", entries)
	}
	store, err := newBackupSetStoreAt(applicationData, destination)
	if err != nil {
		t.Fatalf("newBackupSetStoreAt() error = %v", err)
	}
	loaded, err := store.Load()
	if err != nil || loaded.Status != BackupSetLoadValid || len(loaded.State.Archives) != 1 || loaded.State.Archives[0].Lifecycle != BackupArchiveConfirmed {
		t.Fatalf("backup state = %+v, %v, want one confirmed archive", loaded, err)
	}
	if got := snapshotTreeBytes(t, source); got != sourceBefore {
		t.Fatalf("source changed after backup: before=%q after=%q", sourceBefore, got)
	}
	if !strings.Contains(logOutput.String(), "backup_cycle_started") || !strings.Contains(logOutput.String(), "backup_archive_verified") || !strings.Contains(logOutput.String(), "backup_archive_confirmed") {
		t.Fatalf("backup lifecycle logs = %q, want start, verification, and confirmation", logOutput.String())
	}
}

func TestBackupCycleValidatesBeforeMutationAndPreservesExistingDestinationOnFailure(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "missing-source")
	destination := filepath.Join(root, "destination")
	applicationData := filepath.Join(root, "application-data")
	for _, directory := range []string{destination, applicationData} {
		if err := os.Mkdir(directory, 0o700); err != nil {
			t.Fatalf("Mkdir(%q) error = %v", directory, err)
		}
	}
	foreignPath := filepath.Join(destination, "foreign.txt")
	if err := os.WriteFile(foreignPath, []byte("keep"), 0o600); err != nil {
		t.Fatalf("WriteFile(foreign) error = %v", err)
	}
	policy := BackupPolicySnapshot{Source: source, Destination: destination, SynchronizationIntervalSeconds: 1, BackupRetentionCount: 3}
	if err := runBackupCycle(policy, BackupCycleOptions{ApplicationDataDirectory: applicationData}); err == nil {
		t.Fatal("runBackupCycle(missing source) error = nil, want source failure")
	}
	if content, err := os.ReadFile(foreignPath); err != nil || string(content) != "keep" {
		t.Fatalf("foreign destination after source failure = %q, %v, want unchanged", content, err)
	}
	entries, err := os.ReadDir(destination)
	if err != nil || len(entries) != 1 || entries[0].Name() != "foreign.txt" {
		t.Fatalf("destination after source failure = %+v, %v, want no backup mutation", entries, err)
	}
}

func TestBackupWatchUsesStartupConfigurationOwnershipAndCompletionWait(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	destination := filepath.Join(root, "destination")
	applicationData := filepath.Join(root, "application-data")
	logs := filepath.Join(root, "logs")
	for _, directory := range []string{source, applicationData, logs} {
		if err := os.Mkdir(directory, 0o700); err != nil {
			t.Fatalf("Mkdir(%q) error = %v", directory, err)
		}
	}
	if err := os.WriteFile(filepath.Join(source, "file.txt"), []byte("content"), 0o600); err != nil {
		t.Fatalf("WriteFile(source) error = %v", err)
	}
	configurationStore := newConfigurationStore(filepath.Join(applicationData, "config.json"))
	if err := configurationStore.Save(ConfigurationSnapshot{SchemaVersion: 2, SynchronizationIntervalSeconds: 2, SynchronizationMode: SynchronizationModeBackup, BackupRetentionCount: 1}); err != nil {
		t.Fatalf("Save(configuration) error = %v", err)
	}
	confirmedStateStore := newConfirmedStateStoreAt(filepath.Join(root, "legacy-state"))
	stop := make(chan struct{})
	close(stop)
	if err := runWatchCommand([]string{"watch", source, destination}, WatchCommandOptions{
		Output:             io.Discard,
		Stop:               stop,
		StateStore:         &confirmedStateStore,
		ConfigurationStore: &configurationStore,
		Logging:            LoggingCoordinatorOptions{FileDirectory: logs},
	}); err != nil {
		t.Fatalf("runWatchCommand(backup immediate) error = %v", err)
	}
	entries, err := os.ReadDir(destination)
	if err != nil || len(entries) != 1 {
		t.Fatalf("watch destination entries = %+v, %v, want one immediate backup", entries, err)
	}

	secondDestination := filepath.Join(root, "second-destination")
	waitStop := make(chan struct{})
	waits := make([]time.Duration, 0, 1)
	if err := runWatchCommand([]string{"watch", source, secondDestination}, WatchCommandOptions{
		Output:             io.Discard,
		Stop:               waitStop,
		StateStore:         &confirmedStateStore,
		ConfigurationStore: &configurationStore,
		Logging:            LoggingCoordinatorOptions{FileDirectory: logs},
		Wait: func(interval time.Duration) {
			waits = append(waits, interval)
			close(waitStop)
		},
	}); err != nil {
		t.Fatalf("runWatchCommand(backup wait) error = %v", err)
	}
	if len(waits) != 1 || waits[0] != 2*time.Second {
		t.Fatalf("watch waits = %v, want one configured completion wait", waits)
	}
}

func TestBackupWatchRejectsDestinationOwnerWithoutMutation(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	destination := filepath.Join(root, "destination")
	applicationData := filepath.Join(root, "application-data")
	logs := filepath.Join(root, "logs")
	for _, directory := range []string{source, applicationData, logs} {
		if err := os.Mkdir(directory, 0o700); err != nil {
			t.Fatalf("Mkdir(%q) error = %v", directory, err)
		}
	}
	if err := os.WriteFile(filepath.Join(source, "file.txt"), []byte("content"), 0o600); err != nil {
		t.Fatalf("WriteFile(source) error = %v", err)
	}
	configurationStore := newConfigurationStore(filepath.Join(applicationData, "config.json"))
	if err := configurationStore.Save(ConfigurationSnapshot{SchemaVersion: 2, SynchronizationIntervalSeconds: 1, SynchronizationMode: SynchronizationModeBackup, BackupRetentionCount: 1}); err != nil {
		t.Fatalf("Save(configuration) error = %v", err)
	}
	backupStore, err := newBackupSetStoreAt(applicationData, destination)
	if err != nil {
		t.Fatalf("newBackupSetStoreAt() error = %v", err)
	}
	owner, err := acquireBackupDestinationOwnership(backupStore.LockPath())
	if err != nil {
		t.Fatalf("acquire owner error = %v", err)
	}
	defer owner.Release()
	confirmedStateStore := newConfirmedStateStoreAt(filepath.Join(root, "legacy-state"))
	stop := make(chan struct{})
	close(stop)
	if err := runWatchCommand([]string{"watch", source, destination}, WatchCommandOptions{Output: io.Discard, Stop: stop, StateStore: &confirmedStateStore, ConfigurationStore: &configurationStore, Logging: LoggingCoordinatorOptions{FileDirectory: logs}}); err == nil {
		t.Fatal("runWatchCommand(owner conflict) error = nil, want ownership failure")
	}
	if _, err := os.Stat(destination); !os.IsNotExist(err) {
		t.Fatalf("destination stat error = %v, want no mutation after ownership rejection", err)
	}
}

func snapshotTreeBytes(t *testing.T, root string) string {
	t.Helper()
	inventory, err := buildBackupLogicalInventory(root)
	if err != nil {
		t.Fatalf("buildBackupLogicalInventory(snapshot) error = %v", err)
	}
	return inventory.Digest
}
