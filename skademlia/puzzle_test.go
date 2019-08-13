package skademlia

import (
	"github.com/stretchr/testify/assert"
	"math"
	"testing"
)

func TestGenerateKeys(t *testing.T) {
	t.Parallel()

	c1 := 8
	c2 := 8

	pub, priv, id, checksum, err := generateKeys(c1)
	assert.NoError(t, err)
	assert.NotNil(t, pub)
	assert.NotNil(t, priv)
	assert.NotNil(t, id)
	assert.NotNil(t, checksum)

	assert.True(t, prefixLen(checksum[:]) >= c1)
	nonce, err := generateNonce(checksum, c2)
	assert.NoError(t, err)
	assert.NoError(t, verifyPuzzle(checksum, nonce, c1, c2))

	assert.Error(t, verifyPuzzle(checksum, nonce, math.MaxInt32, c2))
	assert.Error(t, verifyPuzzle(checksum, nonce, c1, math.MaxInt32))
}
