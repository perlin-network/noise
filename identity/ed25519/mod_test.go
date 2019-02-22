package ed25519_test

import (
	"github.com/perlin-network/noise/identity/ed25519"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestEd25519(t *testing.T) {
	t.Parallel()
	p := ed25519.RandomKeys()

	publicKey := p.PublicKey()
	privateKey := p.PrivateKey()
	assert.True(t, len(p.PublicKey()) > 0)
	assert.True(t, len(p.String()) > 0)

	// try reloading the private key, should make the same object
	mgr := ed25519.LoadKeys(privateKey)
	assert.NotNil(t, mgr)
	assert.EqualValues(t, mgr.PublicKey(), publicKey)
}
