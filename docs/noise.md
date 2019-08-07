# Noise

Noise extends gRPC's transport protocol to enable you to create robust and secure decentralized applications on top of gRPC.

The way Noise does this is by creating `Credentials` type which implements `TransportCredentials` of gRPC. 

To use the protocols on a gRPC server, you can create an instance of `Credentials` and pass the protocols you want. 
Then, you can then pass the `Credentials` instance to the gRPC server.

```go
import "github.com/perlin-network/noise"
import "google.golang.org/grpc"

creds := noise.NewCredentials(addr, myProtocol)
server := grpc.NewServer(
    grpc.Creds(creds),
)
```

Usually though, you wouldn't use Noise like that. Instead, you'll use Noise's protocols together with S/Kademlia protocol which will be explained further on S/Kademlia section below.

<br />

You can also implement a new protocol by fulfilling `Protocol` interface:
```go
type Protocol interface {
	Client(Info, context.Context, string, net.Conn) (net.Conn, error)
	Server(Info, net.Conn) (net.Conn, error)
}
```

<br />

Noise provides a set of protocols that can be used together to build a secure decentralized application :
1) S/Kademlia overlay networks
2) ECDH handshake 
3) AEAD AES-256 GCM

We will go through each of the protocol below (skademlia)[].

## S/Kademlia
The S/Kademlia protocol is a overlay network protocol which allows p2p applications to easily achieve reliability in:
1) routing messages to/from peers,
2) bootstrapping new peers into the network,
3) and maintaining a clear overview of the liveness of all peers a node is connected to.

Create a new node and passing the protocol:
```go
import "github.com/perlin-network/noise"
import "github.com/perlin-network/noise/skademlia"

// Create new keys with `c1` and `c2` both 1
keys, err := skademlia.NewKeys(1, 1)
if err != nil {
    panic(err)
}

client := skademlia.NewClient(addr, keys)

// Pass the S/Kademlia protocols
client.SetCredentials(noise.NewCredentials(addr, client.Protocol()))
```

For further details, customizations and functionalities of the S/Kademlia protocol, consult the [docs](protocol_skademlia.md).

## ECDH handshake 
The Elliptic Curve Diffie-Hellman (ECDH) handshake protocol can be used to generate a shared symmetric key amongst two peers.

The elliptic curve used is the Twisted-Edwards (Ed25519) curve, accompanied by the EdDSA signature scheme to verify handshake requests.

After a handshake is successful, the shared key established is saved in Noise `noise.Info` (which implements gRPC `credentials.AuthInfo`) with the key as defined in the Noise constant `handshake.SharedKey`.

If an error occurred during the handshake, the peer will be disconnected.

Using handshake protocol with S/Kademlia:
```go
import "github.com/perlin-network/noise"
import "github.com/perlin-network/noise/handshake"
import "github.com/perlin-network/skademlia"

client := skademlia.NewClient(addr, keys, skademlia.WithC1(C1), skademlia.WithC2(C2))
client.SetCredentials(noise.NewCredentials(addr, handshake.NewECDH(), client.Protocol()))
```

## AEAD AES-256 GCM
Noise provides a AEAD protocol to encrypt the data send through the gRPC connection between two peers.
It works by wrapping the `net.Conn` in a custom `net.Conn` implementation to automatically handle all the encryption/decryption.

AEAD protocol relies on a prior protocol providing an established shared key between two peers.
The shared key must be saved in the `noise.Info` with the key as defined in the Noise constant `handshake.SharedKey`

If an established shared key could not be found upon entry to the AEAD protocol, the peer will be disconnected.

The shared key would be passed through a SHA256-based HKDF, and thereafter used to encrypt/decrypt all future communications amongst a pair of peers

Using AEAD protocol with ECDH handshake and S/Kademlia protocols:
```go
import "github.com/perlin-network/noise"
import "github.com/perlin-network/noise/cipher"
import "github.com/perlin-network/noise/handshake"
import "github.com/perlin-network/skademlia"


client := skademlia.NewClient(addr, keys, skademlia.WithC1(C1), skademlia.WithC2(C2))

// Notice we pass the handshake protocol before the AEAD protocol.
client.SetCredentials(noise.NewCredentials(addr, handshake.NewECDH(), cipher.NewAEAD(), client.Protocol()))
```