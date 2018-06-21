package main

import (
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/perlin-network/noise/crypto"
	"github.com/perlin-network/noise/grpc_utils"
	"github.com/perlin-network/noise/log"
	"github.com/perlin-network/noise/network"
)

func filterPeers(host string, port int, peers []string) []string {
	currAddress := fmt.Sprintf("%s:%d", host, port)
	peersLen := len(peers)
	filtered := make([]string, peersLen)
	visitedSet := make(map[string]struct{}, peersLen)
	for _, peer := range peers {
		if peer != currAddress {
			// remove if it is the current host and port
			if _, ok := visitedSet[peer]; !ok {
				// remove if it is a duplicate in the list
				filtered = append(filtered, peer)
				visitedSet[peer] = struct{}{}
			}
		}
	}
	return filtered
}

func main() {
	portFlag := flag.Int("port", 3000, "port to listen to")
	hostFlag := flag.String("host", "localhost", "host to listen to")
	peersFlag := flag.String("peers", "", "peers to connect to")
	flag.Parse()

	port := *portFlag
	host := *hostFlag
	peers := strings.Split(*peersFlag, ",")
	peers = filterPeers(host, port, peers)

	keys := crypto.RandomKeyPair()

	log.Print("Private Key: " + keys.PrivateKeyHex())
	log.Print("Public Key: " + keys.PublicKeyHex())

	builder := network.NetworkBuilder{}
	builder.SetKeys(keys)
	builder.SetAddress(host)
	builder.SetPort(port)

	net, err := builder.BuildNetwork()
	if err != nil {
		log.Print(err)
		return
	}

	net.Listen()

	if err := grpc_utils.BlockUntilServerReady(host, port, 10*time.Second); err != nil {
		log.Print(fmt.Sprintf("Error: %v", err))
		return
	}

	if len(peers) > 0 {
		net.Bootstrap(peers...)
	}

	select {}
}
