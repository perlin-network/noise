package cluster_fragmentation

package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/golang/glog"
	"github.com/perlin-network/noise/crypto"
	"github.com/perlin-network/noise/examples/chat/messages"
	"github.com/perlin-network/noise/grpc_utils"
	"github.com/perlin-network/noise/network"
	"github.com/perlin-network/noise/network/builders"
	"github.com/perlin-network/noise/network/discovery"
	"os"
	"strings"
	"time"
)

type ChatMessageProcessor struct{}

func (*ChatMessageProcessor) Handle(client *network.PeerClient, raw *network.IncomingMessage) error {
	message := raw.Message.(*messages.ChatMessage)

	glog.Infof("<%s> %s", client.Id.Address, message.Message)

	return nil
}

func main() {
	// glog defaults to logging to a file, override this flag to log to console for testing
	flag.Set("logtostderr", "true")

	// process other flags
	portFlag := flag.Int("port", 3000, "port to listen to")
	hostFlag := flag.String("host", "localhost", "host to listen to")
	peersFlag := flag.String("peers", "", "peers to connect to")
	flag.Parse()

	port := *portFlag
	host := *hostFlag
	peers := strings.Split(*peersFlag, ",")

	keys := crypto.RandomKeyPair()

	glog.Infof("Private Key: %s", keys.PrivateKeyHex())
	glog.Infof("Public Key: %s", keys.PublicKeyHex())

	builder := &builders.NetworkBuilder{}
	builder.SetKeys(keys)
	builder.SetHost(host)
	builder.SetPort(port)

	// Register peer discovery RPC handlers.
	discovery.BootstrapPeerDiscovery(builder)

	builder.AddProcessor((*messages.ChatMessage)(nil), new(ChatMessageProcessor))

	net, err := builder.BuildNetwork()
	if err != nil {
		glog.Fatal(err)
		return
	}

	go net.Listen()

	if len(peers) > 0 {
		blockTimeout := 10 * time.Second
		if err := grpc_utils.BlockUntilConnectionReady(host, port, blockTimeout); err != nil {
			glog.Warningf(fmt.Sprintf("Error: port was not available, cannot bootstrap peers, err=%+v", err))
		}

		net.Bootstrap(peers...)
	}

	reader := bufio.NewReader(os.Stdin)
	for {
		input, _ := reader.ReadString('\n')

		glog.Infof("<%s> %s", net.Address(), input)

		net.Broadcast(&messages.ChatMessage{Message: input})
	}

	glog.Flush()
}

type ClusterMessage struct {
	Value string
}
type ClusterNode struct {
	Host string
	Port int
	Net *network.Network
}

func (c* ClusterNode) Handle(client *network.PeerClient, raw *network.IncomingMessage) error {
	message := raw.Message.(*messages.ChatMessage)

	glog.Infof("<%s> %s", client.Id.Address, message.Message)

	return nil
}

var blockTimeout = 10 * time.Second

func setupCluster(t *testing.T) ([]*ClusterNode, error) {
	// glog defaults to logging to a file, override this flag to log to console for testing
	flag.Set("logtostderr", "true")

	host := "localhost"
	cluster1StartPort := 3001
	cluster1NumPorts := 3
	networks := []*network.Network{}
	peers := []string{}
	nodes := []*ClusterNode{}

	for portOffset := 0; portOffset < cluster1NumPorts; portOffset++ {
		node := &ClusterNode{}
		node.Host = host
		node.Port = cluster1StartPort + portOffset
		keys := crypto.RandomKeyPair()
		peers = append(peers, fmt.Sprintf("%s:%d", node.Host , node.Port))
		
		builder := &builders.NetworkBuilder{}
		builder.SetKeys(keys)
		builder.SetHost(node.Host)
		builder.SetPort(node.Port)

		discovery.BootstrapPeerDiscovery(builder)

		builder.AddProcessor((*messages.ChatMessage)(nil), new(ChatMessageProcessor))

		net, err := builder.BuildNetwork()
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, net)

		go net.Listen()
	}

	for i := 0; i < len(nodes); i++ {
		if err := grpc_utils.BlockUntilConnectionReady(node.Host, node.Port, blockTimeout); err != nil {
			return nil, fmt.Error("Error: port was not available, cannot bootstrap peers, err=%+v", err)
		}
	}

	for i := 0; i < len(nodes); i++ {
		nodes[i].Net.Bootstrap(peers...)
	}

	return nodes, nil
}

func TestClusters(t *testing.T) {
	nodes, err := setupCluster(t)

	// TODO
}
