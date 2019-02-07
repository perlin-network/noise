package basic

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/perlin-network/noise"
	"github.com/perlin-network/noise/callbacks"
	"github.com/perlin-network/noise/payload"
	"github.com/perlin-network/noise/protocol"
	"github.com/pkg/errors"
)

var (
	_ protocol.IdentityPolicy = (*identityPolicy)(nil)
)

type identityPolicy struct {
}

func NewIdentityPolicy() *identityPolicy {
	return &identityPolicy{}
}

func (p *identityPolicy) EnforceIdentityPolicy(node *noise.Node) {
	if node.ID == nil {
		panic("basic: identity policy enforced but no identity provider assigned to node")
	}

	id := NewID(node.ExternalAddress(), node.ID.PublicID())

	protocol.SetNodeID(node, id)

	simpleActivity := func(node *noise.Node, peer *noise.Peer, id protocol.ID) error {
		AddPeer(node, id)
		peer.BeforeMessageReceived(func(node *noise.Node, peer *noise.Peer, msg []byte) (i []byte, e error) {
			return msg, nil
		})
		return nil
	}

	protocol.OnEachPeerAuthenticated(node, simpleActivity)
}

func (p *identityPolicy) OnSessionEstablished(node *noise.Node, peer *noise.Peer) error {
	// Place node ID at the header of every single message.
	peer.OnEncodeHeader(func(node *noise.Node, peer *noise.Peer, header []byte, msg []byte) (i []byte, e error) {
		return append(header, protocol.NodeID(node).Write()...), nil
	})

	// Validate all peer IDs situated inside every single messages header.
	peer.OnDecodeHeader(func(node *noise.Node, peer *noise.Peer, reader payload.Reader) error {
		raw, err := ID{}.Read(reader)
		if err != nil {
			peer.Disconnect()
			return errors.Wrap(err, "basic: failed to read peer ID")
		}

		id, currentID := raw.(ID), protocol.PeerID(peer)

		if currentID != nil {
			if !bytes.Equal(id.Hash(), currentID.Hash()) {
				peer.Disconnect()
				return errors.New("basic: peer gave a different ID to what they originally had")
			}
		} else {
			protocol.AuthenticatePeer(peer, id)
		}

		return nil
	})

	return callbacks.DeregisterCallback
}

type ID struct {
	address   string
	publicKey []byte
}

func (a ID) Equals(other protocol.ID) bool {
	if other, ok := other.(ID); ok {
		return bytes.Equal(a.Hash(), other.Hash())
	}

	return false
}

func (a ID) PublicID() []byte {
	return a.publicKey
}

func NewID(address string, publicKey []byte) ID {
	return ID{address: address, publicKey: publicKey}
}

func (a ID) String() string {
	return fmt.Sprintf("%s(%s)", a.address, hex.EncodeToString(a.publicKey)[:16])
}

func (a ID) Read(reader payload.Reader) (msg noise.Message, err error) {
	a.address, err = reader.ReadString()
	if err != nil {
		return nil, errors.Wrap(err, "basic: failed to deserialize ID address")
	}

	a.publicKey, err = reader.ReadBytes()
	if err != nil {
		return nil, errors.Wrap(err, "basic: failed to deserialize ID public key")
	}

	return a, nil
}

func (a ID) Write() []byte {
	return payload.NewWriter(nil).WriteString(a.address).WriteBytes(a.publicKey).Bytes()
}

func (a ID) Hash() []byte {
	return a.publicKey
}
