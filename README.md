# EthBerlinZwei: Profiling Libp2p Gossipsub, Golang version

Hackathon submission by @protolambda, learning libp2p with a non-networking background.

## The bounty problem

> Find and fix bottlenecks and performance hotspots in the Go implementation of gossipsub.

See [this issue on `bounties/EthBerlinZwei`](https://github.com/ethberlinzwei/Bounties/issues/18).

And so there it starts; read up on libp2p knowledge, 
read the [Gossipsub spec](https://github.com/libp2p/specs/tree/master/pubsub/gossipsub)
and then trial-and-error throughout the hackathon. I started late however, since I worked on other Eth 2 issues too.
Thanks to @raulk for getting me up to speed to work on this so fast.

Note that this is a hack, produced with a "see it work first" mindset, not a research paper. You are welcome to fork and improve the profiling.

## Approach

To profile anything at all, some kind of test-run is necessary.
One that stresses go-libp2p with a high throughput, with a good amount of peers and topics.
Then, a PPROF profile can be made of the test-run, and help identify hotspots to optimize.

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

Common settings for the produced *hackathon results* (not claiming perfectness, time constraints to for pretty parametrization apply):

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

## Usage

1. Configure `main.go` options: a `zwei.Experiment` is created with these.
  Message length and interval can be changed in the experiment code, if required.
2. PPROF CPU-Profiling starts after setting up the experiment (starting hosts, starting gossipsub, and subscribing to topics)  
3. Start experiment
4. Wait for stop-signal
5. Stop profiling, save results, see log output for profiling output location. 
6. Stop libp2p tasks and close resources with `Experiment.Close()`

To generate a call-graph:
```bash
go tool pprof -web /tmp/profile......../cpu.pprof 
```

## Profiling results

Early results settings: Yamux, signed GossipSub, small 10 byte messages: signature verification is the clear bottleneck.

This test run published 31k messages, 1500k were received. A 145 seconds run.
[Full callgraph SVG](results/pprof_31k_published_1500k_received.svg)

Then, I disabled GossipSub signatures (`pubsub.WithMessageSigning(false)`) to see what was left.

For small messages, it is Yamux triggering secio encryption, which then writes to a socket connection provided by the kernel, which also forms a bottleneck.

This test run published 8k messages, 600k were received. A 20 seconds run.
[Full callgraph SVG](results/pprof_no_gossipsub_signing_8k_published_600k_received.svg)

Raul then recommended to increase the message size, so repeat this with random 8 - 16 KB messages:

This test run published 5k messages, 200k were received. A 90 seconds run. Note the significantly lower throughput.
[Full callgraph SVG](results/pprof_no_gossipsub_signing_5k_published_200k_received_8k_to_16k_bytelen_yamux.svg)

For larger messages, SHA-256 calls by Yamux become the bottleneck.
However, it looks like it is already using the excellent [Sha-256 SIMD library](https://github.com/minio/sha256-simd) for speed,
so there is not much to gain unless something is being hashed twice and can be cached.

Now try again with Mplex:

This test run published 8k messages, 357k were received. A 33 seconds run. Note the significantly lower throughput.
[Full callgraph SVG](results/pprof_no_gossipsub_signing_8k_published_375k_received_8k_to_16k_bytelen_mplex.svg)

SHA-256 (and general secio crypto) is still by far the biggest bottleneck.


## Conclusion

GossipSub itself is primarily limited by the crypto necessary to verify and encrypt the messages,
and the data-structures used in its implementation do not seem to be worth optimizing at this time.   

There seems to be some interesting difference in mplex vs. yamux to look into at a later moment, if it is not a usage problem from my side.

## LICENSE

MIT, see [`LICENSE` file](./LICENSE).
Some initial code was adapted from the go-libp2p examples repository,
 [here](https://github.com/libp2p/go-libp2p-examples),
 [also licensed with MIT](https://github.com/libp2p/go-libp2p-examples/blob/master/LICENSE).
