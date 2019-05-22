package cleanup

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

// Add queues a cleanup function to run when context is cancelled
func Add(ctx context.Context, wg *sync.WaitGroup, cleanupFunc func()) {
	wg.Add(1)

	go func() {
		defer wg.Done()
		select {
		case <-ctx.Done():
			cleanupFunc()
			return
		}
	}()
}

// Listen starts a cleanup goroutine to gracefully exit program
func Listen(cancel context.CancelFunc, wg *sync.WaitGroup) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, syscall.SIGTERM)
	signal.Notify(c, syscall.SIGKILL)

	go func() {
		<-c
		log.Print("[Info] Shutting down gracefully...")

		if cancel != nil {
			cancel()
			wg.Wait()
		}

		os.Exit(0)
	}()
}
