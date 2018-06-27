package topologies

import (
	"flag"
	"fmt"
	"testing"
	"time"

	"github.com/perlin-network/noise/examples/basic"
	"github.com/perlin-network/noise/examples/basic/messages"
)

const (
	host      = "localhost"
	startPort = 5000
)

func setupRingNodes() []*TopoNode {
	numNodes := 4
	var nodes []*TopoNode

	for i := 0; i < numNodes; i++ {
		node := &TopoNode{}
		node.h = host
		node.p = startPort + i

		// in a ring, each node is only connected to 2 others
		node.ps = append(node.ps, fmt.Sprintf("%s:%d", node.h, (node.p+1)%(startPort+numNodes)))
		node.ps = append(node.ps, fmt.Sprintf("%s:%d", node.h, (node.p-1)%(startPort+numNodes)))

		nodes = append(nodes, node)
	}

	return nodes
}

func setupMeshNodes() []*TopoNode {
	var nodes []*TopoNode

	edges := []struct {
		portOffset  int
		peerOffsets []int
	}{
		{portOffset: 0, peerOffsets: []int{1}},
		{portOffset: 1, peerOffsets: []int{0, 5, 2}},
		{portOffset: 2, peerOffsets: []int{1, 3, 5}},
		{portOffset: 3, peerOffsets: []int{2, 4}},
		{portOffset: 4, peerOffsets: []int{3, 5}},
		{portOffset: 5, peerOffsets: []int{1, 2, 4}},
	}

	for _, edge := range edges {
		node := &TopoNode{}
		node.h = host
		node.p = startPort + edge.portOffset

		nodes = append(nodes, node)

		for _, po := range edge.peerOffsets {
			node.ps = append(node.ps, fmt.Sprintf("%s:%d", node.h, startPort+po))
		}
	}

	return nodes
}

func setupStarNodes() []*TopoNode {
	var nodes []*TopoNode

	edges := []struct {
		portOffset  int
		peerOffsets []int
	}{
		{portOffset: 0, peerOffsets: []int{1, 2, 3, 4}},
		{portOffset: 1, peerOffsets: []int{0}},
		{portOffset: 2, peerOffsets: []int{0}},
		{portOffset: 3, peerOffsets: []int{0}},
		{portOffset: 4, peerOffsets: []int{0}},
	}

	for _, edge := range edges {
		node := &TopoNode{}
		node.h = host
		node.p = startPort + edge.portOffset

		nodes = append(nodes, node)

		for _, po := range edge.peerOffsets {
			node.ps = append(node.ps, fmt.Sprintf("%s:%d", node.h, startPort+po))
		}
	}

	return nodes
}

func setupFullyConnectedNodes() []*TopoNode {
	var nodes []*TopoNode
	var peers []string
	numNodes := 5

	for i := 0; i < numNodes; i++ {
		node := &TopoNode{}
		node.h = host
		node.p = startPort + i

		nodes = append(nodes, node)
		peers = append(peers, fmt.Sprintf("%s:%d", node.h, node.p))
	}

	// got lazy, even connect to itself
	for _, node := range nodes {
		node.ps = peers
	}

	return nodes
}

func setupLineNodes() []*TopoNode {
	var nodes []*TopoNode
	numNodes := 5

	for i := 0; i < numNodes; i++ {
		node := &TopoNode{}
		node.h = host
		node.p = startPort + i

		nodes = append(nodes, node)

		if i > 0 {
			node.ps = append(node.ps, fmt.Sprintf("%s:%d", node.h, node.p-1))
		}
		if i < numNodes-1 {
			node.ps = append(node.ps, fmt.Sprintf("%s:%d", node.h, node.p+1))
		}
	}

	return nodes
}

func setupTreeNodes() []*TopoNode {
	var nodes []*TopoNode

	edges := []struct {
		portOffset  int
		peerOffsets []int
	}{
		{portOffset: 0, peerOffsets: []int{1, 3}},
		{portOffset: 1, peerOffsets: []int{0, 2}},
		{portOffset: 2, peerOffsets: []int{1}},
		{portOffset: 3, peerOffsets: []int{0, 4, 5}},
		{portOffset: 4, peerOffsets: []int{3}},
		{portOffset: 5, peerOffsets: []int{3}},
	}

	for _, edge := range edges {
		node := &TopoNode{}
		node.h = host
		node.p = startPort + edge.portOffset

		nodes = append(nodes, node)

		for _, po := range edge.peerOffsets {
			node.ps = append(node.ps, fmt.Sprintf("%s:%d", node.h, startPort+po))
		}
	}

	return nodes
}

func topoNode2ClusterNode(nodes []*TopoNode) []basic.ClusterNode {
	var cluster []basic.ClusterNode
	for _, node := range nodes {
		cluster = append(cluster, node)
	}
	return cluster
}

func bootstrapNodes(nodes []*TopoNode) error {
	for i, node := range nodes {
		if node.Net() == nil {
			return fmt.Errorf("expected %d nodes, but node %d is missing a network", len(nodes), i)
		}

		// get nodes to start talking with each other
		node.Net().Bootstrap(node.Peers()...)

		// TODO: seems there's another race condition with Bootstrap, use a sleep for now
		time.Sleep(1 * time.Second)
	}
	return nil
}

func TestRing(t *testing.T) {
	// parse to flags to silence the glog library
	flag.Parse()

	nodes := setupRingNodes()

	if err := basic.SetupCluster(topoNode2ClusterNode(nodes)); err != nil {
		t.Fatal(err)
	}

	if err := bootstrapNodes(nodes); err != nil {
		t.Fatal(err)
	}

	// Broadcast is an asynchronous call to send a message to other nodes
	testMessage := "message from node 0"
	nodes[0].Net().Broadcast(&messages.BasicMessage{Message: testMessage})

	// Simplificiation: message broadcasting is asynchronous, so need the messages to settle
	time.Sleep(1 * time.Second)

	// check if you can send a message from node 1 and will it be received only in node 2,3
	if result := nodes[0].PopMessage(); result != nil {
		t.Errorf("expected nothing in node 0, got %v", result)
	}
	if len(nodes[0].Messages) > 0 {
		t.Errorf("expected no messages buffered in node 0, found: %v", nodes[0].Messages)
	}
	for i := 1; i < len(nodes); i++ {
		if result := nodes[i].PopMessage(); result == nil {
			t.Errorf("expected a message in node %d but it was blank", i)
		} else {
			if result.Message != testMessage {
				t.Errorf("expected message %s in node %d but got %v", testMessage, i, result)
			}
		}
		if len(nodes[i].Messages) > 0 {
			t.Errorf("expected no messages buffered in node %d, found: %v", i, nodes[i].Messages)
		}
	}
}
