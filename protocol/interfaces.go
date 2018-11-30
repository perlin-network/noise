package protocol

import (
	"context"
)

type RecvMessageCallback func(ctx context.Context, message []byte)

// ConnectionAdapter is an adapter that establishes real/virtual connections (message adapters), both passively and actively.
type ConnectionAdapter interface {
	EstablishPassively(c *Controller, local []byte) chan MessageAdapter
	EstablishActively(c *Controller, local []byte, remote []byte) (MessageAdapter, error)
	AddPeerID(id []byte, addr string) error
	GetPeerIDs() [][]byte
}

// MessageAdapter is an adapter that sends/receives messages, usually corresponding to a real/virtual connection.
type MessageAdapter interface {
	RemoteEndpoint() []byte
	Metadata() map[string]string
	StartRecvMessage(c *Controller, callback RecvMessageCallback)
	SendMessage(c *Controller, message []byte) error
	Close()
}

// IdentityAdapter is an adapter that provides identity-related operations like signing and verification.
type IdentityAdapter interface {
	MyIdentity() []byte
	Sign(input []byte) []byte
	Verify(id, data, signature []byte) bool
	SignatureSize() int
}

type DoneAction byte

const (
	DoneAction_Invalid DoneAction = iota
	DoneAction_NotDone
	DoneAction_SendMessage
	DoneAction_DoNothing
)

// HandshakeProcessor is called while connections are being made
type HandshakeProcessor interface {
	ActivelyInitHandshake() ([]byte, interface{}, error)                                   // (message, state, err)
	PassivelyInitHandshake() (interface{}, error)                                          // (state, err)
	ProcessHandshakeMessage(state interface{}, payload []byte) ([]byte, DoneAction, error) // (message, doneAction, err)
}

// SendAdapter is an adapter that manages sending messages
type SendAdapter interface {
	Send(ctx context.Context, message *Message) error
	Request(ctx context.Context, target []byte, body *MessageBody) (*MessageBody, error)
	Broadcast(ctx context.Context, body *MessageBody) error
}
