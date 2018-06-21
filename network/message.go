package network

type MessageProcessor interface {
	Handle(client *PeerClient, message *IncomingMessage)
}
