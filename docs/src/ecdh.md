# Elliptic Curve Diffie-Hellman Handshake

**noise** provides a simple implementation of an Elliptic Curve Diffie-Hellman (ECDH) handshake that may be used to generate a shared symmetric key amongst two peers as a `protocol.Block`.

The elliptic curve used is the Twisted-Edwards (Ed25519) curve, accompanied by the EdDSA signature scheme to verify handshake requests.

After a handshake is successful, the shared key established is set within the peers metadata via `protocol.SetSharedKey([]byte)`.

A timeout may be established in terms of how long a peer should wait for a handshake request from a newly established peer like so:

```go
import "github.com/perlin-network/handshake/ecdh"
import "time"

block := ecdh.New().TimeoutAfter(3 * time.Second)

// Optionally, feel free to not use the builder pattern and set the timeout like so.
block.TimeoutAfter(5 * time.Second)

// Additionally, you can change up the handshake message contents to be sent/verified
// like so. The default handshake messages contents is `.noise_handshake`.
block.WithHandshakeMessage("some handshake message here!")
```

A debug message would be printed should the handshake be successful. If at any stage throughout the protocol a peer fails to complete a requested action, they would be immediately disconnected.

## Protocol

Let's define two peers \\( A \\) and \\( B \\).

When peer \\( A \\) manages to connect to peer \\(B \\), or vice versa, both generate an ephemeral Ed25519 keypair comprised of a private key \\(x \\), and a public key \\( g^x \\) where \\( g \\) is a Twisted-Edwards curve generator point.

Both peers would then send a `Handshake` message to one another whose contents comprise of:
 
1. their ephemeral public keys,
2. and an EdDSA signature of the message `.noise_handshake` (customizable).

If either peer \\( A \\) or peer \\( B \\) does not receive the `Handshake` message after a specified period of time, either peer \\(A \\) or peer \\(B \\) would disconnect from one another.

Should either one receive the `Handshake` message, said peer would verify the EdDSA signature within the message. If the signature is invalid, they would disconnect away from the peer.

Afterwards, a shared key then is computed. Assuming that peer A's ephemeral private key is \\(x_a\\) and peer B's ephemeral private key \\(x_b\\), peer A would compute their shared key as \\(x_a * g^{x_b}\\), and peer B would compute their shared key as \\(x_b * g^{x_a} \\).

Both shared keys generated from either sides in this case would be equivalent as elliptic curve point addition is commutative, and would not unrecoverable by an attacker intercepting communication channels as attempting to derive the shared key is equivalent to solving the [discrete logarithm problem](https://en.wikipedia.org/wiki/Discrete_logarithm).
 
 