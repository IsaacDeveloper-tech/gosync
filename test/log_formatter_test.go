package gosync_test

import (
	"testing"
	"time"
)

func TestFormatLogEntryProducesOneDeterministicStructuredRecord(t *testing.T) {
	entry := LogEntry{
		Timestamp: time.Date(2026, time.September, 13, 14, 15, 16, 123456789, time.UTC),
		Severity:  LogSeverityInfo,
		Event:     LogEventChangeSynchronized,
		Message:   "change synchronized",
		Context: map[string]string{
			"root": "first",
			"path": "notes.txt",
		},
	}

	want := "timestamp=2026-09-13T14:15:16.123456789Z severity=INFO event=change_synchronized message=\"change synchronized\" context.path=\"notes.txt\" context.root=\"first\"\n"
	if got := formatLogEntry(entry); got != want {
		t.Fatalf("formatted entry = %q, want %q", got, want)
	}
}
