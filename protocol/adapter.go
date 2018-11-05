package protocol

type RecvMessageCallback func(message []byte)

type ConnectionAdapter interface {
	EstablishPassively(c *Controller, local []byte) chan MessageAdapter
	EstablishActively(c *Controller, local []byte, remote []byte) (MessageAdapter, error)
}

type MessageAdapter interface {
	RemoteEndpoint() []byte
	StartRecvMessage(c *Controller, callback RecvMessageCallback)
	SendMessage(c *Controller, message []byte) error
}

type IdentityAdapter interface {
	MyIdentity() []byte
	Sign(input []byte) []byte
	Verify(id, data, signature []byte) bool
	SignatureSize() int
}
