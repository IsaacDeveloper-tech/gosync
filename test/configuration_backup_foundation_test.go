package gosync_test

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBackupConfigurationDomainDefinesVersionTwoAndConditionalRetention(t *testing.T) {
	if ConfigurationSchemaVersion != 2 || BackupConfigurationSchemaVersion != 2 {
		t.Fatalf("configuration schema version = %d, backup schema version = %d, want 2", ConfigurationSchemaVersion, BackupConfigurationSchemaVersion)
	}
	if MinimumBackupRetentionCount != 1 || MaximumBackupRetentionCount != 100 {
		t.Fatalf("backup retention range = %d..%d, want 1..100", MinimumBackupRetentionCount, MaximumBackupRetentionCount)
	}
	if SynchronizationModeBackup != SynchronizationMode("backup") {
		t.Fatalf("backup mode = %q, want lowercase persisted mode", SynchronizationModeBackup)
	}

	validConfigurations := []ConfigurationSnapshot{
		{SchemaVersion: 1, SynchronizationIntervalSeconds: 60, SynchronizationMode: SynchronizationModeBidirectional},
		{SchemaVersion: 2, SynchronizationIntervalSeconds: 60, SynchronizationMode: SynchronizationModeUnidirectional},
		{SchemaVersion: 2, SynchronizationIntervalSeconds: 60, SynchronizationMode: SynchronizationModeBackup, BackupRetentionCount: 3},
	}
	for _, configuration := range validConfigurations {
		if err := validateConfigurationSnapshot(configuration); err != nil {
			t.Fatalf("validateConfigurationSnapshot(%+v) error = %v", configuration, err)
		}
	}

	invalidConfigurations := []ConfigurationSnapshot{
		{SchemaVersion: 2, SynchronizationIntervalSeconds: 60, SynchronizationMode: SynchronizationModeBackup},
		{SchemaVersion: 2, SynchronizationIntervalSeconds: 60, SynchronizationMode: SynchronizationModeBackup, BackupRetentionCount: 101},
		{SchemaVersion: 2, SynchronizationIntervalSeconds: 60, SynchronizationMode: SynchronizationModeBidirectional, BackupRetentionCount: 3},
	}
	for _, configuration := range invalidConfigurations {
		if err := validateConfigurationSnapshot(configuration); err == nil {
			t.Fatalf("validateConfigurationSnapshot(%+v) error = nil, want rejection", configuration)
		}
	}
}

func TestParseSynchronizationModeAcceptsBackup(t *testing.T) {
	for _, answer := range []string{"BACKUP", "backup"} {
		mode, err := parseSynchronizationMode(answer)
		if err != nil || mode != SynchronizationModeBackup {
			t.Fatalf("parseSynchronizationMode(%q) = %q, %v, want %q", answer, mode, err, SynchronizationModeBackup)
		}
	}
}

func TestParseBackupRetentionCountValidatesBoundsAndWholeNumbers(t *testing.T) {
	for _, testCase := range []struct {
		answer string
		want   int
	}{
		{answer: "1", want: 1},
		{answer: "3", want: 3},
		{answer: "100", want: 100},
	} {
		retention, err := parseBackupRetentionCount(testCase.answer)
		if err != nil || retention != testCase.want {
			t.Fatalf("parseBackupRetentionCount(%q) = %d, %v, want %d", testCase.answer, retention, err, testCase.want)
		}
	}
	for _, answer := range []string{"0", "-1", "101", "1.5", "versions", "9223372036854775808"} {
		if _, err := parseBackupRetentionCount(answer); err == nil {
			t.Fatalf("parseBackupRetentionCount(%q) error = nil, want rejection", answer)
		}
	}
}

func TestCollectBackupConfigurationDraftUsesDefaultAndRepeatsInvalidRetention(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		var output strings.Builder
		draft, err := collectConfigurationDraft(strings.NewReader("60\nBACKUP\n\n"), &output)
		if err != nil {
			t.Fatalf("collectConfigurationDraft(default) error = %v", err)
		}
		if draft.Mode != SynchronizationModeBackup || draft.BackupRetentionCount != 3 {
			t.Fatalf("draft = %+v, want BACKUP with default retention 3", draft)
		}
		if !strings.Contains(output.String(), "retention") {
			t.Fatalf("interactive output = %q, want retention prompt", output.String())
		}
	})

	t.Run("invalid answers", func(t *testing.T) {
		var output strings.Builder
		draft, err := collectConfigurationDraft(strings.NewReader("60\nbackup\n0\n101\n1.5\n100\n"), &output)
		if err != nil {
			t.Fatalf("collectConfigurationDraft(invalid retention) error = %v", err)
		}
		if draft.Mode != SynchronizationModeBackup || draft.BackupRetentionCount != 100 {
			t.Fatalf("draft = %+v, want BACKUP with retention 100", draft)
		}
		if strings.Count(output.String(), "Invalid retention") != 3 {
			t.Fatalf("interactive output = %q, want three retention validation messages", output.String())
		}
	})
}

