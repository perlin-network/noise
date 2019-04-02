package skademlia

import (
	"github.com/perlin-network/noise/edwards25519"
	"github.com/pkg/errors"
	"golang.org/x/crypto/blake2b"
	"math/rand"
)

// generateKeys attempts to randomly generate a suitable Ed25519 keypair which satisfies the
// condition that blake2b(blake2b(publicKey)) has at least c1 prefixed zero bits.
func generateKeys(address string, c1 int) (publicKey edwards25519.PublicKey, privateKey edwards25519.PrivateKey, id [blake2b.Size256]byte, checksum [blake2b.Size256]byte, err error) {
	for {
		publicKey, privateKey, err = edwards25519.GenerateKey(nil)

		if err != nil {
			err = errors.Wrap(err, "failed to generate random keys")
			return
		}

		id = blake2b.Sum256(publicKey[:])
		checksum = blake2b.Sum256(append(id[:], address...))

		if staticPuzzle := prefixLen(checksum[:]); staticPuzzle >= c1 {
			return
		}
	}
}

// generateNonce attempts to randomly generate a suitable nonce which satisfies the condition
// that xor(checksum, nonce) has at least c2 prefixed zero bits.
func generateNonce(checksum [blake2b.Size256]byte, c2 int) ([blake2b.Size256]byte, error) {
	var nonce [blake2b.Size256]byte

	for {
		n, err := rand.Read(nonce[:])

		if err != nil {
			return nonce, err
		}

		if n != blake2b.Size256 {
			return nonce, errors.Errorf("failed to generate %d bytes", blake2b.Size256)
		}

		if dynamicPuzzle := prefixLen(xor(checksum[:], nonce[:])); dynamicPuzzle >= c2 {
			return nonce, nil
		}
	}
}

// verifyPuzzle checks whether or not given the checksum of an id and a corresponding nonce, that
// they suffice both S/Kademlia's static and dynamic puzzle given protocol parameters c1 and c2.
func verifyPuzzle(checksum, nonce [blake2b.Size256]byte, c1, c2 int) error {
	if staticPuzzle := prefixLen(checksum[:]); staticPuzzle < c1 {
		return errors.Errorf("failed to pass static puzzle as prefix length of checksum is %d, yet c1 is %d", staticPuzzle, c1)
	}

	if dynamicPuzzle := prefixLen(xor(checksum[:], nonce[:])); dynamicPuzzle < c2 {
		return errors.Errorf("failed to pass dynamic puzzle as prefix length of xor(checksum, nonce) is %d, yet c2 is %d", dynamicPuzzle, c2)
	}

	return nil
}
