package peer

import (
	"bytes"
	"testing"
)

func TestID(t *testing.T) {
	var (
		publicKey1 = []byte("12345678901234567890123456789012")
		publicKey2 = []byte("12345678901234567890123456789011")
		publicKey3 = []byte("12345678901234567890123456789013")
		address    = "localhost:12345"

		id1 = CreateID(address, publicKey1)
		id2 = CreateID(address, publicKey2)
		id3 = CreateID(address, publicKey3)
	)

	t.Run("CreateID()", func(t *testing.T) {
		if !bytes.Equal(id1.PublicKey, publicKey1) {
			t.Fatalf("wrong public key: %s != %s", id1.PublicKey, publicKey1)
		}
		if id1.Address != address {
			t.Fatalf("wrong address: %s != %s", id1.Address, address)
		}
	})

	t.Run("String()", func(t *testing.T) {
		if id1.String() != "ID{PublicKey: [49 50 51 52 53 54 55 56 57 48 49 50 51 52 53 54 55 56 57 48 49 50 51 52 53 54 55 56 57 48 49 50], Address: localhost:12345}" {
			t.Fatalf("string() error: %s", id1.String())
		}
	})

	t.Run("Equals()", func(t *testing.T) {
		if !id1.Equals(CreateID(address, publicKey1)) {
			t.Fatalf("%s != %s", id1.PublicKeyHex(), id2.PublicKeyHex())
		}
	})

	t.Run("Less()", func(t *testing.T) {
		if id1.Less(id2) {
			t.Fatalf("%s < %s", id1.PublicKeyHex(), id2.PublicKeyHex())
		}

		if !id1.Less(id3) {
			t.Fatalf("%s >= %s", id1.PublicKeyHex(), id3.PublicKeyHex())
		}
	})

	t.Run("PublicKeyHex()", func(t *testing.T) {
		if id1.PublicKeyHex() != "3132333435363738393031323334353637383930313233343536373839303132" {
			t.Fatalf("publickeyhex() error or hex.encodetostring() changed definition? value: %v", id1.PublicKeyHex())
		}
	})

	t.Run("Xor()", func(t *testing.T) {
		xor := CreateID(
			address,
			[]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		)

		result := id1.Xor(id3)

		if !xor.Equals(result) {
			t.Fatalf("%v != %v", xor, result)
		}
	})
}