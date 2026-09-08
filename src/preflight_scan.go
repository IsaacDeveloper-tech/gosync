package gosync

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

func scanRootsForUnsupportedEntries(roots RootPaths) error {
	for _, root := range []string{roots.First, roots.Second} {
		if err := scanRootForUnsupportedEntries(root); err != nil {
			return fmt.Errorf("scan root %q: %w", root, err)
		}
	}

	return nil
}

func scanRootForUnsupportedEntries(root string) error {
	if _, err := os.Lstat(root); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	return filepath.WalkDir(root, func(path string, directoryEntry fs.DirEntry, walkError error) error {
		if walkError != nil {
			return walkError
		}

		entryInfo, err := directoryEntry.Info()
		if err != nil {
			return err
		}
		if !isSupportedEntryMode(entryInfo.Mode()) {
			return fmt.Errorf("unsupported filesystem entry %q", path)
		}

		return nil
	})
}

func isSupportedEntryMode(mode fs.FileMode) bool {
	return mode.IsRegular() || mode.IsDir()
}
