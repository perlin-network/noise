package protocol

import (
	"github.com/perlin-network/noise"
	"github.com/perlin-network/noise/payload"
	"github.com/stretchr/testify/assert"
	"testing"
)

type dummyID struct{}

func (dummyID) String() string {
	panic("unreachable")
}

func (dummyID) Equals(other ID) bool {
	panic("unreachable")
}

func (dummyID) PublicKey() []byte {
	panic("unreachable")
}

func (dummyID) Hash() []byte {
	return []byte{0}
}

func (dummyID) Read(reader payload.Reader) (msg noise.Message, err error) {
	return dummyID{}, nil
}

func (dummyID) Write() []byte {
	return nil
}

func TestSharedKey(t *testing.T) {
	peer := &noise.Peer{}

	assert.False(t, HasSharedKey(peer))
	assert.Nil(t, LoadSharedKey(peer))

	SetSharedKey(peer, []byte{0})
	assert.True(t, HasSharedKey(peer))
	assert.Equal(t, []byte{0}, LoadSharedKey(peer))

	DeleteSharedKey(peer)
	assert.False(t, HasSharedKey(peer))
	assert.Nil(t, LoadSharedKey(peer))
}

func TestNodeID(t *testing.T) {
	node := &noise.Node{}
	assert.Nil(t, node.Get(KeyID))

	SetNodeID(node, dummyID{})
	assert.NotNil(t, NodeID(node))

	DeleteNodeID(node)
	assert.Nil(t, NodeID(node))
}

func TestPeerID(t *testing.T) {
	node := &noise.Node{}
	peer := &noise.Peer{}
	peer.SetNode(node)

	assert.False(t, HasPeerID(peer))
	assert.Nil(t, PeerID(peer))

	SetPeerID(peer, dummyID{})

	assert.True(t, HasPeerID(peer))
	assert.NotNil(t, PeerID(peer))
	assert.Equal(t, peer, Peer(node, dummyID{}))

	DeletePeerID(peer)
	assert.False(t, HasPeerID(peer))
	assert.Nil(t, PeerID(peer))
	assert.Nil(t, Peer(node, dummyID{}))
}
