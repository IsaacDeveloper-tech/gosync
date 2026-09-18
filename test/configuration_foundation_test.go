package gosync_test

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigurationDomainDefinesVersionRangeModesAndSnapshots(t *testing.T) {
	if ConfigurationSchemaVersion != 1 {
		t.Fatalf("schema version = %d, want 1", ConfigurationSchemaVersion)
	}
	if MinimumSynchronizationIntervalSeconds != 1 || MaximumSynchronizationIntervalSeconds != 86400 {
		t.Fatalf("interval range = %d..%d, want 1..86400", MinimumSynchronizationIntervalSeconds, MaximumSynchronizationIntervalSeconds)
	}
	if SynchronizationModeBidirectional == SynchronizationModeUnidirectional || SynchronizationModeBackup == SynchronizationModeBidirectional {
		t.Fatal("configuration modes must be distinct")
	}

	snapshot := ConfigurationSnapshot{
		SchemaVersion:                  ConfigurationSchemaVersion,
		SynchronizationIntervalSeconds: 60,
		SynchronizationMode:            SynchronizationModeBidirectional,
	}
	snapshotCopy := snapshot
	snapshotCopy.SynchronizationIntervalSeconds = 120
	if snapshot.SynchronizationIntervalSeconds != 60 {
		t.Fatal("configuration snapshot was changed through a copied snapshot")
	}
}

func TestParseConfigureCommandAcceptsOnlyArgumentFreeConfigure(t *testing.T) {
	if err := parseConfigureCommand([]string{"configure"}); err != nil {
		t.Fatalf("parseConfigureCommand(configure) error = %v", err)
	}
	for _, arguments := range [][]string{{}, {"configure", "extra"}, {"watch"}, {"configure", "--interval", "60"}} {
		if err := parseConfigureCommand(arguments); err == nil {
			t.Fatalf("parseConfigureCommand(%v) error = nil, want usage error", arguments)
		}
	}
}

func TestParseSynchronizationIntervalAcceptsInclusiveBoundsAndRejectsInvalidAnswers(t *testing.T) {
	for _, answer := range []string{"1", "86400"} {
		interval, err := parseSynchronizationInterval(answer)
		if err != nil || interval < 1 || interval > 86400 {
			t.Fatalf("parseSynchronizationInterval(%q) = %d, %v, want valid interval", answer, interval, err)
		}
	}
	for _, answer := range []string{"0", "86401", "1.5", "", "seconds", "-1"} {
		if _, err := parseSynchronizationInterval(answer); err == nil {
			t.Fatalf("parseSynchronizationInterval(%q) error = nil, want invalid-answer error", answer)
		}
	}
}

func TestParseSynchronizationModeAcceptsPersistedModesAndRejectsBackup(t *testing.T) {
	for _, testCase := range []struct {
		answer string
		mode   SynchronizationMode
	}{
		{answer: "bidirectional", mode: SynchronizationModeBidirectional},
		{answer: "unidirectional", mode: SynchronizationModeUnidirectional},
	} {
		mode, err := parseSynchronizationMode(testCase.answer)
		if err != nil || mode != testCase.mode {
			t.Fatalf("parseSynchronizationMode(%q) = %q, %v, want %q", testCase.answer, mode, err, testCase.mode)
		}
	}
	for _, answer := range []string{"BACKUP", "backup", "unknown", ""} {
		if _, err := parseSynchronizationMode(answer); err == nil {
			t.Fatalf("parseSynchronizationMode(%q) error = nil, want invalid-mode error", answer)
		}
	}
}

func TestCollectConfigurationDraftRepeatsInvalidIntervalAndModeAnswers(t *testing.T) {
	input := strings.NewReader("0\n60\nBACKUP\nunidirectional\n")
	var output strings.Builder

	draft, err := collectConfigurationDraft(input, &output)
	if err != nil {
		t.Fatalf("collectConfigurationDraft() error = %v", err)
	}
	if draft.IntervalSeconds != 60 || draft.Mode != SynchronizationModeUnidirectional {
		t.Fatalf("draft = %+v, want interval 60 and unidirectional mode", draft)
	}
	if !strings.Contains(output.String(), "BACKUP") || !strings.Contains(output.String(), "unavailable") {
		t.Fatalf("interactive output = %q, want unavailable BACKUP explanation", output.String())
	}
}

