package skademlia

import (
	"context"
	"github.com/perlin-network/noise"
	"github.com/perlin-network/noise/edwards25519"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/blake2b"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestClientFields(t *testing.T) {
	keys, err := NewKeys(1, 1)
	if err != nil {
		t.Fatal(err)
	}

	client := NewClient("127.0.0.1", keys)
	assert.NotNil(t, client.Logger())
	assert.Equal(t, client, client.Protocol().client)
	assert.Equal(t, 16, client.BucketSize())
	assert.Equal(t, keys, client.Keys())

	assert.NotNil(t, client.ID())
	assert.Equal(t, keys.id, client.ID().id)

	credential := noise.NewCredentials("127.0.0.1")
	client.SetCredentials(credential)
	assert.Equal(t, client.creds, credential)

	client.OnPeerJoin(func(conn *grpc.ClientConn, id *ID) {})
	assert.NotNil(t, client.onPeerJoin)

	client.OnPeerLeave(func(conn *grpc.ClientConn, id *ID) {})
	assert.NotNil(t, client.onPeerLeave)
}

func TestClient(t *testing.T) {
	c1, lis1 := getClient(t, 1, 1)
	defer lis1.Close()
	c2, lis2 := getClient(t, 1, 1)
	defer lis2.Close()

	var onPeerJoinCalled int32

	c2.OnPeerJoin(func(conn *grpc.ClientConn, id *ID) {
		atomic.StoreInt32(&onPeerJoinCalled, 1)
	})

	onPeerLeave := make(chan struct{})

	c2.OnPeerLeave(func(conn *grpc.ClientConn, id *ID) {
		close(onPeerLeave)
	})

	server := c1.Listen()
	go func() {
		if err := server.Serve(lis1); err != nil {
			t.Fatal(err)
		}
	}()
	defer server.Stop()

	conn, err := c2.Dial(lis1.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	assert.Len(t, c2.Bootstrap(), 1)
	assert.Len(t, c2.AllPeers(), 1)
	assert.Len(t, c2.ClosestPeerIDs(), 1)
	assert.Len(t, c2.ClosestPeers(), 1)

	assert.Equal(t, c1.id.checksum, c2.ClosestPeerIDs()[0].checksum)

	server.Stop()

	assert.Equal(t, int32(1), atomic.LoadInt32(&onPeerJoinCalled))

	select {
	case <-onPeerLeave:
	case <-time.After(100 * time.Millisecond):
		assert.Fail(t, "OnPeerLeave never called")
	}
}

func TestClientEviction(t *testing.T) {
	client, _ := getClient(t, 1, 1)
	client.table.setBucketSize(1)

	type peer struct {
		c *Client
		l net.Listener
	}

	var peers = struct {
		peers []*peer
		sync.RWMutex
	}{}

	var wg sync.WaitGroup

	for i := 0; i < 3; i++ {
		wg.Add(1)

		c, lis := getClient(t, 1, 1)

		peers.Lock()
		peers.peers = append(peers.peers, &peer{
			c: c,
			l: lis,
		})
		peers.Unlock()

		go func(i int) {
			peers.RLock()
			p := peers.peers[i]
			peers.RUnlock()

			s := p.c.Listen()

			wg.Done()

			_ = s.Serve(p.l)
		}(i)
	}

	wg.Wait()

	for _, p := range peers.peers {
		_, _ = client.Dial(p.l.Addr().String())
	}

	client.Bootstrap()

	assert.Len(t, client.ClosestPeerIDs(), 1)
}

func TestInterceptedServerStream(t *testing.T) {
	c, lis := getClient(t, 1, 1)
	defer lis.Close()
	dss := &dummyServerStream{}

	var nodes []*ID

	nodes = append(nodes,
		&ID{address: "0000"},
		&ID{address: "0001"},
		&ID{address: "0002"},
		&ID{address: "0003"},
		&ID{address: "0004"},
		&ID{address: "0005"},
		&ID{address: "0006"},
	)

	var publicKey [blake2b.Size256]byte

	copy(publicKey[:], []byte("12345678901234567890123456789010"))
	nodes[0].checksum = publicKey

	copy(publicKey[:], []byte("12345678901234567890123456789011"))
	nodes[1].checksum = publicKey

	copy(publicKey[:], []byte("12345678901234567890123456789012"))
	nodes[2].checksum = publicKey

	copy(publicKey[:], []byte("12345678901234567890123456789013"))
	nodes[3].checksum = publicKey

	copy(publicKey[:], []byte("12345678901234567890123456789014"))
	nodes[4].checksum = publicKey

	copy(publicKey[:], []byte("12345678901234567890123456789015"))
	nodes[5].checksum = publicKey

	copy(publicKey[:], []byte("12345678901234567890123456789016"))
	nodes[6].checksum = publicKey

	c.table = NewTable(nodes[0])

	for i := 1; i < 5; i++ {
		assert.NoError(t, c.table.Update(nodes[i]))
	}

	// Test SendMsg

	closest := c.table.FindClosest(nodes[4], 2)
	assert.Len(t, closest, 2)
	assert.Equal(t, "0002", closest[0].address)
	assert.Equal(t, "0003", closest[1].address)

	iss := InterceptedServerStream{
		ServerStream: dss,
		client:       c,
		id:           nodes[5],
	}

	assert.NoError(t, iss.SendMsg(nil))

	closest = c.table.FindClosest(nodes[4], 2)
	assert.Len(t, closest, 2)
	assert.Equal(t, "0005", closest[0].address)
	assert.Equal(t, "0002", closest[1].address)

	// Test RecvMsg

	closest = c.table.FindClosest(nodes[5], 2)
	assert.Len(t, closest, 2)
	assert.Equal(t, "0004", closest[0].address)
	assert.Equal(t, "0003", closest[1].address)

	iss = InterceptedServerStream{
		ServerStream: dummyServerStream{},
		client:       c,
		id:           nodes[6],
	}

	assert.NoError(t, iss.RecvMsg(nil))

	closest = c.table.FindClosest(nodes[5], 2)
	assert.Len(t, closest, 2)
	assert.Equal(t, "0004", closest[0].address)
	assert.Equal(t, "0006", closest[1].address)
}

func TestInterceptedClientStream(t *testing.T) {
	c, lis := getClient(t, 1, 1)
	defer lis.Close()

	var nodes []*ID

	nodes = append(nodes,
		&ID{address: "0000"},
		&ID{address: "0001"},
		&ID{address: "0002"},
		&ID{address: "0003"},
		&ID{address: "0004"},
		&ID{address: "0005"},
		&ID{address: "0006"},
	)

	var publicKey edwards25519.PublicKey

	copy(publicKey[:], []byte("12345678901234567890123456789010"))
	nodes[0].checksum = publicKey

	copy(publicKey[:], []byte("12345678901234567890123456789011"))
	nodes[1].checksum = publicKey

	copy(publicKey[:], []byte("12345678901234567890123456789012"))
	nodes[2].checksum = publicKey

	copy(publicKey[:], []byte("12345678901234567890123456789013"))
	nodes[3].checksum = publicKey

	copy(publicKey[:], []byte("12345678901234567890123456789014"))
	nodes[4].checksum = publicKey

	copy(publicKey[:], []byte("12345678901234567890123456789015"))
	nodes[5].checksum = publicKey

	copy(publicKey[:], []byte("12345678901234567890123456789016"))
	nodes[6].checksum = publicKey

	c.table = NewTable(nodes[0])

	for i := 1; i < 5; i++ {
		assert.NoError(t, c.table.Update(nodes[i]))
	}

	// Test SendMsg

	closest := c.table.FindClosest(nodes[4], 2)
	assert.Len(t, closest, 2)
	assert.Equal(t, "0002", closest[0].address)
	assert.Equal(t, "0003", closest[1].address)

	iss := InterceptedClientStream{
		ClientStream: dummyClientStream{},
		client:       c,
		id:           nodes[5],
	}

	assert.NoError(t, iss.SendMsg(nil))

	closest = c.table.FindClosest(nodes[4], 2)
	assert.Len(t, closest, 2)
	assert.Equal(t, "0005", closest[0].address)
	assert.Equal(t, "0002", closest[1].address)

	// Test RecvMsg

	closest = c.table.FindClosest(nodes[5], 2)
	assert.Len(t, closest, 2)
	assert.Equal(t, "0004", closest[0].address)
	assert.Equal(t, "0003", closest[1].address)

	iss = InterceptedClientStream{
		ClientStream: dummyClientStream{},
		client:       c,
		id:           nodes[6],
	}

	assert.NoError(t, iss.RecvMsg(nil))

	closest = c.table.FindClosest(nodes[5], 2)
	assert.Len(t, closest, 2)
	assert.Equal(t, "0004", closest[0].address)
	assert.Equal(t, "0006", closest[1].address)
}

func getClient(t *testing.T, c1, c2 int) (*Client, net.Listener) {
	lis, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatal(err)
	}
	keys, err := NewKeys(c1, c2)
	if err != nil {
		t.Fatalf("error NewKeys(): %v", err)
	}

	c := NewClient(lis.Addr().String(), keys, WithC1(c1), WithC2(c2))
	c.SetCredentials(noise.NewCredentials(lis.Addr().String(), c.Protocol()))

	return c, lis
}

type dummyServerStream struct {
}

func (dummyServerStream) SetHeader(metadata.MD) error  { return nil }
func (dummyServerStream) SendHeader(metadata.MD) error { return nil }
func (dummyServerStream) SetTrailer(metadata.MD)       {}
func (dummyServerStream) Context() context.Context     { return nil }
func (dummyServerStream) SendMsg(m interface{}) error  { return nil }
func (dummyServerStream) RecvMsg(m interface{}) error  { return nil }

type dummyClientStream struct{}

func (dummyClientStream) Header() (metadata.MD, error) { return nil, nil }
func (dummyClientStream) Trailer() metadata.MD         { return nil }
func (dummyClientStream) CloseSend() error             { return nil }
func (dummyClientStream) Context() context.Context     { return nil }
func (dummyClientStream) SendMsg(m interface{}) error  { return nil }
func (dummyClientStream) RecvMsg(m interface{}) error  { return nil }
