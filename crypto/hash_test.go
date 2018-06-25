package crypto

import (
	"bytes"
	"math/big"
	"testing"
)

func TestHash(t *testing.T) {
	r := Hash(big.NewInt(123))

	n := new(big.Int)
	n, ok := n.SetString("89391711502145780362310349925943903708999319576398061903082165979787487688967", 10)
	if ok {
		if n.String() != r.String() {
			t.Fatalf("hash error %v %v", r, n)
		}
	} else {
		t.Fatal("big int error")
	}
}

func TestHashBytes(t *testing.T) {
	r := HashBytes([]byte("123"))
	n := []byte{245, 214, 123, 174, 115, 176, 225, 13, 13, 253, 48, 67, 179, 244, 241, 0, 173, 160, 20, 197, 195, 123, 213, 206, 151, 129, 59, 19, 245, 171, 43, 207}
	if !bytes.Equal(n, r) {
		t.Fatalf("hash error : %v %v", r, n)
	}
}
