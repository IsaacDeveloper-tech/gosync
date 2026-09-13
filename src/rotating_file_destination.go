package gosync

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const (
	logActiveFileName        = "gosync.log"
	logMaximumActiveFileSize = int64(10_000_000)
	logMaximumArchiveCount   = 4
)

type RotatingFileDestination struct {
	directory string
}

func newRotatingFileDestination(directory string) RotatingFileDestination {
	return RotatingFileDestination{directory: directory}
}

func (destination RotatingFileDestination) ActiveFilePath() string {
	return filepath.Join(destination.directory, logActiveFileName)
}

func (destination RotatingFileDestination) Write(formattedRecord string) error {
	if err := os.MkdirAll(destination.directory, 0o700); err != nil {
		return fmt.Errorf("create log directory: %w", err)
	}

	activeFilePath := destination.ActiveFilePath()
	activeFileSize, err := getActiveLogFileSize(activeFilePath)
	if err != nil {
		return err
	}
	if activeFileSize > 0 && activeFileSize+int64(len(formattedRecord)) > logMaximumActiveFileSize {
		if err := rotateActiveLogFile(activeFilePath); err != nil {
			return err
		}
	}

	return appendToActiveLogFile(activeFilePath, formattedRecord)
}

func getActiveLogFileSize(activeFilePath string) (int64, error) {
	fileInformation, err := os.Stat(activeFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("inspect active log file: %w", err)
	}
	if fileInformation.IsDir() {
		return 0, fmt.Errorf("active log path is a directory: %s", activeFilePath)
	}

	return fileInformation.Size(), nil
}

func appendToActiveLogFile(activeFilePath, formattedRecord string) error {
	activeFile, err := os.OpenFile(activeFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return fmt.Errorf("open active log file: %w", err)
	}

	writtenBytes, writeErr := activeFile.Write([]byte(formattedRecord))
	closeErr := activeFile.Close()
	if writeErr != nil {
		return fmt.Errorf("append active log file: %w", writeErr)
	}
	if writtenBytes != len(formattedRecord) {
		return fmt.Errorf("append active log file: %w", io.ErrShortWrite)
	}
	if closeErr != nil {
		return fmt.Errorf("close active log file: %w", closeErr)
	}

	return nil
}

func rotateActiveLogFile(activeFilePath string) error {
	oldestArchivePath := archiveLogFilePath(activeFilePath, logMaximumArchiveCount)
	if err := os.Remove(oldestArchivePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove oldest log archive: %w", err)
	}

	for archiveGeneration := logMaximumArchiveCount - 1; archiveGeneration >= 1; archiveGeneration-- {
		sourceArchivePath := archiveLogFilePath(activeFilePath, archiveGeneration)
		targetArchivePath := archiveLogFilePath(activeFilePath, archiveGeneration+1)
		if err := os.Rename(sourceArchivePath, targetArchivePath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("shift log archive %d: %w", archiveGeneration, err)
		}
	}

	if err := os.Rename(activeFilePath, archiveLogFilePath(activeFilePath, 1)); err != nil {
		return fmt.Errorf("archive active log file: %w", err)
	}

	return nil
}

func archiveLogFilePath(activeFilePath string, generation int) string {
	return fmt.Sprintf("%s.%d", activeFilePath, generation)
}
