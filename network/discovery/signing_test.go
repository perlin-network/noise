package discovery

import (
	"bytes"
	"crypto/rand"
	"testing"
	"time"

	"github.com/perlin-network/noise/peer"
)

func TestSerializeMessageInfoForSigning(t *testing.T) {
	mustReadRand := func(size int) []byte {
		out := make([]byte, size)
		_, err := rand.Read(out)
		if err != nil {
			panic(err)
		}
		return out
	}

	pk1, pk2 := mustReadRand(32), mustReadRand(32)

	ids := []peer.ID{
		peer.CreateID("tcp://127.0.0.1:3001", pk1),
		peer.CreateID("tcp://127.0.0.1:3001", pk2),
		peer.CreateID("tcp://127.0.0.1:3002", pk1),
		peer.CreateID("tcp://127.0.0.1:3002", pk2),
	}

	messages := [][]byte{
		[]byte("hello"),
		[]byte("world"),
	}

	outputs := make([][]byte, 0)

	for _, id := range ids {
		for _, msg := range messages {
			outputs = append(outputs, serializeMessage(&id, msg))
		}
	}

	for i := 0; i < len(outputs); i++ {
		for j := i + 1; j < len(outputs); j++ {
			if bytes.Equal(outputs[i], outputs[j]) {
				t.Fatal("Different inputs produced the same output")
			}
		}
	}
}

func TestSerializePeerIDAndExpiration(t *testing.T) {
	id1 := peer.CreateID("tcp://127.0.0.1:3001", []byte("12341234123412341234123412341234"))
	id2 := peer.CreateID("tcp://127.0.0.1:3001", []byte("43214321432143214321432143214321"))
	id3 := peer.CreateID("tcp://127.0.0.1:3002", []byte("43214321432143214321432143214321"))

	now := time.Now()

	expiration1 := now.Add(60 * time.Second)
	expiration2 := now.Add(1 * time.Minute)
	b1 := serializePeerIDAndExpiration(&id1, &expiration1)
	b2 := serializePeerIDAndExpiration(&id2, &expiration2)
	b3 := serializePeerIDAndExpiration(&id3, &expiration2)

	if !bytes.Equal(b1, b2) {
		t.Errorf("Equal() expected to be true")
	}
	if bytes.Equal(b2, b3) {
		t.Errorf("Equal() expected to be false")
	}
}
