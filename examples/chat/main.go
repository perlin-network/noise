package main

import (
	"bufio"
	"flag"
	"os"
	"strings"

	"github.com/golang/glog"
	"github.com/perlin-network/noise/crypto"
	"github.com/perlin-network/noise/examples/chat/messages"
	"github.com/perlin-network/noise/network"
	"github.com/perlin-network/noise/network/builders"
	"github.com/perlin-network/noise/network/discovery"
)

type ChatPlugin struct{ *network.Plugin }

func (state *ChatPlugin) Receive(ctx *network.MessageContext) error {
	switch msg := ctx.Message().(type) {
	case *messages.ChatMessage:
		glog.Infof("<%s> %s", ctx.Client().ID.Address, msg.Message)
	}

	return nil
}

func main() {
	cryptoProvider := crypto.NewEd25519()

	// glog defaults to logging to a file, override this flag to log to console for testing
	flag.Set("logtostderr", "true")

	// process other flags
	portFlag := flag.Int("port", 3000, "port to listen to")
	hostFlag := flag.String("host", "localhost", "host to listen to")
	protocolFlag := flag.String("protocol", "kcp", "protocol to use (kcp/tcp)")
	peersFlag := flag.String("peers", "", "peers to connect to")
	flag.Parse()

	port := uint16(*portFlag)
	host := *hostFlag
	protocol := *protocolFlag
	peers := strings.Split(*peersFlag, ",")

	keys := crypto.RandomKeyPair(cryptoProvider)

	glog.Infof("Private Key: %s", keys.PrivateKeyHex())
	glog.Infof("Public Key: %s", keys.PublicKeyHex())

	builder := builders.NewNetworkBuilder()
	builder.SetKeys(keys)
	builder.SetAddress(network.FormatAddress(protocol, host, port))
	builder.SetCryptoProvider(cryptoProvider)

	// Register peer discovery plugin.
	builder.AddPlugin(new(discovery.Plugin))

	// Add custom chat plugin.
	builder.AddPlugin(new(ChatPlugin))

	net, err := builder.Build()
	if err != nil {
		glog.Fatal(err)
		return
	}

	go net.Listen()

	if len(peers) > 0 {
		net.Bootstrap(peers...)
	}

	reader := bufio.NewReader(os.Stdin)
	for {
		input, _ := reader.ReadString('\n')

		// skip blank lines
		if len(strings.TrimSpace(input)) == 0 {
			continue
		}

		glog.Infof("<%s> %s", net.Address, input)

		net.Broadcast(&messages.ChatMessage{Message: input})
	}

	glog.Flush()
}
