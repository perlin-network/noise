package blake2b

import (
	"bytes"
	"crypto/rand"
	"math/big"
	"testing"

	"github.com/perlin-network/noise/crypto"
)

func BenchmarkHash(b *testing.B) {
	hp := New()

	message := make([]byte, 64)
	_, err := rand.Read(message)
	if err != nil {
		panic(err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		hp.HashBytes(message)
	}
}

func TestHash(t *testing.T) {
	t.Parallel()
	hp := New()

	r := crypto.Hash(hp, big.NewInt(123))

	n := new(big.Int)
	n, ok := n.SetString("89391711502145780362310349925943903708999319576398061903082165979787487688967", 10)
	if !ok {
		t.Errorf("big int error")
	}
	if n.String() != r.String() {
		t.Errorf("String() n = %v, want %v", n, r)
	}
}

func TestHashBytes(t *testing.T) {
	t.Parallel()
	hp := New()

	r := hp.HashBytes([]byte("123"))

	n := []byte{245, 214, 123, 174, 115, 176, 225, 13, 13, 253, 48, 67, 179, 244, 241, 0, 173, 160, 20, 197, 195, 123, 213, 206, 151, 129, 59, 19, 245, 171, 43, 207}
	if !bytes.Equal(n, r) {
		t.Errorf("Equal() n = %v, want %v", n, r)
	}
}
