package noise

import (
	"github.com/pkg/errors"
	"time"
)

type Locker <-chan struct{}

func (m Locker) Unlock() {
	if len(m) > 0 {
		<-m
	}
}

type evtSend struct {
	opcode byte
	mux    uint64

	msg []byte
}

type evtRecv struct {
	ch   chan Wire
	lock chan struct{}
}

type Mux struct {
	peer *Peer
	id   uint64
}

// Send is equivalent to calling SendWithTimeout(opcode, msg, 0).
func (m Mux) Send(opcode byte, msg []byte) error {
	return m.SendWithTimeout(opcode, msg, 0)
}

// SendWithTimeout sends a message registered under an opcode.
//
// Should a timeout greater than zero be provided, an error will be returned
// if sending a message takes longer than the specified timeout duration.
//
// Setting a timeout sets a deadline for all present and future messages which
// are sent to the peer, including ones that are running in parallel.
//
// The timeout is enforced using the `SetWriteDeadline(time.Duration) error`
// function provided by the transport method underlying the socket of the peer.
//
// It returns an error if an error exists in setting a timeout on the peers underlying
// socket, if the send queue is full, or if an error occurred sending a message to
// a peer.
func (m Mux) SendWithTimeout(opcode byte, msg []byte, timeout time.Duration) error {
	evt := evtSend{opcode: opcode, mux: m.id, msg: msg}

	if timeout > 0 {
		err := m.peer.SetWriteDeadline(time.Now().Add(timeout))

		if err != nil {
			return err
		}
	}

	select {
	case <-m.peer.ctx.stop:
		return ErrDisconnect
	case m.peer.send <- evt:
	default:
		return ErrSendQueueFull
	}

	if timeout > 0 {
		err := m.peer.SetWriteDeadline(time.Time{})

		if err != nil {
			return err
		}
	}

	return nil
}

// Recv returns a receive-only channel that transmits messages under a specified
// opcode upon recipient.
//
// It is designed to return a receive-only channel such that it may be multiplexed
// with a series of other receive-only channel signals through a `select` statement.
//
// It does not spawn or leak any new channels or additional goroutines, and thus
// may be used sparingly without any concern for garbage collection.
func (m Mux) Recv(opcode byte) <-chan Wire {
	return m.peer.getMuxQueue(m.id, opcode).ch
}

func (m Mux) LockOnRecv(opcode byte) Locker {
	hub := m.peer.getMuxQueue(m.id, opcode)
	hub.lock <- struct{}{}
	return hub.lock
}

// Close de-registers the current mux from its associated peer.
func (m Mux) Close() error {
	if m.id == 0 {
		return errors.New("noise: cannot close default mux")
	}

	m.peer.recvLock.Lock()
	delete(m.peer.recv, m.id)
	m.peer.recvLock.Unlock()
	return nil
}
