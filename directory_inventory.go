package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

type DirectoryInventory map[string]SynchronizationEntry

func buildDirectoryInventory(root string) (DirectoryInventory, error) {
	inventory := make(DirectoryInventory)

	err := filepath.WalkDir(root, func(path string, directoryEntry fs.DirEntry, walkError error) error {
		if walkError != nil {
			return walkError
		}
		if path == root {
			return nil
		}

		relativePath, err := filepath.Rel(root, path)
		if err != nil {
			return fmt.Errorf("get relative path for %q: %w", path, err)
		}
		entryInfo, err := directoryEntry.Info()
		if err != nil {
			return fmt.Errorf("read entry information for %q: %w", path, err)
		}

		entry := SynchronizationEntry{
			RelativePath:     relativePath,
			ModificationTime: entryInfo.ModTime(),
		}
		switch {
		case entryInfo.IsDir():
			entry.Kind = EntryKindDirectory
		case entryInfo.Mode().IsRegular():
			contentDigest, err := fileContentDigest(path)
			if err != nil {
				return err
			}
			entry.Kind = EntryKindFile
			entry.ContentDigest = contentDigest
		default:
			return fmt.Errorf("unsupported filesystem entry %q", path)
		}

		inventory[relativePath] = entry
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("build directory inventory for %q: %w", root, err)
	}

	return inventory, nil
}

func compareDirectoryInventories(firstInventory, secondInventory DirectoryInventory) []EntryComparison {
	paths := make(map[string]struct{}, len(firstInventory)+len(secondInventory))
	for path := range firstInventory {
		paths[path] = struct{}{}
	}
	for path := range secondInventory {
		paths[path] = struct{}{}
	}

	sortedPaths := make([]string, 0, len(paths))
	for path := range paths {
		sortedPaths = append(sortedPaths, path)
	}
	sort.Strings(sortedPaths)

	comparisons := make([]EntryComparison, 0, len(sortedPaths))
	for _, path := range sortedPaths {
		comparisons = append(comparisons, EntryComparison{
			RelativePath: path,
			First:        inventoryEntry(firstInventory, path),
			Second:       inventoryEntry(secondInventory, path),
		})
	}

	return comparisons
}

func inventoryEntry(inventory DirectoryInventory, path string) *SynchronizationEntry {
	entry, found := inventory[path]
	if !found {
		return nil
	}

	return &entry
}

func directoryInventoriesHaveEquivalentContents(firstInventory, secondInventory DirectoryInventory) bool {
	if len(firstInventory) != len(secondInventory) {
		return false
	}

	for path, firstEntry := range firstInventory {
		secondEntry, found := secondInventory[path]
		if !found || firstEntry.RelativePath != secondEntry.RelativePath || firstEntry.Kind != secondEntry.Kind {
			return false
		}
		if firstEntry.Kind == EntryKindFile && firstEntry.ContentDigest != secondEntry.ContentDigest {
			return false
		}
	}

	return true
}

func fileContentDigest(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open file %q: %w", path, err)
	}

	digest := sha256.New()
	_, copyError := io.Copy(digest, file)
	closeError := file.Close()
	if copyError != nil {
		return "", fmt.Errorf("read file %q: %w", path, copyError)
	}
	if closeError != nil {
		return "", fmt.Errorf("close file %q: %w", path, closeError)
	}

	return hex.EncodeToString(digest.Sum(nil)), nil
}
