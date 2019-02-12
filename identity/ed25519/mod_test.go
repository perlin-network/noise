package ed25519_test

import (
	"crypto/rand"
	"encoding/hex"
	"github.com/perlin-network/noise/identity/ed25519"
	"github.com/stretchr/testify/assert"
	"testing"
)

func BenchmarkSign(b *testing.B) {
	p := ed25519.Random()

	message := make([]byte, 32)
	if _, err := rand.Read(message); err != nil {
		panic(err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		sig, err := p.Sign(message)
		if err != nil || len(sig) == 0 {
			panic("signing failed")
		}
	}
}

func BenchmarkVerify(b *testing.B) {
	p := ed25519.Random()

	message := make([]byte, 32)
	if _, err := rand.Read(message); err != nil {
		panic(err)
	}

	publicKey := p.PublicID()

	b.ResetTimer()

	sig, err := p.Sign(message)
	if err != nil {
		panic(err)
	}

	for i := 0; i < b.N; i++ {
		if err := p.Verify(publicKey, message, sig); err != nil {
			panic("verification failed")
		}
	}
}

func TestEd25519(t *testing.T) {
	t.Parallel()
	p := ed25519.Random()

	publicKey := p.PublicID()
	privateKeyHex := (*ed25519.Manager)(p).PrivateKey()
	assert.True(t, len(p.PublicID()) > 0)
	assert.True(t, len(p.String()) > 0)

	message := []byte("test message")
	// sign with a bad key should have yield signature with 0 length

	// length of signature should not be 0
	sig, err := p.Sign(message)
	assert.Nil(t, err)
	assert.True(t, len(sig) > 0)

	// correct message should pass verify check
	err = p.Verify(publicKey, message, sig)
	assert.Nil(t, err)

	// wrong public key should fail verify check
	err = p.Verify([]byte("bad key"), message, sig)
	assert.NotNil(t, err)

	// wrong message should fail verify check
	wrongMessage := []byte("wrong message")
	err = p.Verify(publicKey, wrongMessage, sig)
	assert.NotNil(t, err)

	privateKey, err := hex.DecodeString(privateKeyHex)
	assert.Nil(t, err)

	// try reloading the private key, should make the same object
	mgr := ed25519.New(privateKey)
	assert.NotNil(t, mgr)
	assert.EqualValues(t, mgr.PublicID(), publicKey)

	// make sure signing is different
	badMgr := ed25519.New([]byte("bad"))
	badSig, err := badMgr.Sign(message)
	assert.Nil(t, err)
	assert.NotEqual(t, sig, badSig)
}
