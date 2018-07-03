package network

import (
	"net"
	"sync"

	"github.com/golang/glog"
	"github.com/golang/protobuf/ptypes"
	"github.com/perlin-network/noise/peer"
	"github.com/xtaci/smux"
)

// Ingest handles peer registration and processes incoming message streams consisting of
// asynchronous messages or request/responses.
func (n *Network) Ingest(conn net.Conn) {
	session, err := smux.Server(conn, muxConfig())
	if err != nil {
		glog.Error(err)
		return
	}

	defer session.Close()

	var client *PeerClient
	var clientInit sync.Once
	var initError error

	// Handle new streams and process their incoming messages.
	for {
		stream, err := session.AcceptStream()
		if err != nil {
			if client != nil && err.Error() == "broken pipe" {
				client.Close()
			}
			break
		}

		// One goroutine per incoming stream.
		go func(stream *smux.Stream) {
			// Clean up resources.
			defer stream.Close()

			msg, err := n.receiveMessage(stream)

			// Failed to receive message.
			if err != nil {
				glog.Warning(err)
				return
			}

			id := peer.ID(*msg.Sender)

			clientInit.Do(func() {
				var err error
				if client != nil {
					glog.Fatal("Invalid state (1)")
				}
				client, err = n.Client(msg.Sender.Address)
				if err != nil {
					initError = err
					return
				}
				client.ID = &id
				err = client.establishConnection(id.Address)
				if err != nil {
					glog.Errorf("Failed to connect to peer %s err=[%+v]\n", id.Address, err)
					initError = err
					return
				}
			})

			if initError != nil {
				glog.Warning(initError)
				return
			}

			if client == nil {
				glog.Fatal("Invalid state (2)")
			}

			if !client.ID.Equals(id) {
				// Peer sent message with a completely different ID (???)
				glog.Errorf("Message signed by peer %s but client is %s", client.ID.Address, id.Address)
				return
			}

			// Unmarshal protobuf.
			var ptr ptypes.DynamicAny
			if err := ptypes.UnmarshalAny(msg.Message, &ptr); err != nil {
				glog.Error(err)
				return
			}

			// Check if the incoming message is a response.
			if channel, exists := client.Requests.Load(msg.Nonce); exists && msg.Nonce > 0 {
				channel <- ptr.Message
				return
			}

			// Create message execution context.
			ctx := new(MessageContext)
			ctx.client = client
			ctx.stream = stream
			ctx.message = ptr.Message
			ctx.nonce = msg.Nonce

			// Execute 'on receive message' callback for all plugins.
			n.Plugins.Each(func(plugin PluginInterface) {
				err := plugin.Receive(ctx)

				if err != nil {
					glog.Error(err)
				}
			})
		}(stream)
	}
}