func TestConfirmConfigurationRequiresExplicitAcceptanceAndCancelsWithoutConfirmation(t *testing.T) {
	draft := ConfigurationDraft{IntervalSeconds: 60, Mode: SynchronizationModeBidirectional}
	var summary strings.Builder
	confirmed, err := confirmConfiguration(strings.NewReader("yes\n"), &summary, draft)
	if err != nil || !confirmed {
		t.Fatalf("confirmConfiguration(yes) = %t, %v, want confirmed", confirmed, err)
	}
	if !strings.Contains(summary.String(), "60") || !strings.Contains(summary.String(), "bidirectional") {
		t.Fatalf("summary = %q, want draft values", summary.String())
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
				if err == nil || strings.Contains(err.Error(), "configuration cancelled") {
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

func TestRunInteractiveConfigurationReturnsOnlyConfirmedCompleteSnapshot(t *testing.T) {
	snapshot, err := runInteractiveConfiguration(strings.NewReader("120\nunidirectional\nyes\n"), io.Discard)
	if err != nil {
		t.Fatalf("runInteractiveConfiguration() error = %v", err)
	}
	if snapshot.SchemaVersion != ConfigurationSchemaVersion || snapshot.SynchronizationIntervalSeconds != 120 || snapshot.SynchronizationMode != SynchronizationModeUnidirectional {
		t.Fatalf("snapshot = %+v, want confirmed versioned configuration", snapshot)
	}
}

func TestResolveConfigurationFilePathUsesPerUserLocationIndependentOfWorkingDirectory(t *testing.T) {
	userConfigurationDirectory, err := os.UserConfigDir()
	if err != nil {
		t.Fatalf("UserConfigDir() error = %v", err)
	}
	originalWorkingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	defer os.Chdir(originalWorkingDirectory)

	firstPath, err := resolveConfigurationFilePath()
	if err != nil {
		t.Fatalf("resolveConfigurationFilePath() error = %v", err)
	}
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}
	secondPath, err := resolveConfigurationFilePath()
	if err != nil {
		t.Fatalf("resolveConfigurationFilePath() after Chdir error = %v", err)
	}

	want := filepath.Join(userConfigurationDirectory, "gosync", "config.json")
	if firstPath != want || secondPath != want {
		t.Fatalf("configuration paths = %q and %q, want %q", firstPath, secondPath, want)
	}
}

func TestEncodeConfigurationProducesExactVersionOneJSONDocument(t *testing.T) {
	configuration := ConfigurationSnapshot{
		SchemaVersion:                  1,
		SynchronizationIntervalSeconds: 300,
		SynchronizationMode:            SynchronizationModeBidirectional,
	}
	encoded, err := encodeConfiguration(configuration)
	if err != nil {
		t.Fatalf("encodeConfiguration() error = %v", err)
	}
	want := `{"schemaVersion":1,"synchronizationIntervalSeconds":300,"synchronizationMode":"bidirectional"}`
	if encoded != want {
		t.Fatalf("encoded configuration = %q, want %q", encoded, want)
	}
}

func TestDecodeConfigurationAcceptsCompleteVersionOneDocument(t *testing.T) {
	encoded := `{"schemaVersion":1,"synchronizationIntervalSeconds":300,"synchronizationMode":"unidirectional"}`
	configuration, err := decodeConfiguration(encoded)
	if err != nil {
		t.Fatalf("decodeConfiguration() error = %v", err)
	}
	want := ConfigurationSnapshot{SchemaVersion: 1, SynchronizationIntervalSeconds: 300, SynchronizationMode: SynchronizationModeUnidirectional}
	if configuration != want {
		t.Fatalf("decoded configuration = %+v, want %+v", configuration, want)
	}
}

func TestDecodeConfigurationRejectsAmbiguousAndUnsupportedJSON(t *testing.T) {
	testCases := []struct {
		name string
		data string
	}{
		{name: "malformed", data: `{"schemaVersion":1`},
		{name: "trailing value", data: `{"schemaVersion":1,"synchronizationIntervalSeconds":60,"synchronizationMode":"bidirectional"} {}`},
		{name: "duplicate property", data: `{"schemaVersion":1,"schemaVersion":1,"synchronizationIntervalSeconds":60,"synchronizationMode":"bidirectional"}`},
		{name: "unknown property", data: `{"schemaVersion":1,"synchronizationIntervalSeconds":60,"synchronizationMode":"bidirectional","extra":true}`},
		{name: "missing property", data: `{"schemaVersion":1,"synchronizationIntervalSeconds":60}`},
		{name: "wrong capitalization", data: `{"schemaVersion":1,"synchronizationIntervalSeconds":60,"SynchronizationMode":"bidirectional"}`},
		{name: "wrong interval type", data: `{"schemaVersion":1,"synchronizationIntervalSeconds":"60","synchronizationMode":"bidirectional"}`},
		{name: "wrong mode type", data: `{"schemaVersion":1,"synchronizationIntervalSeconds":60,"synchronizationMode":true}`},
		{name: "unsupported version", data: `{"schemaVersion":2,"synchronizationIntervalSeconds":60,"synchronizationMode":"bidirectional"}`},
		{name: "fractional interval", data: `{"schemaVersion":1,"synchronizationIntervalSeconds":60.5,"synchronizationMode":"bidirectional"}`},
		{name: "low interval", data: `{"schemaVersion":1,"synchronizationIntervalSeconds":0,"synchronizationMode":"bidirectional"}`},
		{name: "high interval", data: `{"schemaVersion":1,"synchronizationIntervalSeconds":86401,"synchronizationMode":"bidirectional"}`},
		{name: "backup mode", data: `{"schemaVersion":1,"synchronizationIntervalSeconds":60,"synchronizationMode":"BACKUP"}`},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			if _, err := decodeConfiguration(testCase.data); err == nil {
				t.Fatalf("decodeConfiguration(%s) error = nil, want rejection", testCase.data)
			}
		})
	}
}

type errorReader struct {
	err error
}

func (reader errorReader) Read([]byte) (int, error) {
	return 0, reader.err
}
