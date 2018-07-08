package network

import (
	"bufio"
	"encoding/binary"
	"io"
	"net"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/perlin-network/noise/crypto"
	"github.com/perlin-network/noise/protobuf"
	"github.com/pkg/errors"
)

// sendMessage marshals, signs and sends a message over a stream.
func (n *Network) sendMessage(stream net.Conn, message *protobuf.Message) error {
	bytes, err := proto.Marshal(message)
	if err != nil {
		return errors.Wrap(err, "failed to marshal message")
	}

	// Serialize size.
	buffer := make([]byte, binary.MaxVarintLen64)
	binary.PutUvarint(buffer, uint64(len(bytes)))

	// Prefix message with its size.
	bytes = append(buffer, bytes...)

	stream.SetDeadline(time.Now().Add(3 * time.Second))

	writer := bufio.NewWriter(stream)

	// Send request bytes.
	written, err := writer.Write(bytes)
	if err != nil {
		return errors.Wrap(err, "failed to send request bytes")
	}

	// Flush writer.
	err = writer.Flush()
	if err != nil {
		return errors.Wrap(err, "failed to flush writer")
	}

	if written != len(bytes) {
		return errors.Errorf("only wrote %d / %d bytes to stream", written, len(bytes))
	}

	return nil
}

// receiveMessage reads, unmarshals and verifies a message from a stream.
func (n *Network) receiveMessage(stream net.Conn) (*protobuf.Message, error) {
	reader := bufio.NewReader(stream)

	buffer := make([]byte, binary.MaxVarintLen64)

	_, err := reader.Read(buffer)
	if err != nil {
		return nil, errors.Wrap(err, "failed to recv message size")
	}

	// Decode unsigned varint representing message size.
	size, read := binary.Uvarint(buffer)

	// Check if unsigned varint overflows, or if protobuf message is too large.
	// Message size at most is limited to 4MB. If a big message need be sent,
	// consider partitioning to message into chunks of 4MB.
	if read <= 0 || size > 4e+6 {
		return nil, errors.New("message len is either broken or too large")
	}

	// Read message from buffered I/O completely.
	buffer = make([]byte, size)
	_, err = io.ReadFull(reader, buffer)

	if err != nil {
		// Potentially malicious or dead client; kill it.
		if err == io.ErrUnexpectedEOF {
			stream.Close()
		}
		return nil, errors.Wrap(err, "failed to recv serialized message")
	}

	// Deserialize message.
	msg := new(protobuf.Message)

	err = proto.Unmarshal(buffer, msg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal message")
	}

	// Check if any of the message headers are invalid or null.
	if msg.Message == nil || msg.Sender == nil || msg.Sender.PublicKey == nil || len(msg.Sender.Address) == 0 || msg.Signature == nil {
		return nil, errors.New("received an invalid message (either no message, no sender, or no signature) from a peer")
	}

	// Verify signature of message.
	if !crypto.Verify(
		n.SignaturePolicy,
		n.HashPolicy,
		msg.Sender.PublicKey,
		serializeMessageInfoForSigning(msg.Sender, msg.Message.Value),
		msg.Signature,
	) {
		return nil, errors.New("received message had an malformed signature")
	}

	return msg, nil
}
