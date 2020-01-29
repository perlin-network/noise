package noise

import (
	"crypto/cipher"
	"github.com/valyala/bytebufferpool"
	"math/rand"
)

func encryptAEAD(suite cipher.AEAD, buf []byte) ([]byte, error) {
	dst := bytebufferpool.Get()
	defer bytebufferpool.Put(dst)

	nonce := make([]byte, suite.NonceSize())

	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	return append(nonce, suite.Seal(dst.B[:0], nonce, buf, nil)...), nil
}

func decryptAEAD(suite cipher.AEAD, buf []byte) ([]byte, error) {
	nonce := buf[:suite.NonceSize()]
	text := buf[suite.NonceSize():]

	return suite.Open(text[:0], nonce, text, nil)
}
