package network

import "github.com/perlin-network/noise/protobuf"

type MessageProcessor interface {
	Handle(client *PeerClient, message protobuf.Message)
}