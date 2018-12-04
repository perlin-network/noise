package protocol

import (
	"context"
)

// ConnectionAdapter is an adapter that establishes real/virtual connections (message adapters), both passively and actively.
type ConnectionAdapter interface {
	// Accept returns a message adapter when a remote node connects to this node
	Accept(c *Controller, local []byte) chan MessageAdapter

	// Dial will try to connect to a remote node
	Dial(c *Controller, local []byte, remote []byte) (MessageAdapter, error)

	// AddRemoteIDs adds an ID and metadata to the routing table
	AddRemoteID(id []byte, addr string) error

	// GetRemoteIDs returns all the IDs in the connection adapter's routing table
	GetRemoteIDs() [][]byte
}

// RecvMessageCallback is a callback when a message is received from a peer
type RecvMessageCallback func(ctx context.Context, message []byte)

// MessageAdapter is an adapter that sends/receives messages, usually corresponding to a real/virtual connection.
type MessageAdapter interface {
	// RemoteID is the ID of the remote side of the connection
	RemoteID() []byte

	// Metadata are additional metadata about the connection
	Metadata() map[string]string

	// OnRecvMessage is the handler when a message received
	OnRecvMessage(c *Controller, callback RecvMessageCallback)

	// SendMessage writes the message to the connection
	SendMessage(c *Controller, message []byte) error

	// Close cleans up the connection resources
	Close()
}

// IdentityAdapter is an adapter that provides identity-related operations like signing and verification.
type IdentityAdapter interface {
	// MyIdentity is the node's public ID
	MyIdentity() []byte

	// Sign will cryptographically sign the input and return the result
	Sign(input []byte) []byte

	// Verify checks if given the node ID and data generates the input signature bytes
	Verify(id, data, signature []byte) bool

	// SignatureSize returns the number of bytes in the signature
	SignatureSize() int
}

// DoneAction are states that the handshake process goes through
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
	// Send will deliver a one way message to the recipient node
	Send(ctx context.Context, recipient []byte, body *MessageBody) error

	// Request will send a message to the recipient and wait for a reply
	Request(ctx context.Context, recipient []byte, body *MessageBody) (*MessageBody, error)

	// Broadcast sends a message to all it's currently connected peers
	Broadcast(ctx context.Context, body *MessageBody) error
}
