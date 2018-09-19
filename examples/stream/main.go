package main

import (
	"flag"
	"io"
	"net"
	"strings"
	"time"

	"github.com/perlin-network/noise/crypto/ed25519"
	"github.com/perlin-network/noise/log"
	"github.com/perlin-network/noise/network"
	"github.com/perlin-network/noise/network/discovery"

	"github.com/xtaci/smux"
)

func muxStreamConfig() *smux.Config {
	config := smux.DefaultConfig()
	config.KeepAliveTimeout = 30 * time.Second
	config.KeepAliveInterval = 5 * time.Second

	return config
}

func proxy(a, b io.ReadWriter) {
	ch1 := make(chan struct{})
	ch2 := make(chan struct{})

	go func() {
		io.Copy(a, b)
		close(ch1)
	}()
	go func() {
		io.Copy(b, a)
		close(ch2)
	}()
	select {
	case <-ch1:
	case <-ch2:
	}
}

type ExampleServerPlugin struct {
	network.Plugin
	remoteAddress string
}

func (state *ExampleServerPlugin) PeerConnect(client *network.PeerClient) {
	log.Info().Msgf("new connection from %s.", client.Address)

	go state.handleClient(client)
}

func (state *ExampleServerPlugin) handleClient(client *network.PeerClient) {
	session, err := smux.Server(client, muxStreamConfig())
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}
	for {
		stream, err := session.AcceptStream()
		if err != nil {
			log.Error().Err(err).Msg("")
			break
		}

		log.Info().Msgf("new incoming stream from %s.", client.Address)

		go func() {
			defer stream.Close()

			remote, err := net.Dial("tcp", state.remoteAddress)
			if err != nil {
				log.Error().Err(err).Msg("")
				return
			}
			defer remote.Close()

			proxy(stream, remote)
		}()
	}
}

func (state *ExampleServerPlugin) PeerDisconnect(client *network.PeerClient) {
	log.Info().Msgf("lost connection with %s.", client.Address)
}

type ProxyServerPlugin struct {
	network.Plugin
	listenAddress string
}

func (state *ProxyServerPlugin) PeerConnect(client *network.PeerClient) {
	log.Info().
		Str("address", client.Address).
		Msgf("connected to proxy destination")

	go state.startProxying(client)
}

func (state *ProxyServerPlugin) startProxying(client *network.PeerClient) {
	session, err := smux.Client(client, muxStreamConfig())
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}

	// Open proxy server.
	listener, err := net.Listen("tcp", state.listenAddress)
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Fatal().Err(err).Msg("")
		}

		log.Info().
			Str("address", client.Address).
			Str("remote_address", conn.RemoteAddr().String()).
			Msg("proxying data from remote address")

		go func() {
			defer conn.Close()

			remote, err := session.OpenStream()
			if err != nil {
				log.Error().Err(err).Msg("")
				return
			}
			defer remote.Close()

			proxy(conn, remote)
		}()
	}
}

func (state *ProxyServerPlugin) PeerDisconnect(client *network.PeerClient) {
	log.Info().Msgf("Lost connection with proxy destination %s.", client.Address)
}

// An example showcasing how to use streams in Noise by creating a sample proxying server.
func main() {
	flag.Set("logtostderr", "true")

	portFlag := flag.Int("port", 3000, "port to listen to")
	hostFlag := flag.String("host", "localhost", "host to listen to")
	protocolFlag := flag.String("protocol", "tcp", "protocol to use (kcp/tcp)")
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

	keys := ed25519.RandomKeyPair()

	log.Info().Str("private_key", keys.PrivateKeyHex()).Msg("")
	log.Info().Str("public_key", keys.PublicKeyHex()).Msg("")

	builder := network.NewBuilder()
	builder.SetKeys(keys)
	builder.SetAddress(network.FormatAddress(protocol, host, port))

	// Register peer discovery plugin.
	builder.AddPlugin(new(discovery.Plugin))

	// Add custom port forwarding plugin.
	if mode == "server" {
		builder.AddPlugin(&ExampleServerPlugin{remoteAddress: address})
	} else if mode == "client" {
		builder.AddPlugin(&ProxyServerPlugin{listenAddress: address})
	}

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
