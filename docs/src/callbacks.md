# Callbacks

**noise** comes with a lot of node/peer events that may be intercepted through the provision of a callback function.

As a result, noise provides a flexible, asynchronous callback manager that allows for a linearly ordered set of
callback functions to operate/be invoked when a particular event occurs within the `callbacks` package.

```go
import "github.com/perlin-network/noise"
import "github.com/perlin-network/noise/callbacks"
import "fmt"

func main() {
    var node *noise.Node 
    
    // ... setup node here
    
    node.OnPeerDialed(func(node *noise.Node, peer *noise.Peer) error {
        fmt.Println("This callback will only get called on the very first peer dialed!")
        
    	return callbacks.Deregister
    })

    _, _ = node.Dial("some peer address here")
    
    select {}
}
``` 

One particularly interesting capability of noise's callback managers is that in amidst a callback function, a `callbacks.Deregister` error
may be returned from within the callback to deregister the callback function after it is invoked once/multiple times on some data/event.

> **Note:** Callback functions are not independently executed in their own separate goroutines! Be aware that invoking any blocking operation such as an infinite loop within any callback function will potentially deadlock Noise.

## Callback function signatures

```go
package noise

import "github.com/perlin-network/noise/payload"

type OnErrorCallback func(node *Node, err error) error
type OnPeerErrorCallback func(node *Node, peer *Peer, err error) error
type OnPeerDisconnectCallback func(node *Node, peer *Peer) error
type OnPeerInitCallback func(node *Node, peer *Peer) error

type BeforeMessageSentCallback func(node *Node, peer *Peer, msg []byte) ([]byte, error)
type BeforeMessageReceivedCallback func(node *Node, peer *Peer, msg []byte) ([]byte, error)

type AfterMessageSentCallback func(node *Node, peer *Peer) error
type AfterMessageReceivedCallback func(node *Node, peer *Peer) error

type AfterMessageEncodedCallback func(node *Node, peer *Peer, header, msg []byte) ([]byte, error)

type OnPeerDecodeHeaderCallback func(node *Node, peer *Peer, reader payload.Reader) error
type OnPeerDecodeFooterCallback func(node *Node, peer *Peer, msg []byte, reader payload.Reader) error

type OnMessageReceivedCallback func(node *Node, opcode Opcode, peer *Peer, message Message) error
```

## Interceptable node events

```go
// OnListenerError registers a callback for whenever our nodes listener
// fails to accept an incoming peer.
func (n *Node) OnListenerError(c OnErrorCallback) { ... }

// OnPeerConnected registers a callback for whenever a peer has successfully
// been accepted by our node.
func (n *Node) OnPeerConnected(c OnPeerInitCallback) { ... }

// OnPeerDialed registers a callback for whenever a peer has been successfully dialed.
func (n *Node) OnPeerDialed(c OnPeerInitCallback) { ... }

// OnPeerDisconnected registers a callback whenever a peer has been disconnected.
func (n *Node) OnPeerDisconnected(srcCallbacks ...OnPeerDisconnectCallback) { ... }

// OnPeerInit registers a callback for whenever a peer has either been successfully
// dialed, or otherwise accepted by our node.
//
// In essence a helper function that registers callbacks for both `OnPeerConnected`
// and `OnPeerDialed` at once.
func (n *Node) OnPeerInit(srcCallbacks ...OnPeerInitCallback) { ... }
```

## Interceptable peer events

```go
// BeforeMessageSent registers a callback to be called before a message
// is sent to a specified peer.
func (p *Peer) BeforeMessageSent(c BeforeMessageSentCallback) { ... }

// BeforeMessageReceived registers a callback to be called before a message
// is to be received from a specified peer.
func (p *Peer) BeforeMessageReceived(c BeforeMessageReceivedCallback) { ... }

// AfterMessageSent registers a callback to be called after a message
// is sent to a specified peer.
func (p *Peer) AfterMessageSent(c AfterMessageSentCallback) { ... }

// AfterMessageReceived registers a callback to be called after a message
// is to be received from a specified peer.
func (p *Peer) AfterMessageReceived(c AfterMessageReceivedCallback) { ... }

// OnDecodeHeader registers a callback that is fed in the contents of the
// header portion of an incoming message from a specified peer.
func (p *Peer) OnDecodeHeader(c OnPeerDecodeHeaderCallback) { ... }

// OnDecodeFooter registers a callback that is fed in the contents of the
// footer portion of an incoming message from a specified peer.
func (p *Peer) OnDecodeFooter(c OnPeerDecodeFooterCallback) { ... }

// OnEncodeHeader registers a callback that is fed in the raw contents of
// a message to be sent, which then outputs bytes that are to be appended
// to the header of an outgoing message.
func (p *Peer) OnEncodeHeader(c AfterMessageEncodedCallback) { ... }

// OnEncodeFooter registers a callback that is fed in the raw contents of
// a message to be sent, which then outputs bytes that are to be appended
// to the footer of an outgoing message.
func (p *Peer) OnEncodeFooter(c AfterMessageEncodedCallback) { ... }

// OnConnError registers a callback for whenever something goes wrong with the
// connection to our peer.
func (p *Peer) OnConnError(c OnPeerErrorCallback) { ... }

// OnDisconnect registers a callback for whenever the peer disconnects.
func (p *Peer) OnDisconnect(srcCallbacks ...OnPeerDisconnectCallback) { ... }
```