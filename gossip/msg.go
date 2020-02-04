package gossip

// Message is a message that is being pushed to nodes.
type Message []byte

// Marshal implements noise.Serializable and serializes Message into a slice of bytes.
func (m Message) Marshal() []byte {
	return m
}

// UnmarshalMessage deserializes data into a Message. It never throws an error.
func UnmarshalMessage(data []byte) (Message, error) {
	return data, nil
}
