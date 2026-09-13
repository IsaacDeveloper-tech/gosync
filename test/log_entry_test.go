package gosync_test

import (
	"reflect"
	"testing"
	"time"
)

func TestNewLogEntryUsesInjectedClockAndStoresStructuredFields(t *testing.T) {
	expectedTimestamp := time.Date(2026, time.September, 13, 14, 15, 16, 123456789, time.UTC)
	expectedContext := map[string]string{
		"root": "first",
		"path": "notes.txt",
	}
	clockCalls := 0

	entry := newLogEntry(func() time.Time {
		clockCalls++
		return expectedTimestamp
	}, LogSeverityInfo, LogEventChangeSynchronized, "change synchronized", expectedContext)

	if !entry.Timestamp.Equal(expectedTimestamp) {
		t.Fatalf("timestamp = %v, want %v", entry.Timestamp, expectedTimestamp)
	}
	if entry.Severity != LogSeverityInfo {
		t.Fatalf("severity = %v, want %v", entry.Severity, LogSeverityInfo)
	}
	if entry.Event != LogEventChangeSynchronized {
		t.Fatalf("event = %v, want %v", entry.Event, LogEventChangeSynchronized)
	}
	if entry.Message != "change synchronized" {
		t.Fatalf("message = %q, want %q", entry.Message, "change synchronized")
	}
	if !reflect.DeepEqual(entry.Context, expectedContext) {
		t.Fatalf("context = %#v, want %#v", entry.Context, expectedContext)
	}
	if clockCalls != 1 {
		t.Fatalf("clock calls = %d, want 1", clockCalls)
	}
}
