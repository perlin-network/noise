# NAT Traversal

NAT traversal schemes which Noise provides built-in support for are:

1. NAT-PMP
2. UPnP IGDv1/IDGv2

Using either one of the schemes above is a matter of setting the `NAT` option like so:

```go
import (
	"github.com/perlin-network/noise"
	"github.com/perlin-network/noise/nat"
)

params := noise.DefaultParams()

// Enable NAT-PMP NAT traversal support.
params.NAT = pmp.New()

// Enable UPnP IGDv1/IDGv2 NAT traversal support.
params.NAT = upnp.New()
```

You may additionally implement your own NAT traversal protocol support which your nodes may make use of. Noise only requires the following interface to be implemented:

```go
package nat

import (
	"net"
	"time"
)

// Provider represents a barebones generic interface to a NAT traversal
// network protocol.
type Provider interface {
	ExternalIP() (net.IP, error)
	AddMapping(protocol string, externalPort, internalPort uint16, expiry time.Duration) error
	DeleteMapping(protocol string, externalPort, internalPort uint16) (err error)
}
```

When a NAT traversal protocol is specified, a port mapping is automatically made which maps the internal port `Port` to an external port `ExternalPort`.

The port mapping is destroyed upon calling a `Kill()`, which blocks the current goroutine until the node gracefully comes to a complete stop of all of its operations.

> **Note:** Noise does not provide any utilities as of yet to help you call the `Kill()` function automatically when your application exits! Be sure to do this yourself or risk leaving a port mapping open on your PC.

Should no `Host` option be set before instantiating your node, the NAT traversal protocol will also be queried for your nodes external address.

A quick tip when making your own implementation to use with Noise is to have all device gateway-related setup happen in the constructor of your code.

Additionally, should any errors occur, immediately `panic()` as it is undefined behavior as to what happens if a broken NAT traversal mechanism is used for instantiating and running a node.

All built-in NAT traversal protocol support will invoke `panic()` should the nodes' router not support a specified NAT traversal protocol.