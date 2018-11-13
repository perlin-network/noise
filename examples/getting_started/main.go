package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/perlin-network/noise/connection"
	"github.com/perlin-network/noise/crypto/ed25519"
	"github.com/perlin-network/noise/identity"
	"github.com/perlin-network/noise/log"
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

	keys := ed25519.RandomKeyPair()
	idAdapter := identity.NewDefaultIdentityAdapter(keys)

	log.Info().Str("private_key", keys.PrivateKeyHex()).Msg("")
	log.Info().Str("public_key", keys.PublicKeyHex()).Msg("")

	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}

	connAdapter, err := connection.StartAddressableConnectionAdapter(listener, func(addr string) (net.Conn, error) {
		return net.DialTimeout("tcp", addr, 10*time.Second)
	})
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}

	node := protocol.NewNode(
		protocol.NewController(),
		connAdapter,
		idAdapter,
	)
	node.Start()

	node.AddService(42, func(message *protocol.Message) {
		log.Info().Msgf("received payload from %s: %s", hex.EncodeToString(message.Sender), string(message.Body.Payload))
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
				log.Fatal().Err(err).Msg("")
			}
			remoteAddr := peer[1]
			connAdapter.MapIDToAddress(peerID, remoteAddr)
		}
	}

	select {}

}
