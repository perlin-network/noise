package network

import (
	"bufio"
	"encoding/binary"
	"errors"
	"github.com/golang/protobuf/proto"
	"github.com/perlin-network/noise/crypto"
	"github.com/perlin-network/noise/protobuf"
	"github.com/xtaci/smux"
	"io"
)

// sendMessage marshals and sends a message over a stream.
func (c *PeerClient) sendMessage(stream *smux.Stream, message proto.Message) error {
	req, err := c.prepareMessage(message)
	if err != nil {
		return err
	}

	bytes, err := proto.Marshal(req)
	if err != nil {
		return err
	}

	// Serialize size.
	buffer := make([]byte, binary.MaxVarintLen64)
	binary.PutUvarint(buffer, uint64(len(bytes)))

	// Prefix message with its size.
	bytes = append(buffer, bytes...)

	writer := bufio.NewWriter(stream)

	// Send request bytes.
	n, err := writer.Write(bytes)
	if err != nil {
		return err
	}

	// Flush writer.
	err = writer.Flush()
	if err != nil {
		if err == io.EOF {
			c.close()
		}
		return err
	}

	if n != len(bytes) {
		return errors.New("failed to write all bytes to stream")
	}

	return nil
}

// receiveMessage reads, unmarshals and verifies a message from a stream.
func (c *PeerClient) receiveMessage(stream *smux.Stream) (*protobuf.Message, error) {
	reader := bufio.NewReader(stream)

	buffer := make([]byte, binary.MaxVarintLen64)

	_, err := reader.Read(buffer)
	if err != nil {
		return nil, err
	}

	// Decode unsigned varint representing message size.
	size, n := binary.Uvarint(buffer)

	// Check if unsigned varint overflows, or if protobuf message is too large.
	if n <= 0 || size > 1<<31-1 {
		return nil, errors.New("message len is either broken or too large")
	}

	// Read message from buffered I/O completely.
	buffer = make([]byte, size)
	_, err = io.ReadFull(reader, buffer)

	if err != nil {
		if err == io.EOF {
			c.close()
		}

		return nil, err
	}

	// Deserialize message.
	msg := new(protobuf.Message)

	err = proto.Unmarshal(buffer, msg)
	if err != nil {
		return nil, err
	}

	// Check if any of the message headers are invalid or null.
	if msg.Message == nil || msg.Sender == nil || msg.Sender.PublicKey == nil || len(msg.Sender.Address) == 0 || msg.Signature == nil {
		return nil, errors.New("received an invalid message (either no message, no sender, or no signature) from a peer")
	}

	// Verify signature of message.
	if !crypto.Verify(msg.Sender.PublicKey, msg.Message.Value, msg.Signature) {
		return nil, errors.New("received message had an malformed signature")
	}

	return msg, err
}
