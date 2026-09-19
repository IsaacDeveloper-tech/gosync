package gosync

import (
	"errors"
	"time"
)

const watchCheckInterval = 5 * time.Second

type WatchLoopOptions struct {
	Interval time.Duration
	Stop     <-chan struct{}
	Wait     func(time.Duration)
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
	for {
		select {
		case <-options.Stop:
			return nil
		default:
		}

		if options.Wait != nil {
			options.Wait(interval)
		} else {
			timer := time.NewTimer(interval)
			select {
			case <-timer.C:
			case <-options.Stop:
				if !timer.Stop() {
					<-timer.C
				}
				return nil
			}
		}
		select {
		case <-options.Stop:
			return nil
		default:
		}
		if err := synchronize(); err != nil {
			return err
		}
	}
}
