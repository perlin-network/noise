package network

type Service interface {
	HandleMessage(message *MessageBody)
}
