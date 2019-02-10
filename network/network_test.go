package network_test

import (
	"context"
	"testing"
	"time"

	"github.com/perlin-network/noise/internal/test/protobuf"
	"github.com/perlin-network/noise/network"
	"github.com/perlin-network/noise/types/opcode"

	"github.com/stretchr/testify/assert"
)

func init() {
	opcode.RegisterMessageType(opcode.Opcode(1000), &protobuf.TestMessage{})
}

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
	te.bootstrapNode.Broadcast(context.Background(), &protobuf.TestMessage{Message: expected})

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

	te.bootstrapNode.BroadcastByIDs(context.Background(), &protobuf.TestMessage{Message: expected}, selectedPeers...)

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
	te.bootstrapNode.BroadcastByAddresses(context.Background(), &protobuf.TestMessage{Message: expected}, addresses...)

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

func TestClientRequest(t *testing.T) {
	if testing.Short() {
		t.Skipf("skipping %s in short mode", t.Name())
	}

	numNodes := 2
	for _, e := range allEnvs {
		testClientRequest(t, e, numNodes)
	}
}

func testClientRequest(t *testing.T, e env, numNodes int) {
	te := newTest(t, e, network.WriteTimeout(1*time.Second))
	plugin := new(clientTestPlugin)
	te.startBoostrap(numNodes, plugin)
	defer te.tearDown()

	msgStr := "test message"
	address := te.nodes[0].Address
	client, err := te.bootstrapNode.Client(address)
	assert.Equal(t, nil, err, "expected client error to be nil")
	msg := &protobuf.TestMessage{
		Message:  msgStr,
		Duration: 1,
	}
	response, err := client.Request(context.Background(), msg)
	resp, ok := response.(*protobuf.TestMessage)
	assert.Equal(t, true, ok, "expected response to be cast successfully")
	assert.Equal(t, msgStr, resp.Message, "expected reply message to be '%s', got '%s'", msgStr, resp.Message)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	response, err = client.Request(ctx, msg)
	assert.Equal(t, nil, response, "expected response to be nil")
	assert.Equal(t, "context deadline exceeded", err.Error(), "expected error to be context deadline exceeded")

	ctx, cancel = context.WithTimeout(context.Background(), 2*time.Second)
	go func(ctx context.Context) {
		_, err := client.Request(ctx, msg)
		assert.Equal(t, "context canceled", err.Error(), "expected context canceled error")
	}(ctx)
	cancel()
}
