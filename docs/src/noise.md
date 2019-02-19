# noise 

**noise** is a peer-to-peer (p2p) networking stack with minimal dependencies which allows for extreme granularity in defining, testing, developing and deploying complex, secure, performant, and robust networking protocols in [Go](https://golang.org) written by [the Perlin team](https://perlin.net).

By itself, noise is a low-level, stateless, concurrent networking library that easily allows you to incorporate fundamental features any modern p2p application needs such as:

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

> **noise** is truly open-source and free. You can find the source code on [GitHub](https://github.com/perlin-network/noise). Issues and feature requests can be posted on the [GitHub issue tracker](https://github.com/perlin-network/noise/issues).

```go
package main

import (
	"fmt"
	
	"github.com/perlin-network/noise"
    "github.com/perlin-network/noise/cipher/aead"
    "github.com/perlin-network/noise/handshake/ecdh"
    "github.com/perlin-network/noise/identity/ed25519"
    "github.com/perlin-network/noise/protocol"
    "github.com/perlin-network/noise/rpc"
    "github.com/perlin-network/noise/skademlia"
)

func main() {
    params := noise.DefaultParams()
    params.Keys = ed25519.Random()
    params.Port = uint16(3000)
    
    node, err := noise.NewNode(params)
    if err != nil {
        panic(err)
    }
    
    protocol.New().
    	Register(ecdh.New()).
    	Register(aead.New()).
    	Register(skademlia.New()).
    	Enforce(node)
    
    fmt.Printf("Listening for peers on port %d.\n", node.ExternalPort())
    
    go node.Listen()
    
    select{}
}
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