package gosync

import (
	"fmt"
	"path/filepath"
	"strings"
)

type RootPaths struct {
	First  string
	Second string
}

func validateRootPaths(firstRoot, secondRoot string) (RootPaths, error) {
	normalizedFirstRoot, err := normalizeRootPath(firstRoot)
	if err != nil {
		return RootPaths{}, fmt.Errorf("normalize first root: %w", err)
	}

	normalizedSecondRoot, err := normalizeRootPath(secondRoot)
	if err != nil {
		return RootPaths{}, fmt.Errorf("normalize second root: %w", err)
	}

	firstContainsSecond, err := rootContains(normalizedFirstRoot, normalizedSecondRoot)
	if err != nil {
		return RootPaths{}, err
	}
	secondContainsFirst, err := rootContains(normalizedSecondRoot, normalizedFirstRoot)
	if err != nil {
		return RootPaths{}, err
	}
	if firstContainsSecond || secondContainsFirst {
		return RootPaths{}, fmt.Errorf("synchronization roots must be distinct and not nested")
	}

	return RootPaths{
		First:  normalizedFirstRoot,
		Second: normalizedSecondRoot,
	}, nil
}

func normalizeRootPath(root string) (string, error) {
	if root == "" {
		return "", fmt.Errorf("root path must not be empty")
	}

	normalizedRoot, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}

	return filepath.Clean(normalizedRoot), nil
}

func rootContains(parentRoot, candidateRoot string) (bool, error) {
	if !strings.EqualFold(filepath.VolumeName(parentRoot), filepath.VolumeName(candidateRoot)) {
		return false, nil
	}

	relativePath, err := filepath.Rel(parentRoot, candidateRoot)
	if err != nil {
		return false, fmt.Errorf("compare synchronization roots: %w", err)
	}
	if relativePath == "." {
		return true, nil
	}
	if relativePath == ".." || strings.HasPrefix(relativePath, ".."+string(filepath.Separator)) {
		return false, nil
	}

	return true, nil
}
