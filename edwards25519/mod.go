// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package ed25519 implements the Ed25519 signature algorithm. See
// https://ed25519.cr.yp.to/.
//
// These functions are also compatible with the “Ed25519” function defined in
// https://tools.ietf.org/html/draft-irtf-cfrg-eddsa-05.
package edwards25519

// This code is a port of the public domain, “ref10” implementation of ed25519
// from SUPERCOP.

import (
	"crypto"
	cryptorand "crypto/rand"
	"crypto/sha512"
	"crypto/subtle"
	"errors"
	"io"
)

const (
	// SizePublicKey is the size, in bytes, of public keys as used in this package.
	SizePublicKey = 32
	// SizePrivateKey is the size, in bytes, of private keys as used in this package.
	SizePrivateKey = 64
	// SizeSignature is the size, in bytes, of signatures generated and verified by this package.
	SizeSignature = 64
)

// PublicKey is the type of Ed25519 public keys.
type PublicKey [SizePublicKey]byte

// PrivateKey is the type of Ed25519 private keys.
type PrivateKey [SizePrivateKey]byte

// Signature is type of Ed25519 signatures.
type Signature [SizeSignature]byte

// Public returns the PublicKey corresponding to priv.
func (p PrivateKey) Public() crypto.PublicKey {
	var publicKey PublicKey
	copy(publicKey[:], p[SizePrivateKey/2:])

	return PublicKey(publicKey)
}

// Sign signs the given message with a private key..
//
// Ed25519 performs two passes over messages to be signed and therefore cannot handle
// pre-hashed messages.
//
// Thus opts.HashFunc() must return zero to indicate the message hasn't been hashed.
//
// This can be achieved by passing crypto.Hash(0) as the value for opts.
func (p PrivateKey) Sign(message []byte, opts crypto.SignerOpts) (signature Signature, err error) {
	if opts.HashFunc() != crypto.Hash(0) {
		return Signature{}, errors.New("edwards25519: cannot sign hashed message")
	}

	return Sign(p, message), nil
}

// GenerateKey generates a public/private key pair using entropy from rand.
//
// If rand is nil, crypto/rand.Reader will be used.
func GenerateKey(rand io.Reader) (publicKey PublicKey, privateKey PrivateKey, err error) {
	if rand == nil {
		rand = cryptorand.Reader
	}

	if _, err = io.ReadFull(rand, privateKey[:SizePrivateKey/2]); err != nil {
		return publicKey, privateKey, err
	}

	digest := sha512.Sum512(privateKey[:SizePrivateKey/2])
	digest[0] &= 248
	digest[SizePrivateKey/2-1] &= 127
	digest[SizePrivateKey/2-1] |= 64

	var digestBuf [sha512.Size / 2]byte
	copy(digestBuf[:], digest[:])

	var A ExtendedGroupElement
	GeScalarMultBase(&A, &digestBuf)

	A.ToBytes((*[SizePublicKey]byte)(&publicKey))
	copy(privateKey[SizePrivateKey/2:], publicKey[:])

	return publicKey, privateKey, nil
}

// Sign signs the message with privateKey and returns a signature.
func Sign(privateKey PrivateKey, message []byte) Signature {
	h := sha512.New()
	h.Write(privateKey[:SizePrivateKey/2])

	var digest, messageDigest, hramDigest [sha512.Size]byte
	var expandedSecretKey [SizePrivateKey / 2]byte
	h.Sum(digest[:0])

	copy(expandedSecretKey[:], digest[:])

	expandedSecretKey[0] &= 248
	expandedSecretKey[SizePrivateKey/2-1] &= 63
	expandedSecretKey[SizePrivateKey/2-1] |= 64

	h.Reset()
	h.Write(digest[sha512.Size/2:])
	h.Write(message)
	h.Sum(messageDigest[:0])

	var messageDigestReduced [32]byte
	ScReduce(&messageDigestReduced, &messageDigest)

	var R ExtendedGroupElement
	GeScalarMultBase(&R, &messageDigestReduced)

	var encodedR [SizeSignature / 2]byte
	R.ToBytes(&encodedR)

	h.Reset()
	h.Write(encodedR[:])
	h.Write(privateKey[SizePrivateKey/2:])
	h.Write(message)
	h.Sum(hramDigest[:0])

	var hramDigestReduced [32]byte
	ScReduce(&hramDigestReduced, &hramDigest)

	var s [SizeSignature / 2]byte
	ScMulAdd(&s, &hramDigestReduced, &expandedSecretKey, &messageDigestReduced)

	var signature Signature
	copy(signature[:SizeSignature/2], encodedR[:])
	copy(signature[SizeSignature/2:], s[:])

	return signature
}

// Verify reports whether signature is a valid signature of message by publicKey.
func Verify(publicKey PublicKey, message []byte, signature Signature) bool {
	if signature[SizeSignature-1]&224 != 0 {
		return false
	}

	var A ExtendedGroupElement
	if !A.FromBytes((*[SizePublicKey]byte)(&publicKey)) {
		return false
	}

	FeNeg(&A.X, &A.X)
	FeNeg(&A.T, &A.T)

	h := sha512.New()
	h.Write(signature[:SizeSignature/2])
	h.Write(publicKey[:])
	h.Write(message)

	var digest [sha512.Size]byte
	h.Sum(digest[:0])

	var hReduced [32]byte
	ScReduce(&hReduced, &digest)

	var R ProjectiveGroupElement
	var b [32]byte
	copy(b[:], signature[SizeSignature/2:])
	GeDoubleScalarMultVartime(&R, &hReduced, &A, &b)

	var checkR [32]byte
	R.ToBytes(&checkR)

	return subtle.ConstantTimeCompare(signature[:SizeSignature/2], checkR[:]) == 1
}
