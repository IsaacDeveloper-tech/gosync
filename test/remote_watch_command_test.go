package gosync_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRemoteWatchRejectsInvalidArgumentsBeforeReadingConfiguration(t *testing.T) {
	temporaryDirectory := t.TempDir()
	invalidConfigurationPath := filepath.Join(temporaryDirectory, "config.json")
	if err := os.WriteFile(invalidConfigurationPath, []byte("not json"), 0o600); err != nil {
		t.Fatalf("WriteFile(invalid config) error = %v", err)
	}
	configurationStore := newConfigurationStore(invalidConfigurationPath)
	stateStore := newConfirmedStateStoreAt(filepath.Join(temporaryDirectory, "state"))
	arguments := []struct {
		name string
		args []string
		want string
	}{
		{name: "remote source", args: []string{"watch", "sftp://server/source", filepath.Join(temporaryDirectory, "destination")}, want: "source must be a local"},
		{name: "invalid remote path", args: []string{"watch", filepath.Join(temporaryDirectory, "source"), "ftp://server/a/../archive"}, want: "dot segments"},
		{name: "unsupported endpoint", args: []string{"watch", filepath.Join(temporaryDirectory, "source"), "https://server/archive"}, want: "ftp or sftp"},
	}
	for _, testCase := range arguments {
		t.Run(testCase.name, func(t *testing.T) {
			err := runWatchCommand(testCase.args, WatchCommandOptions{
				Input:              strings.NewReader(""),
				Output:             ioDiscardWriter{},
				StateStore:         &stateStore,
				ConfigurationStore: &configurationStore,
			})
			if err == nil || !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(testCase.want)) {
				t.Fatalf("runWatchCommand() error = %v, want an argument error containing %q", err, testCase.want)
			}
			if strings.Contains(err.Error(), "invalid configuration") {
				t.Fatalf("configuration was read before argument rejection: %v", err)
			}
		})
	}
}

func TestRemoteWatchRejectsBidirectionalModeBeforeRemoteDispatch(t *testing.T) {
	temporaryDirectory := t.TempDir()
	sourceRoot := filepath.Join(temporaryDirectory, "source")
	if err := os.Mkdir(sourceRoot, 0o700); err != nil {
		t.Fatalf("Mkdir(source) error = %v", err)
	}
	configurationStore := newConfigurationStore(filepath.Join(temporaryDirectory, "config.json"))
	configuration := ConfigurationSnapshot{
		SchemaVersion:                  1,
		SynchronizationIntervalSeconds: 5,
		SynchronizationMode:            SynchronizationModeBidirectional,
	}
	if err := configurationStore.Save(configuration); err != nil {
		t.Fatalf("Save(configuration) error = %v", err)
	}
	stateStore := newConfirmedStateStoreAt(filepath.Join(temporaryDirectory, "state"))
	remoteDispatchCalls := 0
	err := runWatchCommand([]string{"watch", sourceRoot, "sftp://server/archive"}, WatchCommandOptions{
		Input:              strings.NewReader("must not be read"),
		Output:             ioDiscardWriter{},
		StateStore:         &stateStore,
		ConfigurationStore: &configurationStore,
		Logging:            LoggingCoordinatorOptions{FileDirectory: filepath.Join(temporaryDirectory, "logs")},
		SynchronizeRemoteWithPolicy: func(RemoteWatchPolicySnapshot) error {
			remoteDispatchCalls++
			return nil
		},
	})
	if err == nil || !strings.Contains(err.Error(), "bidirectional") {
		t.Fatalf("runWatchCommand() error = %v, want bidirectional rejection", err)
	}
	if remoteDispatchCalls != 0 {
		t.Fatalf("remote dispatch calls = %d, want 0", remoteDispatchCalls)
	}
}

