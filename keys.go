package noise

import (
	"encoding/hex"
	"encoding/json"
	"github.com/oasislabs/ed25519"
	"io"
)

const (
	// Size in bytes of a peers public key.
	PublicKeySize = ed25519.PublicKeySize

	// Size in bytes of a peers private key.
	PrivateKeySize = ed25519.PrivateKeySize
)

type (
	// Default peer public key type.
	PublicKey [PublicKeySize]byte

	// Default peer private key type.
	PrivateKey [PrivateKeySize]byte
)

var (
	// Zero-value for a peer public key.
	ZeroPublicKey PublicKey

	// Zero-value for a peer private key.
	ZeroPrivateKey PrivateKey
)

// GenerateKeys randomly generates a new pair of cryptographic keys. Nil may be passed to rand in order to use
// crypto/rand by default. It throws an error if rand is invalid.
func GenerateKeys(rand io.Reader) (publicKey PublicKey, privateKey PrivateKey, err error) {
	pub, priv, err := ed25519.GenerateKey(rand)
	if err != nil {
		return publicKey, privateKey, err
	}

	copy(publicKey[:], pub)
	copy(privateKey[:], priv)

	return publicKey, privateKey, nil
}

// Verify returns true if the cryptographic signature of data is representative of this public key.
func (k PublicKey) Verify(data, signature []byte) bool {
	return ed25519.Verify(k[:], data, signature)
}

// String returns the hexadecimal representation of this public key.
func (k PublicKey) String() string {
	return hex.EncodeToString(k[:])
}

// MarshalJSON returns the hexadecimal representation of this public key in JSON. It should never throw an error.
func (k PublicKey) MarshalJSON() ([]byte, error) {
	return json.Marshal(k.String())
}

// Sign uses this private key to sign data and return its cryptographic signature as a slice of bytes.
func (k PrivateKey) Sign(data []byte) []byte {
	return ed25519.Sign(k[:], data)
}

// String returns the hexadecimal representation of this private key.
func (k PrivateKey) String() string {
	return hex.EncodeToString(k[:])
}

// MarshalJSON returns the hexadecimal representation of this private key in JSON. It should never throw an error.
func (k PrivateKey) MarshalJSON() ([]byte, error) {
	return json.Marshal(hex.EncodeToString(k[:]))
}
