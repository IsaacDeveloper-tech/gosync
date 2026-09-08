package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExecuteSynchronizationPlanAppliesCopiesAndDeletions(t *testing.T) {
	temporaryDirectory := t.TempDir()
	firstRoot := filepath.Join(temporaryDirectory, "first")
	secondRoot := filepath.Join(temporaryDirectory, "second")
	if err := os.MkdirAll(firstRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll(firstRoot) error = %v", err)
	}
	if err := os.MkdirAll(secondRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll(secondRoot) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(firstRoot, "new.txt"), []byte("new content"), 0o644); err != nil {
		t.Fatalf("WriteFile(new.txt) error = %v", err)
	}
	if err := os.MkdirAll(filepath.Join(firstRoot, "archive"), 0o755); err != nil {
		t.Fatalf("MkdirAll(archive) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(firstRoot, "archive", "notes.txt"), []byte("notes"), 0o644); err != nil {
		t.Fatalf("WriteFile(notes.txt) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(firstRoot, "delete-from-first.txt"), []byte("delete"), 0o644); err != nil {
		t.Fatalf("WriteFile(delete-from-first.txt) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(secondRoot, "delete-from-second.txt"), []byte("delete"), 0o644); err != nil {
		t.Fatalf("WriteFile(delete-from-second.txt) error = %v", err)
	}

	plan := SynchronizationPlan{Actions: []SynchronizationAction{
		{Kind: SynchronizationActionCopyToSecond, RelativePath: "new.txt"},
		{Kind: SynchronizationActionCopyToSecond, RelativePath: "archive"},
		{Kind: SynchronizationActionDeleteFromFirst, RelativePath: "delete-from-first.txt"},
		{Kind: SynchronizationActionDeleteFromSecond, RelativePath: "delete-from-second.txt"},
	}}
	result := executeSynchronizationPlan(RootPaths{First: firstRoot, Second: secondRoot}, plan, SynchronizationExecutionOptions{})
	if !result.Completed || result.Failure != nil {
		t.Fatalf("executeSynchronizationPlan() result = %+v, want completed result", result)
	}

	assertFileContent(t, filepath.Join(secondRoot, "new.txt"), "new content")
	assertFileContent(t, filepath.Join(secondRoot, "archive", "notes.txt"), "notes")
	assertPathDoesNotExist(t, filepath.Join(firstRoot, "delete-from-first.txt"))
	assertPathDoesNotExist(t, filepath.Join(secondRoot, "delete-from-second.txt"))
}

func TestExecuteSynchronizationPlanStopsOnNonLockedError(t *testing.T) {
	temporaryDirectory := t.TempDir()
	firstRoot := filepath.Join(temporaryDirectory, "first")
	secondRoot := filepath.Join(temporaryDirectory, "second")
	if err := os.MkdirAll(firstRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll(firstRoot) error = %v", err)
	}
	if err := os.MkdirAll(secondRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll(secondRoot) error = %v", err)
	}

	plan := SynchronizationPlan{Actions: []SynchronizationAction{
		{Kind: SynchronizationActionCopyToSecond, RelativePath: "missing.txt"},
	}}
	result := executeSynchronizationPlan(RootPaths{First: firstRoot, Second: secondRoot}, plan, SynchronizationExecutionOptions{
		IsLockedError: func(error) bool { return false },
	})
	if result.Completed {
		t.Fatal("executeSynchronizationPlan() completed = true, want false")
	}
	if result.Failure == nil {
		t.Fatal("executeSynchronizationPlan() failure = nil, want an execution error")
	}
}

func assertFileContent(t *testing.T, path, want string) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}
	if string(content) != want {
		t.Fatalf("file %q content = %q, want %q", path, content, want)
	}
}

func assertPathDoesNotExist(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("Stat(%q) error = %v, want path not to exist", path, err)
	}
}
