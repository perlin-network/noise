package network

import (
	"net"

	"github.com/golang/glog"
	"github.com/golang/protobuf/ptypes"
	"github.com/perlin-network/noise/peer"
	"github.com/perlin-network/noise/protobuf"
)

// Ingest handles peer registration and processes incoming message streams consisting of
// asynchronous messages or request/responses.
func (n *Network) Ingest(conn net.Conn) {
	var client *PeerClient

	glog.Info("INGEST ENTER")

	for {
		var msg *protobuf.Message
		var err error

		if client == nil {
			msg, err = n.receiveMessage(conn)
			if err != nil {
				glog.Error(err)
				break
			}
			n.EnsureConnectionState(msg.Sender.Address, conn)
			client, err = n.Client(msg.Sender.Address)
			client.ID = (*peer.ID)(msg.Sender)
			glog.Info("client initialized")
		} else {
			msg, err = n.ReadMessage(client.Address)
		}

		if err != nil {
			glog.Error(err)
			return
		}

		/*
		if !client.ID.Equals(id) {
			// Peer sent message with a completely different ID (???)
			glog.Errorf("Message signed by peer %s but client is %s", client.ID.Address, id.Address)
			return
		}*/

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

		switch ptr.Message.(type) {
		case *protobuf.StreamPacket:
			pkt := ptr.Message.(*protobuf.StreamPacket)
			client.handleStreamPacket(pkt.Data)
			return
		}

		// Create message execution context.
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
