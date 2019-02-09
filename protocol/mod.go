package protocol

import (
	"github.com/perlin-network/noise"
	"github.com/perlin-network/noise/log"
	"github.com/pkg/errors"
)

const (
	KeyProtocolCurrentBlockIndex = "protocol.current_block"
	KeyProtocolInstanceIndex     = "protocol.instance"
)

var (
	CompletedAllBlocks = errors.New("completed all blocks")
	DisconnectPeer     = errors.New("peer disconnect requested")
)

type Block func(protocol *ProtocolInstance, peer *noise.Peer) (Block, error)

type Protocol struct {
	blocks []Block
}

type ProtocolInstance struct {
	protocol *Protocol

	incoming map[noise.Opcode]chan noise.Message
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
		inst := &ProtocolInstance{
			protocol: p,
			incoming: make(map[noise.Opcode]chan noise.Message),
		}
		for _, op := range noise.GetOpcodes() {
			ch := make(chan noise.Message)
			inst.incoming[op] = ch
			peer.OnMessageReceived(op, func(node *noise.Node, opcode noise.Opcode, peer *noise.Peer, message noise.Message) error {
				ch <- message
				return nil
			})
		}
		peer.Set(KeyProtocolInstanceIndex, inst)
		go inst.runWorker(peer)
		return nil
	})

	node.OnPeerDisconnected(func(node *noise.Node, peer *noise.Peer) error {
		inst := peer.Get(KeyProtocolInstanceIndex).(*ProtocolInstance)

		for _, ch := range inst.incoming {
			close(ch)
		}

		return nil
	})
}

func (p *ProtocolInstance) Recv(op noise.Opcode) (noise.Message, error) {
	ch := p.incoming[op]
	if ch == nil {
		return nil, errors.New("invalid opcode")
	}

	msg, ok := <-ch
	if !ok {
		return nil, errors.New("peer disconnected")
	}

	return msg, nil
}

func (p *ProtocolInstance) runWorker(peer *noise.Peer) {
	defer peer.Disconnect()

	for _, blk := range p.protocol.blocks {
		maybeNext := blk
		var err error
		for maybeNext != nil {
			maybeNext, err = maybeNext(p, peer)
			if err != nil {
				if err != DisconnectPeer {
					log.Error().Err(err).Msg("got an error while handling protocol session")
				}
				return
			}
		}
	}
}
