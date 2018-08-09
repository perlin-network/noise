package network_test

import (
	"testing"
	"time"

	"github.com/perlin-network/noise/internal/test/protobuf"
	"github.com/perlin-network/noise/network"
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
		t.Errorf("#peers = %d, want %d", count, numNodes-1)
	}
}

func TestNodeBroadcast(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skipf("skipping %s in short mode", t.Name())
	}

	for _, e := range allEnvs {
		testNodeBroadcast(t, e, 4)
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
			if received.Message != expected {
				t.Errorf("Expected message %s to be received by node %d but got %v\n", expected, i+1, received.Message)
			} else {
				// t.Logf("Node %d received a message from Node 0.\n", i+1)
			}
		case <-time.After(100 * time.Millisecond):
			// FIXME(jack0): this can trigger sometimes, flaky
			t.Errorf("Timed out attempting to receive message from Node 0.\n")
		}
	}
}

func TestNodeBroadcastByIDs(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skipf("skipping %s in short mode", t.Name())
	}

	for _, e := range allEnvs[:1] {
		testNodeBroadcastByIDs(t, e, 5, 2)
	}
}

func testNodeBroadcastByIDs(t *testing.T, e env, numNodes, numPeers int) {
	te := newTest(t, e)
	te.startBoostrap(numNodes)
	defer te.tearDown()

	expected := "test message"
	peers := te.getPeers(te.bootstrapNode)

	te.bootstrapNode.BroadcastByIDs(&protobuf.TestMessage{Message: expected}, peers[:numPeers]...)

	time.Sleep(50 * time.Millisecond)

	for _, node := range te.nodes {
		numMsgs := len(te.getMailbox(node).RecvMailbox)

		if isIn(node.Address, peers[:numPeers]...) {
			if numMsgs != 1 {
				t.Errorf("node [%v] got %d messages, expected 1", node.Address, numMsgs)
			}
		} else {
			if numMsgs != 0 {
				t.Errorf("node [%v] got %d messages, expected 0", node.Address, numMsgs)
			}
		}
	}
}

func TestNodeBroadcastByAddresses(t *testing.T) {
	t.Skip()
	t.Parallel()
	if testing.Short() {
		t.Skipf("skipping %s in short mode", t.Name())
	}

	for _, e := range allEnvs {
		testNodeBroadcastByAddresses(t, e)
	}
}

func testNodeBroadcastByAddresses(t *testing.T, e env) {
	te := newTest(t, e, network.WriteTimeout(1*time.Second))
	numNodes := 5
	te.startBoostrap(numNodes)
	defer te.tearDown()

	expected := "test message"
	peers := te.getPeers(te.bootstrapNode)
	if len(peers) != 4 {
		t.Errorf("len(peers) = %d, expected 4", len(peers))
	}

	numPeers := 2
	addresses := []string{}
	for i := 0; i < numPeers; i++ {
		addresses = append(addresses, peers[i].Address)
	}
	te.bootstrapNode.BroadcastByAddresses(&protobuf.TestMessage{Message: expected}, addresses...)

	time.Sleep(50 * time.Millisecond)

	for _, node := range te.nodes {
		numMsgs := len(te.getMailbox(node).RecvMailbox)
		// t.Logf("addresses: %+v address: %s i: %d\n", addresses, node.Address, i+1)
		for _, address := range addresses {
			if address == node.Address {
				if numMsgs != 1 {
					t.Errorf("node [%v] got %d messages, expected 1", node.Address, numMsgs)
				}
			} else if numMsgs != 0 {
				t.Errorf("node [%v] got %d messages, expected 0", node.Address, numMsgs)
			}
		}
	}
}
