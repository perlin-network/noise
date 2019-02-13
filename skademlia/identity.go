package skademlia

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"github.com/perlin-network/noise/identity"
	"github.com/perlin-network/noise/identity/ed25519"
	"github.com/pkg/errors"
	"golang.org/x/crypto/blake2b"
)

const (
	// DefaultC1 is the prefix-matching length for the static cryptopuzzle.
	DefaultC1 = 16
	// DefaultC2 is the prefix-matching length for the dynamic cryptopuzzle.
	DefaultC2 = 16

	maxPuzzleIterations = 100000000
)

var (
	_ identity.Manager = (*IdentityAdapter)(nil)
)

// IdentityAdapter implements the identity interface for S/Kademlia node IDs.
type IdentityAdapter struct {
	keypair identity.Manager
	nodeID  []byte
	Nonce   []byte
	c1, c2  int
}

// NewIdentityRandomDefault creates a new SKademlia IdentityAdapter with sound default values.
func NewIdentityRandomDefault() *IdentityAdapter {
	return NewIdentityRandom(DefaultC1, DefaultC2)
}

// NewIdentityRandom creates a new SKademlia IdentityAdapter with the given cryptopuzzle constants.
func NewIdentityRandom(c1, c2 int) *IdentityAdapter {
	kp := generateKeyPair(c1, c2)
	if kp == nil {
		return nil
	}
	if a, err := NewIdentityFromKeypair(kp, c1, c2); err != nil {
		return a
	}
	return nil
}

// NewIdentityDefaultFromKeypair creates a new SKademlia IdentityAdapter with the given cryptopuzzle
// constants from an existing keypair with sound default values.
func NewIdentityDefaultFromKeypair(kp identity.Manager) (*IdentityAdapter, error) {
	return NewIdentityFromKeypair(kp, DefaultC1, DefaultC2)
}

// NewIdentityFromKeypair creates a new SKademlia IdentityAdapter with the given cryptopuzzle
// constants from an existing keypair.
func NewIdentityFromKeypair(kp identity.Manager, c1, c2 int) (*IdentityAdapter, error) {
	hash := blake2b.Sum256(kp.PublicID())
	id := hash[:]
	if !checkHashedBytesPrefixLen(id, c1) {
		return nil, errors.Errorf("skademlia: provided keypair does not generate a valid node ID for c1: %d", c1)
	}
	return &IdentityAdapter{
		keypair: kp,
		nodeID:  id,
		Nonce:   getNonce(id, c2),
		c1:      c1,
		c2:      c2,
	}, nil
}

// PublicID returns the S/Kademlia public key ID.
func (a *IdentityAdapter) PublicID() []byte {
	return a.keypair.PublicID()
}

// PublicIDHex returns the S/Kademlia hex-encoded node's public key.
func (a *IdentityAdapter) PublicIDHex() string {
	return hex.EncodeToString(a.PublicID())
}

// PrivateKey returns the S/Kademlia private key for this ID.
func (a *IdentityAdapter) PrivateKey() []byte {
	return a.keypair.PrivateKey()
}

// NodeID returns the S/Kademlia node ID. The node ID is used for routing.
func (a *IdentityAdapter) NodeID() []byte {
	return a.nodeID
}

// NodeIDHex returns the S/Kademlia hex-encoded node ID.
func (a *IdentityAdapter) NodeIDHex() string {
	return hex.EncodeToString(a.nodeID)
}

// Sign signs the input bytes with the identity's private key.
func (a *IdentityAdapter) Sign(input []byte) ([]byte, error) {
	return a.keypair.Sign(input)
}

// Verify checks whether the signature matches the signed data
func (a *IdentityAdapter) Verify(publicKey, data, signature []byte) error {
	return a.keypair.Verify(publicKey, data, signature)
}

// SignatureSize specifies the byte length for signatures generated with the keypair
func (a *IdentityAdapter) SignatureSize() int {
	return ed25519.SignatureSize
}

// GetKeyPair returns the key pair used to create the idenity
func (a *IdentityAdapter) GetKeyPair() identity.Manager {
	return a.keypair
}

func (a *IdentityAdapter) String() string {
	return fmt.Sprintf("skademlia-id{keypair: %s, nodeID: %s, Nonce:%s, c1: %d, c2: %d}",
		a.keypair.String(),
		hex.EncodeToString(a.nodeID),
		hex.EncodeToString(a.Nonce),
		a.c1,
		a.c2,
	)
}

// generateKeyPair generates an S/Kademlia keypair with cryptopuzzle
// prefix matching constants c1 and c2.
func generateKeyPair(c1, c2 int) identity.Manager {
	for i := 0; i < maxPuzzleIterations; i++ {
		kp := ed25519.Random()
		hash := blake2b.Sum256(kp.PublicID())
		nodeID := hash[:]
		if checkHashedBytesPrefixLen(nodeID, c1) {
			return kp
		}
	}
	return nil
}

// checkHashedBytesPrefixLen checks if the hashed bytes has prefix length of c.
func checkHashedBytesPrefixLen(a []byte, c int) bool {
	hash := blake2b.Sum256(a)
	p := hash[:]
	return prefixLen(p) >= c
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
	for i := 0; i < maxPuzzleIterations; i++ {
		x, err := randomBytes(len)
		if err != nil {
			continue
		}
		if checkDynamicPuzzle(nodeID, x, c) {
			return x
		}
	}
	return nil
}

// checkDynamicPuzzle checks whether the nodeID and bytes x solves the S/Kademlia dynamic puzzle for c prefix length.
func checkDynamicPuzzle(nodeID, x []byte, c int) bool {
	xored := xor(nodeID, x)
	return checkHashedBytesPrefixLen(xored, c)
}

// VerifyPuzzle checks whether an ID is a valid S/Kademlia node ID with cryptopuzzle constants c1 and c2.
func VerifyPuzzle(publicKey, nodeID, nonce []byte, c1, c2 int) bool {
	// check if static puzzle and dynamic puzzle is solved
	hash := blake2b.Sum256(publicKey)
	return bytes.Equal(hash[:], nodeID) &&
		checkHashedBytesPrefixLen(nodeID, c1) &&
		checkDynamicPuzzle(nodeID, nonce, c2)
}
