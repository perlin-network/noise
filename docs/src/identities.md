# Identities

Apart from your nodes external IP and port, p2p applications typically instill that each node should have some sort of publicly verifiable identity.

The identity could comprise of all sorts of things: from a string of random bytes to a cryptographically verifiable identity.

Most likely, you would want a cryptographic ID for your node.

As of right now, there exists a few identity schemes which Noise provides built-in support for that you may use on the get-go for your p2p application:

1. Ed25519 identities w/ EdDSA signatures
2. S/Kademlia-compatible Ed25519 identities w/ EdDSA signatures

You may additionally create/implement your own identity schemes that any node/peer may support in Noise by simply having your scheme implement the following interface:

```go
package identity

import "fmt"

type Keypair interface {
	fmt.Stringer

	ID() []byte
	PublicKey() []byte
	PrivateKey() []byte

	Sign(buf []byte) ([]byte, error)
	Verify(publicKeyBuf []byte, buf []byte, signature []byte) error
}
```

Should the identity scheme you wish to implement not have any associated signature scheme to it, or any sort of `PrivateKey()` or `PublicKey()` or `ID()` associated to it, you may simply stub out those functions and ignore their implementation.

After picking/implementing an identity scheme of your choice for your p2p application, it is simple to have your node adopt it as follows:

```go
import (
	"github.com/perlin-network/noise"
	"github.com/perlin-network/noise/identity/ed25519"
	"github.com/perlin-network/noise/skademlia"
)

params := noise.DefaultParams()

// Generate a random Ed25519 keypair w/ EdDSA signature scheme support
// for your node.
params.Keys = ed25519.RandomKeys()

// Generate a random S/Kademlia-compatible Ed25519 keypair w/ EdDSA 
// signature scheme support for your node.
params.Keys = skademlia.RandomKeys()

// Load an existing Ed25519 keypair for your node.
params.Keys = ed25519.LoadKeys([]byte{...})

// Load an existing S/Kademlia-compatible keypair for your node.
// Wondering what C1 and C2 are for? Check out S/Kademlia's documentation!
params.Keys = skademlia.LoadKeys([]byte{...}, skademlia.DefaultC1, skademlia.DefaultC2)
```

## Signing/Verifying Messages

Given a node instance instantiated with an identity scheme paired with a signature scheme,
you may sign/verify raw arrays of bytes to see whether or not a signature was generated
by a specified public key like so:

```go
var node *noise.Node

message := "We're going to sign this message with our node!"

signature, err := node.Keys.Sign(message)
if err != nil {
	panic("failed to sign the message")
}

fmt.Println("Signature:", signature)

// Now let's verify that the signature is under our own nodes identity!

fmt.Println("Is the signature valid?", 
	node.Keys.Verify(node.Keys.PublicKey(), message, signature))
```