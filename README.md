# EthBerlinZwei: Profiling Libp2p Gossipsub, Golang version

## The bounty problem

> Find and fix bottlenecks and performance hotspots in the Go implementation of gossipsub.

See [this issue on `bounties/EthBerlinZwei`](https://github.com/ethberlinzwei/Bounties/issues/18).

## Approach

To profile anything at all, some kind of test-run is necessary.
One that stresses go-libp2p with a high throughput, with a good amount of peers and topics.

### Why not use benchmarking?

Since the task is not to benchmark libp2p (discussed options here with @raulk however), but to profile and find (and fix) the hotspots,
a more practical test-run with the actual overhead of opening a connection and not sharing memory for messages helps identify hotspots.

Also, the message-interval and size parameters are less strict: they can definitely affect speed,
 but there are only so many extremes to find hotspots for.

Benchmarking of the isolated gossipsub logic would be better if done with a mock net, 
 [something like this](https://github.com/libp2p/go-libp2p/blob/master/p2p/net/mock/mock_net.go)
This however hides the overhead introduced by passing messages to a real socket, skewing the priorities in what to optimize for.
If practical issues are solved, one could then use Perf to profile a Go benchmark with, and look into the memory allocations and flamegraph of the remaining calls.
The bigger picture found in call-graphs in a non-benchmark setting does not show gossipsub code itself to be the bottleneck in practice however, hence not going the benchmarking route.

### Profiling settings

Common settings for *hackathon results*:

```go
// total hosts
hostCount := 100
// peers per host (randomly assigned)
degree := 10

// pubsub topic chances:
"/libp2p/example/berlin/protolambda/foo":  0.7,
"/libp2p/example/berlin/protolambda/bar":  0.4,
"/libp2p/example/berlin/protolambda/quix": 0.8,

// A no-op logger is used during benchmarking for speed.
logger := zwei.NewDebugLogger(nil)
// For debugging this can be changed to:  
// logger := zwei.NewDebugLogger(log.New(os.Stdout, "experiment: ", log.Lmicroseconds))

// message size
// big: 8 - 15 KB
minMsgByteLen := 8 << 10
maxMsgByteLen := 16 << 10
// small: 10 bytes
minMsgByteLen := 10
maxMsgByteLen := 10

// publish interval range for each simulated host (publish on 1 random topic)
minSleepMs := 100
maxSleepMs := 300

// libp2p settings
// transport:
libp2p.Transport(tcp.NewTCPTransport),
// mux choice:
libp2p.Muxer("/yamux/1.0.0", yamux.DefaultTransport),
//libp2p.Muxer("/mplex/6.7.0", mplex.DefaultTransport), // for some later profiles with mplex
// security:
libp2p.Security(secio.ID, secio.New),

// GossipSub settings
// Initially true, signing with Secp256k1.
// Later disabled, since this was the biggest practical bottleneck, and obfuscates the smaller differences. 
pubsub.WithMessageSigning(true)

// loopback through localhost, with no artificial latency
libp2p.ListenAddrStrings(
    "/ip4/127.0.0.1/tcp/0", // 0: gets a random port assigned on localhost
),

// There also are options to change the RNG seed for both initialization and the testrun itself,
// but libp2p (interaction with machine itself, and go-routine scheduling) is not deterministic enough
// to make the results fully reproducible. 
```

### Profiling usage

1. Configure `main.go` options: a `zwei.Experiment` is created with these.
  Message length and interval can be changed in the experiment code, if required.
2. PPROF CPU-Profiling starts after setting up the experiment (starting hosts, starting gossipsub, and subscribing to topics)  
3. Start experiment
4. Wait for stop-signal
5. Stop profiling, save results, see log output for profiling output location. 
6. Stop libp2p tasks and close resources with `Experiment.Close()`

### Profiling results

// TODO describe results

### Conclusion

// TODO 

### LICENSE

MIT, see [`LICENSE` file](./LICENSE).
Some initial code was adapted from the go-libp2p examples repository,
 [here](https://github.com/libp2p/go-libp2p-examples),
 [also licensed with MIT](https://github.com/libp2p/go-libp2p-examples/blob/master/LICENSE).