func TestRemoteWatchRoutesSupportedModesUsingOneStartupPolicySnapshot(t *testing.T) {
	testCases := []struct {
		name          string
		configuration ConfigurationSnapshot
		changed       ConfigurationSnapshot
	}{
		{
			name: "unidirectional",
			configuration: ConfigurationSnapshot{
				SchemaVersion:                  2,
				SynchronizationIntervalSeconds: 7,
				SynchronizationMode:            SynchronizationModeUnidirectional,
			},
			changed: ConfigurationSnapshot{
				SchemaVersion:                  2,
				SynchronizationIntervalSeconds: 19,
				SynchronizationMode:            SynchronizationModeBackup,
				BackupRetentionCount:           2,
			},
		},
		{
			name: "backup",
			configuration: ConfigurationSnapshot{
				SchemaVersion:                  2,
				SynchronizationIntervalSeconds: 11,
				SynchronizationMode:            SynchronizationModeBackup,
				BackupRetentionCount:           3,
			},
			changed: ConfigurationSnapshot{
				SchemaVersion:                  2,
				SynchronizationIntervalSeconds: 23,
				SynchronizationMode:            SynchronizationModeUnidirectional,
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			temporaryDirectory := t.TempDir()
			sourceRoot := filepath.Join(temporaryDirectory, "source")
			if err := os.Mkdir(sourceRoot, 0o700); err != nil {
				t.Fatalf("Mkdir(source) error = %v", err)
			}
			if err := os.WriteFile(filepath.Join(sourceRoot, "notes.txt"), []byte("source data"), 0o600); err != nil {
				t.Fatalf("WriteFile(source entry) error = %v", err)
			}
			configurationStore := newConfigurationStore(filepath.Join(temporaryDirectory, "config.json"))
			if err := configurationStore.Save(testCase.configuration); err != nil {
				t.Fatalf("Save(initial configuration) error = %v", err)
			}
			stateStore := newConfirmedStateStoreAt(filepath.Join(temporaryDirectory, "state"))
			stop := make(chan struct{})
			receivedPolicies := make([]RemoteWatchPolicySnapshot, 0, 2)
			waitIntervals := make([]time.Duration, 0, 1)

			err := runWatchCommand([]string{"watch", sourceRoot, "FTP://Backup.Example:2121/weekly"}, WatchCommandOptions{
				Input:              strings.NewReader(""),
				Output:             ioDiscardWriter{},
				Stop:               stop,
				StateStore:         &stateStore,
				ConfigurationStore: &configurationStore,
				Logging:            LoggingCoordinatorOptions{FileDirectory: filepath.Join(temporaryDirectory, "logs")},
				Wait: func(interval time.Duration) {
					waitIntervals = append(waitIntervals, interval)
					if err := configurationStore.Save(testCase.changed); err != nil {
						t.Fatalf("Save(changed configuration) error = %v", err)
					}
				},
				SynchronizeRemoteWithPolicy: func(policy RemoteWatchPolicySnapshot) error {
					receivedPolicies = append(receivedPolicies, policy)
					if len(receivedPolicies) == 2 {
						close(stop)
					}
					return nil
				},
			})
			if err != nil {
				t.Fatalf("runWatchCommand() error = %v", err)
			}
			if len(receivedPolicies) != 2 || receivedPolicies[0] != receivedPolicies[1] {
				t.Fatalf("remote policies = %+v, want two identical startup snapshots", receivedPolicies)
			}
			policy := receivedPolicies[0]
			expectedSourceRoot, err := filepath.EvalSymlinks(sourceRoot)
			if err != nil {
				t.Fatalf("EvalSymlinks(source) error = %v", err)
			}
			if filepath.Clean(policy.SourceRoot) != filepath.Clean(expectedSourceRoot) || policy.Destination.Protocol != RemoteProtocolFTP || policy.Destination.Host != "backup.example" || policy.Destination.Port != 2121 || policy.Destination.Path != "/weekly" {
				t.Fatalf("remote policy destination/source = %+v, want source %q and FTP endpoint protocol=%q host=%q port=%d path=%q", policy, expectedSourceRoot, RemoteProtocolFTP, "backup.example", 2121, "/weekly")
			}
			if policy.Mode != testCase.configuration.SynchronizationMode || policy.Interval != time.Duration(testCase.configuration.SynchronizationIntervalSeconds)*time.Second || policy.BackupRetentionCount != testCase.configuration.BackupRetentionCount {
				t.Fatalf("remote policy = %+v, want startup mode, interval, and retention from %+v", policy, testCase.configuration)
			}
			if len(waitIntervals) != 1 || waitIntervals[0] != policy.Interval {
				t.Fatalf("watch waits = %v, want one startup interval %v", waitIntervals, policy.Interval)
			}
		})
	}
}

