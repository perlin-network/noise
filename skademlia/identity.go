package skademlia

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"github.com/perlin-network/noise/crypto"
	"github.com/perlin-network/noise/crypto/blake2b"
	"github.com/perlin-network/noise/crypto/ed25519"
	"github.com/perlin-network/noise/identity"
	"github.com/pkg/errors"
)

const (
	// DefaultC1 is the prefix-matching length for the static cryptopuzzle.
	DefaultC1 = 16
	// DefaultC2 is the prefix-matching length for the dynamic cryptopuzzle.
	DefaultC2 = 16
	// maxPuzzleIterations is an internal limit for performing crytographic puzzles for skademlia
	maxPuzzleIterations = 10000000000
)

var (
	_ identity.Manager = (*IdentityManager)(nil)
)

// IdentityManager implements the identity interface for S/Kademlia node IDs.
type IdentityManager struct {
	keypair *crypto.KeyPair
	nodeID  []byte
	nonce   []byte
	c1, c2  int
	signer  crypto.SignaturePolicy
	hasher  crypto.HashPolicy
}

// NewIdentityRandomDefault creates a new SKademlia IdentityManager with sound default values.
func NewIdentityRandomDefault() (*IdentityManager, error) {
	return NewIdentityRandom(DefaultC1, DefaultC2)
}

// NewIdentityRandom creates a new SKademlia IdentityManager with the given cryptopuzzle constants.
func NewIdentityRandom(c1, c2 int) (*IdentityManager, error) {
	kp := generateKeyPair(c1, c2)
	if kp == nil {
		return nil, errors.New("skademlia: unable to generate a random valid node ID ")
	}
	return NewIdentityFromKeypair(kp, c1, c2)
}

// NewIdentityDefaultFromKeypair creates a new SKademlia IdentityManager with the given cryptopuzzle
// constants from an existing keypair with sound default values.
func NewIdentityDefaultFromKeypair(kp *crypto.KeyPair) (*IdentityManager, error) {
	return NewIdentityFromKeypair(kp, DefaultC1, DefaultC2)
}

// NewIdentityFromKeypair creates a new SKademlia IdentityManager with the given cryptopuzzle
// constants from an existing keypair.
func NewIdentityFromKeypair(kp *crypto.KeyPair, c1, c2 int) (*IdentityManager, error) {
	b := blake2b.New()
	nodeID := b.HashBytes(kp.PublicKey)
	if !checkHashedBytesPrefixLen(nodeID, c1) {
		return nil, errors.Errorf("skademlia: provided keypair does not generate a valid node ID for c1: %d", c1)
	}
	nonce := getNonce(nodeID, c2)
	if nonce == nil {
		return nil, errors.New("skademlia: keypair has an invalid nonce")
	}
	return &IdentityManager{
		keypair: kp,
		nodeID:  nodeID,
		nonce:   nonce,
		c1:      c1,
		c2:      c2,
		signer:  ed25519.New(),
		hasher:  b,
	}, nil
}

// PublicID returns the S/Kademlia public key ID.
func (a *IdentityManager) PublicID() []byte {
	return a.keypair.PublicKey
}

// PrivateKey returns the S/Kademlia private key for this ID.
func (a *IdentityManager) PrivateKey() []byte {
	return a.keypair.PrivateKey
}

// Sign signs the input bytes with the identity's private key.
func (a *IdentityManager) Sign(input []byte) ([]byte, error) {
	return a.keypair.Sign(a.signer, a.hasher, input)
}

// Verify checks whether the signature matches the signed data
func (a *IdentityManager) Verify(publicKeyBuf, data, signature []byte) error {
	if crypto.Verify(a.signer, a.hasher, publicKeyBuf, data, signature) {
		return nil
	}
	return errors.New("unable to verify signature")
}

func (a *IdentityManager) String() string {
	return fmt.Sprintf("skademlia(public: %s, private: %s)", hex.EncodeToString(a.PublicID()), hex.EncodeToString(a.PrivateKey()))
}

// generateKeyPair generates an S/Kademlia keypair with cryptopuzzle
// prefix matching constants c1 and c2.
func generateKeyPair(c1, c2 int) *crypto.KeyPair {
	b := blake2b.New()
	for i := 0; i < maxPuzzleIterations; i++ {
		kp := ed25519.RandomKeyPair()
		nodeID := b.HashBytes(kp.PublicKey)
		if checkHashedBytesPrefixLen(nodeID, c1) {
			return kp
		}
	}
	return nil
}

// checkHashedBytesPrefixLen checks if the hashed bytes has prefix length of c.
func checkHashedBytesPrefixLen(a []byte, c int) bool {
	p := blake2b.New().HashBytes(a)
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
	hash := blake2b.New().HashBytes(publicKey)
	return bytes.Equal(hash, nodeID) &&
		checkHashedBytesPrefixLen(nodeID, c1) &&
		checkDynamicPuzzle(nodeID, nonce, c2)
}
