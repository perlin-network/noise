package protocol

type RecvMessageCallback func(message []byte)

// ConnectionAdapter is an adapter that establishes real/virtual connections (message adapters), both passively and actively.
type ConnectionAdapter interface {
	EstablishPassively(c *Controller, local []byte) chan MessageAdapter
	EstablishActively(c *Controller, local []byte, remote []byte) (MessageAdapter, error)
	AddConnection(id []byte, addr string)
	GetConnectionIDs() [][]byte
	GetAddressByID(id []byte) (string, error)
}

// MessageAdapter is an adapter that sends/receives messages, usually corresponding to a real/virtual connection.
type MessageAdapter interface {
	RemoteEndpoint() []byte
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
