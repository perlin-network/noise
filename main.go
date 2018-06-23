package main

import (
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/perlin-network/noise/crypto"
	"github.com/perlin-network/noise/grpc_utils"
	"github.com/perlin-network/noise/network/builders"
	"github.com/perlin-network/noise/network/discovery"
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
	// glog defaults to logging to a file, override this flag to log to console for testing
	flag.Set("logtostderr", "true")

	// process other flags
	portFlag := flag.Int("port", 3000, "port to listen to")
	hostFlag := flag.String("host", "localhost", "host to listen to")
	peersFlag := flag.String("peers", "", "peers to connect to")
	flag.Parse()

	port := *portFlag
	host := *hostFlag
	peers := strings.Split(*peersFlag, ",")
	peers = filterPeers(host, port, peers)

	keys := crypto.RandomKeyPair()

	glog.Infof("Private Key: %s", keys.PrivateKeyHex())
	glog.Infof("Public Key: %s", keys.PublicKeyHex())

	builder := &builders.NetworkBuilder{}
	builder.SetKeys(keys)
	builder.SetAddress(host)
	builder.SetPort(port)

	// Register peer discovery RPC handlers.
	discovery.BootstrapPeerDiscovery(builder)

	net, err := builder.BuildNetwork()
	if err != nil {
		glog.Warning(err)
		return
	}

	net.Listen()

	blockTimeout := 10 * time.Second
	if err := grpc_utils.BlockUntilConnectionReady(host, port, blockTimeout); err != nil {
		glog.Warningf("Error: port was not available, cannot bootstrap peers, err=%+v", err)
	}

	if len(peers) > 0 {
		net.Bootstrap(peers...)
	}

	select {}

	glog.Flush()
}
