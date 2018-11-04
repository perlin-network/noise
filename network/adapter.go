package network

type ConnectionAdapter interface {
	EstablishPassively(c *Controller) chan MessageAdapter
	EstablishActively(c *Controller, remote []byte) (MessageAdapter, error)
}

type MessageAdapter interface {
	RemoteEndpoint() []byte
	SendMessage(c *Controller, message []byte) error
	RecvMessage(c *Controller) ([]byte, error)
}

type IdentityAdapter interface {
	MyIdentity() []byte
	Sign(input []byte) []byte
	Verify(id, data, signature []byte) bool
}
