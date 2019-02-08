package protocol

import (
	"github.com/perlin-network/noise"
	"github.com/pkg/errors"
)

const (
	KeyProtocolBlocks            = "protocol.blocks"
	KeyProtocolCurrentBlockIndex = "protocol.current_block"
)

var (
	CompletedAllBlocks = errors.New("completed all blocks")
)

type Protocol struct {
	blocks []Block
}

type Block interface {
	OnBegin(node *noise.Node, peer *noise.Peer) error
	OnEnd(node *noise.Node, peer *noise.Peer) error
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
		return p.blocks[0].OnBegin(node, peer)
	})
}

// Next forces a peer to be in the next 'protocol block'.
// To be called inside a 'Block' implementation to signal that the peer is done following the current block
func (p *Protocol) Next(node *noise.Node, peer *noise.Peer) error {
	numBlocks := len(p.blocks)

	currBlock := 0
	if val, ok := peer.Get(KeyProtocolCurrentBlockIndex).(int); ok {
		currBlock = val
	}

	if currBlock >= numBlocks {
		return CompletedAllBlocks
	}

	if err := p.blocks[currBlock].OnEnd(node, peer); err != nil {
		return err
	}

	nextBlock := (currBlock + 1) % numBlocks
	peer.Set(KeyProtocolCurrentBlockIndex, nextBlock)

	if nextBlock == 0 {
		return CompletedAllBlocks
	}

	if err := p.blocks[nextBlock].OnBegin(node, peer); err != nil {
		return err
	}

	return nil
}
