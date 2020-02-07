package noise

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/oasislabs/ed25519"
	"io"
	"reflect"
	"unsafe"
)

const (
	// SizePublicKey is the size in bytes of a nodes/peers public key.
	SizePublicKey = ed25519.PublicKeySize

	// SizePrivateKey is the size in bytes of a nodes/peers private key.
	SizePrivateKey = ed25519.PrivateKeySize

	// SizeSignature is the size in bytes of a cryptographic signature.
	SizeSignature = ed25519.SignatureSize
)

type (
	// PublicKey is the default node/peer public key type.
	PublicKey [SizePublicKey]byte

	// PrivateKey is the default node/peer private key type.
	PrivateKey [SizePrivateKey]byte

	// Signature is the default node/peer cryptographic signature type.
	Signature [SizeSignature]byte
)

var (
	// ZeroPublicKey is the zero-value for a node/peer public key.
	ZeroPublicKey PublicKey

	// ZeroPrivateKey is the zero-value for a node/peer private key.
	ZeroPrivateKey PrivateKey

	// ZeroSignature is the zero-value for a cryptographic signature.
	ZeroSignature Signature
)

// GenerateKeys randomly generates a new pair of cryptographic keys. Nil may be passed to rand in order to use
// crypto/rand by default. It returns an error if rand is invalid.
func GenerateKeys(rand io.Reader) (publicKey PublicKey, privateKey PrivateKey, err error) {
	pub, priv, err := ed25519.GenerateKey(rand)
	if err != nil {
		return publicKey, privateKey, err
	}

	copy(publicKey[:], pub)
	copy(privateKey[:], priv)

	return publicKey, privateKey, nil
}

// LoadKeysFromHex loads a private key from a hex string. It returns an error if secretHex is not hex-encoded or is
// an invalid number of bytes. In the case of the latter error, the error is wrapped as io.ErrUnexpectedEOF. Calling
// this function performs 1 allocation.
func LoadKeysFromHex(secretHex string) (PrivateKey, error) {
	secret, err := hex.DecodeString(secretHex)
	if err != nil {
		return ZeroPrivateKey, fmt.Errorf("private key provided in hex failed to be decoded: %w", err)
	}

	if len(secret) != SizePrivateKey {
		return ZeroPrivateKey, fmt.Errorf("got private key of %d byte(s), but expected %d byte(s): %w",
			len(secret), SizePrivateKey, io.ErrUnexpectedEOF,
		)
	}

	var privateKey PrivateKey
	copy(privateKey[:], secret)

	return privateKey, nil
}

// Verify returns true if the cryptographic signature of data is representative of this public key.
func (k PublicKey) Verify(data []byte, signature Signature) bool {
	return ed25519.Verify(k[:], data, signature[:])
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
func (k PrivateKey) Sign(data []byte) Signature {
	return UnmarshalSignature(ed25519.Sign(k[:], data))
}

// String returns the hexadecimal representation of this private key.
func (k PrivateKey) String() string {
	return hex.EncodeToString(k[:])
}

// MarshalJSON returns the hexadecimal representation of this private key in JSON. It should never throw an error.
func (k PrivateKey) MarshalJSON() ([]byte, error) {
	return json.Marshal(k.String())
}

// Public returns the public key associated to this private key.
func (k PrivateKey) Public() PublicKey {
	var publicKey PublicKey
	copy(publicKey[:], (ed25519.PrivateKey)(k[:]).Public().(ed25519.PublicKey))

	return publicKey
}

// String returns the hexadecimal representation of this signature.
func (s Signature) String() string {
	return hex.EncodeToString(s[:])
}

// MarshalJSON returns the hexadecimal representation of this signature in JSON. It should never throw an error.
func (s Signature) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

// UnmarshalSignature decodes data into a Signature instance. It panics if data is not of expected length by instilling
// a bound check hint to the compiler. It uses unsafe hackery to zero-alloc convert data into a Signature.
func UnmarshalSignature(data []byte) Signature {
	_ = data[SizeSignature-1]
	return *(*Signature)(unsafe.Pointer((*reflect.SliceHeader)(unsafe.Pointer(&data)).Data))
}
