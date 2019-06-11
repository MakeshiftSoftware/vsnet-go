package grace

import (
	"log"
	"os"
	"os/signal"
	"syscall"
)

// HookSignals listen for signals to gracefully exit
func HookSignals(quitc chan os.Signal, cleanup func()) {
	signal.Notify(quitc, os.Interrupt, syscall.SIGTERM, syscall.SIGKILL)

	go func() {
		<-quitc

		log.Print("[info] received quit signal")

		cleanup()

		log.Print("[info] goodbye")

		os.Exit(0)
	}()
}
