package basic

import (
	"time"

	"github.com/perlin-network/noise/crypto"
	"github.com/perlin-network/noise/examples/basic/messages"
	"github.com/perlin-network/noise/network"
	"github.com/perlin-network/noise/network/builders"
	"github.com/perlin-network/noise/network/discovery"
)

// ClusterNode holds the network and message handler for each node
type ClusterNode interface {
	Host() string
	Port() uint16
	Peers() []string
	Net() *network.Network
	SetNet(*network.Network)
	Handle(ctx *network.MessageContext) error
}

var blockTimeout = 10 * time.Second

// SetupCluster sets up a connected group of nodes in a cluster.
func SetupCluster(nodes []ClusterNode) error {
	for i := 0; i < len(nodes); i++ {
		node := nodes[i]
		keys := crypto.RandomKeyPair()

		builder := &builders.NetworkBuilder{}
		builder.SetKeys(keys)
		builder.SetHost(node.Host())
		builder.SetPort(node.Port())

		discovery.BootstrapPeerDiscovery(builder)

		builder.AddProcessor((*messages.BasicMessage)(nil), nodes[i])

		net, err := builder.BuildNetwork()
		if err != nil {
			return err
		}
		node.SetNet(net)

		go net.Listen()
	}

	return nil
}
