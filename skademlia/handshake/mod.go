package handshake

import (
	"encoding/hex"
	"github.com/perlin-network/noise"
	"github.com/perlin-network/noise/log"
	"github.com/perlin-network/noise/protocol"
	"github.com/perlin-network/noise/skademlia"
	"github.com/pkg/errors"
	"time"
)

const KeySkademliaHandshake = ".noise.skademlia.handshake"

var (
	_ protocol.Block = (*Block)(nil)
)

type Block struct {
	timeout         time.Duration
	opcodeHandshake noise.Opcode
	IDMgr           *skademlia.IdentityManager
}

func New() *Block {
	return &Block{
		timeout: 5 * time.Second,
	}
}

func (b *Block) WithTimeout(timeout time.Duration) *Block {
	b.timeout = timeout
	return b
}

func (b *Block) OnRegister(p *protocol.Protocol, node *noise.Node) {
	b.opcodeHandshake = noise.RegisterMessage(noise.NextAvailableOpcode(), (*Handshake)(nil))
	b.IDMgr = node.ID.(*skademlia.IdentityManager)
}

func (b *Block) OnBegin(p *protocol.Protocol, peer *noise.Peer) error {
	if b.IDMgr == nil {
		return errors.New("node not setup with skademlia properly")
	}

	// Send your
	if err := peer.SendMessage(b.makeMessage("init")); err != nil {
		return errors.Wrap(errors.Wrap(protocol.DisconnectPeer, err.Error()), "failed to send our skademlia init to our peer")
	}

	// Wait for handshake response.
	var res Handshake
	var ok bool

	select {
	case <-time.After(b.timeout):
		return errors.Wrap(protocol.DisconnectPeer, "timed out receiving handshake request")
	case msg := <-peer.Receive(b.opcodeHandshake):
		res, ok = msg.(Handshake)
		if !ok {
			return errors.Wrap(protocol.DisconnectPeer, "did not get a handshake response back")
		}
	}

	// verify the handshake is okay
	if err := b.VerifyHandshake(res); err != nil {
		return errors.Wrap(errors.Wrap(protocol.DisconnectPeer, err.Error()), "failed to validate skademlia id")
	}

	// store the public ID in the metadata for further debugging
	publicIDHex := hex.EncodeToString(res.PublicKey)
	peer.Set(KeySkademliaHandshake, publicIDHex)

	log.Debug().
		Str("publicIDHex", publicIDHex).
		Msg("Successfully performed SKademlia ID verification with our peer.")

	return nil
}

func (b *Block) OnEnd(p *protocol.Protocol, peer *noise.Peer) error {
	return nil
}

func (b *Block) makeMessage(msg string) *Handshake {
	return &Handshake{
		Msg:       msg,
		PublicKey: b.IDMgr.PublicID(),
		ID:        b.IDMgr.NodeID,
		Nonce:     b.IDMgr.Nonce,
		C1:        uint16(b.IDMgr.C1),
		C2:        uint16(b.IDMgr.C2),
	}
}

// VerifyHandshake checks if a handshake is valid
func (b *Block) VerifyHandshake(msg Handshake) error {

	if msg.C1 < uint16(b.IDMgr.C1) || msg.C2 < uint16(b.IDMgr.C2) {
		return errors.Errorf("skademlia: S/Kademlia constants (%d, %d) for (c1, c2) do not satisfy local constants (%d, %d)",
			msg.C1, msg.C2, b.IDMgr.C1, b.IDMgr.C2)
	}

	// Verify that the remote peer ID is valid for the current node's c1 and c2 settings
	if ok := skademlia.VerifyPuzzle(msg.PublicKey, msg.ID, msg.Nonce, b.IDMgr.C1, b.IDMgr.C2); !ok {
		return errors.New("skademlia: keypair failed skademlia verification")
	}

	return nil
}
