package main

import (
	"go-libp2p-gossip-berlin/zwei"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	logger := zwei.NewDebugLogger(log.New(os.Stdout, "experiment: ", log.Lmicroseconds))
	start, stop := zwei.Run(logger, 123, 100, 10)
	start()

	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, syscall.SIGINT)

	select {
	// TODO more program events here, maybe change settings on runtime?
	case <-stopCh:
		stop()
		os.Exit(0)
	}
}
