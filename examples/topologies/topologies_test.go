package topologies

import (
	"flag"
	"fmt"
	"testing"
	"time"

	"github.com/perlin-network/noise/crypto"
	"github.com/perlin-network/noise/examples/basic/messages"
	"github.com/perlin-network/noise/network"
	"github.com/perlin-network/noise/network/builders"
	"github.com/perlin-network/noise/network/discovery"
)

const (
	host = "127.0.0.1"
)

// tProcessor implements the message handler
type tProcessor struct {
	Mailbox chan *messages.BasicMessage
}

// Handle implements the network interface callback
func (n *tProcessor) Handle(ctx *network.MessageContext) error {
	message := ctx.Message().(*messages.BasicMessage)
	n.Mailbox <- message
	return nil
}

func setupRingNodes(startPort int) ([]int, map[string][]string) {
	numNodes := 4
	var ports []int
	peers := map[string][]string{}

	for i := 0; i < numNodes; i++ {
		ports = append(ports, startPort+i)
		addr := fmt.Sprintf("%s:%d", host, ports[i])

		// in a ring, each node is only connected to 2 others
		peers[addr] = []string{}
		peers[addr] = append(peers[addr], fmt.Sprintf("%s:%d", host, ports[i]+(numNodes+i+1)%numNodes))
		peers[addr] = append(peers[addr], fmt.Sprintf("%s:%d", host, ports[i]+(numNodes+i-1)%numNodes))
	}

	return ports, peers
}

/*

func setupMeshNodes(startPort int) []*tNode {
	var nodes []*tNode

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
		node := &tNode{}
		node.Host = host
		node.Port = startPort + edge.portOffset

		nodes = append(nodes, node)

		for _, po := range edge.peerOffsets {
			node.Peers = append(node.Peers, fmt.Sprintf("%s:%d", node.Host, startPort+po))
		}
	}

	return nodes
}

func setupStarNodes(startPort int) []*tNode {
	var nodes []*tNode

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
		node := &tNode{}
		node.Host = host
		node.Port = startPort + edge.portOffset

		nodes = append(nodes, node)

		for _, po := range edge.peerOffsets {
			node.Peers = append(node.Peers, fmt.Sprintf("%s:%d", node.Host, startPort+po))
		}
	}

	return nodes
}

func setupFullyConnectedNodes(startPort int) []*tNode {
	var nodes []*tNode
	var peers []string
	numNodes := 5

	for i := 0; i < numNodes; i++ {
		node := &tNode{}
		node.Host = host
		node.Port = startPort + i

		nodes = append(nodes, node)
		peers = append(peers, fmt.Sprintf("%s:%d", node.Host, node.Port))
	}

	// got lazy, even connect to itself
	for _, node := range nodes {
		node.Peers = peers
	}

	return nodes
}

func setupLineNodes(startPort int) []*tNode {
	var nodes []*tNode
	numNodes := 5

	for i := 0; i < numNodes; i++ {
		node := &tNode{}
		node.Host = host
		node.Port = startPort + i

		nodes = append(nodes, node)

		if i > 0 {
			node.Peers = append(node.Peers, fmt.Sprintf("%s:%d", node.Host, node.Port-1))
		}
		if i < numNodes-1 {
			node.Peers = append(node.Peers, fmt.Sprintf("%s:%d", node.Host, node.Port+1))
		}
	}

	return nodes
}

func setupTreeNodes(startPort int) []*tNode {
	var nodes []*tNode

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
		node := &tNode{}
		node.Host = host
		node.Port = startPort + edge.portOffset

		nodes = append(nodes, node)

		for _, po := range edge.peerOffsets {
			node.Peers = append(node.Peers, fmt.Sprintf("%s:%d", node.Host, startPort+po))
		}
	}

	return nodes
}

*/

// setupNodes sets up a connected group of nodes in a cluster.
func setupNodes(ports []int) ([]*network.Network, []*tProcessor, error) {
	var nodes []*network.Network
	var processors []*tProcessor

	for _, port := range ports {
		builder := &builders.NetworkBuilder{}
		builder.SetKeys(crypto.RandomKeyPair())
		builder.SetHost(host)
		builder.SetPort(uint16(port))

		discovery.BootstrapPeerDiscovery(builder)

		processor := &tProcessor{Mailbox: make(chan *messages.BasicMessage, 1)}
		builder.AddProcessor((*messages.BasicMessage)(nil), processor)

		node, err := builder.BuildNetwork()
		if err != nil {
			return nil, nil, err
		}
		nodes = append(nodes, node)
		processors = append(processors, processor)

		go node.Listen()
	}

	//for _, node := range nodes {
	//	node.BlockUntilListening()
	//}
	// Wait for all nodes to finish discovering other peers.
	time.Sleep(1000 * time.Millisecond)

	return nodes, processors, nil
}

func bootstrapNodes(nodes []*network.Network, peers map[string][]string) error {
	for _, node := range nodes {
		if len(peers[node.Address()]) == 0 {
			continue
		}

		// get nodes to start talking with each other
		node.Bootstrap(peers[node.Address()]...)

	}

	// Wait for all nodes to finish discovering other peers.
	time.Sleep(1000 * time.Millisecond)

	return nil
}

func broadcastTest(t *testing.T, nodes []*network.Network, processors []*tProcessor, sender int) {
	// Broadcast is an asynchronous call to send a message to other nodes
	expected := fmt.Sprintf("message from node %d", sender)
	nodes[sender].Broadcast(&messages.BasicMessage{Message: expected})

	// make sure the sender didn't get his own message
	{
		select {
		case received := <-processors[sender].Mailbox:
			t.Errorf("expected nothing in sending node %d, got %v", sender, received)
		case <-time.After(1 * time.Second):
			// this is the good case, don't want to receive anything
		}
	}

	// check the messages
	for i := 0; i < len(nodes); i++ {
		if i == sender {
			// sender is checked after
			continue
		}
		select {
		case received := <-processors[i].Mailbox:
			// this is a receiving node, it should have just the one message buffered up
			if received.Message != expected {
				t.Errorf("expected message '%s' for node %d --> %d, but got %v", expected, sender, i, received)
			}
		case <-time.After(3 * time.Second):
			t.Errorf("expected a message for node %d --> %d, but it timed out", sender, i)
		}
	}
}

func TestRing(t *testing.T) {
	t.Parallel()

	// parse to flags to silence the glog library
	flag.Parse()

	var nodes []*network.Network
	var processors []*tProcessor
	var err error

	ports, peers := setupRingNodes(5010)

	nodes, processors, err = setupNodes(ports)
	if err != nil {
		t.Fatal(err)
	}

	if err := bootstrapNodes(nodes, peers); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < len(nodes); i++ {
		broadcastTest(t, nodes, processors, i)
	}

	// TODO: should close the connection to release the port
}

/*
func TestMesh(t *testing.T) {
	t.Parallel()

	// parse to flags to silence the glog library
	flag.Parse()

	nodes := setupMeshNodes(5020)

	if err := setupCluster(nodes); err != nil {
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

	if err := setupCluster(nodes); err != nil {
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

	if err := setupCluster(nodes); err != nil {
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

	if err := setupCluster(nodes); err != nil {
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

	if err := setupCluster(nodes); err != nil {
		t.Fatal(err)
	}

	if err := bootstrapNodes(nodes); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < len(nodes); i++ {
		broadcastNode(t, nodes, i)
	}
}
*/
