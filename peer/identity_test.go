package peer

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/perlin-network/noise/peer"
)

func TestID(t *testing.T) {

	testPublicKey := []byte("12345678901234567890123456789012")
	testPublicKey1 := []byte("12345678901234567890123456789011")
	testPublicKey2 := []byte("12345678901234567890123456789013")
	testAddr := "localhost:12345"

	id := peer.CreateID(testAddr, testPublicKey)

	if !bytes.Equal(id.PublicKey, testPublicKey) {
		fmt.Printf("%s \n%s", id.PublicKey, testPublicKey)
		panic("Wrong Public Key")
	}

	if id.Address != testAddr {
		panic("Wrong Address")
	}

	if id.String() != "ID{PublicKey: [49 50 51 52 53 54 55 56 57 48 49 50 51 52 53 54 55 56 57 48 49 50 51 52 53 54 55 56 57 48 49 50], Address: localhost:12345}" {
		fmt.Printf(id.String())
		panic("String() error")
	}

	if !id.Equals(peer.CreateID(testAddr, testPublicKey)) {
		panic("Equals() error")
	}

	if id.Less(peer.CreateID(testAddr, testPublicKey1)) {
		panic("Less() error 1")
	}

	if !id.Less(peer.CreateID(testAddr, testPublicKey2)) {
		panic("Less() error 2")
	}

	if id.PublicKeyHex() != "3132333435363738393031323334353637383930313233343536373839303132" {
		fmt.Print(id.PublicKeyHex())
		panic("PublicKeyHex() error or hex.EncodeToString() changed defination?")
	}

	comparee := peer.CreateID(
		testAddr,
		[]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
	)

	if !comparee.Equals(id.Xor(peer.CreateID(testAddr, testPublicKey2))) {
		fmt.Printf("%v\n%v", comparee, id.Xor(peer.CreateID(testAddr, testPublicKey2)))
		panic("Xor() error")
	}

}
