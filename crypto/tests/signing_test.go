package tests

import (
	"crypto/rand"
	"github.com/perlin-network/noise/crypto"
	"github.com/perlin-network/noise/crypto/hashing/blake2b"
	"github.com/perlin-network/noise/crypto/signing/ed25519"
	"testing"
)

func TestSignVerify(t *testing.T) {
	sp := ed25519.New()
	hp := blake2b.New()

	kp := ed25519.RandomKeyPair()
	message := make([]byte, 120)

	_, err := rand.Read(message)
	if err != nil {
		t.Fatal(err)
	}

	sig, err := kp.Sign(sp, hp, message)
	if err != nil {
		t.Fatal(err)
	}

	// Test if signature works.
	ok := crypto.Verify(sp, hp, kp.PublicKey, message, sig)
	if !ok {
		t.Fatal("signature verification failed with correct info")
	}

	ok = crypto.Verify(sp, hp, message, message, sig)
	if ok {
		t.Fatal("signature verification failed with wrong public key size/contents")
	}

	sig[0] = ^sig[0]
	ok = crypto.Verify(sp, hp, kp.PublicKey, message, sig)
	if ok {
		t.Fatal("invalid signature passed verification unexpectedly")
	}
}
