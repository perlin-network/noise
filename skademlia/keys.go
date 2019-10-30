package skademlia

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"github.com/Yayg/noise/identity"
	"github.com/Yayg/noise/internal/edwards25519"
	"github.com/pkg/errors"
	"golang.org/x/crypto/blake2b"
)

const (
	// DefaultC1 is the prefix-matching length for the static crypto puzzle.
	DefaultC1 = 8

	// DefaultC2 is the prefix-matching length for the dynamic crypto puzzle.
	DefaultC2 = 8

	// MaxPuzzleIterations is an internal limit for performing cryptographic puzzles for S/Kademlia.
	// This is set to stop the puzzle after about 1sec of failed continuous checks.
	MaxPuzzleIterations = 1000000
)

var (
	_ identity.Keypair = (*Keypair)(nil)
)

type Keypair struct {
	privateKey edwards25519.PrivateKey
	publicKey  edwards25519.PublicKey

	Nonce  []byte
	C1, C2 int
}

func (p *Keypair) ID() []byte {
	id := blake2b.Sum256(p.publicKey)
	return id[:]
}

// LoadKeys loads a S/Kademlia given an Ed25519 private key, and validates it through both a static and dynamic
// crypto puzzle parameterized by constants C1 and C2 respectively.
func LoadKeys(privateKeyBuf []byte, c1, c2 int) (*Keypair, error) {
	if len(privateKeyBuf) != edwards25519.PrivateKeySize {
		panic(errors.Errorf("skademlia: private key is not %d bytes", edwards25519.PrivateKeySize))
	}

	privateKey := edwards25519.PrivateKey(privateKeyBuf)
	publicKey := privateKey.Public()

	id := blake2b.Sum256(publicKey.(edwards25519.PublicKey))

	if !checkHashedBytesPrefixLen(id[:], c1) {
		return nil, errors.Errorf("skademlia: private key provided does not have a prefix of C1: %x", id)
	}

	nonce := generateNonce(id[:], c2)

	if nonce == nil {
		return nil, errors.New("skademlia: keypair has an invalid nonce")
	}

	return &Keypair{
		privateKey: privateKey,
		publicKey:  privateKey.Public().(edwards25519.PublicKey),

		Nonce: nonce,

		C1: c1,
		C2: c2,
	}, nil
}

// RandomKeys randomly generates a set of cryptographic keys by solving both a static and dynamic
// crypto puzzle parameterized by constants C1 = 8, and C2 = 8 respectively.
func RandomKeys() *Keypair {
	return NewKeys(DefaultC1, DefaultC2)
}

// NewKeys randomly generates a set of cryptographic keys by solving both a static and dynamic
// crypto puzzle parameterized by constants C1 and C2 respectively.
func NewKeys(c1, c2 int) *Keypair {
	var publicKey edwards25519.PublicKey
	var privateKey edwards25519.PrivateKey
	var id [32]byte
	var err error

	for i := 0; i < MaxPuzzleIterations; i++ {
		publicKey, privateKey, err = edwards25519.GenerateKey(nil)
		if err != nil {
			panic(errors.Wrap(err, "ed25519: failed to generate random keypair"))
		}

		id = blake2b.Sum256(publicKey)

		if checkHashedBytesPrefixLen(id[:], c1) {
			break
		}
	}

	keys, err := LoadKeys(privateKey, c1, c2)
	if err != nil {
		panic(err)
	}

	return keys
}

func (p *Keypair) PublicKey() []byte {
	return p.publicKey
}

func (p *Keypair) String() string {
	return fmt.Sprintf("S/Kademlia(public: %s, private: %s)", hex.EncodeToString(p.PublicKey()), hex.EncodeToString(p.PrivateKey()))
}

func (p *Keypair) PrivateKey() []byte {
	return p.privateKey
}

// checkHashedBytesPrefixLen checks if the hashed bytes has prefix length of c.
func checkHashedBytesPrefixLen(a []byte, c int) bool {
	hash := blake2b.Sum256(a)
	return prefixLen(hash[:]) >= c
}

// randomBytes generates a random byte slice with a specified length.
func randomBytes(len int) ([]byte, error) {
	buf := make([]byte, len)
	n, err := rand.Read(buf)

	if err != nil {
		return nil, err
	}

	if n != len {
		return nil, errors.Errorf("failed to generate %d bytes", len)
	}

	return buf, nil
}

// generateNonce attempts to randomly generate a suitable nonce which satisfies the condition
// that xor(hash(id), random_bytes) has at least a prefix length of c.
func generateNonce(id []byte, c int) []byte {
	for i := 0; i < MaxPuzzleIterations; i++ {
		nonce, err := randomBytes(len(id))

		if err != nil {
			continue
		}

		if checkDynamicPuzzle(id, nonce, c) {
			return nonce
		}
	}

	return nil
}

// checkDynamicPuzzle checks whether the id and bytes fulfills the S/Kademlia dynamic puzzle for c prefix length.
func checkDynamicPuzzle(id, buf []byte, c int) bool {
	return checkHashedBytesPrefixLen(xor(id, buf), c)
}

// VerifyPuzzle checks whether or not an id is a valid S/Kademlia id that suffices
// both S/Kademlia's static and dynamic puzzle given constants C1 and C2.
func VerifyPuzzle(publicKey, id, nonce []byte, c1, c2 int) bool {
	hash := blake2b.Sum256(publicKey)

	return bytes.Equal(hash[:], id) &&
		checkHashedBytesPrefixLen(id, c1) &&
		checkDynamicPuzzle(id, nonce, c2)
}
