package noise

import (
	"encoding/hex"
	"encoding/json"
	"github.com/oasislabs/ed25519"
	"io"
)

const (
	PublicKeySize  = ed25519.PublicKeySize
	PrivateKeySize = ed25519.PrivateKeySize
)

type (
	PublicKey  [PublicKeySize]byte
	PrivateKey [PrivateKeySize]byte
)

var (
	ZeroPublicKey  PublicKey
	ZeroPrivateKey PrivateKey
)

func GenerateKeys(rand io.Reader) (publicKey PublicKey, privateKey PrivateKey, err error) {
	pub, priv, err := ed25519.GenerateKey(rand)
	if err != nil {
		return publicKey, privateKey, err
	}

	copy(publicKey[:], pub)
	copy(privateKey[:], priv)

	return publicKey, privateKey, nil
}

func (k PublicKey) Verify(data, signature []byte) bool {
	return ed25519.Verify(k[:], data, signature)
}

func (k PublicKey) String() string {
	return hex.EncodeToString(k[:])
}

func (k PublicKey) MarshalJSON() ([]byte, error) {
	return json.Marshal(hex.EncodeToString(k[:]))
}

func (k PrivateKey) Sign(data []byte) []byte {
	return ed25519.Sign(k[:], data)
}

func (k PrivateKey) String() string {
	return hex.EncodeToString(k[:])
}

func (k PrivateKey) MarshalJSON() ([]byte, error) {
	return json.Marshal(hex.EncodeToString(k[:]))
}
