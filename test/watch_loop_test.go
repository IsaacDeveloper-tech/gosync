package gosync_test

import (
	"errors"
	"testing"
	"time"
)

func TestRunWatchLoopSynchronizesImmediatelyAndPeriodically(t *testing.T) {
	stop := make(chan struct{})
	callCount := 0

	err := runWatchLoop(func() error {
		callCount++
		if callCount == 3 {
			close(stop)
		}
		return nil
	}, WatchLoopOptions{
		Interval: time.Millisecond,
		Stop:     stop,
	})
	if err != nil {
		t.Fatalf("runWatchLoop() error = %v", err)
	}
	if callCount != 3 {
		t.Fatalf("synchronization calls = %d, want 3", callCount)
	}
}

func TestRunWatchLoopStopsOnSynchronizationError(t *testing.T) {
	wantError := errors.New("synchronization failed")
	callCount := 0

	err := runWatchLoop(func() error {
		callCount++
		return wantError
	}, WatchLoopOptions{Interval: time.Millisecond})
	if !errors.Is(err, wantError) {
		t.Fatalf("runWatchLoop() error = %v, want %v", err, wantError)
	}
	if callCount != 1 {
		t.Fatalf("synchronization calls = %d, want 1", callCount)
	}
}
