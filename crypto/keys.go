package crypto

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
)

type KeyPair struct {
	PrivateKey []byte
	PublicKey  []byte
}

func newErrPrivKeySize(p Provider) error {
	return errors.New("private key !=" + strconv.Itoa(p.PrivateKeySize()) + " bytes")
}

func (k *KeyPair) Sign(p Provider, message []byte) ([]byte, error) {
	if len(k.PrivateKey) != p.PrivateKeySize() {
		return nil, newErrPrivKeySize(p)
	}

	message = HashBytes(message)

	signature := p.Sign(k.PrivateKey, message)
	return signature, nil
}

func (k *KeyPair) PrivateKeyHex() string {
	return hex.EncodeToString(k.PrivateKey)
}

func (k *KeyPair) PublicKeyHex() string {
	return hex.EncodeToString(k.PublicKey)
}

func (k *KeyPair) String() string {
	return fmt.Sprintf("Private Key: %s\nPublic Key: %s", k.PrivateKeyHex(), k.PublicKeyHex())
}

func RandomKeyPair(p Provider) *KeyPair {
	k, err := p.GenerateKeyPair()
	if err != nil {
		panic(err)
	}
	return &k
}

func FromPrivateKey(p Provider, privateKey string) (*KeyPair, error) {
	rawPrivateKey, err := hex.DecodeString(privateKey)
	if err != nil {
		return nil, err
	}

	return FromPrivateKeyBytes(p, rawPrivateKey)
}

func FromPrivateKeyBytes(p Provider, rawPrivateKey []byte) (*KeyPair, error) {
	if len(rawPrivateKey) != p.PrivateKeySize() {
		return nil, newErrPrivKeySize(p)
	}

	rawPublicKey, err := p.PrivateToPublic(rawPrivateKey)
	if err != nil {
		return nil, err
	}

	keyPair := &KeyPair{
		PrivateKey: rawPrivateKey,
		PublicKey:  rawPublicKey,
	}

	return keyPair, nil
}

func Verify(p Provider, publicKey []byte, message []byte, signature []byte) bool {
	// Public key must be a set size.
	if len(publicKey) != p.PublicKeySize() {
		return false
	}

	message = HashBytes(message)
	return p.Verify(publicKey, message, signature)
}
