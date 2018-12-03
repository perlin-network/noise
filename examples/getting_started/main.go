package main

import (
	"context"
	"encoding/hex"
	"flag"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/perlin-network/noise/base"
	"github.com/perlin-network/noise/protocol"
)

type StarterService struct {
	protocol.Service
}

func (s *StarterService) Receive(ctx context.Context, message *protocol.Message) (*protocol.MessageBody, error) {
	fmt.Printf("received payload from %s: %s\n", hex.EncodeToString(message.Sender), string(message.Body.Payload))
	return nil, nil
}

func dialTCP(addr string) (net.Conn, error) {
	return net.DialTimeout("tcp", addr, 10*time.Second)
}

func main() {
	// process flags
	portFlag := flag.Int("port", 3000, "port to listen to")
	hostFlag := flag.String("host", "localhost", "host to listen to")
	peersFlag := flag.String("peers", "", "peers to connect to in format: peerKeyHash1=host1:port1,peerKeyHash2=host2:port2,...")
	flag.Parse()

	port := *portFlag
	host := *hostFlag
	peers := strings.Split(*peersFlag, ",")

	idAdapter := base.NewIdentityAdapter()
	fmt.Printf("private_key: %s\n", idAdapter.GetKeyPair().PrivateKeyHex())
	fmt.Printf("public_key: %s\n", idAdapter.GetKeyPair().PublicKeyHex())

	localAddr := fmt.Sprintf("%s:%d", host, port)
	listener, err := net.Listen("tcp", localAddr)
	if err != nil {
		panic(err)
	}
	connAdapter, err := base.NewConnectionAdapter(listener, dialTCP)
	if err != nil {
		panic(err)
	}

	node := protocol.NewNode(
		protocol.NewController(),
		idAdapter,
	)
	connAdapter.RegisterNode(node)
	node.AddService(&StarterService{})

	if len(peers) > 0 {
		for _, peerKV := range peers {
			if len(peerKV) == 0 {
				// this is a blank parameter
				continue
			}
			p := strings.Split(peerKV, "=")
			peerID, err := hex.DecodeString(p[0])
			if err != nil {
				panic(err)
			}
			remoteAddr := p[1]
			connAdapter.AddPeerID(peerID, remoteAddr)
		}
	}

	node.Listen()

	select {}

}
