package skademlia

import (
	"github.com/perlin-network/noise"
	"github.com/perlin-network/noise/log"
	"github.com/perlin-network/noise/payload"
	"github.com/perlin-network/noise/protocol"
	"github.com/pkg/errors"
	"time"
)

const (
	keyKademliaTable = "kademlia.table"
	keyAuthChannel   = "kademlia.auth.ch"
)

var (
	OpcodePing           noise.Opcode
	OpcodeEvict          noise.Opcode
	OpcodeLookupRequest  noise.Opcode
	OpcodeLookupResponse noise.Opcode

	_ protocol.Block = (*block)(nil)
)

type block struct{}

func New() block {
	return block{}
}

func (b block) OnRegister(p *protocol.Protocol, node *noise.Node) {
	OpcodePing = noise.RegisterMessage(noise.NextAvailableOpcode(), (*Ping)(nil))
	OpcodeEvict = noise.RegisterMessage(noise.NextAvailableOpcode(), (*Evict)(nil))
	OpcodeLookupRequest = noise.RegisterMessage(noise.NextAvailableOpcode(), (*LookupRequest)(nil))
	OpcodeLookupResponse = noise.RegisterMessage(noise.NextAvailableOpcode(), (*LookupResponse)(nil))

	var nodeID = NewID(node.ExternalAddress(), node.ID.PublicID())

	protocol.SetNodeID(node, nodeID)
	node.Set(keyKademliaTable, newTable(nodeID))
}

func (b block) OnBegin(p *protocol.Protocol, peer *noise.Peer) error {
	// Send a ping.
	err := peer.SendMessage(OpcodePing, Ping{protocol.NodeID(peer.Node()).(ID)})
	if err != nil {
		return errors.Wrap(errors.Wrap(protocol.DisconnectPeer, err.Error()), "failed to send ping")
	}

	// Receive a ping and set the peers ID.
	var ping Ping

	select {
	case msg := <-peer.Receive(OpcodePing):
		ping = msg.(Ping)
	case <-time.After(3 * time.Second):
		return errors.Wrap(protocol.DisconnectPeer, "skademlia: timed out waiting for pong")
	}

	// Register peer.
	protocol.SetPeerID(peer, ping.ID)
	enforceSignatures(peer, false)

	// Log peer into S/Kademlia table, and have all messages update the S/Kademlia table.
	logPeerActivity(peer)

	peer.BeforeMessageReceived(func(node *noise.Node, peer *noise.Peer, msg []byte) (i []byte, e error) {
		return msg, logPeerActivity(peer)
	})

	go handleLookups(peer)

	close(peer.LoadOrStore(keyAuthChannel, make(chan struct{})).(chan struct{}))

	return nil
}

func (b block) OnEnd(p *protocol.Protocol, peer *noise.Peer) error {
	if protocol.HasPeerID(peer) {
		protocol.DeletePeerID(peer)
	}

	return nil
}

func logPeerActivity(peer *noise.Peer) error {
	err := UpdateTable(peer.Node(), protocol.PeerID(peer))

	if err != nil {
		return errors.Wrap(errors.Wrap(protocol.DisconnectPeer, err.Error()),
			"kademlia: failed to update table with peer ID")
	}

	return nil
}

func enforceSignatures(peer *noise.Peer, enforce bool) {
	if enforce {
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
}

func handleLookups(peer *noise.Peer) {
	for {
		select {
		case msg := <-peer.Receive(OpcodeLookupRequest):
			id := msg.(LookupRequest)

			var res LookupResponse

			for _, peerID := range FindClosestPeers(Table(peer.Node()), id.Hash(), BucketSize()) {
				res.peers = append(res.peers, peerID.(ID))
			}

			log.Info().
				Strs("addrs", Table(peer.Node()).GetPeers()).
				Msg("Connected to peer(s).")

			// Send lookup response back.

			if err := peer.SendMessage(OpcodeLookupResponse, res); err != nil {
				log.Warn().
					AnErr("err", err).
					Interface("peer", protocol.PeerID(peer)).
					Msg("Failed to send lookup response to peer.")
			}
		}
	}
}

func WaitUntilAuthenticated(peer *noise.Peer) {
	<-peer.LoadOrStore(keyAuthChannel, make(chan struct{}, 1)).(chan struct{})
}
