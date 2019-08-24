package main

import (
	"github.com/libp2p/go-libp2p"
	//mplex "github.com/libp2p/go-libp2p-mplex"
	//ws "github.com/libp2p/go-ws-transport"
	secio "github.com/libp2p/go-libp2p-secio"
	yamux "github.com/libp2p/go-libp2p-yamux"
	"github.com/libp2p/go-tcp-transport"
	"go-libp2p-gossip-berlin/zwei"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	defaultOps := []libp2p.Option{
		libp2p.Transport(tcp.NewTCPTransport),
		//libp2p.Transport(ws.New),
		libp2p.Muxer("/yamux/1.0.0", yamux.DefaultTransport),
		//libp2p.Muxer("/mplex/6.7.0", mplex.DefaultTransport),
		libp2p.Security(secio.ID, secio.New),
	}
	logger := zwei.NewDebugLogger(log.New(os.Stdout, "experiment: ", log.Lmicroseconds))
	ex := zwei.CreateExperiment(logger, defaultOps, 123, 100, 10)
	ex.Start()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT)

	select {
	// TODO more program events here, maybe change settings on runtime?
	case <-stop:
		ex.Stop()
		os.Exit(0)
	}
}
