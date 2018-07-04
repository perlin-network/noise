package network

import (
	"net"

	"github.com/golang/glog"
	"github.com/golang/protobuf/ptypes"
	"github.com/perlin-network/noise/peer"
	"github.com/perlin-network/noise/protobuf"
	"github.com/xtaci/smux"
)

// Ingest handles peer registration and processes incoming message streams consisting of
// asynchronous messages or request/responses.
func (n *Network) Ingest(conn net.Conn) {
	var incoming net.Conn
	var outgoing net.Conn

	// Wrap a session around the incoming connection.
	session, err := smux.Server(conn, muxConfig())
	if err != nil {
		return
	}

	// Accept an incoming stream.
	incoming, err = session.AcceptStream()
	if err != nil {
		return
	}

	var msg *protobuf.Message

	/** START COMMENCE HANDSHAKE **/

	// Attempt to receive message.
	if msg, err = n.receiveMessage(incoming); err != nil {
		return
	}

	var ptr ptypes.DynamicAny
	if err := ptypes.UnmarshalAny(msg.Message, &ptr); err != nil {
		return
	}

	senderAddress := msg.Sender.Address

	switch ptr.Message.(type) {
	case *protobuf.Ping:
		// If ping received, we assign the incoming stream to a new worker.
		// We must additionally setup an outgoing stream to fully create a new worker.

		// Dial senders address.
		if outgoing, err = dialAddress(senderAddress); err != nil {
			return
		}

		// Prepare a pong response.
		pong, err := n.prepareMessage(&protobuf.Pong{})

		if err != nil {
			glog.Error(err)
			return
		}

		// Respond with a pong.
		err = n.sendMessage(outgoing, pong)

		if err != nil {
			glog.Error(err)
			return
		}

		// Outgoing and incoming exists. Handshake is successful.
		// Lets now create the worker.
		worker := n.spawnWorker(senderAddress)

		go worker.startSender(n, outgoing)
		go worker.startReceiver(n, incoming)
		go n.handleWorker(senderAddress, worker)
	case *protobuf.Pong:
		// If pong received, we assign the incoming stream to a cached worker.
		// If the worker doesn't exist, then the pong is pointless and we disconnect.

		if worker, exists := n.loadWorker(senderAddress); exists {
			go worker.startReceiver(n, incoming)

			// Send normal ping now.
			ping, err := n.prepareMessage(&protobuf.Ping{})

			if err != nil {
				glog.Error(err)
				return
			}

			n.WriteMessage(senderAddress, ping)
		} else {
			return
		}
	default:
		// Shouldn't be receiving any other messages. Disconnect.
		glog.Error("Got message: ", ptr.Message)
		return
	}

	/** END COMMENCE HANDSHAKE **/

	// Lets now setup our peer client.
	client, err := n.Client(senderAddress)
	if err != nil {
		glog.Error(err)
		return
	}
	client.ID = (*peer.ID)(msg.Sender)

	defer client.Close()

	for {
		msg, err = n.ReadMessage(senderAddress)

		// Disconnections will occur here.
		if err != nil {
			return
		}

		id := (peer.ID)(*msg.Sender)

		// Peer sent message with a completely different ID. Destroy.
		if !client.ID.Equals(id) {
			glog.Errorf("Message signed by peer %s but client is %s", client.ID.Address, id.Address)
			return
		}

		// Unmarshal message.
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

		switch ptr.Message.(type) {
		case *protobuf.StreamPacket: // Handle stream packet message.
			pkt := ptr.Message.(*protobuf.StreamPacket)
			client.handleStreamPacket(pkt.Data)
			return

		default: // Handle other messages.
			ctx := new(MessageContext)
			ctx.client = client
			ctx.message = ptr.Message
			ctx.nonce = msg.Nonce

			// Execute 'on receive message' callback for all plugins.
			n.Plugins.Each(func(plugin PluginInterface) {
				err := plugin.Receive(ctx)

				if err != nil {
					glog.Error(err)
				}
			})
		}
	}
}
