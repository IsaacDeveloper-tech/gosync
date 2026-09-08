package gosync_test

import (
	"path/filepath"
	"testing"
)

func TestValidateRootPathsNormalizesPaths(t *testing.T) {
	temporaryDirectory := t.TempDir()
	firstInput := filepath.Join(temporaryDirectory, "first", "..", "first")
	secondInput := filepath.Join(temporaryDirectory, "second")

	roots, err := validateRootPaths(firstInput, secondInput)
	if err != nil {
		t.Fatalf("validateRootPaths() error = %v", err)
	}

	wantFirst := filepath.Join(temporaryDirectory, "first")
	wantSecond := filepath.Join(temporaryDirectory, "second")
	if roots.First != wantFirst || roots.Second != wantSecond {
		t.Fatalf("validateRootPaths() = %+v, want first %q and second %q", roots, wantFirst, wantSecond)
	}
}

func TestValidateRootPathsRejectsIdenticalRoots(t *testing.T) {
	root := t.TempDir()

	_, err := validateRootPaths(root, filepath.Join(root, "."))
	if err == nil {
		t.Fatal("validateRootPaths() error = nil, want an error for identical roots")
	}
}

func TestValidateRootPathsRejectsNestedRoots(t *testing.T) {
	parentRoot := t.TempDir()
	nestedRoot := filepath.Join(parentRoot, "nested")

	testCases := []struct {
		name   string
		first  string
		second string
	}{
		{name: "second is nested in first", first: parentRoot, second: nestedRoot},
		{name: "first is nested in second", first: nestedRoot, second: parentRoot},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := validateRootPaths(testCase.first, testCase.second)
			if err == nil {
				t.Fatal("validateRootPaths() error = nil, want an error for nested roots")
			}
		})
	}
}

func TestValidateRootPathsAllowsSiblingRoots(t *testing.T) {
	temporaryDirectory := t.TempDir()
	firstRoot := filepath.Join(temporaryDirectory, "first")
	secondRoot := filepath.Join(temporaryDirectory, "second")

	_, err := validateRootPaths(firstRoot, secondRoot)
	if err != nil {
		t.Fatalf("validateRootPaths() error = %v, want nil for sibling roots", err)
	}
}

func TestValidateRootPathsRejectsEmptyRoot(t *testing.T) {
	_, err := validateRootPaths("", t.TempDir())
	if err == nil {
		t.Fatal("validateRootPaths() error = nil, want an error for an empty root")
	}
}
