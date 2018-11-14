package skademlia

import (
	"bytes"
	"crypto/rand"
	"github.com/perlin-network/noise/crypto"
	"github.com/perlin-network/noise/crypto/blake2b"
	"github.com/perlin-network/noise/crypto/ed25519"
	"github.com/perlin-network/noise/peer"
	"github.com/perlin-network/noise/protocol"
	"github.com/pkg/errors"
)

var _ protocol.IdentityAdapter = (*IdentityAdapter)(nil)

type IdentityAdapter struct {
}

func NewIdentityAdapter(*crypto.KeyPair) *IdentityAdapter {
	return nil
}

func (ia *IdentityAdapter) MyIdentity() []byte {
	return nil
}

func (ia *IdentityAdapter) Sign(input []byte) []byte {
	return nil
}

func (ia *IdentityAdapter) Verify(id, data, signature []byte) bool {
	return false
}

func (ia *IdentityAdapter) SignatureSize() int {
	return 0
}

// generateKeyPairAndNonce generates an S/Kademlia keypair and nonce with cryptopuzzle prefix matching constants c1
// and c2
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

// checkHashedBytesPrefixLen checks if the hashed bytes has prefix length of c
func checkHashedBytesPrefixLen(a []byte, c int) bool {
	b := blake2b.New()
	P := b.HashBytes(a)
	return peer.PrefixLen(P) >= c
}

// randomBytes generates a random byte slice with specified length
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
// has at least a prefix length of c
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

// checkDynamicPuzzle checks whether the nodeID and bytes x solves the S/Kademlia dynamic puzzle for c prefix length
func checkDynamicPuzzle(nodeID, x []byte, c int) bool {
	xored := peer.Xor(nodeID, x)
	return checkHashedBytesPrefixLen(xored, c)
}

// VerifyPuzzle checks whether an ID is a valid S/Kademlia node ID with cryptopuzzle constants c1 and c2
func VerifyPuzzle(id peer.ID, c1, c2 int) bool {
	// check if static puzzle and dynamic puzzle is solved
	b := blake2b.New()
	nonce := peer.GetNonce(id)
	return bytes.Equal(b.HashBytes(id.PublicKey), id.Id) && checkHashedBytesPrefixLen(id.Id, c1) && checkDynamicPuzzle(id.Id, nonce, c2)
}