func TestConfirmBackupConfigurationIncludesRetentionAndRequiresConfirmation(t *testing.T) {
	draft := ConfigurationDraft{IntervalSeconds: 60, Mode: SynchronizationModeBackup, BackupRetentionCount: 7}
	var summary strings.Builder
	confirmed, err := confirmConfiguration(strings.NewReader("yes\n"), &summary, draft)
	if err != nil || !confirmed {
		t.Fatalf("confirmConfiguration(yes) = %t, %v, want confirmation", confirmed, err)
	}
	if !strings.Contains(summary.String(), "backup") || !strings.Contains(summary.String(), "7") || !strings.Contains(summary.String(), "retention") {
		t.Fatalf("summary = %q, want backup retention policy", summary.String())
	}

	for name, input := range map[string]io.Reader{
		"rejection": strings.NewReader("no\n"),
		"eof":       strings.NewReader(""),
		"interrupt": errorReader{err: errors.New("input interrupted")},
	} {
		t.Run(name, func(t *testing.T) {
			confirmed, err := confirmConfiguration(input, io.Discard, draft)
			if confirmed {
				t.Fatal("confirmConfiguration() confirmed cancelled session")
			}
			if name == "interrupt" {
				if err == nil || errors.Is(err, errConfigurationCancelled) {
					t.Fatalf("confirmConfiguration() error = %v, want input error", err)
				}
				return
			}
			if !errors.Is(err, errConfigurationCancelled) {
				t.Fatalf("confirmConfiguration() error = %v, want cancellation", err)
			}
		})
	}
}

func TestEncodeConfigurationProducesExactConditionalVersionTwoDocuments(t *testing.T) {
	testCases := []struct {
		name          string
		configuration ConfigurationSnapshot
		want          string
	}{
		{
			name:          "non-backup",
			configuration: ConfigurationSnapshot{SchemaVersion: 2, SynchronizationIntervalSeconds: 300, SynchronizationMode: SynchronizationModeUnidirectional},
			want:          `{"schemaVersion":2,"synchronizationIntervalSeconds":300,"synchronizationMode":"unidirectional"}`,
		},
		{
			name:          "backup",
			configuration: ConfigurationSnapshot{SchemaVersion: 2, SynchronizationIntervalSeconds: 300, SynchronizationMode: SynchronizationModeBackup, BackupRetentionCount: 3},
			want:          `{"schemaVersion":2,"synchronizationIntervalSeconds":300,"synchronizationMode":"backup","backupRetentionCount":3}`,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			encoded, err := encodeConfiguration(testCase.configuration)
			if err != nil {
				t.Fatalf("encodeConfiguration() error = %v", err)
			}
			if encoded != testCase.want {
				t.Fatalf("encoded configuration = %q, want %q", encoded, testCase.want)
			}
		})
	}
}

func TestDecodeConfigurationAcceptsVersionOneAndBothVersionTwoVariants(t *testing.T) {
	testCases := []struct {
		name string
		data string
		want ConfigurationSnapshot
	}{
		{
			name: "version one",
			data: `{"schemaVersion":1,"synchronizationIntervalSeconds":60,"synchronizationMode":"bidirectional"}`,
			want: ConfigurationSnapshot{SchemaVersion: 1, SynchronizationIntervalSeconds: 60, SynchronizationMode: SynchronizationModeBidirectional},
		},
		{
			name: "version two non-backup",
			data: `{"schemaVersion":2,"synchronizationIntervalSeconds":120,"synchronizationMode":"unidirectional"}`,
			want: ConfigurationSnapshot{SchemaVersion: 2, SynchronizationIntervalSeconds: 120, SynchronizationMode: SynchronizationModeUnidirectional},
		},
		{
			name: "version two backup",
			data: `{"schemaVersion":2,"synchronizationIntervalSeconds":300,"synchronizationMode":"backup","backupRetentionCount":5}`,
			want: ConfigurationSnapshot{SchemaVersion: 2, SynchronizationIntervalSeconds: 300, SynchronizationMode: SynchronizationModeBackup, BackupRetentionCount: 5},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			configuration, err := decodeConfiguration(testCase.data)
			if err != nil {
				t.Fatalf("decodeConfiguration() error = %v", err)
			}
			if configuration != testCase.want {
				t.Fatalf("decoded configuration = %+v, want %+v", configuration, testCase.want)
			}
		})
	}
}

