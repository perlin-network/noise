package skademlia

import (
	"encoding/hex"
	"fmt"
	"github.com/perlin-network/noise/identity/ed25519"
	"github.com/perlin-network/noise/internal/edwards25519"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/blake2b"
	"testing"
)

func TestNewIdentityRandom(t *testing.T) {
	t.Parallel()

	keys := NewKeys()

	assert.NotNil(t, keys)
	assert.True(t, VerifyPuzzle(keys.PublicKey(), keys.ID(), keys.Nonce, DefaultC1, DefaultC2))
}

func TestNewSKademliaIdentityFromPrivateKey(t *testing.T) {
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
	for i, tt := range testCases {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			privateKey, err := hex.DecodeString(tt.privateKeyHex)
			assert.Nil(t, err)

			keys, err := LoadKeys(privateKey, tt.c1, 16)

			if tt.valid {
				assert.NoError(t, err)
				assert.NotNil(t, keys)
			} else {
				assert.Error(t, err)
				assert.Nil(t, keys)
			}
		})
	}
}

func TestSignAndVerify(t *testing.T) {
	t.Parallel()

	data, err := randomBytes(1024)
	assert.NoError(t, err)

	privateKeyHex := "1946e455ca6072bcdfd3182799c2ceb1557c2a56c5f810478ac0eb279ad4c93e8e8b6a97551342fd70ec03bea8bae5b05bc5dc0f54b2721dff76f06fab909263"

	privateKey, err := hex.DecodeString(privateKeyHex)
	assert.NoError(t, err)

	keys, err := LoadKeys(privateKey, DefaultC1, DefaultC2)
	assert.NoError(t, err)
	assert.NotNil(t, keys)

	signature, err := keys.Sign([]byte(data))

	assert.NoError(t, err)
	assert.Len(t, signature, edwards25519.SignatureSize)

	assert.Nil(t, keys.Verify(keys.publicKey, data, signature))
}

func TestGenerateKeyPairAndID(t *testing.T) {
	t.Parallel()

	c1 := 8
	c2 := 8

	keys := RandomKeys(c1, c2)
	assert.NotNil(t, keys)

	// Check if we can validate correct nonces.
	assert.True(t, checkHashedBytesPrefixLen(keys.ID(), c1))
	nonce := generateNonce(keys.ID(), c2)
	assert.True(t, len(nonce) > 0)

	// Check what happens if we generate the nonce with an invalid prefix length.
	assert.False(t, checkHashedBytesPrefixLen(keys.ID(), c1*4))
	nonce = generateNonce(keys.ID(), c2*4)
	assert.Equal(t, 0, len(nonce))
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

	for i, tt := range testCases {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			buf, err := hex.DecodeString(tt.privateKeyHex)
			assert.NoError(t, err)

			keys := ed25519.LoadKeys(buf)
			id := blake2b.Sum256(keys.PublicKey())

			assert.Equal(t, tt.valid, checkHashedBytesPrefixLen(id[:], tt.c1))
		})
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
	for i, tt := range testCases {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			randBytes, err := randomBytes(tt.length)
			assert.Nil(t, err)
			assert.Equal(t, tt.length, len(randBytes))
		})
	}
}

func TestCheckDynamicPuzzle(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		privateKeyHex string
		nonceHex      string
		prefixLength  int
	}{
		{"d4a4936c626e53af8d7db5585df855c3f845bf13480f5c18e8dbf228f9d2c56589632630a2c7069424e7fb46a7d9efc1e017f39f72eb119c3c9151edd11787b9", "421e8f9ebab772d12562e0908286ccaef7672640f340f714c0734819bb078c99", 8},
		{"ad2396db40d6207af844959e331aa43b18332c72393c6233ffeb74e8ec19dbd764a488cdfabff04f72cb4dc68e7b9132c19db0675427b69a25c0cb6267d9042d", "12a449a7900717a57b9679ae249ea372ded071b96c0587f292e7e78e56fefce7", 16},
		{"63d26a6e3d6966191e28e17aa36c401d8d36522dd6560a02c6f6c3dd046d035ed2d489b003b5189d5864f2cdeb3afc4f662ab0eb5a0c1e83b991657488ffc71c", "900c6d1e8520f9e7f9908fd9c707c8db81b946393eda40281f2db4425420708f", 16},
		{"63d26a6e3d6966191e28e17aa36c401d8d36522dd6560a02c6f6c3dd046d035ed2d489b003b5189d5864f2cdeb3afc4f662ab0eb5a0c1e83b991657488ffc71c", "900c6d1e8520f9e7f9908fd9c707c8db81b946393eda40281f2db4425420708f", 24},
	}
	for i, tt := range testCases {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			buf, err := hex.DecodeString(tt.privateKeyHex)
			assert.NoError(t, err)

			keys := ed25519.LoadKeys(buf)
			id := blake2b.Sum256(keys.PublicKey())

			nonce, err := hex.DecodeString(tt.nonceHex)
			assert.Nil(t, err)
			assert.True(t, checkDynamicPuzzle(id[:], nonce, tt.prefixLength))
		})
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
	for i, tt := range testCases {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			privateKey, err := hex.DecodeString(tt.privateKeyHex)
			assert.Nil(t, err)

			keys, err := LoadKeys(privateKey, 16, 16)
			assert.NoError(t, err)
			assert.NotNil(t, keys)

			nonce, err := hex.DecodeString(tt.encodedX)
			assert.NoError(t, err)

			assert.Equal(t, tt.valid, VerifyPuzzle(keys.PublicKey(), keys.ID(), nonce, 16, 16))
		})
	}
}
