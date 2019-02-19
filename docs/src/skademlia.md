# S/Kademlia

**noise** provides a full-fledged implementation of the S/Kademlia overlay network protocol which
allows p2p applications to easily achieve reliability in:

1. routing messages to/from peers,
2. bootstrapping new peers into the network,
3. and maintaining a clear overview of the _liveness_ of all peers a node is connected to.

## Identity scheme

Nodes must use a specialized type of identity scheme which S/Kademlia provides to wield S/Kademlia
as their overlay network protocol of choice, as S/Kademlia explicitly requires node identities
to be generated through a [Proof of Work](https://en.wikipedia.org/wiki/Proof-of-work_system)-based static and
dynamic cryptographic puzzle parameterized by some constants \\(c_1\\) and \\(c_2\\).

In order to incorporate S/Kademlia as an overlay network into your protocol, you may set it
up like so:

```go
import "github.com/perlin-network/noise"
import "github.com/perlin-network/skademlia"

func main() {
	params := noise.DefaultParams()
	params.Keys = skademlia.RandomKeys()
	params.Port = uint16(3000)
	
	node, err := noise.NewNode(params)
	if err != nil {
		panic(err)
	}
	
	protocol.New().
		Register(skademlia.New()).
		Enforce(node)
	
	// ... do other stuff with your node here.
}
```

Apart from generating a random S/Kademlia-compatible keypair, you can also load in an existing
keypair as well by calling `skademlia.LoadKeys(privateKey []byte, c1 int, c2 int)`.
 
`c1` and `c2` in this case are representative of the static and dynamic cryptographic puzzles
protocol parameters \\(c_1\\) and \\(c_2\\).

S/Kademlia performs its own handshake procedure to validate node identities, and so other
handshake procedures/message cipher setup blocks may be registered before S/Kademlia like so:

```go
import "github.com/perlin-network/noise"
import "github.com/perlin-network/skademlia"
import "github.com/perlin-network/handshake/ecdh"
import "github.com/perlin-network/cipher/aead"

func main() {
	params := noise.DefaultParams()
	params.Keys = skademlia.RandomKeys()
	params.Port = uint16(3000)
	
	node, err := noise.NewNode(params)
	if err != nil {
		panic(err)
	}
	
	// Setup our protocol and enforce it on our node.
	protocol.New().
		Register(ecdh.New()).
		Register(aead.New())
		Register(skademlia.New()).
		Enforce(node)
	
	// ... do other stuff with your node here.
}
```


Should any form of identity verification fail while performing S/Kademlia's custom handshake,
the peer that failed the identity verification will be disconnected from our node.

Identities are considered to be verified should they pass the static and dynamic crypto puzzle
parameterized by our selected parameter set \\((c_1, c_2, p_{min}, p_{len})\\).

## Routing table

After a successful handshake, our incoming peer is logged into a [Kademlia table data structure](https://en.wikipedia.org/wiki/Kademlia#Routing_tables)
which is an array of LRU caches in which peer identities are stored with respect to the XOR distance between our nodes
identity hash, and our peers identity hash.

A Kademlia table is instantiated when the S/Kademlia block is registered to a protocol which is enforced on a given node instance.

In order to grab an instance of the Kademlia table underlying our node, you may call `skademlia.Table(*noise.Node)`:

```go
import "github.com/perlin-network/noise"
import "github.com/perlin-network/noise/skademlia"

var node *noise.Node

// Grab the Kademlia table underlying our node. It will panic
// if our node does not have S/Kademlia enforced.
table := skademlia.Table(node)

// Print the 16 closest peers to our node based on XOR distance.
fmt.Println(skademlia.FindClosestPeers(table,
	protocol.NodeID(node).(skademlia.ID).Hash(), 16))

// Directly update the liveness of a peer denoted by their ID.
var peerID skademlia.ID
err := skademlia.UpdateTable(node, peerID)
if err != nil {
	panic("failed to update our peers status in our kademlia table")
}
```

The hash function used on top of our S/Kademlia public key for the time being can not be set, and is BLAKE-2b 256 bit.

The routing table gets updated every single time your node receives a message from a peer, and upon completing a handshake.

The routing table will only get updated should the number of prefixed byte differences between your nodes identity hash and an incoming peers identity hash exceed \\(p_{len}\\), and the sum of the prefixes bytes differences exceeds the value \\(p_{min}\\).

By 'update', we refer to the routing table bucket (effectively a LRU cache) marking an incoming peers identity to be most recently used.

By default based on experiments from the original S/Kademlia paper, each routing table bucket may hold at most 16 peer identities.

## Peer eviction policy

Should a routing table bucket be full, we formally follow the peer eviction scheme denoted in the S/Kademlia paper where
the last peer in the bucket would be pinged. If the last peer in the bucket fails to be pinged, then they are evicted from
the routing table, and the incoming peers identity gets put towards the front of the bucket.

If the last peer in the bucket responds to our nodes ping however, the last peer gets moved the front of the bucket,
and the incoming peer is ignored s.t. the incoming peer is disconnected from our node.

## Configuring S/Kademlia

While instantiating a new `skademlia` protocol block, you may customize any of S/Kademlia's security
parameters denoted by the parameter set \\( (c_1, c_2, p_{min}, p_{len}) \\) like so:

```go
var node *noise.Node

block := skademlia.New()

// Setup S/Kademlia security parameters here.
block.WithC1(int(16))
block.WithC2(int(16))
block.WithPrefixDiffLen(int(128))
block.WithPrefixDiffMaxLen(int(32))

// Register the protocol block and enforce it on our node.
protocol.New().Register(block).Enforce(node)
```

## Bootstrapping new peers

In order to bootstrap your newly created node to the peers closest to you within the network, you may make use of the
`FIND_NODE` RPC call described in Section 4.4 of S/Kademlia's paper: "Lookup over disjoint paths".

Given a node instance \\(N\\), and a S/Kademlia ID as a target \\(T\\), \\(\alpha\\) disjoint lookups
take place in parallel over all closest peers to \\(N\\) to target \\(T\\), with at most \\(d\\)
lookups happening at once.

Each disjoint lookup queries at most \\(\alpha\\)  peers, with the RPC call returning at
most \\(B_{size}\\) S/Kademlia peer IDs closest to that of a specified target \\(T\\) where
\\(B_{size}\\) is the maximum number of IDs that may be stored within a routing table bucket.

In the case of our implementation, \\(B_{size}\\) by default is set to 16.

In amidst the disjoint lookup queries, our routing table will be populated with peers we dial/connect to; essentially
enforcing the `FIND_NODE` RPC call to be akin to a bootstrapping mechanism for having our node connect to the peers we are closest
to within our network.

You may invoke the `FIND_NODE` RPC call after setting up your node to work with S/Kademlia like so:

```go
import "github.com/perlin-network/noise"
import "github.com/perlin-network/skademlia"

func main() {
	params := noise.DefaultParams()
	params.Keys = skademlia.RandomKeys()
	params.Port = uint16(3000)
	
	node, err := noise.NewNode(params)
	if err != nil {
		panic(err)
	}
	
	protocol.New().
		Register(skademlia.New()).
		Enforce(node)
	
	go node.Listen()
	
	// Attempt to dial a peer located at the address
	// 127.0.0.1:3001.
	peer, err := node.Dial("127.0.0.1:3001")
	if err != nil {
		panic("failed to connect to the peer we wanted to connect to")
	}
	
	// Block the current goroutine until we finish
	// performing a S/Kademlia handshake with our peer.
	skademlia.WaitUntilAuthenticated(peer)
	
	// Lookup the 16 (skademlia.BucketSize()) closest peers to our
	// node ID throughout the network with at most 8 disjoint lookups
	// happening at once.
	//
	// Calling this method automatically populates our routing table
	// and thus immediately bootstraps us/dials us to the peers closest to us
	// within the network.
	peers := skademlia.FindNode(node,
		protocol.NodeID(node).(skademlia.ID), skademlia.BucketSize(), 8)
	
	// Print the 16 closest peers to us we have found via the `FIND_NODE`
	// RPC call.
    fmt.Printf("Bootstrapped with peers: %+v\n", peers)
	
	// Print the peers we currently are routed/connected to.
	fmt.Printf("Peers we are connected to: %+v\n", table.GetPeers())
}
```

As denoted in the code example above, a helpful function which you may use to block
a particular goroutine until an incoming peer has successfully completed a S/Kademlia handshake is
`skademlia.WaitUntilAuthenticated(*noise.Peer)`.


## Broadcasting messages to peers

After bootstrapping to the closest peers to us within the network, we may make use of
some methods S/Kademlia provides to reliably broadcast to the peers we are routed to.

There exists a `skademlia.Broadcast(*noise.Node, noise.Message) []error` method which broadcasts
to at most 16 of the closest peers to us in parallel a Noise-registered message we specify.

Should we choose to not care about broadcasting errors and simply fire-and-forget a Noise-registered
message we wish to broadcast, we may make use of the `skademlia.BroadcastAsync(*noise.Node, noise.Message)` function.

The `chat` example in Noise calls `BroadcastAsync` to broadcast a chat message to peers closest to us like so:

```go
import "github.com/perlin-network/noise/payload"

var _ noise.Message = (*chatMessage)(nil)

type chatMessage struct {
    text string
}

func (m *chatMessage) Read(reader payload.Reader) (noise.Message, error) {
	var err error
	
	m.text, err = reader.ReadString()
	if err != nil { return nil, err}
	
	return m, nil
}

func (m *chatMessage) Write() []byte {
	return payload.NewWriter(nil).WriteString(m.text).Bytes()
}

func main() {
	noise.RegisterMessage(noise.NextAvailableOpcode(), (*chatMessage)(nil))
	
	var node *noise.Node
	
	// ... setup node, dial a peer, and perform `FIND_NODE` here.
	
	reader := bufio.NewReader(os.Stdin)
    
    for {
        txt, err := reader.ReadString('\n')
    
        if err != nil {
            panic(err)
        }
    
        // Synchronous broadcast.
        errs := skademlia.Broadcast(node, chatMessage{text: strings.TrimSpace(txt)})
        
        if len(errs) > 0 {
        	fmt.Println("Got errors broadcasting our chat message to peers:", errs)
        }
        
        // Asynchronous broadcast.
        // skademlia.BroadcastAsync(node, chatMessage{text: strings.TrimSpace(txt)})
    }
}
```