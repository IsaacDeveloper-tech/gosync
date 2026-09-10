package gosync_test

import "testing"

func TestLogSeverityDefinesEveryRequiredLevel(t *testing.T) {
	severities := []struct {
		name     string
		severity LogSeverity
	}{
		{name: "debug", severity: LogSeverityDebug},
		{name: "info", severity: LogSeverityInfo},
		{name: "warn", severity: LogSeverityWarn},
		{name: "error", severity: LogSeverityError},
	}

	seen := make(map[LogSeverity]struct{}, len(severities))
	for _, testCase := range severities {
		t.Run(testCase.name, func(t *testing.T) {
			if _, alreadySeen := seen[testCase.severity]; alreadySeen {
				t.Fatalf("severity %v is duplicated", testCase.severity)
			}
			seen[testCase.severity] = struct{}{}
		})
	}

	if len(seen) != len(severities) {
		t.Fatalf("defined severities = %d, want %d", len(seen), len(severities))
	}
}

func TestLogEventDefinesEveryPlannedIdentifier(t *testing.T) {
	events := []struct {
		name  string
		event LogEvent
	}{
		{name: "synchronization started", event: LogEventSynchronizationStarted},
		{name: "synchronization completed", event: LogEventSynchronizationCompleted},
		{name: "change synchronized", event: LogEventChangeSynchronized},
		{name: "conflict detected", event: LogEventConflictDetected},
		{name: "retry started", event: LogEventRetryStarted},
		{name: "retry completed", event: LogEventRetryCompleted},
		{name: "warning raised", event: LogEventWarningRaised},
		{name: "operation failed", event: LogEventOperationFailed},
		{name: "persistent destination failure", event: LogEventPersistentDestinationFailure},
		{name: "console destination failure", event: LogEventConsoleDestinationFailure},
	}

	seen := make(map[LogEvent]struct{}, len(events))
	for _, testCase := range events {
		t.Run(testCase.name, func(t *testing.T) {
			if _, alreadySeen := seen[testCase.event]; alreadySeen {
				t.Fatalf("event %v is duplicated", testCase.event)
			}
			seen[testCase.event] = struct{}{}
		})
	}

	if len(seen) != len(events) {
		t.Fatalf("defined events = %d, want %d", len(seen), len(events))
	}
}
