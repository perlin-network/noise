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
	c1 := newClientTestContainer(t, 1, 1)
	c1.serve()
	defer c1.lis.Close()
	c2 := newClientTestContainer(t, 1, 1)
	c2.serve()
	defer c2.lis.Close()

	var onPeerJoinCalled int32

	c1.client.OnPeerJoin(func(conn *grpc.ClientConn, id *ID) {
		atomic.StoreInt32(&onPeerJoinCalled, 1)
	})

	onPeerLeave := make(chan struct{})

	c2.client.OnPeerLeave(func(conn *grpc.ClientConn, id *ID) {
		close(onPeerLeave)
	})

	conn, err := c2.client.Dial(c1.lis.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	assert.Len(t, c2.client.Bootstrap(), 1)
	assert.Len(t, c2.client.AllPeers(), 1)
	assert.Len(t, c2.client.ClosestPeerIDs(), 1)
	assert.Len(t, c2.client.ClosestPeers(), 1)

	assert.Equal(t, c1.client.id.checksum, c2.client.ClosestPeerIDs()[0].checksum)

	c1.server.Stop()

	assert.Equal(t, int32(1), atomic.LoadInt32(&onPeerJoinCalled))

	select {
	case <-onPeerLeave:
	case <-time.After(2000 * time.Millisecond):
		assert.Fail(t, "OnPeerLeave never called")
	}
}

func TestClientEviction(t *testing.T) {
	s := newClientTestContainer(t, 1, 1)
	s.client.table.setBucketSize(1)
	s.serve()
	defer s.cleanup()

	accept := make(chan struct{})

	s.client.OnPeerJoin(func(conn *grpc.ClientConn, id *ID) {
		accept <- struct{}{}
	})

	var clients []*Client
	for i := 0; i < 5; i++ {
		c := newClientTestContainer(t, 1, 1)
		c.serve()

		//noinspection GoDeferInLoop
		defer c.cleanup()

		_, err := c.client.Dial(s.lis.Addr().String())
		assert.NoError(t, err)

		clients = append(clients, c.client)
	}

	for i := 0; i < 5; i++ {
		<-accept
	}

	assert.Len(t, s.client.ClosestPeerIDs(), 1)
}

func TestInterceptedServerStream(t *testing.T) {
	c := newClientTestContainer(t, 1, 1)
	defer c.cleanup()
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

	c.client.table = NewTable(nodes[0])

	for i := 1; i < 5; i++ {
		assert.NoError(t, c.client.table.Update(nodes[i]))
	}

	// Test SendMsg

	closest := c.client.table.FindClosest(nodes[4], 2)
	assert.Len(t, closest, 2)
	assert.Equal(t, "0002", closest[0].address)
	assert.Equal(t, "0003", closest[1].address)

	iss := InterceptedServerStream{
		ServerStream: dss,
		client:       c.client,
		id:           nodes[5],
	}

	assert.NoError(t, iss.SendMsg(nil))

	closest = c.client.table.FindClosest(nodes[4], 2)
	assert.Len(t, closest, 2)
	assert.Equal(t, "0005", closest[0].address)
	assert.Equal(t, "0002", closest[1].address)

	// Test RecvMsg

	closest = c.client.table.FindClosest(nodes[5], 2)
	assert.Len(t, closest, 2)
	assert.Equal(t, "0004", closest[0].address)
	assert.Equal(t, "0003", closest[1].address)

	iss = InterceptedServerStream{
		ServerStream: dummyServerStream{},
		client:       c.client,
		id:           nodes[6],
	}

	assert.NoError(t, iss.RecvMsg(nil))

	closest = c.client.table.FindClosest(nodes[5], 2)
	assert.Len(t, closest, 2)
	assert.Equal(t, "0004", closest[0].address)
	assert.Equal(t, "0006", closest[1].address)
}

func TestInterceptedClientStream(t *testing.T) {
	c := newClientTestContainer(t, 1, 1)
	defer c.cleanup()

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

	c.client.table = NewTable(nodes[0])

	for i := 1; i < 5; i++ {
		assert.NoError(t, c.client.table.Update(nodes[i]))
	}

	// Test SendMsg

	closest := c.client.table.FindClosest(nodes[4], 2)
	assert.Len(t, closest, 2)
	assert.Equal(t, "0002", closest[0].address)
	assert.Equal(t, "0003", closest[1].address)

	iss := InterceptedClientStream{
		ClientStream: dummyClientStream{},
		client:       c.client,
		id:           nodes[5],
	}

	assert.NoError(t, iss.SendMsg(nil))

	closest = c.client.table.FindClosest(nodes[4], 2)
	assert.Len(t, closest, 2)
	assert.Equal(t, "0005", closest[0].address)
	assert.Equal(t, "0002", closest[1].address)

	// Test RecvMsg

	closest = c.client.table.FindClosest(nodes[5], 2)
	assert.Len(t, closest, 2)
	assert.Equal(t, "0004", closest[0].address)
	assert.Equal(t, "0003", closest[1].address)

	iss = InterceptedClientStream{
		ClientStream: dummyClientStream{},
		client:       c.client,
		id:           nodes[6],
	}

	assert.NoError(t, iss.RecvMsg(nil))

	closest = c.client.table.FindClosest(nodes[5], 2)
	assert.Len(t, closest, 2)
	assert.Equal(t, "0004", closest[0].address)
	assert.Equal(t, "0006", closest[1].address)
}

type clientTestContainer struct {
	client   *Client
	lis      net.Listener
	server   *grpc.Server
	onClient func(*noise.Info)
	onServer func(*noise.Info)
}

// Rename to close
func (c *clientTestContainer) cleanup() {
	c.server.Stop()
	c.lis.Close()
}

func (c *clientTestContainer) serve() {
	go func() {
		_ = c.server.Serve(c.lis)
	}()
}

func (c *clientTestContainer) Client(info *noise.Info, ctx context.Context, authority string, conn net.Conn) (net.Conn, error) {
	if c.onClient != nil {
		c.onClient(info)
	}

	return conn, nil
}

func (c *clientTestContainer) Server(info *noise.Info, conn net.Conn) (net.Conn, error) {
	if c.onServer != nil {
		c.onServer(info)
	}

	return conn, nil
}

func newClientTestContainer(t *testing.T, c1, c2 int) *clientTestContainer {
	lis, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatal(err)
	}
	keys, err := NewKeys(c1, c2)
	if err != nil {
		t.Fatalf("error NewKeys(): %v", err)
	}

	c := NewClient(lis.Addr().String(), keys, WithC1(c1), WithC2(c2))
	testClient := &clientTestContainer{
		client: c,
		lis:    lis,
	}
	c.SetCredentials(noise.NewCredentials(lis.Addr().String(), c.Protocol(), testClient))

	server := c.Listen()

	testClient.server = server

	return testClient
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
