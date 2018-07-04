package network

import (
	"github.com/perlin-network/noise/protobuf"
	"net"
)

// Worker dispatches/queues up incoming/outgoing messages for a connection.
type Worker struct {
	sendQueue chan *protobuf.Message
	recvQueue chan *protobuf.Message

	needClose chan struct{}
	closed    chan struct{}
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

func (n *Network) loadWorker(address string) (*Worker, bool) {
	n.WorkersMutex.Lock()
	defer n.WorkersMutex.Unlock()

	if n.Workers == nil {
		return nil, false
	}

	if state, ok := n.Workers[address]; ok {
		return state, true
	} else {
		return nil, false
	}
}

func (n *Network) spawnWorker(address string) *Worker {
	n.WorkersMutex.Lock()
	defer n.WorkersMutex.Unlock()

	if n.Workers == nil {
		n.Workers = make(map[string]*Worker)
	}

	// Return worker if exists.
	if worker, exists := n.Workers[address]; exists {
		return worker
	}

	// Spawn and cache new worker otherwise.
	n.Workers[address] = &Worker{
		// TODO: Make queue size configurable.
		sendQueue: make(chan *protobuf.Message, 4096),
		recvQueue: make(chan *protobuf.Message, 4096),

		needClose: make(chan struct{}),
		closed:    make(chan struct{}),
	}

	return n.Workers[address]
}

func (n *Network) handleWorker(address string, worker *Worker) {
	defer func() {
		n.WorkersMutex.Lock()
		delete(n.Workers, address)
		n.WorkersMutex.Unlock()
	}()

	// Wait until worker is closed.
	<-worker.needClose
	close(worker.closed)
}
