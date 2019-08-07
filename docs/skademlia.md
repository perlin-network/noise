# S/Kademlia Protocol

The S/Kademlia protocol is a overlay network protocol which allows p2p applications to easily achieve reliability in:
1) routing messages to/from peers,
2) bootstrapping new peers into the network,
3) and maintaining a clear overview of the liveness of all peers a node is connected to

## Identity Scheme 

Nodes must use a specialized type of identity scheme which S/Kademlia provides to wield S/Kademlia as their overlay network protocol of choice, as S/Kademlia explicitly requires node identities to be generated through a Proof of Work-based static and dynamic cryptographic puzzle parameterized by some constants c1 and c2.

Create a new node and passing the protocol:
```go
import "github.com/perlin-network/noise"
import "github.com/perlin-network/skademlia"

// Create new keys with `c1` and `c2` both 1
keys, err := skademlia.NewKeys(1, 1)
if err != nil {
    panic(err)
}

client := skademlia.NewClient(addr, keys)

// Pass the S/Kademlia protocols
client.SetCredentials(noise.NewCredentials(addr, client.Protocol()))
```

Apart from generating a random S/Kademlia-compatible keypair, you can also load in an existing keypair as well by calling `skademlia.LoadKeys(privateKey edwards25519.PrivateKey, c1 int, c2 int)`.
Where `edwards25519.PrivateKey` is a byte array of size 64.

c1 and c2 in this case are representative of the static and dynamic cryptographic puzzles protocol parameters c1 and c2.

S/Kademlia performs its own handshake procedure to validate node identities, and so other protocols may be registered before S/Kademlia like so:
```go
import "github.com/perlin-network/noise"
import "github.com/perlin-network/noise/cipher"
import "github.com/perlin-network/noise/handshake"
import "github.com/perlin-network/skademlia"

// Create new keys with `c1` and `c2` both 1
keys, err := skademlia.NewKeys(1, 1)
if err != nil {
    panic(err)
}

client := skademlia.NewClient(addr, keys)

// Pass the ECDH handshake, AEAD, and S/Kademlia protocols
client.SetCredentials(noise.NewCredentials(addr, handshake.NewECDH(), cipher.NewAEAD(), client.Protocol()))
```

Should any form of identity verification fail while performing S/Kademlia's custom handshake, the peer that failed the identity verification will be disconnected from our node.

Identities are considered to be verified should they pass the static and dynamic crypto puzzle parameterized by our selected parameter set (c1,c2,pmin,plen).

## Routing table

After a successful handshake, our incoming peer is logged into a Kademlia table data structure which is an array of LRU caches in which peer identities are stored with respect to the XOR distance between our nodes identity hash, and our peers identity hash.

A Kademlia table is instantiated when the S/Kademlia client is created.

Get the closest peers:
```go
import "github.com/perlin-network/noise"
import "github.com/perlin-network/skademlia"

var client skademlia.Client

// Get closest peers to our node based on XOR distance
ids := client.ClosestPeerIDs()

// A helper method to dial to each of closest peers and get the gRPC connections
conns := client.ClosestPeers()
```

The hash function used on top of our S/Kademlia public key for the time being can not be set, and is BLAKE-2b 256 bit.

The routing table gets updated every single time your node sends message to a peer or receives a message from a peer, and upon completing a handshake.
The way this is done is the S/Kademlia protocol registers interceptors to the node's gRPC server.

The routing table will only get updated should the number of prefixed byte differences between your nodes identity hash and an incoming peers identity hash exceed plen, and the sum of the prefixes bytes differences exceeds the value pmin.

By 'update', we refer to the routing table bucket (effectively a LRU cache) marking an incoming peers identity to be most recently used.

By default based on experiments from the original S/Kademlia paper, each routing table bucket may hold at most 16 peer identities.

## Peer eviction policy

Should a routing table bucket be full, we formally follow the peer eviction scheme denoted in the S/Kademlia paper where the last peer in the bucket would be pinged. If the last peer in the bucket fails to be pinged, then they are evicted from the routing table, and the incoming peers identity gets put towards the front of the bucket.

If the last peer in the bucket responds to our nodes ping however, the last peer gets moved the front of the bucket, and the incoming peer is ignored s.t. the incoming peer is disconnected from our node.

## Configuring S/Kademlia

