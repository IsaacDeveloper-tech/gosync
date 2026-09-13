package gosync

import "fmt"

func logSynchronizedAction(logger *LoggingCoordinator, action SynchronizationAction) error {
	message := "synchronization change completed"
	switch action.Kind {
	case SynchronizationActionCopyToFirst:
		message = "change copied to first directory"
	case SynchronizationActionCopyToSecond:
		message = "change copied to second directory"
	case SynchronizationActionDeleteFromFirst:
		message = "change deleted from first directory"
	case SynchronizationActionDeleteFromSecond:
		message = "change deleted from second directory"
	}

	return logger.Log(LogEntry{
		Severity: LogSeverityInfo,
		Event:    LogEventChangeSynchronized,
		Message:  message,
		Context: map[string]string{
			"path":   action.RelativePath,
			"action": fmt.Sprintf("%d", action.Kind),
		},
	})
}

func logRetryEvent(
	logger *LoggingCoordinator,
	event LogEvent,
	severity LogSeverity,
	filePath string,
	message string,
) error {
	return logger.Log(LogEntry{
		Severity: severity,
		Event:    event,
		Message:  message,
		Context:  map[string]string{"path": filePath},
	})
}

func logSynchronizationConflicts(logger *LoggingCoordinator, comparisons []EntryComparison) error {
	for _, comparison := range comparisons {
		if !comparisonHasConflict(comparison) {
			continue
		}
		if err := logger.Log(LogEntry{
			Severity: LogSeverityWarn,
			Event:    LogEventConflictDetected,
			Message:  "synchronization conflict detected",
			Context:  map[string]string{"path": comparison.RelativePath},
		}); err != nil {
			return err
		}
	}

	return nil
}

func comparisonHasConflict(comparison EntryComparison) bool {
	if comparison.First == nil || comparison.Second == nil {
		return false
	}
	if comparison.First.Kind != comparison.Second.Kind {
		return true
	}
	return comparison.First.Kind == EntryKindFile && comparison.First.ContentDigest != comparison.Second.ContentDigest
}
