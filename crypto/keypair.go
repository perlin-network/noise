package crypto

import (
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/perlin-network/noise/crypto/hashing"
	"github.com/perlin-network/noise/crypto/signing"
)

type KeyPair struct {
	hp         hashing.HashPolicy
	sp         signing.SignaturePolicy
	PrivateKey []byte
	PublicKey  []byte
}

func NewKeyPair(sp signing.SignaturePolicy, hp hashing.HashPolicy) *KeyPair {
	privateKey, publicKey, err := sp.GenerateKeys()
	if err != nil {
		panic(err)
	}
	// generate keys if no private key present in signature policy
	p := &KeyPair{
		hp:         hp,
		sp:         sp,
		PrivateKey: privateKey,
		PublicKey:  publicKey,
	}
	return p
}

func newErrPrivKeySize(length int, sp signing.SignaturePolicy) error {
	return errors.New(fmt.Sprintf("private key length %d does not equal expected key length %d", length, sp.PrivateKeySize()))
}

func (k *KeyPair) Sign(message []byte) ([]byte, error) {
	if len(k.PublicKey) != k.sp.PrivateKeySize() {
		return nil, newErrPrivKeySize(len(k.PrivateKey), k.sp)
	}

	message = k.hp.HashBytes(message)

	signature := k.sp.Sign(k.PrivateKey, message)
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

func (k *KeyPair) Verify(message []byte, signature []byte) bool {
	return Verify(k.sp, k.hp, k.PublicKey, message, signature)
}

func FromPrivateKey(sp signing.SignaturePolicy, hp hashing.HashPolicy, privateKey string) (*KeyPair, error) {
	rawPrivateKey, err := hex.DecodeString(privateKey)
	if err != nil {
		return nil, err
	}

	return FromPrivateKeyBytes(sp, hp, rawPrivateKey)
}

func FromPrivateKeyBytes(sp signing.SignaturePolicy, hp hashing.HashPolicy, rawPrivateKey []byte) (*KeyPair, error) {
	if len(rawPrivateKey) != sp.PrivateKeySize() {
		return nil, newErrPrivKeySize(len(rawPrivateKey), sp)
	}

	rawPublicKey, err := sp.PrivateToPublic(rawPrivateKey)
	if err != nil {
		return nil, err
	}

	keyPair := &KeyPair{
		sp:         sp,
		hp:         hp,
		PrivateKey: rawPrivateKey,
		PublicKey:  rawPublicKey,
	}

	return keyPair, nil
}

func Verify(sp signing.SignaturePolicy, hp hashing.HashPolicy, publicKey []byte, message []byte, signature []byte) bool {
	// Public key must be a set size.
	if len(publicKey) != sp.PublicKeySize() {
		return false
	}

	message = hp.HashBytes(message)
	return sp.Verify(publicKey, message, signature)
}
