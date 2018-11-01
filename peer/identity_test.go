package peer

import (
	"bytes"
	"encoding/binary"
	"github.com/stretchr/testify/assert"
	"testing"

	"github.com/perlin-network/noise/crypto/blake2b"
)

var (
	publicKey1 = []byte("12345678901234567890123456789012")
	publicKey2 = []byte("12345678901234567890123456789011")
	publicKey3 = []byte("12345678901234567890123456789013")
	address    = "localhost:12345"

	id1 = CreateID(address, publicKey1)
	id2 = CreateID(address, publicKey2)
	id3 = CreateID(address, publicKey3)
)

func TestCreateID(t *testing.T) {
	t.Parallel()

	if !bytes.Equal(id1.GetID(), blake2b.New().HashBytes(publicKey1)) {
		t.Errorf("PublicKey = %s, want %s", id1.GetID(), publicKey1)
	}
	if id1.GetAddress() != address {
		t.Errorf("Address = %s, want %s", id1.GetAddress(), address)
	}
}

func TestString(t *testing.T) {
	t.Parallel()

	want := "ID{Address: localhost:12345, Id: [73 44 127 92 143 18 83 102 101 246 108 105 60 227 86 107 128 15 61 7 191 108 178 184 1 152 19 41 78 16 131 58]}"

	if id1.String() != want {
		t.Errorf("String() = %s, want %s", id1.String(), want)
	}
}

func TestEquals(t *testing.T) {
	t.Parallel()

	if !id1.Equals(CreateID(address, publicKey1)) {
		t.Errorf("Equals() = %s, want %s", id1.PublicKeyHex(), id2.PublicKeyHex())
	}
}

func TestLess(t *testing.T) {
	t.Parallel()

	if id2.Less(id1) {
		t.Errorf("'%s'.Less(%s) should be true", id2.PublicKeyHex(), id1.PublicKeyHex())
	}

	if !id1.Less(id2) {
		t.Errorf("'%s'.Less(%s) should be false", id1.PublicKeyHex(), id2.PublicKeyHex())
	}

	if !id1.Less(id3) {
		t.Errorf("'%s'.Less(%s) should be false", id1.PublicKeyHex(), id3.PublicKeyHex())
	}
}

func TestPublicKeyHex(t *testing.T) {
	t.Parallel()

	want := "3132333435363738393031323334353637383930313233343536373839303132"
	if id1.PublicKeyHex() != want {
		t.Errorf("PublicKeyHex() = %s, want %s", id1.PublicKeyHex(), want)
	}
}

func TestXorId(t *testing.T) {
	t.Parallel()

	publicKey1Hash := blake2b.New().HashBytes(publicKey1)
	publicKey3Hash := blake2b.New().HashBytes(publicKey3)
	newID := make([]byte, len(publicKey3Hash))
	for i, b := range publicKey1Hash {
		newID[i] = b ^ publicKey3Hash[i]
	}

	xor := ID{
		Address: address,
		Id:      newID,
	}

	result := id1.XorID(id3)

	if !xor.Equals(result) {
		t.Errorf("Xor() = %v, want %v", xor, result)
	}
}

func TestXor(t *testing.T) {
	t.Parallel()

	xor := ID{
		Address:   address,
		PublicKey: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
	}

	result := id1.Xor(id3)

	if !xor.Equals(result) {
		t.Errorf("Xor() = %v, want %v", xor, result)
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
		id := ID{Address: address, Id: publicKey}
		if id.PrefixLen() != tt.expected {
			t.Errorf("PrefixLen() expected: %d, value: %d", tt.expected, id.PrefixLen())
		}
	}
}

func TestWtihValue(t *testing.T) {
	t.Parallel()
	key := "test"
	value := []byte("testvalue")

	val := id1.Value(key)
	if val != nil {
		t.Errorf("Value(\"%s\") expected to be nil, got %v", key, val)
	}

	id1Val := id1.WithValue(key, value)
	// id1Val should be a copy of id1 with new metadata
	val = id1Val.Value(key)
	if !bytes.Equal(val.([]byte), value) {
		t.Errorf("Value(\"%s\") expected to be %v, got %v", key, value, val)
	}
	val = id1.Value(key)
	if val != nil {
		t.Errorf("Value(\"%s\") expected to be nil, got %v", key, val)
	}

	assert.Panics(t, func() {
		id1.WithValue("", value)
	}, "empty string should panic")

	assert.Panics(t, func() {
		var nilString string
		id1.WithValue(nilString, value)
	}, "nil string should panic")
}

func TestNonce(t *testing.T) {
	t.Parallel()

	expected := []byte("mynoncevalue")
	idWithNonce := WithNonce(id1, expected)
	val := GetNonce(id1)
	assert.NotEqual(t, nil, val, "expected nil nonce")
	val = GetNonce(idWithNonce)
	assert.Equal(t, expected, val, "expected nonce to be found and equal")
}
