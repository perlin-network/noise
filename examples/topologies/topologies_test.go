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

func setupRingNodes(startPort int) ([]int, map[string]map[string]struct{}) {
	var ports []int
	peers := map[string]map[string]struct{}{}
	numNodes := 4

	for i := 0; i < numNodes; i++ {
		ports = append(ports, startPort+i)
		addr := fmt.Sprintf("%s:%d", host, ports[i])

		// in a ring, each node is only connected to 2 others
		peers[addr] = map[string]struct{}{}
		peers[addr][fmt.Sprintf("%s:%d", host, ports[i]+(numNodes+i+1)%numNodes)] = struct{}{}
		peers[addr][fmt.Sprintf("%s:%d", host, ports[i]+(numNodes+i-1)%numNodes)] = struct{}{}
	}

	return ports, peers
}

func setupMeshNodes(startPort int) ([]int, map[string]map[string]struct{}) {
	var ports []int
	peers := map[string]map[string]struct{}{}

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

	for i, edge := range edges {
		ports = append(ports, startPort+edge.portOffset)
		addr := fmt.Sprintf("%s:%d", host, ports[i])

		peers[addr] = map[string]struct{}{}
		for _, po := range edge.peerOffsets {
			peers[addr][fmt.Sprintf("%s:%d", host, startPort+po)] = struct{}{}
		}
	}

	return ports, peers
}

func setupStarNodes(startPort int) ([]int, map[string]map[string]struct{}) {
	var ports []int
	peers := map[string]map[string]struct{}{}

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

	for i, edge := range edges {
		ports = append(ports, startPort+edge.portOffset)
		addr := fmt.Sprintf("%s:%d", host, ports[i])

		peers[addr] = map[string]struct{}{}
		for _, po := range edge.peerOffsets {
			peers[addr][fmt.Sprintf("%s:%d", host, startPort+po)] = struct{}{}
		}
	}

	return ports, peers
}

func setupFullyConnectedNodes(startPort int) ([]int, map[string]map[string]struct{}) {
	var ports []int
	peers := map[string]map[string]struct{}{}
	peerMap := map[string]struct{}{}
	numNodes := 5

	for i := 0; i < numNodes; i++ {
		ports = append(ports, startPort+i)

		peerMap[fmt.Sprintf("%s:%d", host, ports[i])] = struct{}{}
	}

	// got lazy, even connect to itself
	for i := 0; i < numNodes; i++ {
		addr := fmt.Sprintf("%s:%d", host, ports[i])
		peers[addr] = peerMap
	}

	return ports, peers
}

func setupLineNodes(startPort int) ([]int, map[string]map[string]struct{}) {
	var ports []int
	peers := map[string]map[string]struct{}{}
	numNodes := 5

	for i := 0; i < numNodes; i++ {
		ports = append(ports, startPort+i)
		addr := fmt.Sprintf("%s:%d", host, ports[i])

		peers[addr] = map[string]struct{}{}
		if i > 0 {
			peers[addr][fmt.Sprintf("%s:%d", host, ports[i]-1)] = struct{}{}
		}
		if i < numNodes-1 {
			peers[addr][fmt.Sprintf("%s:%d", host, ports[i]+1)] = struct{}{}
		}
	}

	return ports, peers
}

func setupTreeNodes(startPort int) ([]int, map[string]map[string]struct{}) {
	var ports []int
	peers := map[string]map[string]struct{}{}

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

	for i, edge := range edges {
		ports = append(ports, startPort+edge.portOffset)
		addr := fmt.Sprintf("%s:%d", host, ports[i])

		peers[addr] = map[string]struct{}{}
		for _, po := range edge.peerOffsets {
			peers[addr][fmt.Sprintf("%s:%d", host, startPort+po)] = struct{}{}
		}
	}

	return ports, peers
}

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

func bootstrapNodes(nodes []*network.Network, peers map[string]map[string]struct{}) error {
	for _, node := range nodes {
		if len(peers[node.Address()]) == 0 {
			continue
		}

		var peerList []string
		for k := range peers[node.Address()] {
			peerList = append(peerList, k)
		}

		// get nodes to start talking with each other
		node.Bootstrap(peerList...)

	}

	// Wait for all nodes to finish discovering other peers.
	time.Sleep(1000 * time.Millisecond)

	return nil
}

