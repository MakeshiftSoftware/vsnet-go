package task

import (
	"sync"
	"time"
)

// New creates a new repeating background task
func New(task func() bool, interval time.Duration, wg *sync.WaitGroup, cleanupc <-chan struct{}) {
	wg.Add(1)

	go func() {
		ticker := time.NewTicker(interval)

		defer func() {
			ticker.Stop()
			wg.Done()
		}()

		for {
			select {
			case <-ticker.C:
				done := task()

				if done {
					return
				}
			case <-cleanupc:
				return
			}
		}
	}()
}
