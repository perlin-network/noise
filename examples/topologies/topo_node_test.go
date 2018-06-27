package topologies

import (
	"github.com/perlin-network/noise/examples/basic"
	"github.com/perlin-network/noise/examples/basic/messages"
	"github.com/perlin-network/noise/network"
)

// TopoNode holds the variables to create the network and implements the message handler
type TopoNode struct {
	h        string
	p        int
	ps       []string
	net      *network.Network
	Messages []*messages.BasicMessage
}

func (e *TopoNode) Host() string {
	return e.h
}

func (e *TopoNode) Port() int {
	return e.p
}

func (e *TopoNode) Peers() []string {
	return e.ps
}

func (e *TopoNode) Net() *network.Network {
	return e.net
}

func (e *TopoNode) SetNet(n *network.Network) {
	e.net = n
}

// Handle implements the network interface callback
func (e *TopoNode) Handle(client *network.PeerClient, raw *network.IncomingMessage) error {
	message := raw.Message.(*messages.BasicMessage)

	e.Messages = append(e.Messages, message)

	return nil
}

// PopMessage returns the oldest message from it's buffer and removes it from the list
func (e *TopoNode) PopMessage() *messages.BasicMessage {
	if len(e.Messages) == 0 {
		return nil
	}
	var retVal *messages.BasicMessage
	retVal, e.Messages = e.Messages[0], e.Messages[1:]
	return retVal
}

// makes sure the implementation matches the interface at compile time
var _ basic.ClusterNode = (*TopoNode)(nil)
