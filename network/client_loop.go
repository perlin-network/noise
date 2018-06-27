package network

import (
	"github.com/perlin-network/noise/peer"
	"reflect"
	"github.com/xtaci/smux"
	"github.com/golang/glog"
	"github.com/golang/protobuf/ptypes"
)

func (c *PeerClient) handleMessage(stream *smux.Stream) {
	// Clean up resources.
	defer stream.Close()

	msg, err := c.receiveMessage(stream)

	// Failed to receive message.
	if err != nil {
		if err.Error() == "broken pipe" {
			c.close()
		}
		return
	}

	// Derive, set the peer ID, connect to the peer, and additionally
	// store the peer.
	id := peer.ID(*msg.Sender)

	if c.Id == nil {
		c.Id = &id

		err := c.establishConnection(id.Address)

		// Could not connect to peer; disconnect.
		if err != nil {
			glog.Errorf("Failed to connect to peer %s err=[%+v]\n", id.Address, err)
			return
		}
	} else if !c.Id.Equals(id) {
		// Peer sent message with a completely different ID (???)
		glog.Errorf("Message signed by peer %s but client is %s", c.Id.Address, id.Address)
		return
	}

	// Update routing table w/ peer's ID.
	c.Network.Routes.Update(id)

	// Unmarshal protobuf.
	var ptr ptypes.DynamicAny
	if err := ptypes.UnmarshalAny(msg.Message, &ptr); err != nil {
		glog.Error(err)
		return
	}

	// Check if the received request has a message processor. If exists, execute it.
	name := reflect.TypeOf(ptr.Message).String()
	processor, exists := c.Network.Processors.Load(name)

	if exists {
		processor := processor.(MessageProcessor)

		// Create message execution context.
		ctx := new(MessageContext)
		ctx.client = c
		ctx.stream = stream
		ctx.message = ptr.Message

		// Process request.
		err := processor.Handle(ctx)
		if err != nil {
			glog.Errorf("An error occurred handling %x: %x", name, err)
		}
	} else {
		glog.Warning("Unknown message type received:", name)
	}
}
