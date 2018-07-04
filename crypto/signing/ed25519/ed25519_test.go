package ed25519

import (
	"testing"
	"reflect"
	"github.com/perlin-network/noise/crypto"
)

func TestFromPrivateKey(t *testing.T) {
	sp := New()
	kp1 := RandomKeyPair()

	kp2, err := crypto.FromPrivateKey(sp, kp1.PrivateKeyHex())
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(kp1, kp2) {
		t.Fatal("kp1 and kp2 are not deep-equal.")
	}
}