func broadcastTest(t *testing.T, nodes []*network.Network, processors []*tProcessor, sender int, peers map[string]map[string]struct{}) {
	timeout := 250 * time.Millisecond

	// Broadcast is an asynchronous call to send a message to other nodes
	expected := fmt.Sprintf("message from node %d", sender)
	nodes[sender].Broadcast(&messages.BasicMessage{Message: expected})

	// check the messages
	for i := 0; i < len(nodes); i++ {
		if _, isPeer := peers[nodes[i].Address()][nodes[sender].Address()]; !isPeer || i == sender {
			// if not a peer or not the sender, should not receive anything
			select {
			case received := <-processors[sender].Mailbox:
				t.Errorf("expected nothing in sending node %d, got %v", sender, received)
			case <-time.After(timeout):
				// this is the good case, don't want to receive anything
			}
		} else {
			// this is a connected peer, it should receive something
			select {
			case received := <-processors[i].Mailbox:
				// this is a receiving node, it should have just the one message buffered up
				if received.Message != expected {
					t.Errorf("expected message '%s' for node %d --> %d, but got %v", expected, sender, i, received)
				}
			case <-time.After(timeout):
				t.Errorf("expected a message for node %d --> %d, but it timed out", sender, i)
			}
		}
	}
}

func TestRing(t *testing.T) {
	//t.Parallel()

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
		broadcastTest(t, nodes, processors, i, peers)
	}

	// TODO: should close the connection to release the port
}

func TestMesh(t *testing.T) {
	//t.Parallel()

	// parse to flags to silence the glog library
	flag.Parse()

	var nodes []*network.Network
	var processors []*tProcessor
	var err error

	ports, peers := setupMeshNodes(5020)

	nodes, processors, err = setupNodes(ports)
	if err != nil {
		t.Fatal(err)
	}

	if err := bootstrapNodes(nodes, peers); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < len(nodes); i++ {
		broadcastTest(t, nodes, processors, i, peers)
	}

}

func TestStar(t *testing.T) {
	//t.Parallel()

	// parse to flags to silence the glog library
	flag.Parse()

	var nodes []*network.Network
	var processors []*tProcessor
	var err error

	ports, peers := setupStarNodes(5030)

	nodes, processors, err = setupNodes(ports)
	if err != nil {
		t.Fatal(err)
	}

	if err := bootstrapNodes(nodes, peers); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < len(nodes); i++ {
		broadcastTest(t, nodes, processors, i, peers)
	}
}

func TestFullyConnected(t *testing.T) {
	//t.Parallel()

	// parse to flags to silence the glog library
	flag.Parse()

	var nodes []*network.Network
	var processors []*tProcessor
	var err error

	ports, peers := setupFullyConnectedNodes(5040)

	nodes, processors, err = setupNodes(ports)
	if err != nil {
		t.Fatal(err)
	}

	if err := bootstrapNodes(nodes, peers); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < len(nodes); i++ {
		broadcastTest(t, nodes, processors, i, peers)
	}
}

func TestLine(t *testing.T) {
	//t.Parallel()

	// parse to flags to silence the glog library
	flag.Parse()

	var nodes []*network.Network
	var processors []*tProcessor
	var err error

	ports, peers := setupLineNodes(5050)

	nodes, processors, err = setupNodes(ports)
	if err != nil {
		t.Fatal(err)
	}

	if err := bootstrapNodes(nodes, peers); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < len(nodes); i++ {
		broadcastTest(t, nodes, processors, i, peers)
	}
}

func TestTree(t *testing.T) {
	//t.Parallel()

	// parse to flags to silence the glog library
	flag.Parse()

	var nodes []*network.Network
	var processors []*tProcessor
	var err error

	ports, peers := setupTreeNodes(5060)

	nodes, processors, err = setupNodes(ports)
	if err != nil {
		t.Fatal(err)
	}

	if err := bootstrapNodes(nodes, peers); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < len(nodes); i++ {
		broadcastTest(t, nodes, processors, i, peers)
	}
}
