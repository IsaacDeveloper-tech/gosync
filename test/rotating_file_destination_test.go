package gosync_test

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestRotatingFileDestinationCreatesAndAppendsActiveFile(t *testing.T) {
	destination := newRotatingFileDestination(t.TempDir())
	activeFilePath := destination.ActiveFilePath()
	existingRecord := "existing record\n"
	newRecord := "new record\n"

	if err := os.WriteFile(activeFilePath, []byte(existingRecord), 0o600); err != nil {
		t.Fatalf("WriteFile(existing active file) error = %v", err)
	}
	if err := destination.Write(newRecord); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	activeFileContents, err := os.ReadFile(activeFilePath)
	if err != nil {
		t.Fatalf("ReadFile(active file) error = %v", err)
	}
	if string(activeFileContents) != existingRecord+newRecord {
		t.Fatalf("active file contents = %q, want %q", activeFileContents, existingRecord+newRecord)
	}
}

func TestRotatingFileDestinationReturnsActiveFileCreationErrors(t *testing.T) {
	temporaryDirectory := t.TempDir()
	blockedPath := filepath.Join(temporaryDirectory, "blocked")
	if err := os.WriteFile(blockedPath, []byte("not a directory"), 0o600); err != nil {
		t.Fatalf("WriteFile(blocked path) error = %v", err)
	}

	destination := newRotatingFileDestination(filepath.Join(blockedPath, "logs"))
	if err := destination.Write("record\n"); err == nil {
		t.Fatal("Write() error = nil, want active file creation error")
	}
}

func TestRotatingFileDestinationRotatesBeforeActiveFileExceedsTenMegabytes(t *testing.T) {
	destination := newRotatingFileDestination(t.TempDir())
	activeFilePath := destination.ActiveFilePath()
	archiveFilePath := activeFilePath + ".1"
	existingContents := make([]byte, 10_000_000-1)
	for index := range existingContents {
		existingContents[index] = 'x'
	}
	if err := os.WriteFile(activeFilePath, existingContents, 0o600); err != nil {
		t.Fatalf("WriteFile(active file) error = %v", err)
	}

	triggeringRecord := "trigger\n"
	if err := destination.Write(triggeringRecord); err != nil {
		t.Fatalf("Write(triggeringRecord) error = %v", err)
	}

	archiveContents, err := os.ReadFile(archiveFilePath)
	if err != nil {
		t.Fatalf("ReadFile(archive file) error = %v", err)
	}
	if len(archiveContents) != len(existingContents) {
		t.Fatalf("archive size = %d, want %d", len(archiveContents), len(existingContents))
	}
	activeContents, err := os.ReadFile(activeFilePath)
	if err != nil {
		t.Fatalf("ReadFile(rotated active file) error = %v", err)
	}
	if string(activeContents) != triggeringRecord {
		t.Fatalf("rotated active contents = %q, want %q", activeContents, triggeringRecord)
	}
}

func TestRotatingFileDestinationRetainsActiveFileAndFourNewestArchives(t *testing.T) {
	destination := newRotatingFileDestination(t.TempDir())
	activeFilePath := destination.ActiveFilePath()
	maximumActiveFileSize := int64(10_000_000)

	for generation := 1; generation <= 5; generation++ {
		if err := os.WriteFile(activeFilePath, []byte(fmt.Sprintf("generation-%d", generation)), 0o600); err != nil {
			t.Fatalf("WriteFile(generation %d) error = %v", generation, err)
		}
		if err := os.Truncate(activeFilePath, maximumActiveFileSize-1); err != nil {
			t.Fatalf("Truncate(generation %d) error = %v", generation, err)
		}
		if err := destination.Write(fmt.Sprintf("record-%d\n", generation)); err != nil {
			t.Fatalf("Write(generation %d) error = %v", generation, err)
		}
	}

	for archiveGeneration := 1; archiveGeneration <= 4; archiveGeneration++ {
		archivePath := fmt.Sprintf("%s.%d", activeFilePath, archiveGeneration)
		archiveContents, err := os.ReadFile(archivePath)
		if err != nil {
			t.Fatalf("ReadFile(archive %d) error = %v", archiveGeneration, err)
		}
		expectedGeneration := 6 - archiveGeneration
		if string(archiveContents[:len(fmt.Sprintf("generation-%d", expectedGeneration))]) != fmt.Sprintf("generation-%d", expectedGeneration) {
			t.Fatalf("archive %d starts with %q, want generation %d", archiveGeneration, archiveContents[:20], expectedGeneration)
		}
	}
	if _, err := os.Stat(activeFilePath + ".5"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("oldest archive error = %v, want file not to exist", err)
	}
	activeContents, err := os.ReadFile(activeFilePath)
	if err != nil {
		t.Fatalf("ReadFile(active file) error = %v", err)
	}
	if string(activeContents) != "record-5\n" {
		t.Fatalf("active contents = %q, want %q", activeContents, "record-5\n")
	}
}
