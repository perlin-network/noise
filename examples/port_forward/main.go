package main

import (
	"flag"
	"io"
	"net"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/perlin-network/noise/crypto"
	"github.com/perlin-network/noise/network"
	"github.com/perlin-network/noise/network/builders"
	"github.com/perlin-network/noise/network/discovery"
)

type PfServerPlugin struct {
	network.Plugin
	remoteAddress string
}

func (state *PfServerPlugin) PeerConnect(client *network.PeerClient) {
	glog.Infof("New connection from %s", client.Address)
	remote, err := net.Dial("tcp", state.remoteAddress)
	if err != nil {
		// TODO: smux
		glog.Error(err)
		return
	}
	go io.Copy(remote, client)
	go io.Copy(client, remote)
}

func (state *PfServerPlugin) PeerDisconnect(client *network.PeerClient) {
	glog.Infof("Client %s disconnected", client.Address)
}

type PfClientPlugin struct {
	network.Plugin
	listenAddress string
}

func (state *PfClientPlugin) PeerConnect(client *network.PeerClient) {
	glog.Infof("Connected to %s", client.Address)
	go func() {
		for {
			func() {
				listener, err := net.Listen("tcp", state.listenAddress)
				if err != nil {
					glog.Error(err)
					return
				}
				defer listener.Close()

				conn, err := listener.Accept()
				if err != nil {
					glog.Error(err)
					return
				}
				defer conn.Close()

				ch1 := make(chan struct{})
				ch2 := make(chan struct{})

				go func() {
					io.Copy(conn, client)
					close(ch1)
				}()
				go func() {
					io.Copy(client, conn)
					close(ch2)
				}()
				select {
				case <-ch1:
				case <-ch2:
				}
			}()
			time.Sleep(1 * time.Second)
		}
	}()
}

func (state *PfClientPlugin) PeerDisconnect(client *network.PeerClient) {
	glog.Infof("Lost connection with %s", client.Address)
}

func main() {
	// glog defaults to logging to a file, override this flag to log to console for testing
	flag.Set("logtostderr", "true")

	// process other flags
	portFlag := flag.Int("port", 3000, "port to listen to")
	hostFlag := flag.String("host", "localhost", "host to listen to")
	protocolFlag := flag.String("protocol", "kcp", "protocol to use (kcp/tcp)")
	peersFlag := flag.String("peers", "", "peers to connect to")
	modeFlag := flag.String("mode", "server", "mode to use (server/client)")
	addressFlag := flag.String("address", "127.0.0.1:80", "port forwarding connect/listen address")
	flag.Parse()

	port := uint16(*portFlag)
	host := *hostFlag
	protocol := *protocolFlag
	mode := *modeFlag
	address := *addressFlag
	peers := strings.Split(*peersFlag, ",")

	keys := crypto.RandomKeyPair()

	glog.Infof("Private Key: %s", keys.PrivateKeyHex())
	glog.Infof("Public Key: %s", keys.PublicKeyHex())

	builder := builders.NewNetworkBuilder()
	builder.SetKeys(keys)
	builder.SetAddress(network.FormatAddress(protocol, host, port))

	// Register peer discovery plugin.
	builder.AddPlugin(new(discovery.Plugin))

	// Add custom port forwarding plugin.
	if mode == "server" {
		builder.AddPlugin(&PfServerPlugin {
			remoteAddress: address,
		})
	} else if mode == "client" {
		builder.AddPlugin(&PfClientPlugin {
			listenAddress: address,
		})
	}

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
}
