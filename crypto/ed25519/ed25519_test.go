package ed25519

import (
	"reflect"
	"testing"
)

func TestEd25519(t *testing.T) {
	t.Parallel()
	p := NewEd25519()

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
	sig := p.Sign(privateKey, message)
	// length of signature should not be 0
	if len(sig) == 0 {
		t.Errorf("Sign(%s) message length is 0", message)
	}
	// correct message should pass verify check
	if verify := p.Verify(publicKey, message, sig); !verify {
		t.Errorf("Verify(%s, %b) = %v, want true", message, sig, verify)
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
