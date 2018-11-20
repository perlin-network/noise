package skademlia

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"testing"

	"github.com/perlin-network/noise/crypto"
	"github.com/perlin-network/noise/crypto/blake2b"
	"github.com/perlin-network/noise/crypto/ed25519"
)

var (
	idBytes1 = []byte("12345678901234567890123456789012")
	idBytes2 = []byte("12345678901234567890123456789013")
)

func TestNewIdentityAdapter(t *testing.T) {
	t.Parallel()

	id := NewIdentityAdapter(DefaultC1, DefaultC2)
	if !VerifyPuzzle(id.MyIdentity(), id.id(), id.Nonce, DefaultC1, DefaultC2) {
		t.Errorf("GenerateKeyPairAndID() expected ID to be valid")
	}
}

func TestNewSKademliaIdentityFromKeypair(t *testing.T) {
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
		id, err := NewIdentityFromKeypair(kp, tt.c1, DefaultC2)
		if tt.valid {
			if err != nil {
				t.Errorf("NewIdentityFromKeypair() expected to be valid, got: %+v", err)
			} else if id == nil {
				t.Errorf("NewIdentityFromKeypair() expected to have non-nil id")
			}
		} else {
			if err == nil {
				t.Errorf("NewIdentityFromKeypair() expected an error")
			} else if id != nil {
				t.Errorf("NewIdentityFromKeypair() expected nil id")
			}
		}
	}
}

func TestSignAndVerify(t *testing.T) {
	t.Parallel()

	data, err := randomBytes(1024)
	if err != nil {
		t.Errorf("randBytes() expected no error, got %+v", err)
	}
	privateKeyHex := "1946e455ca6072bcdfd3182799c2ceb1557c2a56c5f810478ac0eb279ad4c93e8e8b6a97551342fd70ec03bea8bae5b05bc5dc0f54b2721dff76f06fab909263"
	sp := ed25519.New()
	kp, err := crypto.FromPrivateKey(sp, privateKeyHex)
	if err != nil {
		t.Errorf("FromPrivateKey() expected no error, got: %v", err)
	}
	id, err := NewIdentityFromKeypair(kp, DefaultC1, DefaultC2)
	if err != nil {
		t.Errorf("NewIdentityFromKeypair() expected to be valid, got: %+v", err)
	}
	sign := id.Sign([]byte(data))
	if len(sign) != id.SignatureSize() {
		t.Errorf("SignatureSize() expectec signature size to be %d, got %d", id.SignatureSize(), len(sign))
	}

	valid := id.Verify(kp.PublicKey, data, sign)
	if !valid {
		t.Errorf("Verify() expected to be true")
	}
}

func TestGenerateKeyPairAndID(t *testing.T) {
	t.Parallel()

	id, nonce := generateKeyPairAndNonce(DefaultC1, DefaultC2)
	if id == nil {
		t.Errorf("generateKeyPairAndNonce() expected id not to be nil")
	}
	if len(nonce) == 0 {
		t.Errorf("generateKeyPairAndNonce() expected nonce to not be nil")
	}
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
		b := blake2b.New()
		if tt.valid != checkHashedBytesPrefixLen(b.HashBytes(kp.PublicKey), tt.c1) {
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
		{"d4a4936c626e53af8d7db5585df855c3f845bf13480f5c18e8dbf228f9d2c56589632630a2c7069424e7fb46a7d9efc1e017f39f72eb119c3c9151edd11787b9", "421e8f9ebab772d12562e0908286ccaef7672640f340f714c0734819bb078c99", 8},
		{"ad2396db40d6207af844959e331aa43b18332c72393c6233ffeb74e8ec19dbd764a488cdfabff04f72cb4dc68e7b9132c19db0675427b69a25c0cb6267d9042d", "12a449a7900717a57b9679ae249ea372ded071b96c0587f292e7e78e56fefce7", 16},
		{"63d26a6e3d6966191e28e17aa36c401d8d36522dd6560a02c6f6c3dd046d035ed2d489b003b5189d5864f2cdeb3afc4f662ab0eb5a0c1e83b991657488ffc71c", "900c6d1e8520f9e7f9908fd9c707c8db81b946393eda40281f2db4425420708f", 16},
		{"63d26a6e3d6966191e28e17aa36c401d8d36522dd6560a02c6f6c3dd046d035ed2d489b003b5189d5864f2cdeb3afc4f662ab0eb5a0c1e83b991657488ffc71c", "900c6d1e8520f9e7f9908fd9c707c8db81b946393eda40281f2db4425420708f", 24},
	}
	for _, tt := range testCases {
		sp := ed25519.New()
		kp, err := crypto.FromPrivateKey(sp, tt.privateKeyHex)
		nodeID := blake2b.New().HashBytes(kp.PublicKey)
		x, err := hex.DecodeString(tt.encodedX)
		if err != nil {
			t.Errorf("DecodeString() expected no error, got: %v", err)
		}
		ok := checkDynamicPuzzle(nodeID, x, tt.prefixLength)
		if !ok {
			t.Errorf("checkDynamicPuzzle() expected to be true")
		}
	}
}

func TestVerifyPuzzle(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		privateKeyHex string
		encodedX      string
		valid         bool
	}{
		{"2b56bb7556eaa58d2253d33b34d7ce869c54bb3c946164f6b73adc378cb9eccab37a3bf66608246c5791ebd19bd25169f6b243a6668c6635b0b4bc43474b6dbd", "4e68c698a810ab040232299591c4a902c15245efaae2ebeae34f45f9ca65f1b2", false},
		{"c7147384a46a4e5714b0729019a489521199557143ade85e6e6540d90ac80c6578de0d25fdc274cdff7614dc457333fb7738e29f567e4865f453e2e57c180e67", "ee406641f2d17adb9f970a1bbc4e7a367b18092befc2ff84255941d5324ec584", true},
	}
	for _, tt := range testCases {
		sp := ed25519.New()
		kp, err := crypto.FromPrivateKey(sp, tt.privateKeyHex)
		id, err := NewIdentityFromKeypair(kp, DefaultC1, DefaultC2)
		if err != nil {
			t.Errorf("NewIdentityFromKeypair() expected no error, got: %+v", err)
		}
		nonce, err := hex.DecodeString(tt.encodedX)
		if err != nil {
			t.Errorf("DecodeString() expected no error, got: %v", err)
		}

		id, err = NewIdentityFromKeypair(kp, 16, 16)
		if err != nil {
			t.Errorf("NewIdentityFromKeypair() expected no error, got: %+v", err)
		}
		id.Nonce = nonce
		ok := VerifyPuzzle(id.MyIdentity(), id.id(), id.Nonce, 16, 16)
		if ok != tt.valid {
			t.Errorf("VerifyPuzzle() expected to be %t, got %t", tt.valid, ok)
		}
	}
}

func TestXor(t *testing.T) {
	t.Parallel()

	xorResult := []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}

	result := xor(idBytes1, idBytes2)

	if !bytes.Equal(xorResult, result) {
		t.Errorf("Xor() = %v, want %v", xorResult, result)
	}
}

func TestPrefixLen(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		publicKeyHash uint32
		expected      int
	}{
		{1, 7},
		{2, 6},
		{4, 5},
		{8, 4},
		{16, 3},
		{32, 2},
		{64, 1},
	}
	for _, tt := range testCases {
		publicKey := make([]byte, 4)
		binary.LittleEndian.PutUint32(publicKey, tt.publicKeyHash)
		if prefixLen(publicKey) != tt.expected {
			t.Errorf("PrefixLen() expected: %d, value: %d", tt.expected, prefixLen(publicKey))
		}
	}
}
