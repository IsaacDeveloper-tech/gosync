package gosync

type LogSeverity uint8

const (
	LogSeverityDebug LogSeverity = iota
	LogSeverityInfo
	LogSeverityWarn
	LogSeverityError
)

type LogEvent uint8

const (
	LogEventSynchronizationStarted LogEvent = iota
	LogEventSynchronizationCompleted
	LogEventChangeSynchronized
	LogEventConflictDetected
	LogEventRetryStarted
	LogEventRetryCompleted
	LogEventWarningRaised
	LogEventOperationFailed
	LogEventPersistentDestinationFailure
	LogEventConsoleDestinationFailure
)
