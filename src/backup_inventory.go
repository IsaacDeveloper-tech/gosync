package gosync

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

type BackupInventoryEntryKind string

const (
	BackupInventoryFile      BackupInventoryEntryKind = "file"
	BackupInventoryDirectory BackupInventoryEntryKind = "directory"
)

type BackupInventoryErrorKind string

const (
	BackupInventoryErrorUnsupported BackupInventoryErrorKind = "unsupported-entry"
	BackupInventoryErrorBlocked     BackupInventoryErrorKind = "blocked-entry"
)

type BackupInventoryError struct {
	Path  string
	Kind  BackupInventoryErrorKind
	Cause error
}

func (backupError *BackupInventoryError) Error() string {
	if backupError == nil {
		return "backup inventory error"
	}
	if backupError.Cause == nil {
		return fmt.Sprintf("backup inventory %s at %q", backupError.Kind, backupError.Path)
	}
	return fmt.Sprintf("backup inventory %s at %q: %v", backupError.Kind, backupError.Path, backupError.Cause)
}

func (backupError *BackupInventoryError) Unwrap() error {
	if backupError == nil {
		return nil
	}
	return backupError.Cause
}

func isBackupInventoryBlockedError(err error) bool {
	var backupError *BackupInventoryError
	return errors.As(err, &backupError) && backupError.Kind == BackupInventoryErrorBlocked
}

func IsBackupInventoryBlockedError(err error) bool {
	return isBackupInventoryBlockedError(err)
}

type BackupInventoryEntry struct {
	RelativePath  string
	Kind          BackupInventoryEntryKind
	ContentDigest string
}

type BackupLogicalInventory struct {
	Entries []BackupInventoryEntry
	Digest  string
}

func buildBackupLogicalInventory(root string) (BackupLogicalInventory, error) {
	rootInfo, err := os.Lstat(root)
	if err != nil {
		return BackupLogicalInventory{}, fmt.Errorf("inspect backup inventory root %q: %w", root, err)
	}
	if rootInfo.Mode()&os.ModeSymlink != 0 {
		return BackupLogicalInventory{}, &BackupInventoryError{Path: root, Kind: BackupInventoryErrorUnsupported}
	}
	if !rootInfo.IsDir() {
		return BackupLogicalInventory{}, fmt.Errorf("backup inventory root %q must be a directory", root)
	}

	entries := make([]BackupInventoryEntry, 0)
	err = filepath.WalkDir(root, func(path string, directoryEntry fs.DirEntry, walkError error) error {
		if walkError != nil {
			return classifyBackupInventoryError(path, walkError)
		}
		if path == root {
			return nil
		}
		if directoryEntry.Type()&os.ModeSymlink != 0 {
			return &BackupInventoryError{Path: path, Kind: BackupInventoryErrorUnsupported}
		}

		entryInfo, err := directoryEntry.Info()
		if err != nil {
			return classifyBackupInventoryError(path, err)
		}
		relativePath, err := filepath.Rel(root, path)
		if err != nil {
			return fmt.Errorf("get backup inventory relative path for %q: %w", path, err)
		}
		entry := BackupInventoryEntry{RelativePath: filepath.ToSlash(relativePath)}
		switch {
		case entryInfo.IsDir():
			entry.Kind = BackupInventoryDirectory
		case entryInfo.Mode().IsRegular():
			entry.Kind = BackupInventoryFile
			entry.ContentDigest, err = backupFileContentDigest(path)
			if err != nil {
				return err
			}
		default:
			return &BackupInventoryError{Path: path, Kind: BackupInventoryErrorUnsupported}
		}
		entries = append(entries, entry)
		return nil
	})
	if err != nil {
		return BackupLogicalInventory{}, fmt.Errorf("build backup logical inventory for %q: %w", root, err)
	}

	sort.Slice(entries, func(first, second int) bool {
		return entries[first].RelativePath < entries[second].RelativePath
	})
	return BackupLogicalInventory{Entries: entries, Digest: backupLogicalInventoryDigest(entries)}, nil
}

func backupFileContentDigest(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", classifyBackupInventoryError(path, err)
	}
	digest := sha256.New()
	_, copyErr := io.Copy(digest, file)
	closeErr := file.Close()
	if copyErr != nil {
		return "", classifyBackupInventoryError(path, copyErr)
	}
	if closeErr != nil {
		return "", classifyBackupInventoryError(path, closeErr)
	}
	return hex.EncodeToString(digest.Sum(nil)), nil
}

func classifyBackupInventoryError(path string, err error) error {
	if isLockedFileError(err) {
		return &BackupInventoryError{Path: path, Kind: BackupInventoryErrorBlocked, Cause: err}
	}
	return fmt.Errorf("inspect backup inventory entry %q: %w", path, err)
}

func backupLogicalInventoryDigest(entries []BackupInventoryEntry) string {
	digest := sha256.New()
	for _, entry := range entries {
		fmt.Fprintf(digest, "%s\x00%s\x00%s\n", entry.RelativePath, entry.Kind, entry.ContentDigest)
	}
	return hex.EncodeToString(digest.Sum(nil))
}

func backupLogicalInventoriesEqual(first, second BackupLogicalInventory) bool {
	if first.Digest != second.Digest || len(first.Entries) != len(second.Entries) {
		return false
	}
	for index := range first.Entries {
		if first.Entries[index] != second.Entries[index] {
			return false
		}
	}
	return true
}

func BuildBackupLogicalInventory(root string) (BackupLogicalInventory, error) {
	return buildBackupLogicalInventory(root)
}

func BackupLogicalInventoriesEqual(first, second BackupLogicalInventory) bool {
	return backupLogicalInventoriesEqual(first, second)
}
