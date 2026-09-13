package gosync

import "time"

type LogEntry struct {
	Timestamp time.Time
	Severity  LogSeverity
	Event     LogEvent
	Message   string
	Context   map[string]string
}

func newLogEntry(
	clock func() time.Time,
	severity LogSeverity,
	event LogEvent,
	message string,
	context map[string]string,
) LogEntry {
	return LogEntry{
		Timestamp: clock(),
		Severity:  severity,
		Event:     event,
		Message:   message,
		Context:   context,
	}
}
