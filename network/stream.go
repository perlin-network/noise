package network

import (
	"github.com/xtaci/smux"
	"bufio"
	"io"
	"github.com/perlin-network/noise/protobuf"
	"encoding/binary"
	"github.com/golang/protobuf/proto"
	"errors"
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

// receiveMessage reads and unmarshals a message from a stream.
func (c *PeerClient) receiveMessage(stream *smux.Stream) (*protobuf.Message, error) {
	reader := bufio.NewReader(stream)

	buffer := make([]byte, binary.MaxVarintLen64)

	_, err := reader.Read(buffer)
	if err != nil {
		return nil, err
	}

	size, n := binary.Uvarint(buffer)

	if n <= 0 || size > 1<<31-1 {
		return nil, errors.New("message len is either broken or too large")
	}

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

	return msg, err
}
