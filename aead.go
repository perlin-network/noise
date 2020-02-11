package noise

import (
	"crypto/cipher"
	"io"
	"math/rand"
)

func encryptAEAD(suite cipher.AEAD, buf []byte) ([]byte, error) {
	a, b := suite.NonceSize(), len(buf)

	buf = append(make([]byte, a), append(buf, make([]byte, suite.Overhead())...)...)

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
