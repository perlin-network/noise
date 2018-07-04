package network

import (
	"errors"
	"github.com/perlin-network/noise/protobuf"
	"github.com/xtaci/kcp-go"
	"github.com/xtaci/smux"
	"net"
	"net/url"
)

// Worker dispatches/queues up incoming/outgoing messages for a connection.
type Worker struct {
	sendQueue chan *protobuf.Message
	recvQueue chan *protobuf.Message

	needClose chan struct{}
	closed    chan struct{}
}

func dialAddress(address string) (net.Conn, error) {
	urlInfo, err := url.Parse(address)
	if err != nil {
		return nil, err
	}

	var conn net.Conn

	// Choose scheme.
	if urlInfo.Scheme == "kcp" {
		conn, err = kcp.DialWithOptions(urlInfo.Host, nil, 10, 3)
	} else if urlInfo.Scheme == "tcp" {
		conn, err = net.Dial("tcp", urlInfo.Host)
	} else {
		err = errors.New("Invalid scheme: " + urlInfo.Scheme)
	}

	// Failed to connect.
	if err != nil {
		return nil, err
	}

	// Wrap a session around the outgoing connection.
	session, err := smux.Client(conn, muxConfig())
	if err != nil {
		return nil, err
	}

	// Open an outgoing stream.
	stream, err := session.OpenStream()
	if err != nil {
		return nil, err
	}

	return stream, nil
}

func (s *Worker) IsClosed() bool {
	select {
	case <-s.closed:
		return true
	default:
		return false
	}
}

func (s *Worker) Close() {
	select {
	case s.needClose <- struct{}{}:
	default:
	}
}

func (s *Worker) startReceiver(n *Network, c net.Conn) {
	defer c.Close()
	defer close(s.recvQueue)

	for {
		if s.IsClosed() {
			return
		}

		message, err := n.receiveMessage(c)

		if err != nil {
			s.Close()
			return
		}

		// Dispatch received message to the receive queue.
		s.recvQueue <- message
	}
}

func (s *Worker) startSender(n *Network, c net.Conn) {
	defer c.Close()
	defer close(s.sendQueue)

	for {
		if s.IsClosed() {
			return
		}

		// Dispatch message should a message be available in the send queue.
		select {
		case message := <-s.sendQueue:
			err := n.sendMessage(c, message)

			if err != nil {
				s.Close()
				return
			}
		case <-s.closed:
			return
		}
	}
}
