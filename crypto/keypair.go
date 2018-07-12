package crypto

import (
	"encoding/hex"
	"errors"
	"fmt"
)

type KeyPair struct {
	PrivateKey []byte
	PublicKey  []byte
}

func newErrPrivKeySize(length int, sp SignaturePolicy) error {
	return errors.New(fmt.Sprintf("private key length %d does not equal expected key length %d", length, sp.PrivateKeySize()))
}

func (k *KeyPair) Sign(sp SignaturePolicy, hp HashPolicy, message []byte) ([]byte, error) {
	if len(k.PrivateKey) != sp.PrivateKeySize() {
		return nil, newErrPrivKeySize(len(k.PrivateKey), sp)
	}

	message = hp.HashBytes(message)

	signature := sp.Sign(k.PrivateKey, message)
	return signature, nil
}

func (k *KeyPair) PrivateKeyHex() string {
	return hex.EncodeToString(k.PrivateKey)
}

func (k *KeyPair) PublicKeyHex() string {
	return hex.EncodeToString(k.PublicKey)
}

func (k *KeyPair) String() (string, string) {
	return k.PrivateKeyHex(), k.PublicKeyHex()
}

func FromPrivateKey(sp SignaturePolicy, hp HashPolicy, privateKey string) (*KeyPair, error) {
	rawPrivateKey, err := hex.DecodeString(privateKey)
	if err != nil {
		return nil, err
	}

	return FromPrivateKeyBytes(sp, hp, rawPrivateKey)
}

func FromPrivateKeyBytes(sp SignaturePolicy, hp HashPolicy, rawPrivateKey []byte) (*KeyPair, error) {
	if len(rawPrivateKey) != sp.PrivateKeySize() {
		return nil, newErrPrivKeySize(len(rawPrivateKey), sp)
	}

	rawPublicKey, err := sp.PrivateToPublic(rawPrivateKey)
	if err != nil {
		return nil, err
	}

	keyPair := &KeyPair{
		PrivateKey: rawPrivateKey,
		PublicKey:  rawPublicKey,
	}

	return keyPair, nil
}

func Verify(sp SignaturePolicy, hp HashPolicy, publicKey []byte, message []byte, signature []byte) bool {
	// Public key must be a set size.
	if len(publicKey) != sp.PublicKeySize() {
		return false
	}

	message = hp.HashBytes(message)
	return sp.Verify(publicKey, message, signature)
}
