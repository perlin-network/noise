package main

import (
	"flag"
	"strings"

	"github.com/perlin-network/noise/crypto/ed25519"
	"github.com/perlin-network/noise/log"
	"github.com/perlin-network/noise/network"
	"github.com/perlin-network/noise/network/backoff"
	"github.com/perlin-network/noise/network/discovery"
	"github.com/perlin-network/noise/network/nat"
)

func main() {
	// process flags
	portFlag := flag.Int("port", 3000, "port to listen to")
	hostFlag := flag.String("host", "localhost", "host to listen to")
	protocolFlag := flag.String("protocol", "tcp", "protocol to use (kcp/tcp)")
	peersFlag := flag.String("peers", "", "peers to connect to")
	natFlag := flag.Bool("nat", false, "enable nat traversal")
	reconnectFlag := flag.Bool("reconnect", false, "enable reconnections")
	flag.Parse()

	port := uint16(*portFlag)
	host := *hostFlag
	protocol := *protocolFlag
	natEnabled := *natFlag
	reconnectEnabled := *reconnectFlag
	peers := strings.Split(*peersFlag, ",")

	keys := ed25519.RandomKeyPair()

	log.Info().Str("private_key", keys.PrivateKeyHex()).Str("public_key", keys.PublicKeyHex()).Msg("Generated keypair.")

	builder := network.NewBuilder()
	builder.SetKeys(keys)
	builder.SetAddress(network.FormatAddress(protocol, host, port))

	// Register NAT traversal plugin.
	if natEnabled {
		nat.RegisterPlugin(builder)
	}

	// Register the reconnection plugin
	if reconnectEnabled {
		builder.AddPlugin(new(backoff.Plugin))
	}

	// Register peer discovery plugin.
	builder.AddPlugin(new(discovery.Plugin))

	net, err := builder.Build()
	if err != nil {
		log.Fatal().Err(err).Msg("")
		return
	}

	go net.Listen()

	if len(peers) > 0 {
		net.Bootstrap(peers...)
	}

	select {}

}
