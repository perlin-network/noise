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
	host = "localhost"
)

func setupRingNodes(startPort int) []*TopoNode {
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

func setupMeshNodes(startPort int) []*TopoNode {
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

func setupStarNodes(startPort int) []*TopoNode {
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

func setupFullyConnectedNodes(startPort int) []*TopoNode {
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

func setupLineNodes(startPort int) []*TopoNode {
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

func setupTreeNodes(startPort int) []*TopoNode {
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

func broadcastNode(t *testing.T, nodes []*TopoNode, sendingNodeIdx int) {
	// Broadcast is an asynchronous call to send a message to other nodes
	testMessage := fmt.Sprintf("message from node %d", sendingNodeIdx)
	nodes[sendingNodeIdx].Net().Broadcast(&messages.BasicMessage{Message: testMessage})

	// TODO: remove this simplificiation: message broadcasting is asynchronous, so need the messages to settle
	time.Sleep(1 * time.Second)

	// check the messages
	for i := 1; i < len(nodes); i++ {
		if i == sendingNodeIdx {
			// this is the sending node, it should not have received it's own message
			if result := nodes[sendingNodeIdx].PopMessage(); result != nil {
				t.Errorf("expected nothing in sending node %d, got %v", sendingNodeIdx, result)
			}
			if len(nodes[sendingNodeIdx].Messages) > 0 {
				t.Errorf("expected no messages buffered in sending node %d, found: %v", sendingNodeIdx, nodes[0].Messages)
			}
		} else {
			// this is a receiving node, it should have just the one message buffered up
			if result := nodes[i].PopMessage(); result == nil {
				t.Errorf("expected a message in (sending node %d) receiving node %d but it was blank", sendingNodeIdx, i)
			} else {
				if result.Message != testMessage {
					t.Errorf("expected message %s in (sending node %d) receiving node %d but got %v", testMessage, sendingNodeIdx, i, result)
				}
			}
			if len(nodes[i].Messages) > 0 {
				t.Errorf("expected no messages buffered in (sending node %d) receiving node %d, found: %v", sendingNodeIdx, i, nodes[i].Messages)
			}
		}
	}
}

func TestRing(t *testing.T) {
	t.Parallel()

	// parse to flags to silence the glog library
	flag.Parse()

	nodes := setupRingNodes(5010)

	if err := basic.SetupCluster(topoNode2ClusterNode(nodes)); err != nil {
		t.Fatal(err)
	}

	if err := bootstrapNodes(nodes); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < len(nodes); i++ {
		broadcastNode(t, nodes, i)
	}

	// TODO: should close the connection to release the port
}

func TestMesh(t *testing.T) {
	t.Parallel()

	// TODO: this one times out in network discovery sometimes
	return

	// parse to flags to silence the glog library
	flag.Parse()

	nodes := setupMeshNodes(5020)

	if err := basic.SetupCluster(topoNode2ClusterNode(nodes)); err != nil {
		t.Fatal(err)
	}

	if err := bootstrapNodes(nodes); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < len(nodes); i++ {
		broadcastNode(t, nodes, i)
	}
}

func TestStar(t *testing.T) {
	t.Parallel()

	// parse to flags to silence the glog library
	flag.Parse()

	nodes := setupStarNodes(5030)

	if err := basic.SetupCluster(topoNode2ClusterNode(nodes)); err != nil {
		t.Fatal(err)
	}

	if err := bootstrapNodes(nodes); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < len(nodes); i++ {
		broadcastNode(t, nodes, i)
	}
}

func TestFullyConnected(t *testing.T) {
	t.Parallel()

	// parse to flags to silence the glog library
	flag.Parse()

	nodes := setupFullyConnectedNodes(5040)

	if err := basic.SetupCluster(topoNode2ClusterNode(nodes)); err != nil {
		t.Fatal(err)
	}

	if err := bootstrapNodes(nodes); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < len(nodes); i++ {
		broadcastNode(t, nodes, i)
	}
}

func TestLine(t *testing.T) {
	t.Parallel()

	// parse to flags to silence the glog library
	flag.Parse()

	nodes := setupLineNodes(5050)

	if err := basic.SetupCluster(topoNode2ClusterNode(nodes)); err != nil {
		t.Fatal(err)
	}

	if err := bootstrapNodes(nodes); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < len(nodes); i++ {
		broadcastNode(t, nodes, i)
	}
}

func TestTree(t *testing.T) {
	t.Parallel()

	// parse to flags to silence the glog library
	flag.Parse()

	nodes := setupTreeNodes(5060)

	if err := basic.SetupCluster(topoNode2ClusterNode(nodes)); err != nil {
		t.Fatal(err)
	}

	if err := bootstrapNodes(nodes); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < len(nodes); i++ {
		broadcastNode(t, nodes, i)
	}
}
