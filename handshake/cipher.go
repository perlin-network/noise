package handshake

import (
	"crypto/sha512"
	"github.com/perlin-network/noise/edwards25519"
)

func computeSharedKey(nodePrivateKey edwards25519.PrivateKey, remotePublicKey edwards25519.PublicKey) []byte {
	var nodeSecretKeyBuf, sharedKeyBuf [32]byte
	copy(nodeSecretKeyBuf[:], deriveSecretKey(nodePrivateKey))

	var sharedKeyElement, publicKeyElement edwards25519.ExtendedGroupElement
	publicKeyElement.FromBytes((*[32]byte)(&remotePublicKey))

	edwards25519.GeScalarMult(&sharedKeyElement, &nodeSecretKeyBuf, &publicKeyElement)

	sharedKeyElement.ToBytes(&sharedKeyBuf)

	return sharedKeyBuf[:]
}

func deriveSecretKey(privateKey edwards25519.PrivateKey) []byte {
	digest := sha512.Sum512(privateKey[:32])
	digest[0] &= 248
	digest[31] &= 127
	digest[31] |= 64

	return digest[:32]
}
