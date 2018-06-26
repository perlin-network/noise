package network

import "github.com/golang/protobuf/proto"

type MessageProcessor interface {
	Handle(client *PeerClient, message proto.Message) error
}
