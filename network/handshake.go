package network

import (
	"net"

	"github.com/golang/glog"
	"github.com/perlin-network/noise/peer"
	"github.com/perlin-network/noise/protobuf"
	"github.com/xtaci/smux"
)

func (n *Network) processHandshake(conn net.Conn) *peer.ID {
	var incoming net.Conn
	var outgoing net.Conn

	// Wrap a session around the incoming connection.
	session, err := smux.Server(conn, muxConfig())
	if err != nil {
		return nil
	}

	// Accept an incoming stream.
	incoming, err = session.AcceptStream()
	if err != nil {
		return nil
	}

	var msg *protobuf.Message

	// Attempt to receive message.
	if msg, err = n.receiveMessage(incoming); err != nil {
		return nil
	}

	id := (*peer.ID)(msg.Sender)

	switch msg.Message.TypeUrl {
	case "type.googleapis.com/protobuf.Ping":
		// If ping received, we assign the incoming stream to a new worker.
		// We must additionally setup an outgoing stream to fully create a new worker.

		// Dial senders address.
		if outgoing, err = n.dial(id.Address); err != nil {
			return nil
		}

		// Prepare a pong response.
		pong, err := n.prepareMessage(&protobuf.Pong{})

		if err != nil {
			glog.Error(err)
			return nil
		}

		// Respond with a pong.
		err = n.sendMessage(outgoing, pong)

		if err != nil {
			glog.Error(err)
			return nil
		}

		// Outgoing and incoming exists. Handshake is successful.
		// Lets now create the worker.
		worker := n.spawnWorker(id.Address)

		go worker.startSender(n, outgoing)
		go worker.startReceiver(n, incoming)
		go worker.process(n, id.Address)

		return id
	case "type.googleapis.com/protobuf.Pong":
		// If pong received, we assign the incoming stream to a cached worker.
		// If the worker doesn't exist, then the pong is pointless and we disconnect.

		worker, exists := n.loadWorker(id.Address)
		if !exists {
			glog.Errorf("worker %s does not exist", id.Address)
			return nil
		}

		go worker.startReceiver(n, incoming)

		// Send normal ping now.
		ping, err := n.prepareMessage(&protobuf.Ping{})

		if err != nil {
			glog.Error(err)
			return nil
		}

		err = n.WriteMessage(id.Address, ping)

		if err != nil {
			glog.Error(err)
			return nil
		}

		return id
	default:
		// Shouldn't be receiving any other messages. Disconnect.
		glog.Error("got unexpected message during handshake")
	}

	return nil
}