func TestRemoteWatchAppliesConfigurationAndSourceGuardsBeforeDispatch(t *testing.T) {
	temporaryDirectory := t.TempDir()
	sourceRoot := filepath.Join(temporaryDirectory, "source")
	if err := os.Mkdir(sourceRoot, 0o700); err != nil {
		t.Fatalf("Mkdir(source) error = %v", err)
	}
	configuration := ConfigurationSnapshot{
		SchemaVersion:                  2,
		SynchronizationIntervalSeconds: 1,
		SynchronizationMode:            SynchronizationModeUnidirectional,
	}
	stateStore := newConfirmedStateStoreAt(filepath.Join(temporaryDirectory, "state"))
	remoteDispatchCalls := 0
	commandOptions := WatchCommandOptions{
		Input:      strings.NewReader(""),
		Output:     ioDiscardWriter{},
		StateStore: &stateStore,
		Logging:    LoggingCoordinatorOptions{FileDirectory: filepath.Join(temporaryDirectory, "logs")},
		SynchronizeRemoteWithPolicy: func(RemoteWatchPolicySnapshot) error {
			remoteDispatchCalls++
			return nil
		},
	}

	t.Run("configuration path inside source is rejected before load", func(t *testing.T) {
		configurationStore := newConfigurationStore(filepath.Join(sourceRoot, "config.json"))
		options := commandOptions
		options.ConfigurationStore = &configurationStore
		err := runWatchCommand([]string{"watch", sourceRoot, "ftp://server/archive"}, options)
		if err == nil || !strings.Contains(err.Error(), "inside synchronization root") {
			t.Fatalf("runWatchCommand() error = %v, want configuration/source overlap error", err)
		}
		if remoteDispatchCalls != 0 {
			t.Fatalf("remote dispatch calls = %d, want 0", remoteDispatchCalls)
		}
		if _, err := os.Stat(configurationStore.Path()); !os.IsNotExist(err) {
			t.Fatalf("configuration path state error = %v, want no file created", err)
		}
	})

	t.Run("missing source is rejected before dispatch", func(t *testing.T) {
		configurationStore := newConfigurationStore(filepath.Join(temporaryDirectory, "valid-config.json"))
		if err := configurationStore.Save(configuration); err != nil {
			t.Fatalf("Save(configuration) error = %v", err)
		}
		options := commandOptions
		options.ConfigurationStore = &configurationStore
		err := runWatchCommand([]string{"watch", filepath.Join(temporaryDirectory, "missing-source"), "sftp://server/archive"}, options)
		if err == nil || !strings.Contains(strings.ToLower(err.Error()), "source") {
			t.Fatalf("runWatchCommand() error = %v, want missing-source error", err)
		}
		if remoteDispatchCalls != 0 {
			t.Fatalf("remote dispatch calls = %d, want 0", remoteDispatchCalls)
		}
	})
}

func TestRemoteWatchKeepsConsoleFallbackWhenLogDirectoryOverlapsSource(t *testing.T) {
	temporaryDirectory := t.TempDir()
	sourceRoot := filepath.Join(temporaryDirectory, "source")
	if err := os.Mkdir(sourceRoot, 0o700); err != nil {
		t.Fatalf("Mkdir(source) error = %v", err)
	}
	configurationStore := newConfigurationStore(filepath.Join(temporaryDirectory, "config.json"))
	configuration := ConfigurationSnapshot{
		SchemaVersion:                  2,
		SynchronizationIntervalSeconds: 1,
		SynchronizationMode:            SynchronizationModeUnidirectional,
	}
	if err := configurationStore.Save(configuration); err != nil {
		t.Fatalf("Save(configuration) error = %v", err)
	}
	stateStore := newConfirmedStateStoreAt(filepath.Join(temporaryDirectory, "state"))
	stop := make(chan struct{})
	var output bytes.Buffer
	err := runWatchCommand([]string{"watch", sourceRoot, "ftp://server/archive"}, WatchCommandOptions{
		Output:             &output,
		Stop:               stop,
		StateStore:         &stateStore,
		ConfigurationStore: &configurationStore,
		Logging: LoggingCoordinatorOptions{
			ConsoleWriter: &output,
			FileDirectory: filepath.Join(sourceRoot, "logs"),
		},
		SynchronizeRemoteWithPolicy: func(RemoteWatchPolicySnapshot) error {
			close(stop)
			return nil
		},
	})
	if err != nil {
		t.Fatalf("runWatchCommand() error = %v", err)
	}
	if !strings.Contains(output.String(), "persistent log location overlaps a synchronization root") {
		t.Fatalf("console output = %q, want existing log/source overlap warning", output.String())
	}
	if _, err := os.Stat(filepath.Join(sourceRoot, "logs")); !os.IsNotExist(err) {
		t.Fatalf("overlapping persistent log directory state error = %v, want console-only fallback", err)
	}
}

type ioDiscardWriter struct{}

func (ioDiscardWriter) Write(buffer []byte) (int, error) {
	return len(buffer), nil
}
