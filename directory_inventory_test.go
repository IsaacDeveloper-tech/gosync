package main

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestBuildDirectoryInventoryRecordsPathsTypesContentAndModificationTime(t *testing.T) {
	root := t.TempDir()
	directoryPath := filepath.Join(root, "archive")
	if err := os.Mkdir(directoryPath, 0o755); err != nil {
		t.Fatalf("Mkdir() error = %v", err)
	}
	filePath := filepath.Join(directoryPath, "notes.txt")
	fileContent := []byte("notes")
	if err := os.WriteFile(filePath, fileContent, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	modificationTime := time.Now().Add(-time.Hour).Truncate(time.Second)
	if err := os.Chtimes(filePath, modificationTime, modificationTime); err != nil {
		t.Fatalf("Chtimes() error = %v", err)
	}

	inventory, err := buildDirectoryInventory(root)
	if err != nil {
		t.Fatalf("buildDirectoryInventory() error = %v", err)
	}

	directoryEntry, found := inventory["archive"]
	if !found || directoryEntry.Kind != EntryKindDirectory {
		t.Fatalf("directory entry = %+v, found = %t, want a directory", directoryEntry, found)
	}
	fileEntry, found := inventory[filepath.Join("archive", "notes.txt")]
	if !found {
		t.Fatal("file entry was not recorded")
	}
	expectedDigest := sha256.Sum256(fileContent)
	if fileEntry.Kind != EntryKindFile {
		t.Fatalf("file entry kind = %v, want %v", fileEntry.Kind, EntryKindFile)
	}
	if fileEntry.ContentDigest != hex.EncodeToString(expectedDigest[:]) {
		t.Fatalf("file content digest = %q, want %q", fileEntry.ContentDigest, hex.EncodeToString(expectedDigest[:]))
	}
	if !fileEntry.ModificationTime.Equal(modificationTime) {
		t.Fatalf("file modification time = %v, want %v", fileEntry.ModificationTime, modificationTime)
	}
}

func TestCompareDirectoryInventoriesRecordsEntriesFromBothDirectories(t *testing.T) {
	firstModificationTime := time.Date(2026, time.September, 6, 10, 0, 0, 0, time.UTC)
	secondModificationTime := firstModificationTime.Add(time.Minute)
	firstInventory := DirectoryInventory{
		"notes.txt": {
			RelativePath:     "notes.txt",
			Kind:             EntryKindFile,
			ContentDigest:    "first-content",
			ModificationTime: firstModificationTime,
		},
	}
	secondInventory := DirectoryInventory{
		"notes.txt": {
			RelativePath:     "notes.txt",
			Kind:             EntryKindFile,
			ContentDigest:    "second-content",
			ModificationTime: secondModificationTime,
		},
		"archive": {
			RelativePath: "archive",
			Kind:         EntryKindDirectory,
		},
	}

	comparisons := compareDirectoryInventories(firstInventory, secondInventory)
	if len(comparisons) != 2 {
		t.Fatalf("comparison count = %d, want 2", len(comparisons))
	}

	comparisonsByPath := make(map[string]EntryComparison, len(comparisons))
	for _, comparison := range comparisons {
		comparisonsByPath[comparison.RelativePath] = comparison
	}
	notesComparison := comparisonsByPath["notes.txt"]
	if notesComparison.First == nil || notesComparison.Second == nil {
		t.Fatal("notes comparison must contain both entries")
	}
	if !notesComparison.First.ModificationTime.Equal(firstModificationTime) {
		t.Fatalf("first modification time = %v, want %v", notesComparison.First.ModificationTime, firstModificationTime)
	}
	if !notesComparison.Second.ModificationTime.Equal(secondModificationTime) {
		t.Fatalf("second modification time = %v, want %v", notesComparison.Second.ModificationTime, secondModificationTime)
	}
	if comparisonsByPath["archive"].First != nil || comparisonsByPath["archive"].Second == nil {
		t.Fatal("archive comparison must contain only the second entry")
	}
}

func TestDirectoryInventoriesHaveEquivalentContentsIgnoringModificationTimes(t *testing.T) {
	firstInventory := DirectoryInventory{
		"notes.txt": {
			RelativePath:     "notes.txt",
			Kind:             EntryKindFile,
			ContentDigest:    "matching-content",
			ModificationTime: time.Date(2026, time.September, 6, 10, 0, 0, 0, time.UTC),
		},
	}
	secondInventory := DirectoryInventory{
		"notes.txt": {
			RelativePath:     "notes.txt",
			Kind:             EntryKindFile,
			ContentDigest:    "matching-content",
			ModificationTime: time.Date(2026, time.September, 6, 11, 0, 0, 0, time.UTC),
		},
	}

	if !directoryInventoriesHaveEquivalentContents(firstInventory, secondInventory) {
		t.Fatal("directory inventories with matching paths, types, and content must be equivalent")
	}
}

func TestDirectoryInventoriesRejectDifferentContentOrStructure(t *testing.T) {
	matchingInventory := DirectoryInventory{
		"notes.txt": {RelativePath: "notes.txt", Kind: EntryKindFile, ContentDigest: "content"},
	}
	differentContentInventory := DirectoryInventory{
		"notes.txt": {RelativePath: "notes.txt", Kind: EntryKindFile, ContentDigest: "different"},
	}
	differentStructureInventory := DirectoryInventory{
		"archive": {RelativePath: "archive", Kind: EntryKindDirectory},
	}

	if directoryInventoriesHaveEquivalentContents(matchingInventory, differentContentInventory) {
		t.Fatal("inventories with different file content must not be equivalent")
	}
	if directoryInventoriesHaveEquivalentContents(matchingInventory, differentStructureInventory) {
		t.Fatal("inventories with different paths or types must not be equivalent")
	}
}
