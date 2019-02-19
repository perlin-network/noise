# Authenticated Encryption w/ Authenticated Data

**noise** provides a simple wrapper around Go's de-facto implementation of Authenticated Encryption w/ Authenticated Data (AEAD) as a `protocol.Block`.

`aead` relies on a prior block providing an established shared key between two peers which is set within a given peers metadata through a call to `protocol.SetSharedKey([]byte)`.

If an established shared key could not be found upon entry to the `aead` block, the peer will be disconnected.

The shared key from prior blocks would be passed through a SHA256-based [HKDF](https://en.wikipedia.org/wiki/HKDF), and thereafter used to encrypt/decrypt all future communications amongst a pair of peers.

Any other hash function may optionally be set to perform key derivation like so:

```go
import "github.com/perlin-network/noise"
import "github.com/perlin-network/noise/cipher/aead"
import "crypto/sha256"
import "crypto/sha512"

block := aead.New().WithHash(sha256.New)

// You may optionally opt out from using the builder pattern like so.
block.WithHash(sha512.New)
```

By default, AES-256 GCM (Galois Counter Mode) is used for performing AEAD.

ChaCha20-Poly1305 and XChaCha20-Poly1305 are also supported, but note that the shared key a peer starts off with must be 256 bits.

You may set the cipher to use w/ `aead` like so:

```go
import "github.com/perlin-network/noise"
import "github.com/perlin-network/noise/cipher/aead"
import "crypto/cipher"

block := aead.New()

// Set either one of the 3 cipher suites below to use with the `aead` block.
aes256_GCM := aead.AES256_GCM
chacha20_poly1305 := aead.ChaCha20_Poly1305
xchacha20_poly1305 := aead.XChaCha20_Poly1305

// Or in general, any function of the following format.
customCipher := func(sharedKey []byte) (cipher.AEAD, error) {
	// ... setup your cipher here given a HMAC-derived shared key
}

block.WithSuite(aes256_GCM)
```

## Protocol

An `ACK` message is sent between two peers, and received using the atomic locking operation `peer.LockOnReceive(opcodeACK)` to establish a synchronization point where from a specific time onwards, all messages will be encrypted/decrypted using AEAD.

Timeouts for expecting to receive the `ACK` message can be easily set like so:

```go
import "github.com/perlin-network/noise"
import "github.com/perlin-network/noise/cipher/aead"
import "time"

block := aead.New().WithACKTimeout(3 * time.Second)

// You may optionally opt out from using the builder pattern like so.
block.WithACKTimeout(5 * time.Second)
```

Given that Noise guarantees linearized message delivery which is dependant on the transport layer used, we associate incremental nonces rather than random nonces for establishing authenticated encryption.

Incremental nonces simply imply that on every message received, we increment our nonce and attempt to decrypt the received message with the new nonce. The same applies for whenever we sent a message.

The code associated with handling incremental nonces is like so:

```go
import "crypto/cipher"
import "atomic"
import "binary"

// ...
// Perform ACK handling and locking to establish synchronization point here.
// ...

var ourNonce uint64
var theirNonce uint64
var suite cipher.AEAD

peer.BeforeMessageReceived(func(node *noise.Node, peer *noise.Peer, msg []byte) (buf []byte, err error) {
    theirNonceBuf := make([]byte, suite.NonceSize())
    binary.LittleEndian.PutUint64(theirNonceBuf, atomic.AddUint64(&theirNonce, 1))

    return suite.Open(msg[:0], theirNonceBuf, msg, nil)
})

peer.BeforeMessageSent(func(node *noise.Node, peer *noise.Peer, msg []byte) (buf []byte, err error) {
    ourNonceBuf := make([]byte, suite.NonceSize())
    binary.LittleEndian.PutUint64(ourNonceBuf, atomic.AddUint64(&ourNonce, 1))

    return suite.Seal(msg[:0], ourNonceBuf, msg, nil), nil
})
```

A helpful function that you may choose to use throughout your application is `aead.WaitUntilAuthenticated(*noise.Peer)`
which blocks the current goroutine until a peer we specify has successfully setup AEAD encryption/decryption for all incoming/outgoing messages.

