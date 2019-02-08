package protocol

import "github.com/perlin-network/noise"

const KeyProtocolCurrentBlockIndex = "protocol.current_block"

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

func (p *Protocol) Register(blk Block) {
	p.blocks = append(p.blocks, blk)
}

func (p *Protocol) Enforce(node *noise.Node) {
	initCallbacks := make([]noise.OnPeerInitCallback, len(p.blocks))
	disconnectCallbacks := make([]noise.OnPeerDisconnectCallback, len(p.blocks))

	for i, blk := range p.blocks {
		blk := blk
		initCallbacks[i] = blk.OnBegin
		disconnectCallbacks[i] = blk.OnEnd
	}

	node.OnPeerInit(initCallbacks...)
	node.OnPeerDisconnected(disconnectCallbacks...)
}
