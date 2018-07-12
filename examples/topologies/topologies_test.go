package topologies

import (
	"testing"

	"github.com/perlin-network/noise/network"
)

const basePort = 19700

func TestRing(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("skipping ring test in short mode")
	}

	var nodes []*network.Network
	var processors []*MockPlugin
	var err error

	// create the topology
	ports, peers := setupRingNodes(basePort)

	// setup the cluster
	nodes, processors, err = setupNodes(ports)
	if err != nil {
		t.Fatal(err)
	}

	// setup node connections
	if err := bootstrapNodes(nodes, peers); err != nil {
		t.Fatal(err)
	}

	// have everyone send messages
	for i := 0; i < len(nodes); i++ {
		broadcastTest(t, nodes, processors, peers, i)
	}
}

func TestMesh(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("skipping mesh test in short mode")
	}

	var nodes []*network.Network
	var processors []*MockPlugin
	var err error

	ports, peers := setupMeshNodes(basePort + 10)

	nodes, processors, err = setupNodes(ports)
	if err != nil {
		t.Fatal(err)
	}

	if err := bootstrapNodes(nodes, peers); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < len(nodes); i++ {
		broadcastTest(t, nodes, processors, peers, i)
	}
}

func TestStar(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("skipping star test in short mode")
	}

	var nodes []*network.Network
	var processors []*MockPlugin
	var err error

	ports, peers := setupStarNodes(basePort + 20)

	nodes, processors, err = setupNodes(ports)
	if err != nil {
		t.Fatal(err)
	}

	if err := bootstrapNodes(nodes, peers); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < len(nodes); i++ {
		broadcastTest(t, nodes, processors, peers, i)
	}
}

func TestFullyConnected(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("skipping fully connected test in short mode")
	}

	var nodes []*network.Network
	var processors []*MockPlugin
	var err error

	ports, peers := setupFullyConnectedNodes(basePort + 30)

	nodes, processors, err = setupNodes(ports)
	if err != nil {
		t.Fatal(err)
	}

	if err := bootstrapNodes(nodes, peers); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < len(nodes); i++ {
		broadcastTest(t, nodes, processors, peers, i)
	}
}

func TestLine(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("skipping line test in short mode")
	}

	var nodes []*network.Network
	var processors []*MockPlugin
	var err error

	ports, peers := setupLineNodes(basePort + 40)

	nodes, processors, err = setupNodes(ports)
	if err != nil {
		t.Fatal(err)
	}

	if err := bootstrapNodes(nodes, peers); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < len(nodes); i++ {
		broadcastTest(t, nodes, processors, peers, i)
	}
}

func TestTree(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("skipping tree test in short mode")
	}

	var nodes []*network.Network
	var processors []*MockPlugin
	var err error

	ports, peers := setupTreeNodes(basePort + 50)

	nodes, processors, err = setupNodes(ports)
	if err != nil {
		t.Fatal(err)
	}

	if err := bootstrapNodes(nodes, peers); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < len(nodes); i++ {
		broadcastTest(t, nodes, processors, peers, i)
	}
}
