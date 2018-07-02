package main

import (
	"flag"
	"strings"

	"github.com/golang/glog"
	"github.com/perlin-network/noise/crypto"
	"github.com/perlin-network/noise/network"
	"github.com/perlin-network/noise/network/builders"
	"github.com/perlin-network/noise/network/discovery"
)

func main() {
	// glog defaults to logging to a file, override this flag to log to console for testing
	flag.Set("logtostderr", "true")

	// process other flags
	portFlag := flag.Int("port", 3000, "port to listen to")
	hostFlag := flag.String("host", "localhost", "host to listen to")
	protocolFlag := flag.String("protocol", "kcp", "protocol to use (kcp/tcp)")
	peersFlag := flag.String("peers", "", "peers to connect to")
	upnpFlag := flag.Bool("upnp", false, "enable upnp")
	flag.Parse()

	port := uint16(*portFlag)
	host := *hostFlag
	protocol := *protocolFlag
	upnpEnabled := *upnpFlag
	peers := strings.Split(*peersFlag, ",")

	keys := crypto.RandomKeyPair()

	glog.Infof("Private Key: %s", keys.PrivateKeyHex())
	glog.Infof("Public Key: %s", keys.PublicKeyHex())

	builder := builders.NewNetworkBuilder()
	builder.SetKeys(keys)
	builder.SetAddress(network.FormatAddress(protocol, host, port))
	builder.SetUpnpEnabled(upnpEnabled)

	// Register peer discovery RPC handlers.
	discovery.BootstrapPeerDiscovery(builder)

	net, err := builder.Build()
	if err != nil {
		glog.Fatal(err)
		return
	}

	go net.Listen()

	if len(peers) > 0 {
		net.Bootstrap(peers...)
	}

	select {}

	glog.Flush()
}