While instantiating a new skademlia protocol block, you may customize any of S/Kademlia's security parameters denoted by the parameter set (c1, c2, prefix min, prefix len) like so:
```go
import "github.com/perlin-network/noise"
import "github.com/perlin-network/noise/cipher"
import "github.com/perlin-network/noise/handshake"
import "github.com/perlin-network/skademlia"

var keys *skademlia.Keypair

client := skademlia.NewClient(
    addr,
    keys, 
    skademlia.WithC1(1), 
    skademlia.WithC2(1), 
    skademlia.WithPrefixDiffLen(128), 
    skademlia.WithPrefixDiffMin(32),
)
```

## Bootstrapping new peers
In order to bootstrap your newly created node to the peers closest to you within the network, you may make use of the FIND_NODE RPC call described in Section 4.4 of S/Kademlia's paper: "Lookup over disjoint paths".

Given a node instance N, and a S/Kademlia ID as a target T, α disjoint lookups take place in parallel over all closest peers to N to target T, with at most d lookups happening at once.

Each disjoint lookup queries at most α peers, with the RPC call returning at most Bsize S/Kademlia peer IDs closest to that of a specified target T where Bsize is the maximum number of IDs that may be stored within a routing table bucket.

In the case of our implementation, Bsize by default is set to 16.

In amidst the disjoint lookup queries, our routing table will be populated with peers we dial/connect to; essentially enforcing the FIND_NODE RPC call to be akin to a bootstrapping mechanism for having our node connect to the peers we are closest to within our network.

The S/Kademlia protocol provides a `Bootstrap()` function to perform FIND_NODE RPC call on the closest peers:
```go
import "github.com/perlin-network/noise"
import "github.com/perlin-network/noise/skademlia"

// Create new keys with `c1` and `c2` both 1
keys, err := skademlia.NewKeys(1, 1)
if err != nil {
    panic(err)
}

client := skademlia.NewClient(addr, keys)
client.SetCredentials(noise.NewCredentials(addr, handshake.NewECDH(), cipher.NewAEAD(), client.Protocol()))

if _, err := client.Dial("127.0.0.1:3001"); err != nil {
    panic("failed to connect to the peer we wanted to connect to")
}

peers := client.Bootstrap()
fmt.Printf("Bootstrapped with peers: %+v\n", peers)
```

## Broadcasting messages to peers

After bootstrapping to the closest peers to us within the network, we may make use of some methods S/Kademlia provides to get the peers and their gRPC connections.

Using the gRPC connections, you can create instance of your gRPC clients and call the functions.

```proto
syntax = "proto3";

package main;

message Text {
    string message = 1;
}

service Chat {
    rpc Stream(stream Text) returns (Text) {}
}
```

```go
import "github.com/perlin-network/noise"
import "github.com/perlin-network/noise/cipher"
import "github.com/perlin-network/noise/handshake"
import "github.com/perlin-network/skademlia"
import "google.golang.org/grpc"

// Create new keys with `c1` and `c2` both 1
keys, err := skademlia.NewKeys(1, 1)
if err != nil {
    panic(err)
}

client := skademlia.NewClient(addr, keys)
client.SetCredentials(noise.NewCredentials(addr, handshake.NewECDH(), cipher.NewAEAD(), client.Protocol()))

if _, err := client.Dial("127.0.0.1:3001"); err != nil {
    panic("failed to connect to the peer we wanted to connect to")
}

client.Bootstrap()

reader := bufio.NewReader(os.Stdin)

// For each input message, send it to the closest peers.
for {
    line, _, err := reader.ReadLine()

    if err != nil {
        panic(err)
    }
    
    // Get the closest peer connections
    conns := client.ClosestPeers()

    // For each peer connection, create the chat client and send the message
    for _, conn := range conns {
        chat := NewChatClient(conn)

        stream, err := chat.Stream(context.Background())
        if err != nil {
            continue
        }

        // Send the message
        if err := stream.Send(&Text{Message: string(line)}); err != nil {
            continue
        }
    }
}
```

You can refer to `chat` example for the complete code.

## Callbacks

The S/Kademlia protocol comes with callbacks that you can register to listen to peer connected (join) and disconnected (leave) events.

Register peer connected callback:
```go
import "github.com/perlin-network/noise"
import "github.com/perlin-network/skademlia"

var client skademlia.Client

client.OnPeerJoin(func(conn *grpc.ClientConn, id *skademlia.ID) {
    // Do some works
})
```

<br />

Register peer disconnected callback:
```go
import "github.com/perlin-network/noise"
import "github.com/perlin-network/skademlia"

var client skademlia.Client

client.OnPeerLeave(func(conn *grpc.ClientConn, id *skademlia.ID) {
    // Do some works
})
```