package basic

import (
	"flag"
	"fmt"
	"time"

	"github.com/perlin-network/noise/examples/basic/messages"
)

// ExampleSetupClusters - example of how to use SetupClusters() to automate tests
func ExampleSetupClusters() {
	// parse to flags to silence the glog library
	flag.Parse()

	host := "localhost"
	startPort := 5000
	numNodes := 3
	var nodes []*ClusterNode
	var peers []string

	for i := 0; i < numNodes; i++ {
		node := &ClusterNode{}
		node.Host = host
		node.Port = startPort + i

		nodes = append(nodes, node)
		peers = append(peers, fmt.Sprintf("%s:%d", node.Host, node.Port))
	}

	for _, node := range nodes {
		node.Peers = peers
	}

	if err := SetupCluster(nodes); err != nil {
		fmt.Print(err)
		return
	}

	for i, node := range nodes {
		if node.Net == nil {
			fmt.Printf("expected %d nodes, but node %d is missing a network", len(nodes), i)
			return
		}
	}

	// check if you can send a message from node 1 and will it be received only in node 2,3

	testMessage := "message from node 0"

	// Broadcast is an asynchronous call to send a message to other nodes
	nodes[0].Net.Broadcast(&messages.BasicMessage{Message: testMessage})

	// Simplificiation: message broadcasting is asynchronous, so need the messages to settle
	time.Sleep(1 * time.Second)

	if result := nodes[0].PopMessage(); result != nil {
		fmt.Printf("expected nothing in node 0, got %v", result)
		return
	}
	for i := 1; i < len(nodes); i++ {
		if result := nodes[i].PopMessage(); result == nil {
			fmt.Printf("expected a message in node %d but it was blank", i)
			return
		} else {
			if result.Message != testMessage {
				fmt.Printf("expected message %s in node %d but got %v", testMessage, i, result)
				return
			}
		}
	}

	fmt.Printf("success")
	// Output: success
}
