package crypto

import (
	"crypto/rand"
	"reflect"
	"testing"
)

func TestSignVerify(t *testing.T) {
	p := NewEd25519()

	kp := RandomKeyPair(p)
	message := make([]byte, 120)

	_, err := rand.Read(message)
	if err != nil {
		t.Fatal(err)
	}

	sig, err := kp.Sign(p, message)
	if err != nil {
		t.Fatal(err)
	}

	// Test if signature works.
	ok := Verify(p, kp.PublicKey, message, sig)
	if !ok {
		t.Fatal("signature verification failed with correct info")
	}

	ok = Verify(p, message, message, sig)
	if ok {
		t.Fatal("signature verification failed with wrong public key size/contents")
	}

	sig[0] = ^sig[0]
	ok = Verify(p, kp.PublicKey, message, sig)
	if ok {
		t.Fatal("invalid signature passed verification unexpectedly")
	}
}

func TestFromPrivateKey(t *testing.T) {
	p := NewEd25519()

	kp1 := RandomKeyPair(p)
	kp2, err := FromPrivateKey(p, kp1.PrivateKeyHex())
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(kp1, kp2) {
		t.Fatal("kp1 and kp2 are not deep-equal.")
	}
}
