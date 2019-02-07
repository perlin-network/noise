package basic

import (
	"github.com/perlin-network/noise"
	"github.com/perlin-network/noise/protocol"
)

var (
	_ protocol.CipherPolicy = (*policy)(nil)
)

type policy struct{}

func New() *policy {
	return &policy{}
}

func (p *policy) EnforceCipherPolicy(node *noise.Node) {

}

func (p *policy) Encrypt(peer *noise.Peer, buf []byte) ([]byte, error) {
	return buf, nil
}

func (p *policy) Decrypt(peer *noise.Peer, buf []byte) ([]byte, error) {
	return buf, nil
}
