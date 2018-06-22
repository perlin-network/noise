package crypto

import (
	"testing"
	"crypto/rand"
	"reflect"
)

func TestSignVerify(t *testing.T) {
	kp := RandomKeyPair()
	message := make([]byte, 120)

	_, err := rand.Read(message)
	if err != nil {
		panic(err)
	}

	sig, err := kp.Sign(message)
	if err != nil {
		panic(err)
	}

	ok := Verify(kp.PublicKey, message, sig)
	if !ok {
		panic("Signature verification failed")
	}

	sig[0] = ^sig[0]
	ok = Verify(kp.PublicKey, message, sig)
	if ok {
		panic("Invalid signature passed verification unexpectedly")
	}
}

func TestFromPrivateKey(t *testing.T) {
	kp1 := RandomKeyPair()
	kp2, err := FromPrivateKey(kp1.PrivateKeyHex())
	if err != nil {
		panic(err)
	}

	if !reflect.DeepEqual(kp1, kp2) {
		panic("kp1 and kp2 are not deep-equal.")
	}
}
