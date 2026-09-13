package gosync

import (
	"fmt"
	"os"
	"path/filepath"
)

func resolveLogDirectory() (string, error) {
	userConfigurationDirectory, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("find user application-data directory: %w", err)
	}

	return filepath.Join(userConfigurationDirectory, "gosync", "logs"), nil
}

func logLocationOverlapsRoots(logLocation string, roots RootPaths) (bool, error) {
	normalizedLogLocation, err := normalizeRootPath(logLocation)
	if err != nil {
		return false, fmt.Errorf("normalize log location: %w", err)
	}

	for _, synchronizationRoot := range []string{roots.First, roots.Second} {
		normalizedSynchronizationRoot, err := normalizeRootPath(synchronizationRoot)
		if err != nil {
			return false, fmt.Errorf("normalize synchronization root: %w", err)
		}

		containsLogLocation, err := rootContains(normalizedSynchronizationRoot, normalizedLogLocation)
		if err != nil {
			return false, err
		}
		if containsLogLocation {
			return true, nil
		}
	}

	return false, nil
}
