# noise

[![GoDoc][1]][2] [![Discord][7]][8] [![MIT licensed][5]][6] [![Go Report Card][11]][12] [![Coverage Statusd][13]][14]

[1]: https://godoc.org/github.com/perlin-network/noise?status.svg
[2]: https://godoc.org/github.com/perlin-network/noise
[5]: https://img.shields.io/badge/license-MIT-blue.svg
[6]: LICENSE
[7]: https://img.shields.io/discord/458332417909063682.svg
[8]: https://discord.gg/dMYfDPM
[11]: https://goreportcard.com/badge/github.com/perlin-network/noise
[12]: https://goreportcard.com/report/github.com/perlin-network/noise
[13]: https://codecov.io/gh/perlin-network/noise/branch/master/graph/badge.svg
[14]: https://codecov.io/gh/perlin-network/noise

**noise** is an opinionated, easy-to-use P2P network stack for decentralized applications, and cryptographic protocols written in Go.

**noise** is made to be minimal, robust, developer-friendly, performant, secure, and cross-platform across multitudes of devices by making use of a small amount of well-tested, production-grade dependencies.

## Features

- Listen for incoming peers, query peers, and ping peers.
- Request for/respond to messages, fire-and-forget messages, and optionally automatically serialize/deserialize messages across peers.
- Optionally cancel/timeout pinging peers, sending messages to peers, receiving messages from peers, or requesting messages from peers via `context` support.
- Fine-grained control over a node and peers lifecycle and goroutines and resources (synchronously/asynchronously/gracefully start listening for new peers, stop listening for new peers, send messages to a peer, disconnect an existing peer, wait for a peer to be ready, wait for a peer to have disconnected).
- Limit resource consumption by pooling connections and specifying the max number of inbound/outbound connections allowed at any given time.
- Reclaim resources exhaustively by timing out idle peers with a configurable timeout.
- Establish a shared secret by performing an Elliptic-Curve Diffie-Hellman Handshake over Curve25519.
- Establish an encrypted session amongst a pair of peers via authenticated-encryption-with-associated-data (AEAD). Built-in support for AES 256-bit Galois Counter Mode (GCM).
- Peer-to-peer routing, discovery, identities, and handshake protocol via Kademlia overlay network protocol.

## Defaults

- No logs are printed by default. Set a logger via `noise.WithNodeLogger(*zap.Logger)`.
- A random Ed25519 keypair is generated for a new node.
- Peers attempt to be dialed at most three times.
- A total of 128 outbound connections are allowed at any time.
- A total of 128 inbound connections are allowed at any time.
- Connections timeout after 10 seconds if no reads/writes occur.

## Dependencies

- Logging is handled by [uber-go/zap](https://github.com/uber-go/zap).
- Byte buffer pooling is handled by [valyala/bytebufferpool](https://github.com/valyala/bytebufferpool).
- Unit tests are handled by [stretchr/testify](https://github.com/stretchr/testify).
- Ed25519 signatures are handled by [oasislabs/ed25519](https://github.com/oasislabs/ed25519).
- Elliptic-curve Diffie Hellman Key Exchange (ECDH) over Curve25519 is handled by [agl/ed25519](https://github.com/agl/ed25519).

## Setup

**noise** was intended to be used in Go projects that utilize Go modules. You may incorporate noise into your project as a library dependency by executing the following:

```shell
% go get -u github.com/perlin-network/noise
```
 
## Example

```go
package main

import (
    "context"
    "fmt"
    "github.com/perlin-network/noise"
)

func check(err error) {
    if err != nil {
        panic(err)
    }
}

// This example demonstrates how to send/handle RPC requests across peers, how to listen for incoming
// peers, how to check if a message received is a request or not, how to reply to a RPC request, and
// how to cleanup node instances after you are done using them.
func main() { 
    // Let there be nodes Alice and Bob.

    alice, err := noise.NewNode()
    check(err)

    bob, err := noise.NewNode()
    check(err)

    // Gracefully release resources for Alice and Bob at the end of the example.

    defer alice.Close()
    defer bob.Close()

    // When Bob gets a message from Alice, print it out and respond to Alice with 'Hi Alice!'

    bob.Handle(func(ctx noise.HandlerContext) error {
        if !ctx.IsRequest() {
            return nil
        }

        fmt.Printf("Got a message from Alice: '%s'\n", string(ctx.Data()))

        return ctx.Send([]byte("Hi Alice!"))
    })

    // Have Alice and Bob start listening for new peers.

    check(alice.Listen())
    check(bob.Listen())

    // Have Alice send Bob a request with the message 'Hi Bob!'

    res, err := alice.Request(context.TODO(), bob.Addr(), []byte("Hi Bob!"))
    check(err)

    // Print out the response Bob got from Alice.

    fmt.Printf("Got a message from Bob: '%s'\n", string(res))

    // Output:
    // Got a message from Alice: 'Hi Bob!'
    // Got a message from Bob: 'Hi Alice!'
}
```

For documentation and more examples, refer to noise's godoc [here](https://godoc.org/github.com/perlin-network/noise).

## Benchmarks

Benchmarks measure CPU time and allocations of a single node sending messages, requests, and responses to/from itself over 8 logical cores on a loopback adapter.

Take these benchmark numbers with a grain of salt.

```shell
% cat /proc/cpuinfo | grep 'model name' | uniq
model name	: Intel(R) Core(TM) i7-7700HQ CPU @ 2.80GHz

% go test -bench=. -benchtime=30s -benchmem
goos: linux
goarch: amd64
pkg: github.com/perlin-network/noise
BenchmarkRPC-8           2978550             14136 ns/op            1129 B/op         27 allocs/op
BenchmarkSend-8          9239581              4546 ns/op             503 B/op         12 allocs/op
PASS
ok      github.com/perlin-network/noise 101.966s
```

## Versioning

Breaking changes will comprise of an increment of the MINOR version, with bug patches/improvements involving an increment of the PATCH version. Semantic versioning will be adopted starting from v1.0.0.


## License

**noise**, and all of its source code is released under the MIT [License](https://github.com/perlin-network/noise/blob/master/LICENSE).