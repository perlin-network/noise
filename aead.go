package noise

import (
	"crypto/cipher"
	"io"
	"math/rand"
)

func encryptAEAD(suite cipher.AEAD, buf []byte) ([]byte, error) {
	nonce := make([]byte, suite.NonceSize())

	if _, err := rand.Read(nonce[:]); err != nil {
		return nil, err
	}

	return append(nonce, suite.Seal(buf[:0], nonce[:suite.NonceSize()], buf, nil)...), nil
}

func decryptAEAD(suite cipher.AEAD, buf []byte) ([]byte, error) {
	if len(buf) < suite.NonceSize() {
		return nil, io.ErrUnexpectedEOF
	}

	nonce := buf[:suite.NonceSize()]
	text := buf[suite.NonceSize():]

	return suite.Open(text[:0], nonce, text, nil)
}
