package protocol

// ServiceInterface is used to proxy callbacks to a particular Plugin instance.
type ServiceInterface interface {
	// Callback for when the network starts listening for peers.
	Startup(node *Node)

	// Callback for when an incoming message is received.
	// Returns a message body to reply or whether there was an error.
	Receive(request *Message) (*MessageBody, error)

	// Callback for when the network stops listening for peers.
	Cleanup(node *Node)

	// Callback for when a peer connects to the node
	PeerConnect(id []byte)

	// Callback for when a peer disconnects from the node.
	PeerDisconnect(id []byte)
}

// Service is an abstract class which all services extend.
type Service struct{}

// Hook callbacks of network builder services

// Startup is called only once when the service is loaded
func (*Service) Startup(node *Node) {}

// Receive is called every time when messages are received
func (*Service) Receive(request *Message) (*MessageBody, error) { return nil, nil }

// Cleanup is called only once after network stops listening
func (*Service) Cleanup(node *Node) {}

// PeerConnect is called every time a PeerClient is initialized and connected
func (*Service) PeerConnect(id []byte) {}

// PeerDisconnect is called every time a PeerClient connection is closed
func (*Service) PeerDisconnect(id []byte) {}
