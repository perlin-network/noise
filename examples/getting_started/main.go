package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/perlin-network/noise/base"
	"github.com/perlin-network/noise/protocol"
)

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
	keys := idAdapter.GetKeyPair()

	fmt.Printf("private_key: %s\n", keys.PrivateKeyHex())
	fmt.Printf("public_key: %s\n", keys.PublicKeyHex())

	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		panic(err)
	}

	connAdapter, err := base.NewConnectionAdapter(listener, func(addr string) (net.Conn, error) {
		return net.DialTimeout("tcp", addr, 10*time.Second)
	})
	if err != nil {
		panic(err)
	}

	node := protocol.NewNode(
		protocol.NewController(),
		connAdapter,
		idAdapter,
	)

	node.AddService(42, func(message *protocol.Message) {
		fmt.Printf("received payload from %s: %s\n", hex.EncodeToString(message.Sender), string(message.Body.Payload))
	})

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

	node.Start()

	select {}

}
