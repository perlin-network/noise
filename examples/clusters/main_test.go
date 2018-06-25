package main

import (
	"flag"
	"fmt"
	"testing"
	"time"

	"github.com/perlin-network/noise/crypto"
	"github.com/perlin-network/noise/examples/clusters/messages"
	"github.com/perlin-network/noise/grpc_utils"
	"github.com/perlin-network/noise/network"
	"github.com/perlin-network/noise/network/builders"
	"github.com/perlin-network/noise/network/discovery"
)

type ClusterNode struct {
	Host             string
	Port             int
	Net              *network.Network
	BufferedMessages []*messages.ClusterTestMessage
}

func (c *ClusterNode) Handle(client *network.PeerClient, raw *network.IncomingMessage) error {
	message := raw.Message.(*messages.ClusterTestMessage)

	if c.BufferedMessages == nil {
		c.BufferedMessages = []*messages.ClusterTestMessage{}
	}
	c.BufferedMessages = append(c.BufferedMessages, message)

	return nil
}

func (c *ClusterNode) PopMessage() *messages.ClusterTestMessage {
	if len(c.BufferedMessages) == 0 {
		return nil
	}
	var retVal *messages.ClusterTestMessage
	retVal, c.BufferedMessages = c.BufferedMessages[0], c.BufferedMessages[1:]
	return retVal
}

var blockTimeout = 10 * time.Second

func setupCluster(t *testing.T, nodes []*ClusterNode) error {
	peers := []string{}

	for i := 0; i < len(nodes); i++ {
		node := nodes[i]
		keys := crypto.RandomKeyPair()
		peers = append(peers, fmt.Sprintf("%s:%d", node.Host, node.Port))

		builder := &builders.NetworkBuilder{}
		builder.SetKeys(keys)
		builder.SetHost(node.Host)
		builder.SetPort(node.Port)

		discovery.BootstrapPeerDiscovery(builder)

		builder.AddProcessor((*messages.ClusterTestMessage)(nil), nodes[i])

		net, err := builder.BuildNetwork()
		if err != nil {
			return err
		}
		node.Net = net

		go net.Listen()
	}

	for i := 0; i < len(nodes); i++ {
		if err := grpc_utils.BlockUntilConnectionReady(nodes[i].Host, nodes[i].Port, blockTimeout); err != nil {
			return fmt.Errorf("Error: port was not available, cannot bootstrap node %d peers, err=%+v", i, err)
		}
	}

	for i := 0; i < len(nodes); i++ {
		nodes[i].Net.Bootstrap(peers...)
	}

	return nil
}

func TestClusters(t *testing.T) {
	flag.Parse()

	host := "localhost"
	cluster1StartPort := 3001
	cluster1NumPorts := 3
	nodes := []*ClusterNode{}

	for i := 0; i < cluster1NumPorts; i++ {
		node := &ClusterNode{}
		node.Host = host
		node.Port = cluster1StartPort + i

		nodes = append(nodes, node)
	}

	if err := setupCluster(t, nodes); err != nil {
		t.Fatal(err)
	}

	for i, node := range nodes {
		if node.Net == nil {
			t.Fatalf("Expected %d nodes, but node %d is missing a network", len(nodes), i)
		}
	}

	// check if you can send a message from node 1 and will it be received only in node 2,3
	{
		testMessage := "message from node 0"
		nodes[0].Net.Broadcast(&messages.ClusterTestMessage{Message: testMessage})

		// HACK: TODO: replace sleep with something else
		time.Sleep(1 * time.Second)

		if result := nodes[0].PopMessage(); result != nil {
			t.Errorf("Expected nothing in node 0, got %v", result)
		}
		for i := 1; i < len(nodes); i++ {
			if result := nodes[i].PopMessage(); result == nil {
				t.Errorf("Expected a message in node %d but it was blank", i)
			} else {
				if result.Message != testMessage {
					t.Errorf("Expected message %s in node %d but got %v", testMessage, i, result)
				}
			}
		}
	}
}
