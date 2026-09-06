package main

import (
	"fmt"
	"os"
)

func ensureRootDirectories(roots RootPaths) error {
	for _, root := range []string{roots.First, roots.Second} {
		if err := os.MkdirAll(root, 0o755); err != nil {
			return fmt.Errorf("create root directory %q: %w", root, err)
		}
	}

	return nil
}
