package network

import "github.com/golang/protobuf/proto"

// MessageChannel represents a channel for arbitrary protobuf messages.
type MessageChannel chan proto.Message
