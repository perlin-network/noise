package network

import (
	"encoding/binary"
	"net"
	"github.com/gogo/protobuf/proto"
	"github.com/perlin-network/noise/crypto"
	"github.com/perlin-network/noise/protobuf"
	"github.com/pkg/errors"
	"sync"
	"io"
)

// sendMessage marshals, signs and sends a message over a stream.
func (n *Network) sendMessage(conn io.Writer, message *protobuf.Message, writerMutex *sync.Mutex) error {
	bytes, err := proto.Marshal(message)
	if err != nil {
		return errors.Wrap(err, "failed to marshal message")
	}

	// Serialize size.
	buffer := make([]byte, 4)
	binary.BigEndian.PutUint32(buffer, uint32(len(bytes)))

	buffer = append(buffer, bytes...)

	// Write until all bytes have been written.
	bytesWritten, totalBytesWritten := 0, 0

	writerMutex.Lock()

	for totalBytesWritten < len(buffer) && err == nil {
		bytesWritten, err = conn.Write(buffer[totalBytesWritten:])
		totalBytesWritten += bytesWritten
	}

	writerMutex.Unlock()

	if err != nil {
		return errors.Wrap(err, "failed to write to socket")
	}

	return nil
}

// receiveMessage reads, unmarshals and verifies a message from a net.Conn.
func (n *Network) receiveMessage(conn net.Conn) (*protobuf.Message, error) {
	var err error

	// Read until all header bytes have been read.
	buffer := make([]byte, 4)

	bytesRead, totalBytesRead := 0, 0

	for totalBytesRead < 4 && err == nil {
		bytesRead, err = conn.Read(buffer[totalBytesRead:])
		totalBytesRead += bytesRead
	}

	// Decode message size.
	size := binary.BigEndian.Uint32(buffer)

	// Message size at most is limited to 4MB. If a big message need be sent,
	// consider partitioning to message into chunks of 4MB.
	if size > 4e+6 {
		return nil, errors.Errorf("message has length of %d which is either broken or too large", size)
	}

	// Read until all message bytes have been read.
	buffer = make([]byte, size)

	bytesRead, totalBytesRead = 0, 0

	for totalBytesRead < int(size) && err == nil {
		bytesRead, err = conn.Read(buffer[totalBytesRead:])
		totalBytesRead += bytesRead
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
		SerializeMessage(msg.Sender, msg.Message.Value),
		msg.Signature,
	) {
		return nil, errors.New("received message had an malformed signature")
	}

	return msg, nil
}
