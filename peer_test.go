package noise

import (
	"bytes"
	"crypto/rand"
	"github.com/fortytw2/leaktest"
	"github.com/perlin-network/noise/internal/iotest"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"net"
	"testing"
	"time"
)

func TestPeerDisconnectsProperly(t *testing.T) {
	defer leaktest.Check(t)()

	conn, _ := net.Pipe()

	p := newPeer(nil, nil, conn, conn, conn)

	go p.Disconnect(nil)
	p.Start()

	conn, _ = net.Pipe()

	p = newPeer(nil, nil, conn, conn, conn)

	go p.Start()
	go p.Disconnect(nil)
}

//func TestPeerSendsCorrectly(t *testing.T) {
//	defer leaktest.Check(t)()
//
//	w := bytes.NewBuffer(nil)
//
//	c, _ := net.Pipe()
//
//	node := NewNode(nil)
//	node.Handle(0x01, nil)
//
//	p := newPeer(node, nil, w, new(iotest.NopReader), c)
//	defer p.Disconnect(nil)
//
//	go p.Start()
//
//	msg := []byte("lorem ipsum")
//	assert.NoError(t, p.Send(0x01, msg))
//
//	for w.Len() == 0 {}
//
//	L: for {
//		for _, b := range w.Bytes() {
//			if b != 0 {
//				break L
//			}
//		}
//	}
//
//	r := bytes.NewReader(w.Bytes())
//
//	var receivedLength uint32
//	assert.NoError(t, binary.Read(r, binary.BigEndian, &receivedLength))
//	assert.Equal(t, uint32(len(msg)+5), receivedLength)
//
//	var receivedNonce uint32
//	assert.NoError(t, binary.Read(r, binary.BigEndian, &receivedNonce))
//	assert.Equal(t, receivedNonce, uint32(0))
//
//	var receivedOpcode byte
//	assert.NoError(t, binary.Read(r, binary.BigEndian, &receivedOpcode))
//	assert.Equal(t, uint8(0x01), receivedOpcode)
//
//	receivedBuf := make([]byte, len(msg))
//
//	n, err := r.Read(receivedBuf)
//	assert.NoError(t, err)
//	assert.Equal(t, len(msg), n)
//
//	assert.Equal(t, msg, receivedBuf)
//}

func TestPeerSendAndReceivesCorrectly(t *testing.T) {
	defer leaktest.Check(t)()

	a, b := net.Pipe()

	alice := newPeer(nil, a.RemoteAddr(), a, a, a)
	bob := newPeer(nil, b.RemoteAddr(), b, b, b)

	go alice.Start()
	go bob.Start()

	defer bob.Disconnect(nil)
	defer alice.Disconnect(nil)

	msg := []byte("lorem ipsum")
	go assert.NoError(t, alice.Send(0x16, msg))

	select {
	case buf := <-bob.Recv(0x16):
		assert.Equal(t, msg, buf)
	case <-time.After(1 * time.Second):
		t.Fail()
	}
}

// send with timeout won't work this way since actual write is performed in flush, which is asynchronous
//
//func TestPeerSetWriteDeadline(t *testing.T) {
//	defer leaktest.Check(t)()
//
//	rw := iotest.NewBlockingReadWriter()
//
//	p := newPeer(nil, nil, rw, new(iotest.NopReader), rw)
//	defer p.Disconnect(nil)
//
//	go p.Start()
//
//	err := p.SendWithTimeout(0x01, []byte("lorem ipsum"), 1*time.Millisecond)
//	assert.Equal(t, io.EOF, errors.Cause(err))
//}

func TestPeerSetReadDeadline(t *testing.T) {
	defer leaktest.Check(t)()

	rw := iotest.NewBlockingReadWriter()

	p := newPeer(nil, nil, new(iotest.NopWriter), rw, rw)
	defer p.Disconnect(nil)

	go p.Start()

	assert.NoError(t, p.SetReadDeadline(time.Now()))
	<-rw.Unblock
}

func TestPeerEnsureFollowsProtocol(t *testing.T) {
	defer leaktest.Check(t)()

	n := NewNode(nil)
	p := n.NewPeer(nil, new(iotest.NopWriter), new(iotest.NopReader), nil)

	// Have the peer follow a protocol where the peer should immediately
	// disconnect such that Start() synchronously returns.

	n.FollowProtocol(func(Context) (Protocol, error) {
		p.Disconnect(nil)
		return nil, nil
	})

	p.Start()
}

func TestPeerDropMessageWhenReceiveQueueFull(t *testing.T) {
	defer leaktest.Check(t)()

	_, b := net.Pipe()

	n := NewNode(nil)
	n.Handle(0x01, nil)
	p := newPeer(n, nil, b, b, b)
	defer p.Disconnect(nil)

	go p.Start()

	for i := 0; i < 1025; i++ {
		func() {
			msg := make([]byte, 128)
			_, err := rand.Read(msg)
			assert.NoError(t, err)

			buf := bytes.NewBuffer(nil)
			assert.NoError(t, p.Send(0x01, buf.Bytes()))
		}()
	}
}

func TestPeerSetAddr(t *testing.T) {
	p := newPeer(nil, new(net.TCPAddr), nil, nil, nil)
	assert.NotNil(t, p.Addr())
}

func TestPeerReportAndInterceptErrors(t *testing.T) {
	p := newPeer(nil, nil, new(iotest.NopWriter), new(iotest.NopReader), nil)

	p.InterceptErrors(func(err error) {
		assert.Equal(t, "test error", err.Error())
	})

	p.reportError(errors.New("test error"))
}
