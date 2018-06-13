package crypto

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"golang.org/x/crypto/ed25519"
	"strconv"
)

var (
	ErrPrivKeySize = errors.New("private key !=" + strconv.Itoa(ed25519.PrivateKeySize) + " bytes")
)

type KeyPair struct {
	PrivateKey []byte
	PublicKey  []byte
}

func (k *KeyPair) Sign(message []byte) ([]byte, error) {
	message = HashBytes(message)
	if len(k.PrivateKey) != ed25519.PrivateKeySize {
		return nil, ErrPrivKeySize
	}

	signature := ed25519.Sign(ed25519.PrivateKey(k.PrivateKey), message)
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

func RandomKeyPair() *KeyPair {
	publicKey, privateKey, _ := ed25519.GenerateKey(rand.Reader)

	return &KeyPair{
		PublicKey:  publicKey,
		PrivateKey: privateKey,
	}
}

func FromPrivateKey(privateKey string) (*KeyPair, error) {
	rawPrivateKey, err := hex.DecodeString(privateKey)
	if err != nil {
		return nil, err
	}

	return FromPrivateKeyBytes(rawPrivateKey)
}

func FromPrivateKeyBytes(rawPrivateKey []byte) (*KeyPair, error) {
	if len(rawPrivateKey) != ed25519.PrivateKeySize {
		return nil, ErrPrivKeySize
	}

	rawPublicKey := GetPublicKey(rawPrivateKey)

	keyPair := &KeyPair{
		PrivateKey: rawPrivateKey,
		PublicKey:  rawPublicKey,
	}

	return keyPair, nil
}

func GetPublicKey(rawPrivateKey []byte) []byte {
	return ed25519.PrivateKey(rawPrivateKey).Public().([]byte)
}

func Verify(publicKey []byte, message []byte, signature []byte) bool {
	message = HashBytes(message)
	return ed25519.Verify(ed25519.PublicKey(publicKey), message, signature)
}
