package network_test

import (
	"testing"
	"time"

	"github.com/perlin-network/noise/internal/test/protobuf"
	"github.com/perlin-network/noise/network"

	"github.com/stretchr/testify/assert"
)

func TestNodeConnect(t *testing.T) {
	t.Parallel()

	for _, e := range allEnvs {
		testNodeConnect(t, e, 10)
	}
}

func testNodeConnect(t *testing.T, e env, numNodes int) {
	te := newTest(t, e)
	te.startBoostrap(numNodes)
	defer te.tearDown()

	peers := te.getPeers(te.bootstrapNode)

	count := len(peers)
	if count != numNodes-1 {
		assert.Equalf(t, count, numNodes-1, "#peers = %d, want %d", count, numNodes-1)
	}
}

func TestNodeBroadcast(t *testing.T) {
	if testing.Short() {
		t.Skipf("skipping %s in short mode", t.Name())
	}

	numNodes := 4
	for _, e := range allEnvs {
		testNodeBroadcast(t, e, numNodes)
	}
}

func testNodeBroadcast(t *testing.T, e env, numNodes int) {
	te := newTest(t, e, network.WriteTimeout(1*time.Second))
	te.startBoostrap(numNodes)
	defer te.tearDown()

	expected := "test message"
	te.bootstrapNode.Broadcast(&protobuf.TestMessage{Message: expected})

	for i, node := range te.nodes {
		select {
		case received := <-te.getMailbox(node).RecvMailbox:
			assert.Equalf(t, received.Message, expected, "Expected message %s to be received by node %d but got %v\n", expected, i+1, received.Message)
		case <-time.After(100 * time.Millisecond):
			// FIXME(jack0): this can trigger sometimes, flaky
			t.Errorf("Timed out attempting to receive message from Node 0.\n")
		}
	}
}

func TestNodeBroadcastByIDs(t *testing.T) {
	if testing.Short() {
		t.Skipf("skipping %s in short mode", t.Name())
	}

	numNodes, numPeers := 5, 2
	for _, e := range allEnvs {
		testNodeBroadcastByIDs(t, e, numNodes, numPeers)
	}
}

func testNodeBroadcastByIDs(t *testing.T, e env, numNodes, numPeers int) {
	te := newTest(t, e)
	te.startBoostrap(numNodes)
	defer te.tearDown()

	expected := "test message"
	peers := te.getPeers(te.bootstrapNode)
	selectedPeers := peers[:numPeers]

	te.bootstrapNode.BroadcastByIDs(&protobuf.TestMessage{Message: expected}, selectedPeers...)

	time.Sleep(50 * time.Millisecond)

	for _, node := range te.nodes {
		numMsgs := len(te.getMailbox(node).RecvMailbox)

		if isIn(node.Address, selectedPeers...) {
			assert.Equalf(t, numMsgs, 1, "node [%v] got %d messages, expected 1", node.Address, numMsgs)
		} else {
			assert.Equalf(t, numMsgs, 0, "node [%v] got %d messages, expected 0", node.Address, numMsgs)
		}
	}
}

func TestNodeBroadcastByAddresses(t *testing.T) {
	if testing.Short() {
		t.Skipf("skipping %s in short mode", t.Name())
	}

	numNodes, numPeers := 5, 2
	for _, e := range allEnvs[:1] {
		testNodeBroadcastByAddresses(t, e, numNodes, numPeers)
	}
}

func testNodeBroadcastByAddresses(t *testing.T, e env, numNodes, numPeers int) {
	te := newTest(t, e, network.WriteTimeout(1*time.Second))
	te.startBoostrap(numNodes)
	defer te.tearDown()

	expected := "test message"
	peers := te.getPeers(te.bootstrapNode)

	selectedPeers := peers[:numPeers]
	addresses := make([]string, numPeers)
	for i := range selectedPeers {
		addresses[i] = peers[i].Address
	}
	te.bootstrapNode.BroadcastByAddresses(&protobuf.TestMessage{Message: expected}, addresses...)

	time.Sleep(50 * time.Millisecond)

	for _, node := range te.nodes {
		numMsgs := len(te.getMailbox(node).RecvMailbox)

		if isInAddress(node.Address, addresses...) {
			assert.Equalf(t, numMsgs, 1, "node [%v] got %d messages, expected 1", node.Address, numMsgs)
		} else {
			assert.Equalf(t, numMsgs, 0, "node [%v] got %d messages, expected 0", node.Address, numMsgs)
		}
	}
}
