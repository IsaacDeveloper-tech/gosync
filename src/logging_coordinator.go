package gosync

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

var ErrLoggingUnavailable = errors.New("logging unavailable")

type LoggingCoordinatorOptions struct {
	ConsoleWriter   io.Writer
	FileDirectory   string
	FileDestination *RotatingFileDestination
	Clock           func() time.Time
	Sanitizer       *LogSanitizer
}

type LoggingCoordinator struct {
	consoleDestination  ConsoleDestination
	fileDestination     *RotatingFileDestination
	sanitizer           *LogSanitizer
	clock               func() time.Time
	consoleAvailable    bool
	persistentAvailable bool
}

func newLoggingCoordinator(options LoggingCoordinatorOptions) (*LoggingCoordinator, error) {
	return buildLoggingCoordinator(options, false)
}

func initializeLoggingCoordinator(roots RootPaths, options LoggingCoordinatorOptions) (*LoggingCoordinator, error) {
	logLocation := options.FileDirectory
	if options.FileDestination != nil {
		logLocation = filepath.Dir(options.FileDestination.ActiveFilePath())
	}
	if logLocation == "" {
		resolvedDirectory, err := resolveLogDirectory()
		if err != nil {
			return nil, err
		}
		logLocation = resolvedDirectory
		options.FileDirectory = resolvedDirectory
	}

	overlapsRoots, err := logLocationOverlapsRoots(logLocation, roots)
	if err != nil {
		return nil, err
	}
	if overlapsRoots {
		coordinator, err := buildLoggingCoordinator(options, true)
		if err != nil {
			return nil, err
		}
		if err := coordinator.writeWarning("persistent log location overlaps a synchronization root"); err != nil {
			return nil, err
		}
		return coordinator, nil
	}

	return buildLoggingCoordinator(options, false)
}

func buildLoggingCoordinator(options LoggingCoordinatorOptions, skipPersistentDestination bool) (*LoggingCoordinator, error) {
	clock := options.Clock
	if clock == nil {
		clock = func() time.Time { return time.Now().UTC() }
	}
	sanitizer := options.Sanitizer
	if sanitizer == nil {
		sanitizer = newLogSanitizer()
	}

	coordinator := &LoggingCoordinator{
		sanitizer: sanitizer,
		clock:     clock,
	}
	if options.ConsoleWriter != nil {
		coordinator.consoleDestination = newConsoleDestination(options.ConsoleWriter)
		coordinator.consoleAvailable = true
	}

	if !skipPersistentDestination {
		if options.FileDestination != nil {
			coordinator.fileDestination = options.FileDestination
			coordinator.persistentAvailable = true
		} else {
			fileDirectory := options.FileDirectory
			if fileDirectory == "" {
				resolvedDirectory, err := resolveLogDirectory()
				if err != nil {
					return nil, err
				}
				fileDirectory = resolvedDirectory
			}
			fileDestination := newRotatingFileDestination(fileDirectory)
			if err := ensureLogDirectory(fileDirectory); err != nil {
				if !coordinator.consoleAvailable {
					return nil, fmt.Errorf("%w: initialize persistent destination: %v", ErrLoggingUnavailable, err)
				}
				if reportErr := coordinator.writePersistentFailure(err); reportErr != nil {
					coordinator.consoleAvailable = false
					return nil, fmt.Errorf("%w: report persistent destination failure: %v", ErrLoggingUnavailable, reportErr)
				}
			} else {
				coordinator.fileDestination = &fileDestination
				coordinator.persistentAvailable = true
			}
		}
	}

	if !coordinator.consoleAvailable && !coordinator.persistentAvailable {
		return nil, ErrLoggingUnavailable
	}

	return coordinator, nil
}

func ensureLogDirectory(directory string) error {
	return os.MkdirAll(directory, 0o700)
}

func (coordinator *LoggingCoordinator) Log(entry LogEntry) error {
	if coordinator == nil {
		return ErrLoggingUnavailable
	}

	sanitizedEntry := coordinator.sanitizer.SanitizeEntry(entry)
	return coordinator.Write(formatLogEntry(sanitizedEntry))
}

func (coordinator *LoggingCoordinator) Write(formattedRecord string) error {
	if coordinator == nil || (!coordinator.consoleAvailable && !coordinator.persistentAvailable) {
		return ErrLoggingUnavailable
	}

	if coordinator.consoleAvailable {
		if err := coordinator.consoleDestination.Write(formattedRecord); err != nil {
			coordinator.consoleAvailable = false
			if coordinator.persistentAvailable {
				if reportErr := coordinator.writeConsoleFailure(err); reportErr != nil {
					coordinator.persistentAvailable = false
				}
			}
		}
	}

	if coordinator.persistentAvailable {
		if err := coordinator.fileDestination.Write(formattedRecord); err != nil {
			coordinator.persistentAvailable = false
			if coordinator.consoleAvailable {
				if reportErr := coordinator.writePersistentFailure(err); reportErr != nil {
					coordinator.consoleAvailable = false
				}
			}
		}
	}

	if !coordinator.consoleAvailable && !coordinator.persistentAvailable {
		return ErrLoggingUnavailable
	}
	return nil
}

func (coordinator *LoggingCoordinator) ConsoleAvailable() bool {
	return coordinator != nil && coordinator.consoleAvailable
}

func (coordinator *LoggingCoordinator) PersistentAvailable() bool {
	return coordinator != nil && coordinator.persistentAvailable
}

func (coordinator *LoggingCoordinator) ActiveFilePath() string {
	if coordinator == nil || coordinator.fileDestination == nil {
		return ""
	}
	return coordinator.fileDestination.ActiveFilePath()
}

func (coordinator *LoggingCoordinator) writePersistentFailure(destinationError error) error {
	if !coordinator.consoleAvailable {
		return ErrLoggingUnavailable
	}

	entry := newLogEntry(
		coordinator.clock,
		LogSeverityError,
		LogEventPersistentDestinationFailure,
		"persistent logging failed and was disabled",
		map[string]string{"error": destinationError.Error()},
	)
	sanitizedEntry := coordinator.sanitizer.SanitizeEntry(entry)
	return coordinator.consoleDestination.Write(formatLogEntry(sanitizedEntry))
}

func (coordinator *LoggingCoordinator) writeConsoleFailure(destinationError error) error {
	if !coordinator.persistentAvailable {
		return ErrLoggingUnavailable
	}

	entry := newLogEntry(
		coordinator.clock,
		LogSeverityError,
		LogEventConsoleDestinationFailure,
		"console logging failed and was disabled",
		map[string]string{"error": destinationError.Error()},
	)
	sanitizedEntry := coordinator.sanitizer.SanitizeEntry(entry)
	return coordinator.fileDestination.Write(formatLogEntry(sanitizedEntry))
}

func (coordinator *LoggingCoordinator) writeWarning(message string) error {
	entry := newLogEntry(
		coordinator.clock,
		LogSeverityWarn,
		LogEventWarningRaised,
		message,
		nil,
	)
	return coordinator.consoleDestination.Write(formatLogEntry(coordinator.sanitizer.SanitizeEntry(entry)))
}
