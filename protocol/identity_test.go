package protocol

import (
	"github.com/perlin-network/noise"
	"github.com/perlin-network/noise/payload"
	"github.com/stretchr/testify/assert"
	"testing"
)

type DummyID struct{}

func (DummyID) String() string {
	panic("")
}

func (DummyID) Equals(other ID) bool {
	panic("")
}

func (DummyID) PublicID() []byte {
	panic("")
}

func (DummyID) Hash() []byte {
	return []byte{0}
}

func (DummyID) Read(reader payload.Reader) (msg noise.Message, err error) {
	return DummyID{}, nil
}

func (DummyID) Write() []byte {
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

	SetNodeID(node, DummyID{})
	assert.NotNil(t, NodeID(node))

	DeleteNodeID(node)
	assert.Nil(t, NodeID(node))
}

func TestPeerID(t *testing.T) {
	node := &noise.Node{}
	peer := &noise.Peer{}
	peer.Debug_SetNode(node)

	assert.False(t, HasPeerID(peer))
	assert.Nil(t, PeerID(peer))

	SetPeerID(peer, DummyID{})

	assert.True(t, HasPeerID(peer))
	assert.NotNil(t, PeerID(peer))
	assert.Equal(t, peer, Peer(node, DummyID{}))

	DeletePeerID(peer)
	assert.False(t, HasPeerID(peer))
	assert.Nil(t, PeerID(peer))
	assert.Nil(t, Peer(node, DummyID{}))
}
