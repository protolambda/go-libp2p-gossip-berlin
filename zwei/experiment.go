package zwei

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	//mplex "github.com/libp2p/go-libp2p-mplex"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	//ws "github.com/libp2p/go-ws-transport"
	//"github.com/multiformats/go-multiaddr"
)


type SimHost struct {
	host.Host
	ps *pubsub.PubSub
	ctx context.Context
	logger Logger
}

func NewSimHost(ctx context.Context, h host.Host, logger Logger) *SimHost {
	return &SimHost{Host: h, ctx: ctx, logger: logger}
}

func (s *SimHost) StartPubsub() error {
	optsPS := []pubsub.Option{
		pubsub.WithMessageSigning(true),
	}
	ps, err := pubsub.NewGossipSub(s.ctx, s, optsPS...)
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

		// if the peer is currently not subbed to any topic, don't publish anything for a while
		if len(topics) == 0 {
			time.Sleep(time.Duration(minSleepMs) * time.Millisecond * 10)
			continue
		}

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
	stop func()
	opts []libp2p.Option
	hosts []*SimHost
	logger Logger
}

func (ex *Experiment) CreateHosts(count int) error {
	for i := 0; i < count; i++ {
		opts, err := ex.selectOpts()
		if err != nil {
			return err
		}
		h, err := libp2p.New(ex.ctx, opts...)
		if err != nil {
			return err
		}
		ex.hosts = append(ex.hosts, NewSimHost(ex.ctx, h, ex.logger.SubLogger("from: " + h.ID().Pretty())))
	}
	return nil
}

func (ex *Experiment) selectOpts() (out []libp2p.Option, err error) {
	priv, _, err := crypto.GenerateSecp256k1Key(nil)
	if err != nil {
		return nil, err
	}
	out = append(out, ex.opts...)
	out = append(out,
		libp2p.Identity(priv),
		libp2p.ListenAddrStrings(
			"/ip4/127.0.0.1/tcp/0", // 0: gets a random port assigned on localhost
		),
	)
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
			ex.logger.Printf("Connected %v to %v", hostA.ID(), hostB.ID())
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

func (ex *Experiment) ActRandomlyAll(seed int64) {
	for _, h := range ex.hosts {
		go h.ActRandom(seed)
		seed++
		ex.logger.Printf("started random acting for %v", h.ID())
	}
}


func CreateExperiment(logger *DebugLogger, opts []libp2p.Option, seed int64, hostCount int, degree int) *Experiment {
	ctx, stop := context.WithCancel(context.Background())

	ex := &Experiment{ctx: ctx, opts: opts, logger: logger, stop: stop}

	if err := ex.CreateHosts(hostCount); err != nil {
		panic(err)
	}

	if err := ex.StartPubsubAll(); err != nil {
		panic(err)
	}

	if err := ex.RandomPeering(seed, degree); err != nil {
		panic(err)
	}
	// TODO: experiment with 100% all subscriptions.
	topics := map[string]float64{
		"/libp2p/example/berlin/protolambda/foo": 0.7,
		"/libp2p/example/berlin/protolambda/bar": 0.4,
		"/libp2p/example/berlin/protolambda/quix": 0.8,
	}
	for topic, chance := range topics {
		if err := ex.SubRandomly(seed, topic, chance); err != nil {
			panic(err)
		}
		seed++
	}

	return ex
}

func (ex *Experiment) Start() {
	ex.logger.Printf("starting experiment...")
	ex.ActRandomlyAll(123)
}

func (ex *Experiment) Stop() {
	if ex.stop != nil {
		ex.logger.Printf("stopping experiment...")
		ex.stop()
	} else {
		ex.logger.Printf("already stopped")
	}
}
