package protocol

import (
	"github.com/perlin-network/noise"
	"github.com/pkg/errors"
)

const (
	KeyProtocolCurrentBlockIndex = "protocol.current_block"
)

var (
	CompletedAllBlocks = errors.New("completed all blocks")
)

type Protocol struct {
	blocks []Block
}

type Block interface {
	OnBegin(protocol *Protocol, peer *noise.Peer) error
	OnEnd(protocol *Protocol, peer *noise.Peer) error
}

func NewProtocol() *Protocol {
	return &Protocol{}
}

// Register registers a block to this protocol sequentially.
func (p *Protocol) Register(blk Block) {
	p.blocks = append(p.blocks, blk)
}

// Enforce enforces that all peers of a node follow the given protocol.
func (p *Protocol) Enforce(node *noise.Node) {
	node.OnPeerInit(func(node *noise.Node, peer *noise.Peer) error {
		peer.Set(KeyProtocolCurrentBlockIndex, 0)
		return p.blocks[0].OnBegin(p, peer)
	})

	node.OnPeerDisconnected(func(node *noise.Node, peer *noise.Peer) error {
		return p.blocks[peer.LoadOrStore(KeyProtocolCurrentBlockIndex, 0).(int)].OnEnd(p, peer)
	})
}

// Next forces a peer to be in the next 'protocol block'.
// To be called inside a 'Block' implementation to signal that the peer is done following the current block
func (p *Protocol) Next(node *noise.Node, peer *noise.Peer) error {
	numBlocks := len(p.blocks)

	currBlock := peer.LoadOrStore(KeyProtocolCurrentBlockIndex, 0).(int)

	if err := p.blocks[currBlock].OnEnd(p, peer); err != nil {
		return err
	}

	if currBlock >= numBlocks {
		return CompletedAllBlocks
	}

	nextBlock := (currBlock + 1) % numBlocks
	peer.Set(KeyProtocolCurrentBlockIndex, nextBlock)

	if nextBlock == 0 {
		return CompletedAllBlocks
	}

	if err := p.blocks[nextBlock].OnBegin(p, peer); err != nil {
		return err
	}

	return nil
}
