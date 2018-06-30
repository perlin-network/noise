package crypto

import (
	"crypto/rand"
	"reflect"
	"testing"
)

func TestSignVerify(t *testing.T) {
	kp := RandomKeyPair()
	message := make([]byte, 120)

	_, err := rand.Read(message)
	if err != nil {
		t.Fatal(err)
	}

	sig, err := kp.Sign(message)
	if err != nil {
		t.Fatal(err)
	}

	// Test if signature works.
	ok := Verify(kp.PublicKey, message, sig)
	if !ok {
		t.Fatal("signature verification failed with correct info")
	}

	ok = Verify(message, message, sig)
	if ok {
		t.Fatal("signature verification failed with wrong public key")
	}

	sig[0] = ^sig[0]
	ok = Verify(kp.PublicKey, message, sig)
	if ok {
		t.Fatal("invalid signature passed verification unexpectedly")
	}
}

func TestFromPrivateKey(t *testing.T) {
	kp1 := RandomKeyPair()
	kp2, err := FromPrivateKey(kp1.PrivateKeyHex())
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(kp1, kp2) {
		t.Fatal("kp1 and kp2 are not deep-equal.")
	}
}
