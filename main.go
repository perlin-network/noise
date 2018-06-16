package main

import (
	"flag"
	"github.com/perlin-network/noise/crypto"
	"github.com/perlin-network/noise/log"
	"github.com/perlin-network/noise/network"
	"strings"
)

func main() {
	portFlag := flag.Int("port", 3000, "port to listen to")
	peersFlag := flag.String("peers", "", "peers to connect to")
	flag.Parse()

	port := *portFlag
	peers := strings.Split(*peersFlag, ",")

	keys := crypto.RandomKeyPair()

	log.Print("Private Key: " + keys.PrivateKeyHex())
	log.Print("Public Key: " + keys.PublicKeyHex())

	net := network.CreateNetwork(keys, "localhost", port)

	net.Listen()

	if len(*peersFlag) > 0 {
		net.Bootstrap(peers...)
	}

	select {}
}
