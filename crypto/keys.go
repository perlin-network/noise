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

func newErrPrivKeySize(p SignaturePolicy) error {
	return errors.New("private key !=" + strconv.Itoa(p.PrivateKeySize()) + " bytes")
}

func (k *KeyPair) Sign(sp SignaturePolicy, hp HashPolicy, message []byte) ([]byte, error) {
	if len(k.PrivateKey) != sp.PrivateKeySize() {
		return nil, newErrPrivKeySize(sp)
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

func (k *KeyPair) String() string {
	return fmt.Sprintf("Private Key: %s\nPublic Key: %s", k.PrivateKeyHex(), k.PublicKeyHex())
}

func FromPrivateKey(p SignaturePolicy, privateKey string) (*KeyPair, error) {
	rawPrivateKey, err := hex.DecodeString(privateKey)
	if err != nil {
		return nil, err
	}

	return FromPrivateKeyBytes(p, rawPrivateKey)
}

func FromPrivateKeyBytes(p SignaturePolicy, rawPrivateKey []byte) (*KeyPair, error) {
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

func Verify(sp SignaturePolicy, hp HashPolicy, publicKey []byte, message []byte, signature []byte) bool {
	// Public key must be a set size.
	if len(publicKey) != sp.PublicKeySize() {
		return false
	}

	message = hp.HashBytes(message)
	return sp.Verify(publicKey, message, signature)
}
