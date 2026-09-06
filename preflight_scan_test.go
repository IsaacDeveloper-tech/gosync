package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScanRootsForUnsupportedEntriesAllowsFilesAndDirectories(t *testing.T) {
	temporaryDirectory := t.TempDir()
	firstRoot := filepath.Join(temporaryDirectory, "first")
	secondRoot := filepath.Join(temporaryDirectory, "second")
	if err := os.MkdirAll(filepath.Join(firstRoot, "nested"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.MkdirAll(secondRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(firstRoot, "nested", "notes.txt"), []byte("notes"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	err := scanRootsForUnsupportedEntries(RootPaths{First: firstRoot, Second: secondRoot})
	if err != nil {
		t.Fatalf("scanRootsForUnsupportedEntries() error = %v", err)
	}
}

func TestScanRootsForUnsupportedEntriesAllowsMissingRoot(t *testing.T) {
	temporaryDirectory := t.TempDir()
	missingRoot := filepath.Join(temporaryDirectory, "missing")
	existingRoot := filepath.Join(temporaryDirectory, "existing")
	if err := os.Mkdir(existingRoot, 0o755); err != nil {
		t.Fatalf("Mkdir() error = %v", err)
	}

	err := scanRootsForUnsupportedEntries(RootPaths{First: missingRoot, Second: existingRoot})
	if err != nil {
		t.Fatalf("scanRootsForUnsupportedEntries() error = %v", err)
	}
}

func TestScanRootsForUnsupportedEntriesRejectsSymbolicLinks(t *testing.T) {
	temporaryDirectory := t.TempDir()
	firstRoot := filepath.Join(temporaryDirectory, "first")
	secondRoot := filepath.Join(temporaryDirectory, "second")
	if err := os.MkdirAll(firstRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.MkdirAll(secondRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	targetPath := filepath.Join(firstRoot, "target.txt")
	if err := os.WriteFile(targetPath, []byte("target"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	symlinkPath := filepath.Join(secondRoot, "linked.txt")
	if err := os.Symlink(targetPath, symlinkPath); err != nil {
		t.Skipf("symbolic links are unavailable: %v", err)
	}

	err := scanRootsForUnsupportedEntries(RootPaths{First: firstRoot, Second: secondRoot})
	if err == nil {
		t.Fatal("scanRootsForUnsupportedEntries() error = nil, want an error for a symbolic link")
	}
	if !strings.Contains(err.Error(), symlinkPath) {
		t.Fatalf("scanRootsForUnsupportedEntries() error = %q, want path %q", err, symlinkPath)
	}
}

func TestIsSupportedEntryMode(t *testing.T) {
	testCases := []struct {
		name string
		mode fs.FileMode
		want bool
	}{
		{name: "regular file", mode: 0, want: true},
		{name: "directory", mode: fs.ModeDir, want: true},
		{name: "symbolic link", mode: fs.ModeSymlink, want: false},
		{name: "named pipe", mode: fs.ModeNamedPipe, want: false},
		{name: "device", mode: fs.ModeDevice, want: false},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			if got := isSupportedEntryMode(testCase.mode); got != testCase.want {
				t.Fatalf("isSupportedEntryMode(%v) = %t, want %t", testCase.mode, got, testCase.want)
			}
		})
	}
}
