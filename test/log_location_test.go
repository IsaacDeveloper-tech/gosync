package gosync_test

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveLogDirectoryUsesPerUserConfigurationLocation(t *testing.T) {
	userConfigurationDirectory, err := os.UserConfigDir()
	if err != nil {
		t.Fatalf("UserConfigDir() error = %v", err)
	}

	originalWorkingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	defer os.Chdir(originalWorkingDirectory)

	firstLocation, err := resolveLogDirectory()
	if err != nil {
		t.Fatalf("resolveLogDirectory() error = %v", err)
	}

	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}
	secondLocation, err := resolveLogDirectory()
	if err != nil {
		t.Fatalf("resolveLogDirectory() after changing directory error = %v", err)
	}

	expectedLocation := filepath.Join(userConfigurationDirectory, "gosync", "logs")
	if firstLocation != expectedLocation {
		t.Fatalf("first log location = %q, want %q", firstLocation, expectedLocation)
	}
	if secondLocation != expectedLocation {
		t.Fatalf("second log location = %q, want %q", secondLocation, expectedLocation)
	}
}

func TestLogLocationOverlapsEitherSynchronizationRootWhenEqualOrContained(t *testing.T) {
	temporaryDirectory := t.TempDir()
	roots := RootPaths{
		First:  filepath.Join(temporaryDirectory, "first"),
		Second: filepath.Join(temporaryDirectory, "second"),
	}

	testCases := []struct {
		name          string
		logLocation   string
		overlapsRoots bool
	}{
		{name: "equals first root", logLocation: roots.First, overlapsRoots: true},
		{name: "is contained in first root", logLocation: filepath.Join(roots.First, "logs"), overlapsRoots: true},
		{name: "equals second root", logLocation: roots.Second, overlapsRoots: true},
		{name: "is contained in second root", logLocation: filepath.Join(roots.Second, "logs"), overlapsRoots: true},
		{name: "is outside both roots", logLocation: filepath.Join(temporaryDirectory, "logs"), overlapsRoots: false},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			got, err := logLocationOverlapsRoots(testCase.logLocation, roots)
			if err != nil {
				t.Fatalf("logLocationOverlapsRoots() error = %v", err)
			}
			if got != testCase.overlapsRoots {
				t.Fatalf("overlap = %t, want %t", got, testCase.overlapsRoots)
			}
		})
	}
}
