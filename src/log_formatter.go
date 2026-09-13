package gosync

import (
	"sort"
	"strconv"
	"strings"
)

func formatLogEntry(entry LogEntry) string {
	var formattedEntry strings.Builder

	formattedEntry.WriteString("timestamp=")
	formattedEntry.WriteString(entry.Timestamp.Format("2006-01-02T15:04:05.999999999Z07:00"))
	formattedEntry.WriteString(" severity=")
	formattedEntry.WriteString(logSeverityName(entry.Severity))
	formattedEntry.WriteString(" event=")
	formattedEntry.WriteString(logEventName(entry.Event))
	formattedEntry.WriteString(" message=")
	formattedEntry.WriteString(strconv.Quote(entry.Message))

	contextKeys := make([]string, 0, len(entry.Context))
	for contextKey := range entry.Context {
		contextKeys = append(contextKeys, contextKey)
	}
	sort.Strings(contextKeys)
	for _, contextKey := range contextKeys {
		formattedEntry.WriteString(" context.")
		formattedEntry.WriteString(contextKey)
		formattedEntry.WriteByte('=')
		formattedEntry.WriteString(strconv.Quote(entry.Context[contextKey]))
	}

	formattedEntry.WriteByte('\n')
	return formattedEntry.String()
}

func logSeverityName(severity LogSeverity) string {
	switch severity {
	case LogSeverityDebug:
		return "DEBUG"
	case LogSeverityInfo:
		return "INFO"
	case LogSeverityWarn:
		return "WARN"
	case LogSeverityError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

func logEventName(event LogEvent) string {
	switch event {
	case LogEventSynchronizationStarted:
		return "synchronization_started"
	case LogEventSynchronizationCompleted:
		return "synchronization_completed"
	case LogEventChangeSynchronized:
		return "change_synchronized"
	case LogEventConflictDetected:
		return "conflict_detected"
	case LogEventRetryStarted:
		return "retry_started"
	case LogEventRetryCompleted:
		return "retry_completed"
	case LogEventWarningRaised:
		return "warning_raised"
	case LogEventOperationFailed:
		return "operation_failed"
	case LogEventPersistentDestinationFailure:
		return "persistent_destination_failure"
	case LogEventConsoleDestinationFailure:
		return "console_destination_failure"
	default:
		return "unknown_event"
	}
}
