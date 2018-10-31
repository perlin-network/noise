package discovery

import (
	"encoding/hex"
	"github.com/perlin-network/noise/peer"
	"testing"

	"github.com/perlin-network/noise/crypto"
	"github.com/perlin-network/noise/crypto/ed25519"
)

func TestGenerateKeyPairAndID(t *testing.T) {
	t.Parallel()

	kp, id := GenerateKeyPairAndID("tcp://127.0.0.1:8000")

	t.Logf("%s %v", kp.PrivateKeyHex(), id)
}

func TestCheckHashedBytesPrefixLen(t *testing.T) {
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
		if tt.valid != checkHashedBytesPrefixLen(kp.PublicKey, tt.c1) {
			t.Errorf("isValidKeyPair expected %t, got: %t", tt.valid, !tt.valid)
		}
	}
}

func TestRandomBytes(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		length int
	}{
		{8},
		{16},
		{24},
		{32},
	}
	for _, tt := range testCases {
		randBytes, err := randomBytes(tt.length)
		if err != nil {
			t.Errorf("randomBytes() expected no error, got: %v", err)
		}
		if len(randBytes) != tt.length {
			t.Errorf("randomBytes() expected length to be %d, got: %d", tt.length, len(randBytes))
		}
	}
}

func TestCheckDynamicPuzzle(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		privateKeyHex string
		encodedX      string
		prefixLength  int
	}{
		{"2b56bb7556eaa58d2253d33b34d7ce869c54bb3c946164f6b73adc378cb9eccab37a3bf66608246c5791ebd19bd25169f6b243a6668c6635b0b4bc43474b6dbd", "4e68c698a810ab040232299591c4a902c15245efaae2ebeae34f45f9ca65f1b2", 8},
		{"88b0fcebc02d4f61927ab466e32a67872e2f474dfd7ca5a2286e43c30fc65c9165a8020be000cd7d4697487759b320f8711ea84b29af7f98faecc3490b2a73d6", "20d906ea8a8c6bae5dabecc9815ff826cdf2d73991ff731b8b9ddb90564b523b", 16},
		{"692116c1f969108bf48d6ea05cf08906da8ec1e4a832000f931204048d3d08c95e0ae842174bf34eddd8afbee04005751704f05a27893e15727a4d77de4eff9e", "b4369f03cfd2ac977f5d941e1459896d90cbaf979f243ed7eb24058e450d36d7", 16},
		{"692116c1f969108bf48d6ea05cf08906da8ec1e4a832000f931204048d3d08c95e0ae842174bf34eddd8afbee04005751704f05a27893e15727a4d77de4eff9e", "b4369f03cfd2ac977f5d941e1459896d90cbaf979f243ed7eb24058e450d36d7", 24},
	}
	for _, tt := range testCases {
		sp := ed25519.New()
		kp, err := crypto.FromPrivateKey(sp, tt.privateKeyHex)
		id := peer.CreateID("tcp://localhost:8000", kp.PublicKey)
		x, err := hex.DecodeString(tt.encodedX)
		if err != nil {
			t.Errorf("DecodeString() expected no error, got: %v", err)
		}
		ok := checkDynamicPuzzle(id.Id, x, tt.prefixLength)
		if !ok {
			t.Errorf("checkDynamicPuzzle() expected to be true")
		}
	}
}
