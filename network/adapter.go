package network

type ConnectionAdapter interface {
	EstablishPassively(c *Controller, onRecvMessage func(message []byte)) chan MessageAdapter
	EstablishActively(c *Controller, remote []byte, onRecvMessage func(message []byte)) (MessageAdapter, error)
}

type MessageAdapter interface {
	RemoteEndpoint() []byte
	SendMessage(c *Controller, message []byte) error
}

type IdentityAdapter interface {
	MyIdentity() []byte
	Sign(input []byte) []byte
	Verify(id, data, signature []byte) bool
}
