package protocol

import (
	"sync"
	"sync/atomic"

	"github.com/perlin-network/noise"
	"github.com/perlin-network/noise/log"
	"github.com/pkg/errors"
)

const (
	KeyProtocolCurrentBlockIndex = "protocol.current_block_index"
	KeyProtocolEnforceOnce       = "protocol.enforce_once"
)

var (
	DisconnectPeer = errors.New("peer disconnect requested")
)

type Block interface {
	OnRegister(p *Protocol, node *noise.Node)
	OnBegin(p *Protocol, peer *noise.Peer) error
	OnEnd(p *Protocol, peer *noise.Peer) error
}

type Protocol struct {
	blocks       []Block
	blocksSealed uint32
}

func New() *Protocol {
	return &Protocol{}
}

// Register registers a block to this protocol sequentially.
func (p *Protocol) Register(blk Block) *Protocol {
	// This is not a strict check. Only here to help users find their mistakes.
	if atomic.LoadUint32(&p.blocksSealed) == 1 {
		panic("Register() cannot be called after Enforce().")
	}

	p.blocks = append(p.blocks, blk)
	return p
}

// Enforce enforces that all peers of a node follow the given protocol.
func (p *Protocol) Enforce(node *noise.Node) {
	atomic.StoreUint32(&p.blocksSealed, 1)

	node.LoadOrStore(KeyProtocolEnforceOnce, new(sync.Once)).(*sync.Once).Do(func() {
		for _, block := range p.blocks {
			block.OnRegister(p, node)
		}

		node.OnPeerInit(func(node *noise.Node, peer *noise.Peer) error {
			go func() {
				peer.OnDisconnect(func(node *noise.Node, peer *noise.Peer) error {
					blockIndex := peer.LoadOrStore(KeyProtocolCurrentBlockIndex, 0).(int)

					if blockIndex >= len(p.blocks) {
						return nil
					}

					return p.blocks[blockIndex].OnEnd(p, peer)
				})

				for {
					blockIndex := peer.LoadOrStore(KeyProtocolCurrentBlockIndex, 0).(int)

					if blockIndex >= len(p.blocks) {
						return
					}

					err := p.blocks[blockIndex].OnBegin(p, peer)

					if err != nil {
						switch errors.Cause(err) {
						case DisconnectPeer:
							peer.Disconnect()
						default:
							log.Warn().Err(err).Msg("Received an error following protocol.")
						}

						return
					} else {
						peer.Set(KeyProtocolCurrentBlockIndex, blockIndex+1)
					}
				}
			}()

			return nil
		})
	})
}
