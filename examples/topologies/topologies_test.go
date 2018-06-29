package topologies

import (
	"fmt"
	"time"

	"github.com/perlin-network/noise/crypto"
	"github.com/perlin-network/noise/examples/basic/messages"
	"github.com/perlin-network/noise/network"
	"github.com/perlin-network/noise/network/builders"
)

const (
	host = "127.0.0.1"
)

// TopologyProcessor implements the message handler
type TopologyProcessor struct {
	Mailbox chan *messages.BasicMessage
}

// Handle implements the network interface callback
func (n *TopologyProcessor) Handle(ctx *network.MessageContext) error {
	message := ctx.Message().(*messages.BasicMessage)
	n.Mailbox <- message
	return nil
}

// setupRingNodes setups the parameters of a ring topology
func setupRingNodes(startPort int) ([]int, map[string]map[string]struct{}) {
	var ports []int
	peers := map[string]map[string]struct{}{}
	numNodes := 4

	for i := 0; i < numNodes; i++ {
		ports = append(ports, startPort+i)
		addr := fmt.Sprintf("%s:%d", host, ports[i])

		// in a ring, each node is only connected to 2 others
		peers[addr] = map[string]struct{}{}
		peers[addr][fmt.Sprintf("%s:%d", host, startPort+(numNodes+i+1)%numNodes)] = struct{}{}
		peers[addr][fmt.Sprintf("%s:%d", host, startPort+(numNodes+i-1)%numNodes)] = struct{}{}
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

// setupNodes sets up the networks and processors
func setupNodes(ports []int) ([]*network.Network, []*TopologyProcessor, error) {
	var nodes []*network.Network
	var processors []*TopologyProcessor

	for _, port := range ports {
		builder := &builders.NetworkBuilder{}
		builder.SetKeys(crypto.RandomKeyPair())
		builder.SetHost(host)
		builder.SetPort(uint16(port))

		// excluding peer discovery to test non-fully connected topology
		//discovery.BootstrapPeerDiscovery(builder)

		processor := &TopologyProcessor{Mailbox: make(chan *messages.BasicMessage, 1)}
		builder.AddProcessor((*messages.BasicMessage)(nil), processor)

		node, err := builder.BuildNetwork()
		if err != nil {
			return nil, nil, err
		}
		nodes = append(nodes, node)
		processors = append(processors, processor)

		go node.Listen()
	}

	// make sure all the servers are listening
	for _, node := range nodes {
		node.BlockUntilListening()
	}

	return nodes, processors, nil
}

// bootstrapNodes will
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
	time.Sleep(time.Duration(100*len(nodes)) * time.Millisecond)

	return nil
}

// broadcastTest will broadcast a message from the sender node, checks if the right peers receive it
func broadcastTest(nodes []*network.Network, processors []*TopologyProcessor, peers map[string]map[string]struct{}, sender int) {
	timeout := 250 * time.Millisecond

	// Broadcast is an asynchronous call to send a message to other nodes
	expected := fmt.Sprintf("This is a broadcasted message from Node %d", sender)
	nodes[sender].Broadcast(&messages.BasicMessage{Message: expected})

	// check the messages
	for i := 0; i < len(nodes); i++ {
		if _, isPeer := peers[nodes[i].Address()][nodes[sender].Address()]; !isPeer || i == sender {
			// if not a peer or not the sender, should not receive anything
			select {
			case received := <-processors[sender].Mailbox:
				fmt.Printf("Expected nothing in sending node %d, got %v\n", sender, received)
			case <-time.After(timeout):
				// this is the good case, don't want to receive anything
			}
		} else {
			// this is a connected peer, it should receive something
			select {
			case received := <-processors[i].Mailbox:
				// this is a receiving node, it should have just the one message buffered up
				if received.Message != expected {
					fmt.Printf("Expected message '%s' for node %d --> %d, but got %v\n", expected, sender, i, received)
				}
			case <-time.After(timeout):
				fmt.Printf("Expected a message for node %d --> %d, but it timed out\n", sender, i)
			}
		}
	}
}

func ExampleRing() {
	var nodes []*network.Network
	var processors []*TopologyProcessor
	var err error

	// create the topology
	ports, peers := setupRingNodes(5010)

	// setup the cluster
	nodes, processors, err = setupNodes(ports)
	if err != nil {
		fmt.Println(err)
		return
	}

	// setup node connections
	if err := bootstrapNodes(nodes, peers); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("Nodes setup as a ring topology.")

	// have everyone send messages
	for i := 0; i < len(nodes); i++ {
		broadcastTest(nodes, processors, peers, i)
	}
	fmt.Printf("Messages sent from each node.")

	// Output:
	// Nodes setup as a ring topology.
	// Messages sent from each node.
}

func ExampleMesh() {
	var nodes []*network.Network
	var processors []*TopologyProcessor
	var err error

	ports, peers := setupMeshNodes(5020)

	nodes, processors, err = setupNodes(ports)
	if err != nil {
		fmt.Println(err)
		return
	}

	if err := bootstrapNodes(nodes, peers); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("Nodes setup as a mesh topology.")

	for i := 0; i < len(nodes); i++ {
		broadcastTest(nodes, processors, peers, i)
	}
	fmt.Printf("Messages sent from each node.")

	// Output:
	// Nodes setup as a mesh topology.
	// Messages sent from each node.
}

func ExampleStar() {
	var nodes []*network.Network
	var processors []*TopologyProcessor
	var err error

	ports, peers := setupStarNodes(5030)

	nodes, processors, err = setupNodes(ports)
	if err != nil {
		fmt.Println(err)
		return
	}

	if err := bootstrapNodes(nodes, peers); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("Nodes setup as a star topology.")

	for i := 0; i < len(nodes); i++ {
		broadcastTest(nodes, processors, peers, i)
	}
	fmt.Printf("Messages sent from each node.")

	// Output:
	// Nodes setup as a star topology.
	// Messages sent from each node.
}

func ExampleFullyConnected() {
	var nodes []*network.Network
	var processors []*TopologyProcessor
	var err error

	ports, peers := setupFullyConnectedNodes(5040)

	nodes, processors, err = setupNodes(ports)
	if err != nil {
		fmt.Println(err)
		return
	}

	if err := bootstrapNodes(nodes, peers); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("Nodes setup as a fully connected topology.")

	for i := 0; i < len(nodes); i++ {
		broadcastTest(nodes, processors, peers, i)
	}
	fmt.Printf("Messages sent from each node.")

	// Output:
	// Nodes setup as a fully connected topology.
	// Messages sent from each node.
}

func ExampleLine() {
	var nodes []*network.Network
	var processors []*TopologyProcessor
	var err error

	ports, peers := setupLineNodes(5050)

	nodes, processors, err = setupNodes(ports)
	if err != nil {
		fmt.Println(err)
		return
	}

	if err := bootstrapNodes(nodes, peers); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("Nodes setup as a line topology.")

	for i := 0; i < len(nodes); i++ {
		broadcastTest(nodes, processors, peers, i)
	}
	fmt.Printf("Messages sent from each node.")

	// Output:
	// Nodes setup as a line topology.
	// Messages sent from each node.
}

func ExampleTree() {
	var nodes []*network.Network
	var processors []*TopologyProcessor
	var err error

	ports, peers := setupTreeNodes(5060)

	nodes, processors, err = setupNodes(ports)
	if err != nil {
		fmt.Println(err)
		return
	}

	if err := bootstrapNodes(nodes, peers); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("Nodes setup as a tree topology.")

	for i := 0; i < len(nodes); i++ {
		broadcastTest(nodes, processors, peers, i)
	}
	fmt.Printf("Messages sent from each node.")

	// Output:
	// Nodes setup as a tree topology.
	// Messages sent from each node.
}
