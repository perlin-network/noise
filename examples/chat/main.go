package main

import (
	"bufio"
	"encoding/hex"
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/perlin-network/noise/base"
	"github.com/perlin-network/noise/examples/chat/messages"
	"github.com/perlin-network/noise/log"
	"github.com/perlin-network/noise/protocol"
)

const (
	serviceID = 44
)

type ChatNode struct {
	Node        *protocol.Node
	Address     string
	ConnAdapter protocol.ConnectionAdapter
}

func (n *ChatNode) service(message *protocol.Message) {
	if message.Body.Service != serviceID {
		return
	}
	if len(message.Body.Payload) == 0 {
		return
	}
	var msg messages.ChatMessage
	if err := proto.Unmarshal(message.Body.Payload, &msg); err != nil {
		return
	}
	log.Info().Msgf("<%s> %s", n.Address, msg.Message)
}

func makeMessageBody(value string) *protocol.MessageBody {
	msg := &messages.ChatMessage{
		Message: value,
	}
	payload, err := proto.Marshal(msg)
	if err != nil {
		return nil
	}
	body := &protocol.MessageBody{
		Service: serviceID,
		Payload: payload,
	}
	return body
}

func main() {
	// process other flags
	portFlag := flag.Int("port", 3000, "port to listen to")
	hostFlag := flag.String("host", "localhost", "host to listen to")
	peersFlag := flag.String("peers", "", "peers to connect to in format: peerKeyHash1=host1:port1,peerKeyHash2=host2:port2,...")
	flag.Parse()

	port := *portFlag
	host := *hostFlag
	peers := strings.Split(*peersFlag, ",")

	idAdapter := base.NewIdentityAdapter()
	keys := idAdapter.GetKeyPair()

	log.Info().Msgf("Private Key: %s", keys.PrivateKeyHex())
	log.Info().Msgf("Public Key: %s", keys.PublicKeyHex())

	addr := fmt.Sprintf("%s:%d", host, port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		panic(err)
	}

	connAdapter, err := base.NewConnectionAdapter(listener, func(addr string) (net.Conn, error) {
		return net.DialTimeout("tcp", addr, 10*time.Second)
	})
	if err != nil {
		panic(err)
	}

	node := &ChatNode{
		Node: protocol.NewNode(
			protocol.NewController(),
			connAdapter,
			idAdapter,
		),
		Address:     addr,
		ConnAdapter: connAdapter,
	}

	node.Node.AddService(serviceID, node.service)

	if len(peers) > 0 {
		for _, peerKV := range peers {
			if len(peerKV) == 0 {
				// this is a blank parameter
				continue
			}
			peer := strings.Split(peerKV, "=")
			peerID, err := hex.DecodeString(peer[0])
			if err != nil {
				panic(err)
			}
			remoteAddr := peer[1]
			connAdapter.AddConnection(peerID, remoteAddr)
		}
	}

	node.Node.Start()

	reader := bufio.NewReader(os.Stdin)
	for {
		input, _ := reader.ReadString('\n')

		// skip blank lines
		if len(strings.TrimSpace(input)) == 0 {
			continue
		}

		log.Info().Msgf("<%s> %s", addr, input)

		node.Node.Broadcast(makeMessageBody(input))
	}
}
