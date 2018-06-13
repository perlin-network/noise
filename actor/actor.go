package actor

import (
	"github.com/perlin-network/noise/peer"
	"github.com/perlin-network/noise/protobuf"
)

type MessageReceiver interface {
	Receive(client protobuf.Noise_StreamClient, sender peer.ID, message interface{})
}

type Actor struct {
	MessageReceiver
}
