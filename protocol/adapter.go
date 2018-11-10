package protocol

type RecvMessageCallback func(message []byte)

// A ConnectionAdapter is an adapter that establishes real/virtual connections (message adapters), both passively and actively.
type ConnectionAdapter interface {
	EstablishPassively(c *Controller, local []byte) chan MessageAdapter
	EstablishActively(c *Controller, local []byte, remote []byte) (MessageAdapter, error)
}

// A MessageAdapter is an adapter that sends/receives messages, usually corresponding to a real/virtual connection.
type MessageAdapter interface {
	RemoteEndpoint() []byte
	StartRecvMessage(c *Controller, callback RecvMessageCallback)
	SendMessage(c *Controller, message []byte) error
	Close()
}

// An IdentityAdapter is an adapter that provides identity-related operations like signing and verification.
type IdentityAdapter interface {
	MyIdentity() []byte
	Sign(input []byte) []byte
	Verify(id, data, signature []byte) bool
	SignatureSize() int
}