func TestDecodeConfigurationRejectsInvalidConditionalVersionTwoSchemas(t *testing.T) {
	testCases := []struct {
		name string
		data string
	}{
		{name: "backup missing retention", data: `{"schemaVersion":2,"synchronizationIntervalSeconds":60,"synchronizationMode":"backup"}`},
		{name: "non-backup has retention", data: `{"schemaVersion":2,"synchronizationIntervalSeconds":60,"synchronizationMode":"bidirectional","backupRetentionCount":3}`},
		{name: "backup uppercase", data: `{"schemaVersion":2,"synchronizationIntervalSeconds":60,"synchronizationMode":"BACKUP","backupRetentionCount":3}`},
		{name: "retention zero", data: `{"schemaVersion":2,"synchronizationIntervalSeconds":60,"synchronizationMode":"backup","backupRetentionCount":0}`},
		{name: "retention too high", data: `{"schemaVersion":2,"synchronizationIntervalSeconds":60,"synchronizationMode":"backup","backupRetentionCount":101}`},
		{name: "retention fractional", data: `{"schemaVersion":2,"synchronizationIntervalSeconds":60,"synchronizationMode":"backup","backupRetentionCount":1.5}`},
		{name: "retention wrong type", data: `{"schemaVersion":2,"synchronizationIntervalSeconds":60,"synchronizationMode":"backup","backupRetentionCount":"3"}`},
		{name: "retention wrong capitalization", data: `{"schemaVersion":2,"synchronizationIntervalSeconds":60,"synchronizationMode":"backup","BackupRetentionCount":3}`},
		{name: "duplicate retention", data: `{"schemaVersion":2,"synchronizationIntervalSeconds":60,"synchronizationMode":"backup","backupRetentionCount":3,"backupRetentionCount":4}`},
		{name: "unknown property", data: `{"schemaVersion":2,"synchronizationIntervalSeconds":60,"synchronizationMode":"backup","backupRetentionCount":3,"extra":true}`},
		{name: "unsupported version", data: `{"schemaVersion":3,"synchronizationIntervalSeconds":60,"synchronizationMode":"backup","backupRetentionCount":3}`},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			if _, err := decodeConfiguration(testCase.data); err == nil {
				t.Fatalf("decodeConfiguration(%s) error = nil, want rejection", testCase.data)
			}
		})
	}
}

func TestConfigurationServicePersistsConfirmedBackupAndPreservesPreviousDocument(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	store := newConfigurationStore(configPath)
	service := newConfigurationService(ConfigurationServiceOptions{
		Store:  store,
		Input:  strings.NewReader("60\nBACKUP\n7\nyes\n"),
		Output: io.Discard,
	})
	configuration, err := service.Configure()
	if err != nil {
		t.Fatalf("Configure(backup) error = %v", err)
	}
	want := ConfigurationSnapshot{SchemaVersion: 2, SynchronizationIntervalSeconds: 60, SynchronizationMode: SynchronizationModeBackup, BackupRetentionCount: 7}
	if configuration != want {
		t.Fatalf("configuration = %+v, want %+v", configuration, want)
	}
	contents, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("ReadFile(config) error = %v", err)
	}
	if string(contents) != `{"schemaVersion":2,"synchronizationIntervalSeconds":60,"synchronizationMode":"backup","backupRetentionCount":7}` {
		t.Fatalf("stored configuration = %q, want exact version-two backup JSON", contents)
	}

	previousContents := append([]byte(nil), contents...)
	cancelledService := newConfigurationService(ConfigurationServiceOptions{
		Store:  store,
		Input:  strings.NewReader("120\nunidirectional\nno\n"),
		Output: io.Discard,
	})
	if _, err := cancelledService.Configure(); !errors.Is(err, errConfigurationCancelled) {
		t.Fatalf("Configure(cancelled) error = %v, want cancellation", err)
	}
	contents, err = os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("ReadFile(config after cancellation) error = %v", err)
	}
	if string(contents) != string(previousContents) {
		t.Fatalf("configuration changed after cancellation: %q -> %q", previousContents, contents)
	}

	loaded, err := store.Load()
	if err != nil || loaded.Status != ConfigurationLoadValid || loaded.Configuration != want {
		t.Fatalf("loaded backup configuration = %+v, error = %v, want persisted backup snapshot", loaded, err)
	}
}
