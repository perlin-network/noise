package skademlia

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/perlin-network/noise"
	"github.com/perlin-network/noise/callbacks"
	"github.com/perlin-network/noise/payload"
	"github.com/perlin-network/noise/protocol"
	"github.com/pkg/errors"
	"golang.org/x/crypto/blake2b"
	"math/bits"
)

var (
	_ protocol.IdentityPolicy = (*identityPolicy)(nil)
	_ protocol.ID             = (*ID)(nil)
)

const KeyKademliaTable = "kademlia.table"

type identityPolicy struct {
	enableSignatures bool
}

func NewIdentityPolicy() identityPolicy {
	return identityPolicy{enableSignatures: false}
}

func (p identityPolicy) EnableSignatures() identityPolicy {
	p.enableSignatures = true
	return p
}

func (p identityPolicy) EnforceIdentityPolicy(node *noise.Node) {
	if node.ID == nil {
		panic("kademlia: identity policy enforced but no identity provider assigned to node")
	}

	id := NewID(node.ExternalAddress(), node.ID.PublicID())

	protocol.SetNodeID(node, id)
	node.Set(KeyKademliaTable, newTable(id))

	protocol.OnEachPeerAuthenticated(node, func(node *noise.Node, peer *noise.Peer, id protocol.ID) error {
		logPeerActivity(node, peer, id)

		peer.BeforeMessageReceived(func(node *noise.Node, peer *noise.Peer, msg []byte) (i []byte, e error) {
			return msg, logPeerActivity(node, peer, id)
		})

		return nil
	})
}

func (p identityPolicy) OnSessionEstablished(node *noise.Node, peer *noise.Peer) error {
	// Place node ID at the header of every single message.
	peer.OnEncodeHeader(func(node *noise.Node, peer *noise.Peer, header []byte, msg []byte) (i []byte, e error) {
		return append(header, protocol.NodeID(node).Write()...), nil
	})

	// Validate all peer IDs situated inside every single messages header.
	peer.OnDecodeHeader(func(node *noise.Node, peer *noise.Peer, reader payload.Reader) error {
		raw, err := ID{}.Read(reader)
		if err != nil {
			peer.Disconnect()
			return errors.Wrap(err, "kademlia: failed to read peer ID")
		}

		id, currentID := raw.(ID), protocol.PeerID(peer)

		if currentID != nil {
			if !bytes.Equal(id.Hash(), currentID.Hash()) {
				peer.Disconnect()
				return errors.New("kademlia: peer gave a different ID to what they originally had")
			}
		} else {
			protocol.AuthenticatePeer(peer, id)
		}

		return nil
	})

	if p.enableSignatures {
		// Place signature at the footer of every single message.
		peer.OnEncodeFooter(func(node *noise.Node, peer *noise.Peer, header, msg []byte) (i []byte, e error) {
			signature, err := node.ID.Sign(msg)

			if err != nil {
				panic(errors.Wrap(err, "signature: failed to sign message"))
			}

			return payload.NewWriter(header).WriteBytes(signature).Bytes(), nil
		})

		// Validate signature situated inside every single messages header.
		peer.OnDecodeFooter(func(node *noise.Node, peer *noise.Peer, msg []byte, reader payload.Reader) error {
			signature, err := reader.ReadBytes()
			if err != nil {
				peer.Disconnect()
				return errors.Wrap(err, "signature: failed to read message signature")
			}

			if err = node.ID.Verify(protocol.PeerID(peer).PublicID(), msg, signature); err != nil {
				peer.Disconnect()
				return errors.Wrap(err, "signature: peer sent an invalid signature")
			}

			return nil
		})
	}

	return callbacks.DeregisterCallback
}

func logPeerActivity(node *noise.Node, peer *noise.Peer, id protocol.ID) error {
	err := UpdateTable(node, id)

	if err != nil {
		peer.Disconnect()
		return errors.Wrap(err, "kademlia: failed to update table with peer ID")
	}

	return nil
}

type ID struct {
	address   string
	publicKey []byte

	hash []byte
}

func (a ID) Equals(other protocol.ID) bool {
	if other, ok := other.(ID); ok {
		return bytes.Equal(a.hash, other.hash)
	}

	return false
}

func (a ID) PublicID() []byte {
	return a.publicKey
}

func NewID(address string, publicKey []byte) ID {
	hash := blake2b.Sum256(publicKey)
	return ID{address: address, publicKey: publicKey, hash: hash[:]}
}

func (a ID) String() string {
	return fmt.Sprintf("%s(%s)(%s)", a.address, hex.EncodeToString(a.publicKey)[:16], hex.EncodeToString(a.hash)[:16])
}

func (a ID) Read(reader payload.Reader) (msg noise.Message, err error) {
	a.address, err = reader.ReadString()
	if err != nil {
		return nil, errors.Wrap(err, "kademlia: failed to deserialize ID address")
	}

	a.publicKey, err = reader.ReadBytes()
	if err != nil {
		return nil, errors.Wrap(err, "kademlia: failed to deserialize ID public key")
	}

	hash := blake2b.Sum256(a.publicKey)
	a.hash = hash[:]

	return a, nil
}

func (a ID) Write() []byte {
	return payload.NewWriter(nil).WriteString(a.address).WriteBytes(a.publicKey).Bytes()
}

func (a ID) Hash() []byte {
	return a.hash
}

func prefixLen(buf []byte) int {
	for i, b := range buf {
		if b != 0 {
			return i*8 + bits.LeadingZeros8(uint8(b))
		}
	}

	return len(buf)*8 - 1
}

func xor(a, b []byte) []byte {
	if len(a) != len(b) {
		panic("kademlia: len(a) and len(b) must be equal for xor(a, b)")
	}

	c := make([]byte, len(a))

	for i := 0; i < len(a); i++ {
		c[i] = a[i] ^ b[i]
	}

	return c
}

func prefixDiff(a, b []byte, n int) int {
	bytes, total := xor(a, b), 0

	for i, b := range bytes {
		if n <= 8*i {
			break
		} else if n > 8*i && n < 8*(i+1) {
			shift := 8 - uint(n%8)
			b = b >> shift
		}
		total += bits.OnesCount8(uint8(b))
	}
	return total
}
