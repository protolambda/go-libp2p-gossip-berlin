package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	//mplex "github.com/libp2p/go-libp2p-mplex"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	secio "github.com/libp2p/go-libp2p-secio"
	yamux "github.com/libp2p/go-libp2p-yamux"
	"github.com/libp2p/go-tcp-transport"
	//ws "github.com/libp2p/go-ws-transport"
	//"github.com/multiformats/go-multiaddr"
)

type SimHost struct {
	host.Host
	ps *pubsub.PubSub
	ctx context.Context
	logger *log.Logger
}

func NewSimHost(ctx context.Context, h host.Host, logger *log.Logger) *SimHost {
	return &SimHost{Host: h, ctx: ctx, logger: logger}
}

func (s *SimHost) StartPubsub() error {
	ps, err := pubsub.NewGossipSub(s.ctx, s)
	if err != nil {
		return err
	}
	s.ps = ps
	return nil
}

func (s *SimHost) SubTopic(topic string) error {
	sub, err := s.ps.Subscribe(topic)
	if err != nil {
		return err
	}
	go s.pubsubHandler(sub)
	return nil
}

func (s *SimHost) pubsubHandler(sub *pubsub.Subscription) {
	for {
		ctx, _ := context.WithTimeout(s.ctx, 5 * time.Second)
		msg, err := sub.Next(ctx)
		if err != nil {
			s.logger.Printf("pubsub read err: %v", err)
			continue
		}

		s.logger.Printf("received msg (%s): %x", msg.TopicIDs, msg.Data)
		// TODO act on msg ?
	}
}

func (s *SimHost) ActRandom(seed int64) {
	minSleepMs := 100
	maxSleepMs := 300
	minMsgByteLen := 10
	maxMsgByteLen := 10
	rng := rand.New(rand.NewSource(seed))
	msgData := make([]byte, maxMsgByteLen, maxMsgByteLen)
	for {
		// get a random topic
		topics := s.ps.GetTopics()
		topic := topics[rng.Intn(len(topics))]

		// make a random msg
		size := minMsgByteLen + rng.Intn(1 + maxMsgByteLen - minMsgByteLen)
		msgData = msgData[:size]
		rng.Read(msgData)

		s.logger.Printf("publishing msg (%s): %x", topic, msgData)

		if err := s.ps.Publish(topic, msgData); err != nil {
			s.logger.Printf("publish failed: %v", err)
		}

		// wait random time before publishing next message
		time.Sleep(time.Duration(minSleepMs + rng.Intn(1 + maxSleepMs - minSleepMs)) * time.Millisecond)
	}
}

type Experiment struct {
	ctx context.Context
	hosts []*SimHost
	logger *log.Logger
}

func (ex *Experiment) CreateHosts(count int) error {
	for i := 0; i < count; i++ {
		h, err := libp2p.New(ex.ctx, ex.selectOpts()...)
		if err != nil {
			return err
		}
		ex.hosts = append(ex.hosts, NewSimHost(ex.ctx, h, log.New(ex.logger.Writer(), ex.logger.Prefix() + " > " + h.ID().Pretty() + " > ", log.Lmicroseconds)))
	}
	return nil
}

func (ex *Experiment) selectOpts() (out []libp2p.Option) {
	out = append(out,
		libp2p.Transport(tcp.NewTCPTransport),
		//libp2p.Transport(ws.New),
		libp2p.Muxer("/yamux/1.0.0", yamux.DefaultTransport),
		//libp2p.Muxer("/mplex/6.7.0", mplex.DefaultTransport),
		libp2p.Security(secio.ID, secio.New),

		libp2p.ListenAddrStrings(
			"/ip4/127.0.0.1/tcp/0", // 0: gets a random port assigned on localhost
		),
	)
	// TODO could add more/different options based on input choices?
	// TODO: or randomize option selection?
	return
}

func (ex *Experiment) RandomPeering(seed int64, degree int) error {
	rng := rand.New(rand.NewSource(seed))
	if degree < 1 {
		return fmt.Errorf("invalid degree %d", degree)
	}
	if len(ex.hosts) <= degree {
		return fmt.Errorf("not enough hosts to peer them with degree %d", degree)
	}
	for i, hostA := range ex.hosts {
		// Increase the peer count to the degree.
		for j := len(hostA.Network().Conns()); j < degree; {
			// pick a random *other* node to peer with.
			offset := rng.Intn(len(ex.hosts) - 2) + 1
			hostB := ex.hosts[(i + offset) % len(ex.hosts)]

			// If hostB is already connected, don't connect a second time.
			if len(hostA.Network().ConnsToPeer(hostB.ID())) != 0 {
				continue
			}

			// TODO: could support multiple protocols in peers, and peer based on support
			//addressesB := hostB.Addrs()
			//protocolsB := addressesB[0].Protocols()
			if err := hostA.Connect(ex.ctx, peer.AddrInfo{ID: hostB.ID(), Addrs: hostB.Addrs()}); err != nil {
				return err
			}
			ex.logger.Println("Connected ", hostA.ID(), "to", hostB.ID())
			j++
		}
	}
	return nil
}

func (ex *Experiment) StartPubsubAll() error {
	for _, h := range ex.hosts {
		if err := h.StartPubsub(); err != nil {
			return err
		}
		ex.logger.Printf("started pubsub for %v", h.ID())
	}
	return nil
}

func (ex *Experiment) SubRandomly(seed int64, topic string, chance float64) error {
	rng := rand.New(rand.NewSource(seed))
	for _, h := range ex.hosts {
		if rng.Float64() <= chance {
			if err := h.SubTopic(topic); err != nil {
				return err
			} else {
				ex.logger.Printf("subbed %v to %v", h.ID(), topic)
			}
		}
	}
	return nil
}

func (ex *Experiment) ActRandomlyAll(seed int64) error {
	for _, h := range ex.hosts {
		go h.ActRandom(seed)
		seed++
		ex.logger.Printf("started random acting for %v", h.ID())
	}
	return nil
}

func main() {
	ctx, cancelAll := context.WithCancel(context.Background())

	ex := Experiment{ctx: ctx, logger: log.New(os.Stdout, "experiment", log.LstdFlags)}

	hostCount := 10
	if err := ex.CreateHosts(hostCount); err != nil {
		panic(err)
	}
	degree := 4
	if err := ex.RandomPeering(1234, degree); err != nil {
		panic(err)
	}
	if err := ex.StartPubsubAll(); err != nil {
		panic(err)
	}
	// TODO: experiment with 100% all subscriptions.
	topics := map[string]float64{
		"/libp2p/example/berlin/protolambda/foo": 0.7,
		"/libp2p/example/berlin/protolambda/bar": 0.4,
		"/libp2p/example/berlin/protolambda/quix": 0.8,
	}
	seed := int64(42)
	for topic, chance := range topics {
		if err := ex.SubRandomly(seed, topic, chance); err != nil {
			panic(err)
		}
		seed++
	}

	if err := ex.ActRandomlyAll(123); err != nil {
		panic(err)
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT)

	select {
	// TODO more program events here, maybe change settings on runtime?
	case <-stop:
		cancelAll()
		os.Exit(0)
	}
}
