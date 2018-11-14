package skademlia

import (
	"bytes"
	"crypto/rand"
	"math/bits"

	"github.com/perlin-network/noise/crypto"
	"github.com/perlin-network/noise/crypto/blake2b"
	"github.com/perlin-network/noise/crypto/ed25519"
	"github.com/perlin-network/noise/protocol"

	"github.com/pkg/errors"
)

var _ protocol.IdentityAdapter = (*SKademliaIdentityAdapter)(nil)

type SKademliaIdentityAdapter struct {
	keypair *crypto.KeyPair
	ID      []byte
	Nonce   []byte
}

func NewSKademliaIdentityAdapter(c1, c2 int) *SKademliaIdentityAdapter {
	kp, nonce := generateKeyPairAndNonce(c1, c2)
	return &SKademliaIdentityAdapter{
		keypair: kp,
		Nonce:   nonce,
	}
}

func (a *SKademliaIdentityAdapter) MyIdentity() []byte {
	return a.keypair.PublicKey
}

func (a *SKademliaIdentityAdapter) Sign(input []byte) []byte {
	ret, err := a.keypair.Sign(ed25519.New(), blake2b.New(), input)
	if err != nil {
		panic(err)
	}
	return ret
}

func (a *SKademliaIdentityAdapter) Verify(id, data, signature []byte) bool {
	return crypto.Verify(ed25519.New(), blake2b.New(), id, data, signature)
}

func (a *SKademliaIdentityAdapter) SignatureSize() int {
	return ed25519.SignatureSize
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
	return prefixLen(P) >= c
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
	xored := xor(nodeID, x)
	return checkHashedBytesPrefixLen(xored, c)
}

// VerifyPuzzle checks whether an ID is a valid S/Kademlia node ID with cryptopuzzle constants c1 and c2
func VerifyPuzzle(id *SKademliaIdentityAdapter, c1, c2 int) bool {
	// check if static puzzle and dynamic puzzle is solved
	b := blake2b.New()
	return bytes.Equal(b.HashBytes(id.keypair.PublicKey), id.MyIdentity()) && checkHashedBytesPrefixLen(id.MyIdentity(), c1) && checkDynamicPuzzle(id.MyIdentity(), id.Nonce, c2)
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
