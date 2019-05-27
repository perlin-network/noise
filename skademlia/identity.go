// Copyright (c) 2019 Perlin
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package skademlia

import (
	"encoding/binary"
	"fmt"
	"github.com/perlin-network/noise/edwards25519"
	"github.com/pkg/errors"
	"golang.org/x/crypto/blake2b"
	"io"
)

type ID struct {
	address   string
	publicKey edwards25519.PublicKey

	id, checksum, nonce [blake2b.Size256]byte
}

func NewID(address string, publicKey edwards25519.PublicKey, nonce [blake2b.Size256]byte) *ID {
	id := blake2b.Sum256(publicKey[:])
	checksum := blake2b.Sum256(id[:])

	return &ID{
		address:   address,
		publicKey: publicKey,

		id:       id,
		checksum: checksum,
		nonce:    nonce,
	}
}

func (m ID) Address() string {
	return m.address
}

func (m ID) PublicKey() edwards25519.PublicKey {
	return m.publicKey
}

func (m ID) Checksum() [blake2b.Size256]byte {
	return m.checksum
}

func (m ID) Nonce() [blake2b.Size256]byte {
	return m.nonce
}

func (m ID) String() string {
	return fmt.Sprintf("%s[%x]", m.address, m.publicKey)
}

func (m ID) Marshal() []byte {
	buf := make([]byte, 2+len(m.address)+edwards25519.SizePublicKey+blake2b.Size256)

	binary.BigEndian.PutUint16(buf[0:2], uint16(len(m.address)))
	copy(buf[2:2+len(m.address)], m.address)
	copy(buf[2+len(m.address):2+len(m.address)+edwards25519.SizePublicKey], m.publicKey[:])
	copy(buf[2+len(m.address)+edwards25519.SizePublicKey:2+len(m.address)+edwards25519.SizePublicKey+blake2b.Size256], m.nonce[:])

	return buf
}

func UnmarshalID(b io.Reader) (m ID, err error) {
	var buf [2]byte

	if _, err = io.ReadFull(b, buf[:]); err != nil {
		return ID{}, err
	}

	length := binary.BigEndian.Uint16(buf[:])

	address := make([]byte, length)

	if _, err = io.ReadFull(b, address); err != nil {
		return ID{}, err
	}

	m.address = string(address)

	if _, err = io.ReadFull(b, m.publicKey[:]); err != nil {
		return ID{}, err
	}

	m.id = blake2b.Sum256(m.publicKey[:])
	m.checksum = blake2b.Sum256(m.id[:])

	if _, err = io.ReadFull(b, m.nonce[:]); err != nil {
		return ID{}, err
	}

	return
}

type Keypair struct {
	privateKey edwards25519.PrivateKey
	publicKey  edwards25519.PublicKey

	id, checksum, nonce [blake2b.Size256]byte
	c1, c2              int
}

func (k *Keypair) ID(addr string) *ID {
	return NewID(addr, k.publicKey, k.nonce)
}

func (k *Keypair) PrivateKey() edwards25519.PrivateKey {
	return k.privateKey
}

func (k *Keypair) PublicKey() edwards25519.PublicKey {
	return k.publicKey
}

func NewKeys(c1, c2 int) (*Keypair, error) {
	publicKey, privateKey, id, checksum, err := generateKeys(c1)

	if err != nil {
		return nil, err
	}

	nonce, err := generateNonce(checksum, c2)

	if err != nil {
		return nil, errors.Wrap(err, "failed to generate valid puzzle nonce")
	}

	keys := &Keypair{
		privateKey: privateKey,
		publicKey:  publicKey,

		id:       id,
		checksum: checksum,
		nonce:    nonce,

		c1: c1,
		c2: c2,
	}

	return keys, nil
}

func LoadKeys(privateKey edwards25519.PrivateKey, c1, c2 int) (*Keypair, error) {
	publicKey := privateKey.Public()

	id := blake2b.Sum256(publicKey[:])
	checksum := blake2b.Sum256(id[:])

	nonce, err := generateNonce(checksum, c2)
	if err != nil {
		return nil, errors.Wrap(err, "faiuled to generate valid puzzle nonce")
	}

	if err := verifyPuzzle(checksum, nonce, c1, c2); err != nil {
		return nil, errors.Wrap(err, "keys are invalid")
	}

	keys := &Keypair{
		privateKey: privateKey,
		publicKey:  publicKey,

		id:       id,
		checksum: checksum,
		nonce:    nonce,

		c1: c1,
		c2: c2,
	}

	return keys, nil
}
