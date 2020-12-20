package noise

import (
	"crypto/cipher"
	"crypto/rand"
	"io"
)

type aeadEncryption struct {
	suite cipher.AEAD
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
	a, b := e.suite.NonceSize(), len(buf)

	buf = extendFront(buf, a)
	buf = extendBack(buf, b)

	if _, err := rand.Read(buf[:a]); err != nil {
		return nil, err
	}

	return append(buf[:a], e.suite.Seal(buf[a:a], buf[:a], buf[a:a+b], nil)...), nil
}

func (e *aeadEncryption) decrypt(buf []byte) ([]byte, error) {
	if len(buf) < e.suite.NonceSize() {
		return nil, io.ErrUnexpectedEOF
	}

	nonce := buf[:e.suite.NonceSize()]
	text := buf[e.suite.NonceSize():]

	return e.suite.Open(text[:0], nonce, text, nil)
}
