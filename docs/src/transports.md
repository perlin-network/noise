# Transports

The `transport` package provides Noise built-in transport layer support for:

1. TCP
2. In-Memory

To use either one of the transport layers, it is a matter of setting the option `Transport`:

```go
import (
	"github.com/perlin-network/noise"
	"github.com/perlin-network/noise/transport"
)

params := noise.DefaultParams()

// Use TCP as your nodes transport layer.
params.Transport = transport.NewTCP()

// Have your nodes transport layer be in-memory.
params.Transport = transport.NewBuffered()
```

## A small note.

Noise for the time being really only currently supports specific types of network transport layer protocols.

Specifically, the ones that guarantee message ordering, and guarantee reliable message delivery through error-checking schemes.

_Unsurprisingly_, ones like TCP.

The reasoning for it is simple: you sacrifice performance having to reliably guarantee message ordering should you do it on the application layer (in which Noise operates within).

Though what's more, one of the biggest reasons a large number of complex networking protocol constructs can be so succinctly represented in Noise is precisely because Noise relies on the transport layer for linearized message ordering.

> _However_, that does not mean only TCP is supported by Noise.

There still exists the option of introducing reliable ordering on top of an unreliable transport layer and plugging it into Noise, or really just plugging in any type of transport layer into Noise that you can think of.

That being said though, be aware that you're crossing unexplored waters should you attempt to plug an unreliable transport layer into Noise. 

In spite of my succinct warning, there exists a large number of transport layers that would be interesting to see coupled with Noise.

Examples of those transport layers would include QUIC, or Tor.

Hence, we made it simple for you to be able to plug in a custom transport layer implementation into Noise.

All you have to do is have your transport layer-related implementation code implement the following interface:

```go
package transport

import (
	"fmt"
	"net"
)

type Layer interface {
	fmt.Stringer

	Listen(host string, port uint16) (net.Listener, error)
	Dial(address string) (net.Conn, error)

	IP(address net.Addr) net.IP
	Port(address net.Addr) uint16
}
```

Once implemented, set an instance of your implementation to the `Transport` option and enjoy.

**Note:** There is a good chance that if you're looking to plug-n-play an existing transport layer Go implementation into Noise, be sure to look out for the ones that export `Listen()` or `Dial()` functions that are interoperable with Go's `net` standard library package.