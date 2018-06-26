package basic

import (
	"fmt"
	"time"

	"github.com/perlin-network/noise/crypto"
	"github.com/perlin-network/noise/examples/basic/messages"
	"github.com/perlin-network/noise/grpc_utils"
	"github.com/perlin-network/noise/network"
	"github.com/perlin-network/noise/network/builders"
	"github.com/perlin-network/noise/network/discovery"
)

type ClusterNode struct {
	Host     string
	Port     int
	Peers    []string
	Net      *network.Network
	Messages []*messages.BasicMessage
}

// Handle implements the network interface callback
func (c *ClusterNode) Handle(client *network.PeerClient, raw *network.IncomingMessage) error {
	message := raw.Message.(*messages.BasicMessage)

	c.Messages = append(c.Messages, message)

	return nil
}

// PopMessage returns the oldest message from it's buffer and removes it from the list
func (c *ClusterNode) PopMessage() *messages.BasicMessage {
	if len(c.Messages) == 0 {
		return nil
	}
	var retVal *messages.BasicMessage
	retVal, c.Messages = c.Messages[0], c.Messages[1:]
	return retVal
}

var blockTimeout = 10 * time.Second

// SetupCluster sets up a connected group of nodes in a cluster.
func SetupCluster(nodes []*ClusterNode) error {
	for i := 0; i < len(nodes); i++ {
		node := nodes[i]
		keys := crypto.RandomKeyPair()

		builder := &builders.NetworkBuilder{}
		builder.SetKeys(keys)
		builder.SetHost(node.Host)
		builder.SetPort(node.Port)

		discovery.BootstrapPeerDiscovery(builder)

		builder.AddProcessor((*messages.BasicMessage)(nil), nodes[i])

		net, err := builder.BuildNetwork()
		if err != nil {
			return err
		}
		node.Net = net

		go net.Listen()
	}

	for i := 0; i < len(nodes); i++ {
		if err := grpc_utils.BlockUntilConnectionReady(nodes[i].Host, nodes[i].Port, blockTimeout); err != nil {
			return fmt.Errorf("port was not available, cannot bootstrap node %d peers: %+v", i, err)
		}
	}

	for _, node := range nodes {
		node.Net.Bootstrap(node.Peers...)

		// TODO: seems there's another race condition with Bootstrap, use a sleep for now
		time.Sleep(1 * time.Second)
	}

	return nil
}
