package gosync

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type BackupRootPaths struct {
	Source      string
	Destination string
}

type BackupPolicySnapshot struct {
	Source                         string
	Destination                    string
	SynchronizationIntervalSeconds int
	BackupRetentionCount           int
}

type BackupDestinationStatus string

const (
	BackupDestinationMissing   BackupDestinationStatus = "missing"
	BackupDestinationDirectory BackupDestinationStatus = "directory"
)

func buildBackupPolicySnapshot(roots RootPaths, configuration ConfigurationSnapshot) (BackupPolicySnapshot, error) {
	if configuration.SynchronizationMode != SynchronizationModeBackup {
		return BackupPolicySnapshot{}, fmt.Errorf("backup policy requires BACKUP mode")
	}
	if err := validateConfigurationSnapshot(configuration); err != nil {
		return BackupPolicySnapshot{}, fmt.Errorf("validate backup configuration: %w", err)
	}
	backupRoots, err := deriveBackupRootPaths(roots.First, roots.Second)
	if err != nil {
		return BackupPolicySnapshot{}, err
	}
	return BackupPolicySnapshot{
		Source:                         backupRoots.Source,
		Destination:                    backupRoots.Destination,
		SynchronizationIntervalSeconds: configuration.SynchronizationIntervalSeconds,
		BackupRetentionCount:           configuration.BackupRetentionCount,
	}, nil
}

func deriveBackupRootPaths(sourceRoot, destinationRoot string) (BackupRootPaths, error) {
	canonicalSource, err := canonicalizeBackupPath(sourceRoot)
	if err != nil {
		return BackupRootPaths{}, fmt.Errorf("canonicalize backup source: %w", err)
	}
	canonicalDestination, err := canonicalizeBackupPath(destinationRoot)
	if err != nil {
		return BackupRootPaths{}, fmt.Errorf("canonicalize backup destination: %w", err)
	}
	return BackupRootPaths{Source: canonicalSource, Destination: canonicalDestination}, nil
}

func canonicalizeBackupPath(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", fmt.Errorf("backup path must not be empty")
	}
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("make backup path absolute: %w", err)
	}
	normalizedPath := filepath.Clean(absolutePath)

	if _, err := os.Lstat(normalizedPath); err == nil {
		resolvedPath, err := filepath.EvalSymlinks(normalizedPath)
		if err != nil {
			return "", fmt.Errorf("resolve backup path aliases: %w", err)
		}
		return filepath.Clean(resolvedPath), nil
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("inspect backup path: %w", err)
	}

	missingParts := make([]string, 0)
	currentPath := normalizedPath
	for {
		_, err := os.Lstat(currentPath)
		if err == nil {
			resolvedPath, resolveErr := filepath.EvalSymlinks(currentPath)
			if resolveErr != nil {
				return "", fmt.Errorf("resolve backup path parent aliases: %w", resolveErr)
			}
			for index := len(missingParts) - 1; index >= 0; index-- {
				resolvedPath = filepath.Join(resolvedPath, missingParts[index])
			}
			return filepath.Clean(resolvedPath), nil
		}
		if !os.IsNotExist(err) {
			return "", fmt.Errorf("inspect backup path parent: %w", err)
		}
		parentPath := filepath.Dir(currentPath)
		if parentPath == currentPath {
			return "", fmt.Errorf("backup path has no existing parent")
		}
		missingParts = append(missingParts, filepath.Base(currentPath))
		currentPath = parentPath
	}
}

func validateBackupSource(sourceRoot string) error {
	entryInfo, err := os.Lstat(sourceRoot)
	if err != nil {
		return fmt.Errorf("inspect backup source %q: %w", sourceRoot, err)
	}
	if entryInfo.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("backup source %q must not be a symbolic link", sourceRoot)
	}
	if !entryInfo.IsDir() {
		return fmt.Errorf("backup source %q must be a directory", sourceRoot)
	}

	directory, err := os.Open(sourceRoot)
	if err != nil {
		return fmt.Errorf("open backup source %q: %w", sourceRoot, err)
	}
	_, readErr := directory.Readdirnames(-1)
	closeErr := directory.Close()
	if readErr != nil && readErr != io.EOF {
		return fmt.Errorf("inspect backup source %q: %w", sourceRoot, readErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close backup source %q: %w", sourceRoot, closeErr)
	}
	return nil
}

func inspectBackupDestination(destinationRoot string) (BackupDestinationStatus, error) {
	entryInfo, err := os.Lstat(destinationRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return BackupDestinationMissing, nil
		}
		return "", fmt.Errorf("inspect backup destination %q: %w", destinationRoot, err)
	}
	if entryInfo.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("backup destination %q must not be a symbolic link", destinationRoot)
	}
	if !entryInfo.IsDir() {
		return "", fmt.Errorf("backup destination %q must be a directory", destinationRoot)
	}

	directory, err := os.Open(destinationRoot)
	if err != nil {
		return "", fmt.Errorf("open backup destination %q: %w", destinationRoot, err)
	}
	_, readErr := directory.Readdirnames(-1)
	closeErr := directory.Close()
	if readErr != nil && readErr != io.EOF {
		return "", fmt.Errorf("inspect backup destination %q: %w", destinationRoot, readErr)
	}
	if closeErr != nil {
		return "", fmt.Errorf("close backup destination %q: %w", destinationRoot, closeErr)
	}
	return BackupDestinationDirectory, nil
}

func validateBackupRootPaths(sourceRoot, destinationRoot, configurationPath string) (BackupRootPaths, error) {
	if err := validateBackupSource(sourceRoot); err != nil {
		return BackupRootPaths{}, err
	}
	if _, err := inspectBackupDestination(destinationRoot); err != nil {
		return BackupRootPaths{}, err
	}

	backupRoots, err := deriveBackupRootPaths(sourceRoot, destinationRoot)
	if err != nil {
		return BackupRootPaths{}, err
	}
	firstContainsSecond, err := rootContains(backupRoots.Source, backupRoots.Destination)
	if err != nil {
		return BackupRootPaths{}, err
	}
	secondContainsFirst, err := rootContains(backupRoots.Destination, backupRoots.Source)
	if err != nil {
		return BackupRootPaths{}, err
	}
	if firstContainsSecond || secondContainsFirst {
		return BackupRootPaths{}, fmt.Errorf("backup roots must be distinct and not nested")
	}

	canonicalConfigurationPath, err := canonicalizeBackupPath(configurationPath)
	if err != nil {
		return BackupRootPaths{}, fmt.Errorf("canonicalize configuration path: %w", err)
	}
	for _, root := range []string{backupRoots.Source, backupRoots.Destination} {
		containsConfiguration, err := rootContains(root, canonicalConfigurationPath)
		if err != nil {
			return BackupRootPaths{}, err
		}
		if containsConfiguration {
			return BackupRootPaths{}, fmt.Errorf("configuration path %q is inside backup root %q", configurationPath, root)
		}
	}
	return backupRoots, nil
}
