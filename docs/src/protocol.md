# Protocol

Now that we have gone over the concepts of nodes and peers, you know how to setup node identities, listen for/connect to peers, send messages to/receive messages from peers, deal with message serialization/deserialization, and a lot more through Noise.

On a high level, **noise** comes with a `protocol` package that builds on top of a variety of node/peer events to allow you to easily compose your p2p applications networking protocol out of composable `protocol.Block`'s.

```go
package protocol

import "github.com/perlin-network/noise"

type Block interface {
	OnRegister(p *Protocol, node *noise.Node)
	OnBegin(p *Protocol, peer *noise.Peer) error
	OnEnd(p *Protocol, peer *noise.Peer) error
}
```

A `protocol.Block` represents a implementation of a modular, well-defined sub-protocol defined by 3 events: `OnRegister`, `OnBegin`, and `OnEnd`.

When a block gets registered into a protocol, `OnRegister` gets called. Typically, you would define variables, register opcodes, register callback functions to your `*noise.Node`, or register `noise.Message`'s inside `OnRegister`.

`OnBegin` gets called when a peer has completed executing all other prior blocks. You would define the core block logic inside `OnBegin` you would expect a peer to follow, or set custom metadata to a peer, or even spawn infinite-loop receive workers to handle messages from peers here.

`OnEnd` gets called when a peer disconnects, in the case that you wish to clean up any particular tracked data of a peer should they depart from your network.

For both `OnBegin` and `OnEnd`, you can return a `protocol.DisconnectPeer` error at any time which will directly halt execution of the protocols logic and disconnect the peer from your node should they misbehave/error out in any way.

Returning any other sort of error otherwise would directly log the error onto your console as a warning.

Returning nil in `OnBegin` would have peers transition into the next block within the protocol.

When a peer runs out of blocks to execute, they remain dormant for the time being.

## Composition

Your p2p application would compose together these blocks in an order you may choose to allow you to easily build a custom, secure, and performant networking protocol.

On a high level, examples of what a `protocol.Block` may represent are:

1. a handshake protocol section,
2. a session establishment protocol section,
3. a message broadcasting protocol section,
4. or an overlay network section.

Noise provides a number of high-level, well-tested basic protocol building blocks any p2p application would need to help you kickstart building your p2p application.

After picking a few blocks, or even implementing your own `protocol.Block`'s, you may then define your protocol like so:

```go
import (
	"github.com/perlin-network/noise/protocol"
	"github.com/perlin-network/noise/handshake/ecdh"
	"github.com/perlin-network/noise/cipher/aead"
	"github.com/perlin-network/noise/skademlia"
)

policy := protocol.New()

// First, all nodes must perform an Elliptic-Curve Diffie-Hellman (ECDH) handshake
// between each other to establish a shared symmetric encryption key.
policy.Register(ecdh.New())

// The shared key from ECDH then gets derived by a HMAC to establish a new encrypted
// session where all communication between a pair of nodes is then encrypted using
// Authenticated Encryption w/ Authenticated Data (AEAD) via AES-256 GCM.
policy.Register(aead.New())

// After all communication has been readily encrypted, perform a S/Kademlia handshake
// to exchange S/Kademlia node IDs, setup a S/Kademlia routing table, and start
// handling S/Kademlia-related overlay network RPC messages.
policy.Register(skademlia.New())
```

The order in which you register the blocks as you may be able to tell matters, as `protocol` for the time being linearly traverses through all blocks in the order they are registered.

After registering the ordering in which you expect peers to execute blocks, you may then enforce your custom networking protocol on multiple `*noise.Node` instances at once like so:

```go
import "github.com/perlin-network/noise"

var node1, node2 *noise.Node

// Enforce both node1 and node2 to follow your custom policy
// when it comes to dealing with peers.
policy.Enforce(node1)
policy.Enforce(node2)
```

All peers which connect to either `node1` or `node2` from then on will follow your policy, and be disconnected from you should they fail to otherwise.

Typically, you would define and enforce your protocol on your node before you start listening for peers and start dialing/connecting to other peers.

> **Note:** All protocol logic is handled in a separate goroutine from where nodes/peers are instantiated, so it is safe to execute blocking code inside a block so long as you intend to block a peer from progressing further from within a protocol until some condition is met.

## Helpers

A set of helper functions are provided in the `protocol` package to help you manage a small set of well-defined metadata we at Perlin found to be common-place in all kinds of networking protocools.

Shared keys, node IDs, peer IDs, and mapping peer IDs to `*noise.Peer` instances are what we provide off-the-shelf.

We additionally provide a standardized interface for defining what an ID should comprise of:

```go
package protocol

import "github.com/perlin-network/noise"
import "fmt"

type ID interface {
	fmt.Stringer
	noise.Message

	Equals(other ID) bool

	PublicKey() []byte
	Hash() []byte
}


func HasSharedKey(peer *noise.Peer) bool { ... }
func LoadSharedKey(peer *noise.Peer) []byte { ... }
func MustSharedKey(peer *noise.Peer) []byte { ... } 
func SetSharedKey(peer *noise.Peer, sharedKey []byte) { ... }
func DeleteSharedKey(peer *noise.Peer) { ... }

func SetNodeID(node *noise.Node, id ID) { ... }
func DeleteNodeID(node *noise.Node) { ... }
func HasPeerID(peer *noise.Peer) bool { ... }
func SetPeerID(peer *noise.Peer, id ID) { ... }
func DeletePeerID(peer *noise.Peer) { ... }

func NodeID(node *noise.Node) ID { ... }
func PeerID(peer *noise.Peer) ID { ... }
func Peer(node *noise.Node, id ID) *noise.Peer { ... }
```

If you would like suggest any other type of metadata you think is common-place within a large variety of networking protocols that could be managed from within the `protocol` package, feel free to post up a Github issue.

For the next couple of pages, we'll go over a couple of high-level protocol blocks which Noise provides from the get-go.

