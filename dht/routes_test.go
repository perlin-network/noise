package dht

import (
	"bytes"
	"fmt"
	"strconv"
	"testing"

	"github.com/perlin-network/noise/crypto"
	"github.com/perlin-network/noise/dht"
	"github.com/perlin-network/noise/peer"
)

var (
	keys               = crypto.RandomKeyPair()
	host               = "127.0.0.1"
	port               = 12345
	expectedBucketSize = 20
)

func TestBucketSize(t *testing.T) {
	if dht.BucketSize != expectedBucketSize {
		t.Fatalf("bucket size is expected %d but found %d", expectedBucketSize, dht.BucketSize)
	}
}

func TestCreateRoutingTable(t *testing.T) {

	id := peer.CreateID(host+":"+strconv.Itoa(port), keys.PublicKey)
	routes := dht.CreateRoutingTable(id)
	if routes.Self().Address != fmt.Sprintf("%s:%d", host, port) {
		t.Fatalf("wrong address: %s", routes.Self().Address)
	}
	if !bytes.Equal(routes.Self().PublicKey, keys.PublicKey) {
		t.Fatalf("wrong public key: %v", routes.Self().PublicKey)
	}
}

//

func TestPeerExists(t *testing.T) {

	id1 := peer.CreateID(host+":"+strconv.Itoa(port), keys.PublicKey)
	routes := dht.CreateRoutingTable(id1)
	if !routes.PeerExists(id1) {
		t.Fatal("peerexists() failed")
	}
}
func TestGetPeers(t *testing.T) {

	id1 := peer.CreateID(host+":"+strconv.Itoa(port), keys.PublicKey)
	//id2
	routes := dht.CreateRoutingTable(id1)

	peer := routes.GetPeers()
	fmt.Printf("%v", peer)

	// id2 := peer.CreateID(host+":"+strconv.Itoa(port+1), keys.PublicKey)
	// id3 := peer.CreateID(host+":"+strconv.Itoa(port+2), keys.PublicKey)

	// routes.Update(id2)
	// routes.Update(id3)
	// bucketID := id2.Xor(id1).PrefixLen()
	// bucket := routes.Bucket(bucketID)
	// fmt.Printf("%v", bucket.List.Len())

	// fmt.Printf("%v", bucket.List.Front())

	// fmt.Printf("%v", bucket.List.Back())

}

// routes.Bucket(1)
// routes.FindClosestPeers()
// routes.GetPeerAddresses()
// routes.GetPeers()
// routes.PeerExists()
// routes.RemovePeer()
// routes.Self()
// routes.Update()
// net := &network.Network{
// 	Keys: keys,
// 	Host: host,
// 	Port: port,
// 	ID:   id,

// 	Processors: &network.StringMessageProcessorSyncMap{},

// 	Routes: dht.CreateRoutingTable(id),

// 	Peers: &network.StringPeerClientSyncMap{},

// 	Listening: make(chan struct{}),
// }
