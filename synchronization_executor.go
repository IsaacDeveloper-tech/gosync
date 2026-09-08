package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

type SynchronizationExecutionOptions struct {
	Notify        func(string)
	IsLockedError func(error) bool
	Sleep         func()
}

func executeSynchronizationPlan(roots RootPaths, plan SynchronizationPlan, options SynchronizationExecutionOptions) SynchronizationResult {
	for _, action := range plan.Actions {
		err := retryLockedFile(action.RelativePath, func() error {
			return executeSynchronizationAction(roots, action)
		}, options)
		if err != nil {
			return SynchronizationResult{Failure: fmt.Errorf("execute action for %q: %w", action.RelativePath, err)}
		}
	}

	return SynchronizationResult{Completed: true}
}

func executeSynchronizationAction(roots RootPaths, action SynchronizationAction) error {
	firstPath := filepath.Join(roots.First, action.RelativePath)
	secondPath := filepath.Join(roots.Second, action.RelativePath)

	switch action.Kind {
	case SynchronizationActionCopyToFirst:
		return copySynchronizationPath(secondPath, firstPath)
	case SynchronizationActionCopyToSecond:
		return copySynchronizationPath(firstPath, secondPath)
	case SynchronizationActionDeleteFromFirst:
		return removeSynchronizationPath(firstPath)
	case SynchronizationActionDeleteFromSecond:
		return removeSynchronizationPath(secondPath)
	default:
		return fmt.Errorf("unknown synchronization action kind %d", action.Kind)
	}
}

func copySynchronizationPath(sourcePath, destinationPath string) error {
	sourceInfo, err := os.Lstat(sourcePath)
	if err != nil {
		return fmt.Errorf("inspect source %q: %w", sourcePath, err)
	}

	switch {
	case sourceInfo.IsDir():
		if err := os.RemoveAll(destinationPath); err != nil {
			return fmt.Errorf("replace destination directory %q: %w", destinationPath, err)
		}
		if err := os.MkdirAll(destinationPath, sourceInfo.Mode().Perm()); err != nil {
			return fmt.Errorf("create destination directory %q: %w", destinationPath, err)
		}

		entries, err := os.ReadDir(sourcePath)
		if err != nil {
			return fmt.Errorf("read source directory %q: %w", sourcePath, err)
		}
		for _, entry := range entries {
			sourceEntryPath := filepath.Join(sourcePath, entry.Name())
			destinationEntryPath := filepath.Join(destinationPath, entry.Name())
			if err := copySynchronizationPath(sourceEntryPath, destinationEntryPath); err != nil {
				return err
			}
		}
		return nil
	case sourceInfo.Mode().IsRegular():
		if err := os.MkdirAll(filepath.Dir(destinationPath), 0o755); err != nil {
			return fmt.Errorf("create destination parent for %q: %w", destinationPath, err)
		}
		if err := os.RemoveAll(destinationPath); err != nil {
			return fmt.Errorf("replace destination file %q: %w", destinationPath, err)
		}

		sourceFile, err := os.Open(sourcePath)
		if err != nil {
			return fmt.Errorf("open source file %q: %w", sourcePath, err)
		}
		destinationFile, err := os.OpenFile(destinationPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, sourceInfo.Mode().Perm())
		if err != nil {
			sourceFile.Close()
			return fmt.Errorf("create destination file %q: %w", destinationPath, err)
		}

		_, copyError := io.Copy(destinationFile, sourceFile)
		destinationCloseError := destinationFile.Close()
		sourceCloseError := sourceFile.Close()
		if copyError != nil {
			return fmt.Errorf("copy file %q to %q: %w", sourcePath, destinationPath, copyError)
		}
		if destinationCloseError != nil {
			return fmt.Errorf("close destination file %q: %w", destinationPath, destinationCloseError)
		}
		if sourceCloseError != nil {
			return fmt.Errorf("close source file %q: %w", sourcePath, sourceCloseError)
		}
		return nil
	default:
		return fmt.Errorf("unsupported source filesystem entry %q", sourcePath)
	}
}

func removeSynchronizationPath(path string) error {
	if err := os.RemoveAll(path); err != nil {
		return fmt.Errorf("remove path %q: %w", path, err)
	}
	return nil
}

func retryLockedFile(filePath string, operation func() error, options SynchronizationExecutionOptions) error {
	isLockedError := options.IsLockedError
	if isLockedError == nil {
		isLockedError = isLockedFileError
	}
	notify := options.Notify
	if notify == nil {
		notify = func(string) {}
	}
	sleep := options.Sleep
	if sleep == nil {
		sleep = func() { time.Sleep(time.Second) }
	}

	lockedNotificationSent := false
	for {
		err := operation()
		if err == nil {
			if lockedNotificationSent {
				notify(fmt.Sprintf("File %q synchronized successfully.", filePath))
			}
			return nil
		}
		if !isLockedError(err) {
			return err
		}
		if !lockedNotificationSent {
			notify(fmt.Sprintf("File %q is locked; retrying synchronization.", filePath))
			lockedNotificationSent = true
		}
		sleep()
	}
}
