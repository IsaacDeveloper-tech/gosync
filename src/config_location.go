package gosync

import (
	"fmt"
	"os"
	"path/filepath"
)

func resolveConfigurationFilePath() (string, error) {
	userConfigurationDirectory, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("find user configuration directory: %w", err)
	}
	return filepath.Join(userConfigurationDirectory, "gosync", "config.json"), nil
}
