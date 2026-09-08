package main

import (
	"errors"
	"strings"
	"testing"
)

func TestRetryLockedFileNotifiesAndRetriesUntilSuccess(t *testing.T) {
	lockedError := errors.New("file is locked")
	attempts := 0
	notifications := make([]string, 0, 2)
	sleeps := 0

	err := retryLockedFile("notes.txt", func() error {
		attempts++
		if attempts == 1 {
			return lockedError
		}
		return nil
	}, SynchronizationExecutionOptions{
		Notify: func(message string) { notifications = append(notifications, message) },
		IsLockedError: func(err error) bool {
			return errors.Is(err, lockedError)
		},
		Sleep: func() { sleeps++ },
	})
	if err != nil {
		t.Fatalf("retryLockedFile() error = %v", err)
	}
	if attempts != 2 {
		t.Fatalf("attempts = %d, want 2", attempts)
	}
	if sleeps != 1 {
		t.Fatalf("sleep calls = %d, want 1", sleeps)
	}
	if len(notifications) != 2 {
		t.Fatalf("notifications = %v, want locked and success notifications", notifications)
	}
	if !strings.Contains(notifications[0], "locked") || !strings.Contains(notifications[1], "synchronized") {
		t.Fatalf("notifications = %v, want locked then synchronized messages", notifications)
	}
}

func TestRetryLockedFileStopsImmediatelyOnNonLockedError(t *testing.T) {
	wantError := errors.New("permission denied")
	attempts := 0

	err := retryLockedFile("notes.txt", func() error {
		attempts++
		return wantError
	}, SynchronizationExecutionOptions{
		IsLockedError: func(error) bool { return false },
	})
	if !errors.Is(err, wantError) {
		t.Fatalf("retryLockedFile() error = %v, want %v", err, wantError)
	}
	if attempts != 1 {
		t.Fatalf("attempts = %d, want 1", attempts)
	}
}
