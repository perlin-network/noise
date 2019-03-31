# Noise

[![GoDoc][1]][2] [![Discord][7]][8] [![MIT licensed][5]][6] [![Build Status][9]][10] [![Go Report Card][11]][12] [![Coverage Statusd][13]][14]

[1]: https://godoc.org/github.com/perlin-network/noise?status.svg
[2]: https://godoc.org/github.com/perlin-network/noise
[5]: https://img.shields.io/badge/license-MIT-blue.svg
[6]: LICENSE
[7]: https://img.shields.io/discord/458332417909063682.svg
[8]: https://discord.gg/dMYfDPM
[9]: https://travis-ci.org/perlin-network/noise.svg?branch=master
[10]: https://travis-ci.org/perlin-network/noise
[11]: https://goreportcard.com/badge/github.com/perlin-network/noise
[12]: https://goreportcard.com/report/github.com/perlin-network/noise
[13]: https://codecov.io/gh/perlin-network/noise/branch/master/graph/badge.svg
[14]: https://codecov.io/gh/perlin-network/noise


<img align="right" width=400 src="media/chat.gif">

**noise** is an opinionated, easy-to-use P2P network stack for
*decentralized applications, and cryptographic protocols* written in
[Go](https://golang.org/) by [the Perlin team](https://perlin.net).

**noise** is made to be robust, developer-friendly, performant, secure, and
cross-platform across multitudes of devices by making use of well-tested,
production-grade dependencies.

<hr/>

By itself, **noise** is a low-level, stateless, concurrent networking library that easily allows you to incorporate fundamental features any modern p2p application needs such as:

1) cryptographic primitives (Ed25519, PoW, AES-256),
2) message serialization/deserialization schemes (byte-order little endian, protobuf, msgpack),
3) network timeout/error management (on dial, on receive message, on send buffer full),
4) network-level atomic operations (receive-then-lock),
5) and NAT traversal support (NAT-PMP, UPnP).

Out of its own low-level constructs, noise additionally comes bundled with a high-level `protocol` package comprised of a large number of production-ready, high-level protocol building blocks such as:

1) handshake protocol implementations (Elliptic-Curve Diffie Hellman),
2) peer routing/discovery protocol implementations (S/Kademlia),
3) message broadcasting protocol implementations (S/Kademlia),
4) overlay network protocol implementations (S/Kademlia),
5) cryptographic identity schemes (Ed25519 w/ EdDSA signatures),
6) and authenticated encryption schemes (AES-256 GCM AEAD).

Every single building block is easily configurable, and may be mixed and matched together to help you kickstart your journey on developing secure, debuggable, and highly-performant p2p applications.

## Setup

Make sure to have at the bare minimum [Go 1.11](https://golang.org/dl/) installed before incorporating **noise** into your project.

After installing _Go_, you may choose to either:

1. directly incorporate noise as a library dependency to your project,

```bash
# Be sure to have Go modules enabled: https://github.com/golang/go/wiki/Modules
export GO111MODULE=on

# Run this inside your projects directory.
go get github.com/perlin-network/noise
```

2. or checkout the source code on Github and run any of the following commands below.

```bash
# Be sure to have Go modules enabled: https://github.com/golang/go/wiki/Modules
export GO111MODULE=on

# Run an example creating a cluster of 3 peers automatically
# discovering one another.
[terminal 1] go run examples/chat/main.go -p 3000
[terminal 2] go run examples/chat/main.go -p 3001 127.0.0.1:3000
[terminal 3] go run examples/chat/main.go -p 3002 127.0.0.1:3001

# Optionally run test cases.
go test -v -count=1 -race ./...
```

## We're hiring!

Here at [Perlin](https://perlin.net), we spend days and weeks debating, tinkering, and researching what is out there in academia to bring to industries truly resilient, open-source, secure, economic, and decentralized software to empower companies, startups, and users.
                                                        
Our doors are open to academics that have a knack for distributed systems, engineers that want to explore unknown waters, frontend developers that want to make and evangelize the next generation of customer-facing applications, and graphics designers that yearn to instrument together greater user experiences for decentralized applications.

## Contributions

First of all, _thank you so much_ for taking part in our efforts for creating a p2p networking stack that can meet everyones needs without sacrificing developer productivity!

All code contributions to _noise_ should comply with all idiomatic Go standards listed [here](https://github.com/golang/go/wiki/CodeReviewComments).

All commit messages should be in the format:

```bash
module_name_1, module_name_2: description of the changes you made to the two
    modules here as a sentence
```

Be sure to use only imperative, present tense within your commit messages and optionally include motivation for your changes _two lines breaks_ away from your commit message.

This allows other maintainers and contributors to know which modules you are modifying/creating within the code/docs repository.

Lastly, be sure to consider backwards compatibility.

New modules/methods are perfectly fine, but changing code living in `noise.Node` or `noise.Peer` radically for example would break a lot of existing projects utilizing _noise_.

Additionally, if you'd like to talk to us or any of the team in real-time, be sure to join our [Discord server](https://discord.gg/dMYfDPM)!

We are heavily active, ready to answer any questions/assist you with any code/doc contributions at almost any time.

## License

**noise**, and all of its source code is released under the MIT [License](https://github.com/perlin-network/noise/blob/master/LICENSE).
