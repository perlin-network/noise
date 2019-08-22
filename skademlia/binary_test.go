package skademlia

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestPrefixLen(t *testing.T) {
	assert.Equal(t, 0, prefixLen(nil))

	assert.Equal(t, 8, prefixLen([]byte{0}))

	assert.Equal(t, 16, prefixLen([]byte{0, 0}))

	assert.Equal(t, 15, prefixLen([]byte{0, 1}))
}
