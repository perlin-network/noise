package cipher

import (
	"crypto/sha256"
	"github.com/stretchr/testify/assert"
	"testing"
	"testing/quick"
)

func TestDeriveSharedKey(t *testing.T) {
	check := func(ephemeralSharedKey []byte, context []byte) bool {
		_, _, err := deriveCipherSuite(Aes256Gcm, sha256.New, ephemeralSharedKey, context)
		return assert.NoError(t, err)
	}

	assert.NoError(t, quick.Check(check, &quick.Config{MaxCount: 1000}))
}
