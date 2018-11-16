package skademlia

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"math/bits"

	"github.com/perlin-network/noise/crypto"
	"github.com/perlin-network/noise/crypto/blake2b"
	"github.com/perlin-network/noise/crypto/ed25519"
	"github.com/perlin-network/noise/protocol"

	"github.com/pkg/errors"
)

const (
	// DefaultC1 is the prefix-matching length for the static cryptopuzzle.
	DefaultC1 = 16
	// DefaultC2 is the prefix-matching length for the dynamic cryptopuzzle.
	DefaultC2 = 16
)

var _ protocol.IdentityAdapter = (*IdentityAdapter)(nil)

// IdentityAdapter implements the identity interface for S/Kademlia node IDs.
type IdentityAdapter struct {
	keypair *crypto.KeyPair
	nodeID  []byte
	Nonce   []byte
	signer  crypto.SignaturePolicy
	hasher  crypto.HashPolicy
}

// NewIdentityAdapter creates a new SKademlia IdentityAdapter with the given cryptopuzzle constants.
func NewIdentityAdapter(c1, c2 int) *IdentityAdapter {
	kp, nonce := generateKeyPairAndNonce(c1, c2)
	b := blake2b.New()
	return &IdentityAdapter{
		keypair: kp,
		nodeID:  b.HashBytes(kp.PublicKey),
		Nonce:   nonce,
		signer:  ed25519.New(),
		hasher:  b,
	}
}

// NewIdentityFromKeypair creates a new SKademlia IdentityAdapter with the given cryptopuzzle
// constants from an existing keypair.
func NewIdentityFromKeypair(kp *crypto.KeyPair, c1, c2 int) (*IdentityAdapter, error) {
	b := blake2b.New()
	id := b.HashBytes(kp.PublicKey)
	if !checkHashedBytesPrefixLen(id, c1) {
		return nil, errors.Errorf("skademlia: provided keypair does not generate a valid node ID for c1: %d", c1)
	}
	return &IdentityAdapter{
		keypair: kp,
		nodeID:  id,
		Nonce:   getNonce(id, c2),
		signer:  ed25519.New(),
		hasher:  b,
	}, nil
}

// MyIdentity returns the S/Kademlia public key ID.
func (a IdentityAdapter) MyIdentity() []byte {
	return a.keypair.PublicKey
}

// MyIdentityHex returns the S/Kademlia hex-encoded node's public key.
func (a IdentityAdapter) MyIdentityHex() string {
	return hex.EncodeToString(a.keypair.PublicKey)
}

// id returns the S/Kademlia node ID. The node ID is used for routing.
func (a IdentityAdapter) id() []byte {
	return a.nodeID
}

// idHex returns the S/Kademlia hex-encoded node ID.
func (a IdentityAdapter) idHex() string {
	return hex.EncodeToString(a.nodeID)
}

// Sign signs the input bytes with the identity's private key.
func (a IdentityAdapter) Sign(input []byte) []byte {
	ret, err := a.keypair.Sign(a.signer, a.hasher, input)
	if err != nil {
		panic(err)
	}
	return ret
}

// Verify checks whether the signature matches the signed data
func (a IdentityAdapter) Verify(publicKey, data, signature []byte) bool {
	return crypto.Verify(a.signer, a.hasher, publicKey, data, signature)
}

// SignatureSize specifies the byte length for signatures generated with the keypair
func (a IdentityAdapter) SignatureSize() int {
	return ed25519.SignatureSize
}

// GetKeyPair returns the key pair used to create the idenity
func (a IdentityAdapter) GetKeyPair() *crypto.KeyPair {
	return a.keypair
}

// generateKeyPairAndNonce generates an S/Kademlia keypair and nonce with cryptopuzzle
// prefix matching constants c1 and c2.
func generateKeyPairAndNonce(c1, c2 int) (*crypto.KeyPair, []byte) {
	b := blake2b.New()
	for {
		kp := ed25519.RandomKeyPair()
		nodeID := b.HashBytes(kp.PublicKey)
		if checkHashedBytesPrefixLen(nodeID, c1) {
			return kp, getNonce(nodeID, c2)
		}
	}
}

// checkHashedBytesPrefixLen checks if the hashed bytes has prefix length of c.
func checkHashedBytesPrefixLen(a []byte, c int) bool {
	b := blake2b.New()
	P := b.HashBytes(a)
	return prefixLen(P) >= c
}

// randomBytes generates a random byte slice with specified length.
func randomBytes(len int) ([]byte, error) {
	randBytes := make([]byte, len)
	n, err := rand.Read(randBytes)
	if err != nil {
		return nil, err
	}
	if n != len {
		return nil, errors.Errorf("failed to generate %d bytes", len)
	}
	return randBytes, nil
}

// getNonce returns random bytes X which satisfies that the hash of the nodeID xored with X
// has at least a prefix length of c.
func getNonce(nodeID []byte, c int) []byte {
	len := len(nodeID)
	for {
		x, err := randomBytes(len)
		if err != nil {
			continue
		}
		if checkDynamicPuzzle(nodeID, x, c) {
			return x
		}
	}
}

// checkDynamicPuzzle checks whether the nodeID and bytes x solves the S/Kademlia dynamic puzzle for c prefix length.
func checkDynamicPuzzle(nodeID, x []byte, c int) bool {
	xored := xor(nodeID, x)
	return checkHashedBytesPrefixLen(xored, c)
}

// VerifyPuzzle checks whether an ID is a valid S/Kademlia node ID with cryptopuzzle constants c1 and c2.
func VerifyPuzzle(id *IdentityAdapter, c1, c2 int) bool {
	// check if static puzzle and dynamic puzzle is solved
	b := blake2b.New()
	return bytes.Equal(b.HashBytes(id.keypair.PublicKey), id.id()) &&
		checkHashedBytesPrefixLen(id.id(), c1) &&
		checkDynamicPuzzle(id.id(), id.Nonce, c2)
}

// xor performs an xor operation on two byte slices.
func xor(a, b []byte) []byte {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}

	dst := make([]byte, n)
	for i := 0; i < n; i++ {
		dst[i] = a[i] ^ b[i]
	}
	return dst
}

// prefixLen returns the number of prefixed zeroes in a byte slice.
func prefixLen(bytes []byte) int {
	for i, b := range bytes {
		if b != 0 {
			return i*8 + bits.LeadingZeros8(uint8(b))
		}
	}
	return len(bytes)*8 - 1
}
