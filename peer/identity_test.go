package peer

import (
	"bytes"
	"testing"
)

var (
	testPublicKey  = []byte("12345678901234567890123456789012")
	testPublicKey1 = []byte("12345678901234567890123456789011")
	testPublicKey2 = []byte("12345678901234567890123456789013")
	testAddr       = "localhost:12345"

	id = CreateID(testAddr, testPublicKey)
)

func TestIDEqual(t *testing.T) {
	if !bytes.Equal(id.PublicKey, testPublicKey) {
		t.Fatalf("wrong public key: %s != %s", id.PublicKey, testPublicKey)
	}
	if id.Address != testAddr {
		t.Fatalf("wrong address: %s != %s", id.Address, testAddr)
	}
}

func TestIDString(t *testing.T) {
	if id.String() != "ID{PublicKey: [49 50 51 52 53 54 55 56 57 48 49 50 51 52 53 54 55 56 57 48 49 50 51 52 53 54 55 56 57 48 49 50], Address: localhost:12345}" {
		t.Fatalf("string() error: %s", id.String())
	}
}

func TestIDEquals(t *testing.T) {
	if !id.Equals(CreateID(testAddr, testPublicKey)) {
		t.Fatal("equals() error")
	}
}

func TestIDLess(t *testing.T) {
	if id.Less(CreateID(testAddr, testPublicKey1)) {
		t.Fatal("less() error 1")
	}

	if !id.Less(CreateID(testAddr, testPublicKey2)) {
		t.Fatal("less() error 2")
	}
}

func TestIDPublicKeyHex(t *testing.T) {
	if id.PublicKeyHex() != "3132333435363738393031323334353637383930313233343536373839303132" {
		t.Fatalf("publickeyhex() error or hex.encodetostring() changed definition? value: %v", id.PublicKeyHex())
	}
}

func TestIDXor(t *testing.T) {
	xor := CreateID(
		testAddr,
		[]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
	)

	if !xor.Equals(id.Xor(CreateID(testAddr, testPublicKey2))) {
		t.Fatalf("xor() error : %v != %v", xor, id.Xor(CreateID(testAddr, testPublicKey2)))
	}

}
