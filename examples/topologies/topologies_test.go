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
	host = "localhost"
)

// tNode holds the variables to create the network and implements the message handler
type tNode struct {
	Host    string
	Port    int
	Peers   []string
	Net     *network.Network
	Mailbox chan *messages.BasicMessage
}

// Handle implements the network interface callback
func (n *tNode) Handle(ctx *network.MessageContext) error {
	message := ctx.Message().(*messages.BasicMessage)
	n.Mailbox <- message
	return nil
}

func setupRingNodes(startPort int) []*tNode {
	numNodes := 4
	var nodes []*tNode

	for i := 0; i < numNodes; i++ {
		node := &tNode{}
		node.Host = host
		node.Port = startPort + i

		// in a ring, each node is only connected to 2 others
		//node.Peers = append(node.Peers, fmt.Sprintf("%s:%d", node.Host, (node.Port+1)%(startPort+numNodes)))
		//node.Peers = append(node.Peers, fmt.Sprintf("%s:%d", node.Host, (node.Port-1)%(startPort+numNodes)))

		// only connect to the first node for now
		if i > 0 {
			node.Peers = append(node.Peers, fmt.Sprintf("%s:%d", node.Host, startPort))
		}

		nodes = append(nodes, node)
	}

	return nodes
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

// setupCluster sets up a connected group of nodes in a cluster.
func setupCluster(nodes []*tNode) error {
	for _, node := range nodes {
		builder := &builders.NetworkBuilder{}
		builder.SetKeys(crypto.RandomKeyPair())
		builder.SetHost(node.Host)
		builder.SetPort(uint16(node.Port))

		discovery.BootstrapPeerDiscovery(builder)

		builder.AddProcessor((*messages.BasicMessage)(nil), node)

		net, err := builder.BuildNetwork()
		if err != nil {
			return err
		}
		node.Net = net

		go net.Listen()

		// TODO: seems there's another race condition with Bootstrap, use a sleep for now
		time.Sleep(1000 * time.Millisecond)
	}

	// Wait for all nodes to finish discovering other peers.
	//time.Sleep(1 * time.Second)

	return nil
}

func bootstrapNodes(nodes []*tNode) error {
	for i, node := range nodes {
		if node.Net == nil {
			return fmt.Errorf("expected %d nodes, but node %d is missing a network", len(nodes), i)
		}

		if len(node.Peers) == 0 {
			continue
		}

		// get nodes to start talking with each other
		node.Net.Bootstrap(node.Peers...)

		// TODO: seems there's another race condition with Bootstrap, use a sleep for now
		time.Sleep(1000 * time.Millisecond)
	}
	return nil
}

func broadcastTest(t *testing.T, nodes []*tNode, sender int) {
	// Broadcast is an asynchronous call to send a message to other nodes
	expected := fmt.Sprintf("message from node %d", sender)
	nodes[sender].Net.Broadcast(&messages.BasicMessage{Message: expected})

	// make sure the sender didn't get his own message
	{
		select {
		case received := <-nodes[sender].Mailbox:
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
		case received := <-nodes[i].Mailbox:
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

	// glog defaults to logging to a file, override this flag to log to console for testing
	flag.Set("logtostderr", "true")

	// parse to flags to silence the glog library
	flag.Parse()

	nodes := setupRingNodes(5010)

	if err := setupCluster(nodes); err != nil {
		t.Fatal(err)
	}

	if err := bootstrapNodes(nodes); err != nil {
		t.Fatal(err)
	}

	broadcastTest(t, nodes, 0)
	/*
		for i := 0; i < len(nodes); i++ {
			broadcastNode(t, nodes, i)
		}
	*/

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
