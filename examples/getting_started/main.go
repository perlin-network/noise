package main

import (
	"context"
	"encoding/hex"
	"flag"
	"fmt"
	"strings"

	"github.com/perlin-network/noise"
)

type starterService struct {
	*noise.Noise
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

	// setup the node
	config := &noise.Config{
		Host:            host,
		Port:            port,
		EnableSKademlia: false,
	}
	n, err := noise.NewNoise(config)
	if err != nil {
		panic(err)
	}
	service := &starterService{
		Noise: n,
	}
	service.OnReceive(noise.OpCode(1234), func(ctx context.Context, message *noise.Message) (*noise.MessageBody, error) {
		fmt.Printf("received payload from %s: %s\n", hex.EncodeToString(message.Sender), string(message.Body.Payload))
		return nil, nil
	})

	if len(peers) > 0 {
		var peerIDs []noise.PeerID
		for _, peerKV := range peers {
			if len(peerKV) == 0 {
				// this is a blank parameter
				continue
			}
			p := strings.Split(peerKV, "=")
			publicKey, err := hex.DecodeString(p[0])
			if err != nil {
				panic(err)
			}
			remoteAddr := p[1]
			peerIDs = append(peerIDs, noise.CreatePeerID(publicKey, remoteAddr))
		}
		service.Bootstrap(peerIDs...)
	}

	select {}

}
