package gosync_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestBidirectionalSynchronizationPersistsStateAcrossAdditionsModificationsAndDeletions(t *testing.T) {
	temporaryDirectory := t.TempDir()
	firstRoot := filepath.Join(temporaryDirectory, "first")
	secondRoot := filepath.Join(temporaryDirectory, "second")
	if err := os.MkdirAll(firstRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll(firstRoot) error = %v", err)
	}
	if err := os.MkdirAll(secondRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll(secondRoot) error = %v", err)
	}
	if err := os.MkdirAll(filepath.Join(firstRoot, "archive"), 0o755); err != nil {
		t.Fatalf("MkdirAll(archive) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(firstRoot, "notes.txt"), []byte("first version"), 0o644); err != nil {
		t.Fatalf("WriteFile(notes.txt) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(firstRoot, "archive", "old.txt"), []byte("old"), 0o644); err != nil {
		t.Fatalf("WriteFile(old.txt) error = %v", err)
	}

	roots := RootPaths{First: firstRoot, Second: secondRoot}
	store := newConfirmedStateStoreAt(filepath.Join(temporaryDirectory, "state"))
	if err := synchronizeDirectories(roots, store, bytes.NewBufferString("1\n"), bytes.NewBuffer(nil)); err != nil {
		t.Fatalf("initial synchronizeDirectories() error = %v", err)
	}
	assertFileContent(t, filepath.Join(secondRoot, "notes.txt"), "first version")
	assertFileContent(t, filepath.Join(secondRoot, "archive", "old.txt"), "old")

	if err := os.WriteFile(filepath.Join(firstRoot, "notes.txt"), []byte("updated version"), 0o644); err != nil {
		t.Fatalf("WriteFile(updated notes.txt) error = %v", err)
	}
	futureModificationTime := time.Now().Add(time.Hour)
	if err := os.Chtimes(filepath.Join(firstRoot, "notes.txt"), futureModificationTime, futureModificationTime); err != nil {
		t.Fatalf("Chtimes(notes.txt) error = %v", err)
	}
	if err := os.Remove(filepath.Join(firstRoot, "archive", "old.txt")); err != nil {
		t.Fatalf("Remove(old.txt) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(secondRoot, "new.txt"), []byte("second addition"), 0o644); err != nil {
		t.Fatalf("WriteFile(new.txt) error = %v", err)
	}

	if err := synchronizeDirectories(roots, store, bytes.NewBuffer(nil), bytes.NewBuffer(nil)); err != nil {
		t.Fatalf("second synchronizeDirectories() error = %v", err)
	}
	assertFileContent(t, filepath.Join(firstRoot, "notes.txt"), "updated version")
	assertFileContent(t, filepath.Join(secondRoot, "notes.txt"), "updated version")
	assertFileContent(t, filepath.Join(firstRoot, "new.txt"), "second addition")
	assertPathDoesNotExist(t, filepath.Join(firstRoot, "archive", "old.txt"))
	assertPathDoesNotExist(t, filepath.Join(secondRoot, "archive", "old.txt"))

	firstInventory, err := buildDirectoryInventory(firstRoot)
	if err != nil {
		t.Fatalf("buildDirectoryInventory(firstRoot) error = %v", err)
	}
	secondInventory, err := buildDirectoryInventory(secondRoot)
	if err != nil {
		t.Fatalf("buildDirectoryInventory(secondRoot) error = %v", err)
	}
	if !directoryInventoriesHaveEquivalentContents(firstInventory, secondInventory) {
		t.Fatal("directories are not equivalent after the complete synchronization cycle")
	}
}
