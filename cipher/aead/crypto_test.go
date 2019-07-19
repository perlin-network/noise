package aead

import (
	"crypto/sha256"
	"github.com/stretchr/testify/assert"
	"testing"
	"testing/quick"
)

func TestDeriveSharedKey(t *testing.T) {
	block := New()

	check := func(ephemeralSharedKey []byte, context []byte) bool {
		_, _, err := block.deriveCipherSuite(sha256.New, ephemeralSharedKey, context)

		return err == nil
	}

	assert.NoError(t, quick.Check(check, nil))
}
