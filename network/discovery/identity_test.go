package discovery

import (
	"testing"

	"github.com/perlin-network/noise/crypto"
	"github.com/perlin-network/noise/crypto/ed25519"
)

func TestGenerateKeyPairAndID(t *testing.T) {
	t.Parallel()

	kp, id := GenerateKeyPairAndID("tcp://127.0.0.1:8000")

	t.Logf("%s %v", kp.PrivateKeyHex(), id)
}

func TestIsValidKeyPair(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		privateKeyHex string
		c1            int
		valid         bool
	}{
		{"078e11ac002673b20922a777d827a68191163fa87ce897f55be672a508b5c5a017246e17eb3aa6d3eed0150044d426e899525665b86574f11dbcf150ac65a988", 8, true},
		{"1946e455ca6072bcdfd3182799c2ceb1557c2a56c5f810478ac0eb279ad4c93e8e8b6a97551342fd70ec03bea8bae5b05bc5dc0f54b2721dff76f06fab909263", 16, true},
		{"1946e455ca6072bcdfd3182799c2ceb1557c2a56c5f810478ac0eb279ad4c93e8e8b6a97551342fd70ec03bea8bae5b05bc5dc0f54b2721dff76f06fab909263", 10, true},
		{"078e11ac002673b20922a777d827a68191163fa87ce897f55be672a508b5c5a017246e17eb3aa6d3eed0150044d426e899525665b86574f11dbcf150ac65a988", 16, false},
	}
	for _, tt := range testCases {
		sp := ed25519.New()
		kp, err := crypto.FromPrivateKey(sp, tt.privateKeyHex)
		if err != nil {
			t.Errorf("FromPrivateKey() expected no error, got: %v", err)
		}
		if tt.valid != isValidKeyPair(kp.PublicKey, tt.c1) {
			t.Errorf("isValidKeyPair expected %t, got: %t", tt.valid, !tt.valid)
		}
	}
}
