package skademlia

import (
	"github.com/perlin-network/noise"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
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
	c1, lis1 := getClient(t)
	defer lis1.Close()
	c2, lis2 := getClient(t)
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
