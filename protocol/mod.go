package protocol

import (
	"github.com/perlin-network/noise"
	"github.com/perlin-network/noise/log"
	"github.com/pkg/errors"
	"sync"
)

const (
	KeyProtocolCurrentBlockIndex = "protocol.current_block_index"
)

var (
	CompletedAllBlocks = errors.New("completed all blocks")
	DisconnectPeer     = errors.New("peer disconnect requested")
)

type Block interface {
	OnRegister(p *Protocol, node *noise.Node)
	OnBegin(p *Protocol, peer *noise.Peer) error
	OnEnd(p *Protocol, peer *noise.Peer) error
}

type Protocol struct {
	enforceOnce sync.Once

	blocks      []Block
	blocksMutex sync.Mutex
}

func New() *Protocol {
	return &Protocol{}
}

// Register registers a block to this protocol sequentially.
func (p *Protocol) Register(blk Block) {
	p.blocksMutex.Lock()
	defer p.blocksMutex.Unlock()

	p.blocks = append(p.blocks, blk)
}

// Enforce enforces that all peers of a node follow the given protocol.
func (p *Protocol) Enforce(node *noise.Node) {
	p.enforceOnce.Do(func() {
		p.blocksMutex.Lock()
		for _, block := range p.blocks {
			block.OnRegister(p, node)
		}
		p.blocksMutex.Unlock()

		node.OnPeerInit(func(node *noise.Node, peer *noise.Peer) error {
			peer.Set(KeyProtocolCurrentBlockIndex, 0)

			go func() {
				peer.OnDisconnect(func(node *noise.Node, peer *noise.Peer) error {
					blockIndex := peer.Get(KeyProtocolCurrentBlockIndex).(int)

					p.blocksMutex.Lock()
					defer p.blocksMutex.Unlock()

					if blockIndex >= len(p.blocks) {
						return nil
					}

					// FIXME: Iterate through all blocks

					return p.blocks[blockIndex].OnEnd(p, peer)
				})

				for {
					blockIndex := peer.Get(KeyProtocolCurrentBlockIndex).(int)

					if blockIndex >= len(p.blocks) {
						return
					}

					p.blocksMutex.Lock()
					err := p.blocks[blockIndex].OnBegin(p, peer)
					p.blocksMutex.Unlock()

					if err != nil {
						log.Warn().Err(err).Msg("Received an error following protocol.")

						switch errors.Cause(err) {
						case DisconnectPeer:
							peer.Disconnect()
							return
						case CompletedAllBlocks:
							return
						}
					} else {
						peer.Set(KeyProtocolCurrentBlockIndex, blockIndex+1)
					}
				}
			}()

			return nil
		})
	})
}
