package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureRootDirectoriesCreatesMissingRoot(t *testing.T) {
	temporaryDirectory := t.TempDir()
	firstRoot := filepath.Join(temporaryDirectory, "first")
	secondRoot := filepath.Join(temporaryDirectory, "second")
	if err := os.Mkdir(firstRoot, 0o755); err != nil {
		t.Fatalf("Mkdir() error = %v", err)
	}
	markerPath := filepath.Join(firstRoot, "marker.txt")
	if err := os.WriteFile(markerPath, []byte("existing"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	err := ensureRootDirectories(RootPaths{First: firstRoot, Second: secondRoot})
	if err != nil {
		t.Fatalf("ensureRootDirectories() error = %v", err)
	}

	secondInfo, err := os.Stat(secondRoot)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if !secondInfo.IsDir() {
		t.Fatalf("created root %q is not a directory", secondRoot)
	}
	if _, err := os.Stat(markerPath); err != nil {
		t.Fatalf("existing root content was not preserved: %v", err)
	}
}

func TestEnsureRootDirectoriesCreatesBothMissingRoots(t *testing.T) {
	temporaryDirectory := t.TempDir()
	roots := RootPaths{
		First:  filepath.Join(temporaryDirectory, "first"),
		Second: filepath.Join(temporaryDirectory, "second"),
	}

	err := ensureRootDirectories(roots)
	if err != nil {
		t.Fatalf("ensureRootDirectories() error = %v", err)
	}

	for _, root := range []string{roots.First, roots.Second} {
		rootInfo, err := os.Stat(root)
		if err != nil {
			t.Fatalf("Stat(%q) error = %v", root, err)
		}
		if !rootInfo.IsDir() {
			t.Fatalf("created root %q is not a directory", root)
		}
	}
}

func TestEnsureRootDirectoriesRejectsFileRoot(t *testing.T) {
	temporaryDirectory := t.TempDir()
	fileRoot := filepath.Join(temporaryDirectory, "file-root")
	if err := os.WriteFile(fileRoot, []byte("file"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	directoryRoot := filepath.Join(temporaryDirectory, "directory-root")
	if err := os.Mkdir(directoryRoot, 0o755); err != nil {
		t.Fatalf("Mkdir() error = %v", err)
	}

	err := ensureRootDirectories(RootPaths{First: fileRoot, Second: directoryRoot})
	if err == nil {
		t.Fatal("ensureRootDirectories() error = nil, want an error for a file root")
	}
}
