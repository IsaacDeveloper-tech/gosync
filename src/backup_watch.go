package gosync

import (
	"errors"
	"time"
)

func runBackupWatchLoop(policy BackupPolicySnapshot, options BackupCycleOptions, ownership BackupDestinationOwnership, watchOptions WatchLoopOptions) error {
	var firstCycleError error
	cycle := func() error {
		err := runBackupCycle(policy, options)
		if err != nil && firstCycleError == nil {
			firstCycleError = err
		}
		return nil
	}
	loopError := runWatchLoop(cycle, watchOptions)
	releaseError := ownership.Release()
	return errors.Join(firstCycleError, loopError, releaseError)
}

func backupWatchInterval(seconds int) time.Duration {
	return time.Duration(seconds) * time.Second
}
