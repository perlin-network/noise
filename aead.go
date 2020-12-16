package noise

import (
	"crypto/cipher"
	"crypto/rand"
	"io"
)

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

func encryptAEAD(suite cipher.AEAD, buf []byte) ([]byte, error) {
	a, b := suite.NonceSize(), len(buf)

	buf = extendFront(buf, a)
	buf = extendBack(buf, b)

	if _, err := rand.Read(buf[:a]); err != nil {
		return nil, err
	}

	return append(buf[:a], suite.Seal(buf[a:a], buf[:a], buf[a:a+b], nil)...), nil
}

func decryptAEAD(suite cipher.AEAD, buf []byte) ([]byte, error) {
	if len(buf) < suite.NonceSize() {
		return nil, io.ErrUnexpectedEOF
	}

	nonce := buf[:suite.NonceSize()]
	text := buf[suite.NonceSize():]

	return suite.Open(text[:0], nonce, text, nil)
}
