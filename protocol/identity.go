package protocol

import (
	"fmt"

	"github.com/perlin-network/noise"
)

const (
	KeySharedKey = "identity.shared_key"
	KeyID        = "node.id"
	KeyPeerID    = "peer.id"
)

type ID interface {
	fmt.Stringer
	noise.Message

	Equals(other ID) bool

	PublicKey() []byte
	Hash() []byte
}

func HasSharedKey(peer *noise.Peer) bool {
	return peer.Has(KeySharedKey)
}

func LoadSharedKey(peer *noise.Peer) []byte {
	sharedKey := peer.Get(KeySharedKey)

	if sharedKey == nil {
		return nil
	}

	if sharedKey, ok := sharedKey.([]byte); ok {
		return sharedKey
	}

	return nil
}

func MustSharedKey(peer *noise.Peer) []byte {
	if !HasSharedKey(peer) {
		panic("noise: shared key must be established via protocol for peer")
	}

	return LoadSharedKey(peer)
}

func SetSharedKey(peer *noise.Peer, sharedKey []byte) {
	peer.Set(KeySharedKey, sharedKey)
}

func DeleteSharedKey(peer *noise.Peer) {
	peer.Delete(KeySharedKey)
}

func SetNodeID(node *noise.Node, id ID) {
	node.Set(KeyID, id)
}

func DeleteNodeID(node *noise.Node) {
	node.Delete(KeyID)
}

func HasPeerID(peer *noise.Peer) bool {
	return peer.Has(KeyID)
}

func SetPeerID(peer *noise.Peer, id ID) {
	peer.Node().Set(KeyPeerID+string(id.Hash()), peer)
	peer.Set(KeyID, id)
}

func DeletePeerID(peer *noise.Peer) {
	peer.Node().Delete(KeyPeerID + string(PeerID(peer).Hash()))
	peer.Delete(KeyID)
}

func NodeID(node *noise.Node) ID {
	t := node.Get(KeyID)

	if t == nil {
		return nil
	}

	if t, ok := t.(ID); ok {
		return t
	}

	return nil
}

func PeerID(peer *noise.Peer) ID {
	t := peer.Get(KeyID)

	if t == nil {
		return nil
	}

	if t, ok := t.(ID); ok {
		return t
	}

	return nil
}

func Peer(node *noise.Node, id ID) *noise.Peer {
	t := node.Get(KeyPeerID + string(id.Hash()))

	if t == nil {
		return nil
	}

	if t, ok := t.(*noise.Peer); ok {
		return t
	}

	return nil
}
