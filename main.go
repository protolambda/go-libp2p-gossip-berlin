package main

import (
	"fmt"
	"github.com/libp2p/go-libp2p"
	"github.com/pkg/profile"
	"time"

	mplex "github.com/libp2p/go-libp2p-mplex"
	//ws "github.com/libp2p/go-ws-transport"
	secio "github.com/libp2p/go-libp2p-secio"
	//yamux "github.com/libp2p/go-libp2p-yamux"
	"github.com/libp2p/go-tcp-transport"
	"go-libp2p-gossip-berlin/zwei"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// TODO: experiment with 100% all subscriptions.
	topics := map[string]float64{
		"/libp2p/example/berlin/protolambda/foo":  0.7,
		"/libp2p/example/berlin/protolambda/bar":  0.4,
		"/libp2p/example/berlin/protolambda/quix": 0.8,
	}

	defaultOps := []libp2p.Option{
		libp2p.Transport(tcp.NewTCPTransport),
		//libp2p.Transport(ws.New),
		//libp2p.Muxer("/yamux/1.0.0", yamux.DefaultTransport),
		libp2p.Muxer("/mplex/6.7.0", mplex.DefaultTransport),
		libp2p.Security(secio.ID, secio.New),
	}

	// pretty prints all actions for debugging
	//logger := zwei.NewDebugLogger(log.New(os.Stdout, "experiment: ", log.Lmicroseconds))

	// disables logging for better bench speed
	logger := zwei.NewDebugLogger(nil)

	hostCount := 100
	degree := 10

	ex := zwei.CreateExperiment(logger, defaultOps, topics, 123, hostCount, degree)

	// start profiling after creating the experiment
	prof := profile.Start(profile.CPUProfile, profile.NoShutdownHook)
	ex.Start(1234)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT)

	select {
	// TODO more program events here, maybe change settings on runtime?
	case <-stop:
		prof.Stop()
		sentCount, recvCount := ex.Stats()
		fmt.Printf("total sent: %d\n", sentCount)
		fmt.Printf("total received: %d\n", recvCount)
		ex.Stop()
		time.Sleep(time.Second)
		os.Exit(0)
	}
}
