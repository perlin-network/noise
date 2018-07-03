package discovery

import (
	"github.com/golang/glog"
	"github.com/perlin-network/noise/crypto"
	"github.com/perlin-network/noise/network"
	"github.com/perlin-network/noise/protobuf"
)

var (
	keys     = crypto.RandomKeyPair()
	host     = "localhost"
	protocol = "kcp"
	port     = 12345
)

// MockPlugin handles handshake requests only.
type MockPlugin struct{ *network.Plugin }

func (p *MockPlugin) Handle(ctx *network.MessageContext) error {
	switch ctx.Message().(type) {
	case *protobuf.Ping:
		// Send handshake response to peer.
		err := ctx.Reply(&protobuf.Pong{})

		if err != nil {
			glog.Error(err)
			return err
		}
	}

	return nil
}

// TODO.
