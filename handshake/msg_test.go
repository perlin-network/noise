package handshake

import (
	"github.com/perlin-network/noise/edwards25519"
	"github.com/stretchr/testify/assert"
	"testing"
	"testing/quick"
)

func TestMarshalUnmarshalHandshake(t *testing.T) {
	f := func(pub edwards25519.PublicKey, sig edwards25519.Signature) bool {
		m := Handshake{publicKey: pub, signature: sig}
		m2, err := UnmarshalHandshake(m.Marshal())

		return assert.NoError(t, err) && assert.EqualValues(t, m, m2)
	}

	assert.NoError(t, quick.Check(f, &quick.Config{MaxCount: 1000}))
}
