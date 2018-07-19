package crypto_test

import (
	"bytes"
	"encoding/hex"
	"reflect"
	"testing"

	"github.com/perlin-network/noise/crypto"
	"github.com/perlin-network/noise/crypto/mocks"

	gomock "github.com/golang/mock/gomock"
)

var (
	privateKey    []byte
	privateKeyHex string
	publicKey     []byte
	publicKeyHex  string
	message       []byte
	hashed        []byte
	signature     []byte
)

func init() {
	// mock inputs
	privateKey = []byte("1234567890")
	privateKeyHex = "31323334353637383930"
	publicKey = []byte("12345678901234567890")
	publicKeyHex = "3132333435363738393031323334353637383930"

	message = []byte("test message")
	hashed = []byte("hashed test message")
	signature = []byte("signed test message")
}

func TestKeyPair(t *testing.T) {
	t.Parallel()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	sp := mocks.NewMockSignaturePolicy(mockCtrl)
	hp := mocks.NewMockHashPolicy(mockCtrl)

	// setup expected mock return values
	sp.EXPECT().PrivateKeySize().Return(len(privateKey)).AnyTimes()
	sp.EXPECT().PublicKeySize().Return(len(publicKey)).AnyTimes()
	sp.EXPECT().Sign(privateKey, hashed).Return(signature).AnyTimes()
	sp.EXPECT().Verify(publicKey, hashed, signature).Return(true).Times(1)

	hp.EXPECT().HashBytes(message).Return(hashed).AnyTimes()

	kp := crypto.KeyPair{
		PrivateKey: privateKey,
		PublicKey:  publicKey,
	}

	sig, err := kp.Sign(sp, hp, message)
	if err != nil {
		t.Errorf("Sign() = %v, expected <nil>", err)
	}
	if !bytes.Equal(sig, []byte("signed test message")) {
		t.Errorf("Sign() = '%s', expected '%s'", sig, []byte("signed test message"))
	}

	if !crypto.Verify(sp, hp, publicKey, message, signature) {
		t.Errorf("Verify('%s', '%s') = false, expected true", message, signature)
	}

	if kp.PrivateKeyHex() != privateKeyHex {
		t.Errorf("PrivateKeyHex() = %s, want %s", kp.PrivateKeyHex(), privateKeyHex)
	}

	if kp.PublicKeyHex() != publicKeyHex {
		t.Errorf("PublicKeyHex() = %s, want %s", kp.PublicKeyHex(), publicKeyHex)
	}

	privateKeyHexCheck, publicKeyHexCheck := kp.String()
	if privateKeyHexCheck != privateKeyHex || publicKeyHexCheck != publicKeyHex {
		t.Errorf("String() = (%s, %s), want (%s, %s)", privateKeyHexCheck, privateKeyHex, publicKeyHexCheck, publicKeyHex)
	}
}

func TestKeyPairErrors(t *testing.T) {
	t.Parallel()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	sp := mocks.NewMockSignaturePolicy(mockCtrl)
	hp := mocks.NewMockHashPolicy(mockCtrl)

	sp.EXPECT().PrivateKeySize().Return(100).Times(2)
	sp.EXPECT().PublicKeySize().Return(100).Times(1)
	sp.EXPECT().Sign(privateKey, hashed).Return(signature).AnyTimes()

	hp.EXPECT().HashBytes(message).Return(hashed).AnyTimes()

	kp := crypto.KeyPair{
		PrivateKey: []byte{},
		PublicKey:  []byte{},
	}

	// private key size does not match signature size
	_, err := kp.Sign(sp, hp, message)
	if err != crypto.PrivateKeySizeErr {
		t.Errorf("Sign() = %v, expected %v", err, crypto.PrivateKeySizeErr)
	}

	// private key is a bad format
	errorString := "encoding/hex: invalid byte: U+0020 ' '"
	_, err = crypto.FromPrivateKey(sp, "bad key")
	if err.Error() != errorString {
		t.Errorf("FromPrivateKey() = %v, expected %s", err, errorString)
	}

	// private key size does not match signature size
	_, err = crypto.FromPrivateKey(sp, string(privateKey))
	if err != crypto.PrivateKeySizeErr {
		t.Errorf("FromPrivateKey() = %v, expected %v", err, crypto.PrivateKeySizeErr)
	}

	// public key size does not match signature size
	if crypto.Verify(sp, hp, []byte{}, message, signature) {
		t.Errorf("Verify() = true, expected false")
	}
}

func TestFromPrivateKey(t *testing.T) {
	t.Parallel()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	sp := mocks.NewMockSignaturePolicy(mockCtrl)

	// mock inputs
	privateKeyHexBytes, _ := hex.DecodeString(string(privateKey))

	// setup expected mock return values
	sp.EXPECT().PrivateKeySize().Return(len(privateKeyHexBytes)).Times(1)
	sp.EXPECT().PrivateToPublic(privateKeyHexBytes).Return(publicKey, nil).Times(1)

	kp1 := &crypto.KeyPair{
		PrivateKey: privateKeyHexBytes,
		PublicKey:  publicKey,
	}

	kp2, err := crypto.FromPrivateKey(sp, string(privateKey))
	if err != nil {
		t.Errorf("FromPrivateKey() = %v, expected <nil>", err)
	}

	// assert that NewKeyPair matches FromPrivateKey
	if !reflect.DeepEqual(kp1, kp2) {
		t.Errorf("expected keypair %+v = %+v", kp1, kp2)
	}
}
