package noise

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"io"
)

type aeadEncryption struct {
	suite cipher.AEAD
	// GCM takes a 12 byte nonce by default
	fixed   [4]byte //Random part, 4 bytes
	counter int64   //Counter	   8 bytes
}

func newAEAD(key []byte) (aeadEncryption, error) {
	core, err := aes.NewCipher(key)
	if err != nil {
		return aeadEncryption{}, err
	}

	suite, err := cipher.NewGCM(core)
	if err != nil {
		return aeadEncryption{}, err
	}

	var encryption aeadEncryption
	encryption.suite = suite

	//Generate fixed portion of repetition resistant nonce
	//https://tools.ietf.org/id/draft-mcgrew-iv-gen-01.html
	if _, err := rand.Read(encryption.fixed[:]); err != nil {
		return aeadEncryption{}, err
	}

	return encryption, nil
}

func extendFront(buf []byte, n int) []byte {
	if len(buf) < n {
		clone := make([]byte, n+len(buf))
		copy(clone[n:], buf)

		return clone
	}

	return append(buf[:n], buf...)
}

func extendBack(buf []byte, n int) []byte {
	n += len(buf)
	if nn := n - cap(buf); nn > 0 {
		buf = append(buf[:cap(buf)], make([]byte, nn)...)
	}
	return buf[:n]
}

func (e *aeadEncryption) initialised() bool {
	return e.suite == nil
}

func (e *aeadEncryption) encrypt(buf []byte) ([]byte, error) {
	nonceSize, plaintextSize := e.suite.NonceSize(), len(buf)

	buf = extendFront(buf, nonceSize)
	buf = extendBack(buf, plaintextSize)

	//Repetition resistant nonce https://tools.ietf.org/html/rfc5116#section-3.2

	copy(buf[:nonceSize], e.fixed[:])

	binary.BigEndian.PutUint64(buf[len(e.fixed):nonceSize], uint64(e.counter))

	//Increment Nonce counter
	e.counter++

	//Reuse the storage of buf, taking nonce buf[:nonceSize] and plaintext[nonceSize:nonceSize+plaintextSize]
	//Put nonce on the front of the ciphertext
	return append(buf[:nonceSize], e.suite.Seal(buf[nonceSize:nonceSize], buf[:nonceSize], buf[nonceSize:nonceSize+plaintextSize], nil)...), nil
}

func (e *aeadEncryption) decrypt(buf []byte) ([]byte, error) {
	if len(buf) < e.suite.NonceSize() {
		return nil, io.ErrUnexpectedEOF
	}

	nonce := buf[:e.suite.NonceSize()]
	text := buf[e.suite.NonceSize():]

	return e.suite.Open(text[:0], nonce, text, nil)
}
