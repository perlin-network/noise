package noise

import (
	"crypto/cipher"
	"github.com/valyala/bytebufferpool"
	"io"
	"math/rand"
)

func encryptAEAD(suite cipher.AEAD, buf []byte) ([]byte, error) {
	dst := bytebufferpool.Get()
	defer bytebufferpool.Put(dst)

	dst.B = append(dst.B[:0], make([]byte, suite.NonceSize())...)
	dst.B = dst.B[:suite.NonceSize()]

	if _, err := rand.Read(dst.B[:suite.NonceSize()]); err != nil {
		return nil, err
	}

	return append(dst.B, suite.Seal(buf[:0], dst.B[:suite.NonceSize()], buf, nil)...), nil
}

func decryptAEAD(suite cipher.AEAD, buf []byte) ([]byte, error) {
	if len(buf) < suite.NonceSize() {
		return nil, io.ErrUnexpectedEOF
	}

	nonce := buf[:suite.NonceSize()]
	text := buf[suite.NonceSize():]

	return suite.Open(text[:0], nonce, text, nil)
}
