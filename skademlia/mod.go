package skademlia

import (
	"github.com/perlin-network/noise"
	"github.com/perlin-network/noise/log"
	"github.com/perlin-network/noise/payload"
	"github.com/perlin-network/noise/protocol"
	"github.com/perlin-network/noise/signature"
	"github.com/pkg/errors"
	"time"
)

const (
	DefaultPrefixDiffLen = 128
	DefaultPrefixDiffMin = 32

	keyKademliaTable = "kademlia.table"
	keyAuthChannel   = "kademlia.auth.ch"
)

var (
	_ protocol.Block = (*block)(nil)
)

type block struct {
	opcodePing           noise.Opcode
	opcodeEvict          noise.Opcode
	opcodeLookupRequest  noise.Opcode
	opcodeLookupResponse noise.Opcode

	scheme signature.Scheme

	c1, c2 int

	prefixDiffLen, prefixDiffMin int
}

func New() *block {
	return &block{c1: DefaultC1, c2: DefaultC2, prefixDiffLen: DefaultPrefixDiffLen, prefixDiffMin: DefaultPrefixDiffMin}
}

func (b *block) WithC1(c1 int) *block {
	b.c1 = c1
	return b
}

func (b *block) WithC2(c2 int) *block {
	b.c2 = c2
	return b
}

func (b *block) WithPrefixDiffLen(prefixDiffLen int) *block {
	b.prefixDiffLen = prefixDiffLen
	return b
}

func (b *block) WithPrefixDiffMin(prefixDiffMin int) *block {
	b.prefixDiffMin = prefixDiffMin
	return b
}

func (b *block) WithSignatureScheme(scheme signature.Scheme) *block {
	b.scheme = scheme
	return b
}

func (b *block) OnRegister(p *protocol.Protocol, node *noise.Node) {
	b.opcodePing = noise.RegisterMessage(noise.NextAvailableOpcode(), (*Ping)(nil))
	b.opcodeEvict = noise.RegisterMessage(noise.NextAvailableOpcode(), (*Evict)(nil))
	b.opcodeLookupRequest = noise.RegisterMessage(noise.NextAvailableOpcode(), (*LookupRequest)(nil))
	b.opcodeLookupResponse = noise.RegisterMessage(noise.NextAvailableOpcode(), (*LookupResponse)(nil))

	if _, ok := node.Keys.(*Keypair); !ok {
		panic("skademlia: node should set `params := noise.DefaultParams(); params.Keys = skademlia.NewKeys()`")
	}

	var nodeID = NewID(node.ExternalAddress(), node.Keys.PublicKey(), node.Keys.(*Keypair).Nonce)

	protocol.SetNodeID(node, nodeID)
	node.Set(keyKademliaTable, newTable(nodeID))
}

func (b *block) OnBegin(p *protocol.Protocol, peer *noise.Peer) error {
	// Send a ping.
	err := peer.SendMessage(Ping{ID: protocol.NodeID(peer.Node()).(ID)})
	if err != nil {
		return errors.Wrap(errors.Wrap(protocol.DisconnectPeer, err.Error()), "failed to send ping")
	}

	// Receive a ping and set the peers id.
	var id Ping

	select {
	case msg := <-peer.Receive(b.opcodePing):
		id = msg.(Ping)
	case <-time.After(3 * time.Second):
		return errors.Wrap(protocol.DisconnectPeer, "skademlia: timed out waiting for pong")
	}

	// Verify that the remote peer id is valid for the current node's c1 and c2 settings
	if ok := VerifyPuzzle(id.PublicKey(), id.Hash(), id.nonce, b.c1, b.c2); !ok {
		return errors.New("skademlia: peer connected with ID that fails to solve static/dynamic crpyo tpuzzle")
	}

	// Register peer.
	protocol.SetPeerID(peer, id.ID)
	enforceSignatures(peer, b.scheme)

	// Log peer into S/Kademlia table, and have all messages update the S/Kademlia table.
	_ = b.logPeerActivity(peer)

	peer.BeforeMessageReceived(func(node *noise.Node, peer *noise.Peer, msg []byte) (i []byte, e error) {
		return msg, b.logPeerActivity(peer)
	})

	go b.handleLookups(peer)

	close(peer.LoadOrStore(keyAuthChannel, make(chan struct{})).(chan struct{}))

	return nil
}

func (b *block) OnEnd(p *protocol.Protocol, peer *noise.Peer) error {
	if protocol.HasPeerID(peer) {
		protocol.DeletePeerID(peer)
	}

	return nil
}

func (b *block) logPeerActivity(peer *noise.Peer) error {
	if prefixDiff(protocol.NodeID(peer.Node()).Hash(), protocol.PeerID(peer).Hash(), b.prefixDiffLen) > b.prefixDiffMin {
		err := UpdateTable(peer.Node(), protocol.PeerID(peer))

		if err != nil {
			return errors.Wrap(errors.Wrap(protocol.DisconnectPeer, err.Error()),
				"kademlia: failed to update table with peer ID")
		}
	}

	return nil
}

func enforceSignatures(peer *noise.Peer, scheme signature.Scheme) {
	if scheme != nil {
		// Place signature at the footer of every single message.
		peer.OnEncodeFooter(func(node *noise.Node, peer *noise.Peer, header, msg []byte) (i []byte, e error) {
			signature, err := scheme.Sign(node.Keys.PrivateKey(), msg)

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

			if err = scheme.Verify(protocol.PeerID(peer).PublicKey(), msg, signature); err != nil {
				peer.Disconnect()
				return errors.Wrap(err, "signature: peer sent an invalid signature")
			}

			return nil
		})
	}
}

func (b *block) handleLookups(peer *noise.Peer) {
	for {
		select {
		case msg := <-peer.Receive(b.opcodeLookupRequest):
			id := msg.(LookupRequest)

			var res LookupResponse

			for _, peerID := range FindClosestPeers(Table(peer.Node()), id.Hash(), BucketSize()) {
				res.peers = append(res.peers, peerID.(ID))
			}

			log.Info().
				Strs("addrs", Table(peer.Node()).GetPeers()).
				Msg("Connected to peer(s).")

			// Send lookup response back.

			if err := peer.SendMessage(res); err != nil {
				log.Warn().
					AnErr("err", err).
					Interface("peer", protocol.PeerID(peer)).
					Msg("Failed to send lookup response to peer.")
			}
		}
	}
}

func WaitUntilAuthenticated(peer *noise.Peer) {
	<-peer.LoadOrStore(keyAuthChannel, make(chan struct{})).(chan struct{})
}
