package main

import (
	"errors"
	"time"
)

const watchCheckInterval = 5 * time.Second

type WatchLoopOptions struct {
	Interval time.Duration
	Stop     <-chan struct{}
}

func runWatchLoop(synchronize func() error, options WatchLoopOptions) error {
	if synchronize == nil {
		return errors.New("watch synchronization function is required")
	}

	if err := synchronize(); err != nil {
		return err
	}

	interval := options.Interval
	if interval <= 0 {
		interval = watchCheckInterval
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := synchronize(); err != nil {
				return err
			}
		case <-options.Stop:
			return nil
		}
	}
}
