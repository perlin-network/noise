package network

import (
	"github.com/golang/glog"
	"github.com/golang/protobuf/ptypes"
	"github.com/perlin-network/noise/peer"
	"github.com/xtaci/smux"
	"net"
	"reflect"
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
				return
			}

			// Create a client if not exists.
			if client == nil {
				client, err = n.Client(msg.Sender.Address)

				if err != nil {
					return
				}
			}

			// Derive, set the peer ID, connect to the peer, and additionally
			// store the peer.
			id := peer.ID(*msg.Sender)

			if client.Id == nil {
				client.Id = &id

				err := client.establishConnection(id.Address)

				// Could not connect to peer; disconnect.
				if err != nil {
					glog.Errorf("Failed to connect to peer %s err=[%+v]\n", id.Address, err)
					return
				}
			} else if !client.Id.Equals(id) {
				// Peer sent message with a completely different ID (???)
				glog.Errorf("Message signed by peer %s but client is %s", client.Id.Address, id.Address)
				return
			}

			// Update routing table w/ peer's ID.
			if client.Network.Routes != nil {
				client.Network.Routes.Update(id)
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

			// Check if the received request has a message processor. If exists, execute it.
			name := reflect.TypeOf(ptr.Message).String()
			processor, exists := client.Network.Processors.Load(name)

			if exists {
				processor := processor.(MessageProcessor)

				// Create message execution context.
				ctx := new(MessageContext)
				ctx.client = client
				ctx.stream = stream
				ctx.message = ptr.Message
				ctx.nonce = msg.Nonce

				// Process request.
				err := processor.Handle(ctx)
				if err != nil {
					glog.Errorf("An error occurred handling %x: %x", name, err)
				}
			} else {
				glog.Warning("Unknown message type received:", name)
			}
		}(stream)
	}
}
