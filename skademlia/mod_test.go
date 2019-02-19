package skademlia

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"testing/quick"
)

func TestNew(t *testing.T) {
	block := New()

	assert.Equal(t, false, block.enforceSignatures)
	assert.Equal(t, DefaultC1, block.c1)
	assert.Equal(t, DefaultC2, block.c2)
	assert.Equal(t, DefaultPrefixDiffLen, block.prefixDiffLen)
	assert.Equal(t, DefaultPrefixDiffMin, block.prefixDiffMin)

	assert.NoError(t, quick.Check(func(c1 int) bool {
		return block.WithC1(c1).c1 == c1
	}, nil))

	assert.NoError(t, quick.Check(func(c2 int) bool {
		return block.WithC2(c2).c2 == c2
	}, nil))

	assert.NoError(t, quick.Check(func(prefixDiffLen int) bool {
		return block.WithPrefixDiffLen(prefixDiffLen).prefixDiffLen == prefixDiffLen
	}, nil))

	assert.NoError(t, quick.Check(func(prefixDiffMin int) bool {
		return block.WithPrefixDiffMin(prefixDiffMin).prefixDiffMin == prefixDiffMin
	}, nil))
}
