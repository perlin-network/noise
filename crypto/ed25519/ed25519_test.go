package ed25519

import (
	"crypto/rand"
	"reflect"
	"testing"
)

func BenchmarkSign(b *testing.B) {
	p := New()
	privateKey, _, err := p.GenerateKeys()
	if err != nil {
		panic(err)
	}

	message := make([]byte, 32)
	_, err = rand.Read(message)
	if err != nil {
		panic(err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		sig := p.Sign(privateKey, message)
		if len(sig) == 0 {
			panic("signing failed")
		}
	}
}

func BenchmarkVerify(b *testing.B) {
	p := New()
	privateKey, publicKey, err := p.GenerateKeys()
	if err != nil {
		panic(err)
	}

	message := make([]byte, 32)
	_, err = rand.Read(message)
	if err != nil {
		panic(err)
	}

	b.ResetTimer()

	sig := p.Sign(privateKey, message)

	for i := 0; i < b.N; i++ {
		ok := p.Verify(publicKey, message, sig)
		if !ok {
			panic("verification failed")
		}
	}
}

func TestEd25519(t *testing.T) {
	t.Parallel()
	p := New()

	privateKey, publicKey, err := p.GenerateKeys()
	if err != nil {
		t.Errorf("GenerateKeys() = %v, want <nil>", err)
	}
	if len(privateKey) != p.PrivateKeySize() {
		t.Errorf("PrivateKeySize() = %d, want %d", len(privateKey), p.PrivateKeySize())
	}
	if len(publicKey) != p.PublicKeySize() {
		t.Errorf("PublicKeySize() = %d, want %d", len(publicKey), p.PublicKeySize())
	}

	message := []byte("test message")
	// sign with a bad key should have yield signature with 0 length
	sig := p.Sign([]byte("bad key"), message)
	if len(sig) != 0 {
		t.Errorf("Sign(%s) message length should be 0", message)
	}

	// length of signature should not be 0
	sig = p.Sign(privateKey, message)
	if len(sig) == 0 {
		t.Errorf("Sign(%s) message length is 0", message)
	}

	// correct message should pass verify check
	if verify := p.Verify(publicKey, message, sig); !verify {
		t.Errorf("Verify(%s, %b) = %v, want true", message, sig, verify)
	}

	// wrong public key should fail verify check
	if verify := p.Verify([]byte("bad key"), message, sig); verify {
		t.Errorf("Verify(%s, %b) = %v, want false", message, sig, verify)
	}

	// wrong message should fail verify check
	wrongMessage := []byte("wrong message")
	if verify := p.Verify(publicKey, wrongMessage, sig); verify {
		t.Errorf("Verify(%s, %b) = %v, want false", wrongMessage, sig, verify)
	}

	publicKeyCheck, err := p.PrivateToPublic(privateKey)
	if err != nil {
		t.Errorf("privateToPublic() = %v, want <nil>", err)
	}
	if !reflect.DeepEqual(publicKeyCheck, publicKey) {
		t.Errorf("PrivateToPublic() = %v, want %v", publicKeyCheck, publicKey)
	}
}

func TestRandomKeyPair(t *testing.T) {
	t.Parallel()

	kp := New().RandomKeyPair()
	if len(kp.PrivateKey) == 0 {
		t.Errorf("private key length should not be 0")
	}
	if len(kp.PublicKey) == 0 {
		t.Errorf("public key length should not be 0")
	}
}
