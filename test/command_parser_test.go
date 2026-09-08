package gosync_test

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestParseWatchCommandRequiresWatchAndTwoRoots(t *testing.T) {
	temporaryDirectory := t.TempDir()
	firstRoot := filepath.Join(temporaryDirectory, "first")
	secondRoot := filepath.Join(temporaryDirectory, "second")

	roots, err := parseWatchCommand([]string{"watch", firstRoot, secondRoot})
	if err != nil {
		t.Fatalf("parseWatchCommand() error = %v", err)
	}
	if roots.First != firstRoot || roots.Second != secondRoot {
		t.Fatalf("parseWatchCommand() = %+v, want first %q and second %q", roots, firstRoot, secondRoot)
	}
}

func TestParseWatchCommandShowsUsageForInvalidArguments(t *testing.T) {
	testCases := [][]string{
		{},
		{"watch"},
		{"watch", "first"},
		{"watch", "first", "second", "extra"},
		{"sync", "first", "second"},
	}

	for _, arguments := range testCases {
		t.Run(strings.Join(arguments, "-"), func(t *testing.T) {
			_, err := parseWatchCommand(arguments)
			if err == nil {
				t.Fatal("parseWatchCommand() error = nil, want usage error")
			}
			if !strings.Contains(err.Error(), watchUsage) {
				t.Fatalf("parseWatchCommand() error = %q, want usage %q", err, watchUsage)
			}
		})
	}
}
