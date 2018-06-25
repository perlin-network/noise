package main

import (
	"flag"
	"fmt"
	"testing"
	"time"

	"github.com/golang/glog"
	"github.com/perlin-network/noise/crypto"
	"github.com/perlin-network/noise/examples/clusters/messages"
	"github.com/perlin-network/noise/grpc_utils"
	"github.com/perlin-network/noise/network"
	"github.com/perlin-network/noise/network/builders"
	"github.com/perlin-network/noise/network/discovery"
)

type ClusterNode struct {
	Host string
	Port int
	Net  *network.Network
}

func (c *ClusterNode) Handle(client *network.PeerClient, raw *network.IncomingMessage) error {
	message := raw.Message.(*messages.ClusterTestMessage)

	glog.Infof("<%s> %s", client.Id.Address, message.Message)

	return nil
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
			return fmt.Errorf("Error: port was not available, cannot bootstrap peers, err=%+v", err)
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
	if len(nodes) != cluster1NumPorts {
		t.Errorf("Should have only %d nodes, but had %d", len(nodes), cluster1NumPorts)
	}

	if err := setupCluster(t, nodes); err != nil {
		t.Fatal(err)
	}

	// TODO
}
