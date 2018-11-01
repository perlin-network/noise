package network

import (
	"bytes"
	"crypto/rand"
	"reflect"
	"testing"
	"time"

	"github.com/perlin-network/noise/internal/protobuf"
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

	ids := []protobuf.ID{
		protobuf.ID(peer.CreateID("tcp://127.0.0.1:3001", pk1)),
		protobuf.ID(peer.CreateID("tcp://127.0.0.1:3001", pk2)),
		protobuf.ID(peer.CreateID("tcp://127.0.0.1:3002", pk1)),
		protobuf.ID(peer.CreateID("tcp://127.0.0.1:3002", pk2)),
	}

	messages := [][]byte{
		[]byte("hello"),
		[]byte("world"),
	}

	outputs := make([][]byte, 0)

	for _, id := range ids {
		for _, msg := range messages {
			outputs = append(outputs, SerializeMessage(&id, msg))
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

func TestPrepareWeakSignature(t *testing.T) {
	id1 := protobuf.ID(peer.CreateID("tcp://127.0.0.1:3001", []byte("12341234123412341234123412341234")))
	id2 := protobuf.ID(peer.CreateID("tcp://127.0.0.1:3001", []byte("43214321432143214321432143214321")))
	id3 := protobuf.ID(peer.CreateID("tcp://127.0.0.1:3002", []byte("43214321432143214321432143214321")))

	now := time.Now()

	expiration1 := now.Add(60 * time.Second)
	expiration2 := now.Add(1 * time.Minute)
	b1 := PrepareWeakSignature(&id1, &expiration1)
	b2 := PrepareWeakSignature(&id2, &expiration2)
	b3 := PrepareWeakSignature(&id3, &expiration2)

	if !bytes.Equal(b1, b2) {
		t.Errorf("Equal() expected to be true")
	}
	if bytes.Equal(b2, b3) {
		t.Errorf("Equal() expected to be false")
	}
}

func TestFilterPeers(t *testing.T) {
	result := FilterPeers("tcp://10.0.0.3:3000", []string{
		"tcp://10.0.0.5:3000",
		"tcp://10.0.0.1:3000",
		"tcp://10.0.0.1:3000",
		"",
		"tcp://10.0.0.1:2000",
		"tcp://10.0.0.3:3000",
		"kcp://10.0.0.3:3000",
		"",
		"tcp://10.0.0.6:3000",
		"tcp://localhost:3004",
		"tcp://::1:3005",
	})

	expected := []string{
		"tcp://10.0.0.5:3000",
		"tcp://10.0.0.1:3000",
		// "tcp://10.0.0.1:3000" is a duplicate
		"tcp://10.0.0.1:2000",
		// "tcp://10.0.0.3:3000" is filtered
		"kcp://10.0.0.3:3000",
		"tcp://10.0.0.6:3000",
		"tcp://127.0.0.1:3004",
		// "tcp://::1:3005" will be removed
	}

	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("Unexpected got %v, but expected %v", result, expected)
	}
}
