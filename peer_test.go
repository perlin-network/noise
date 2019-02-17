package noise

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"github.com/perlin-network/noise/identity/ed25519"
	"github.com/perlin-network/noise/log"
	"github.com/perlin-network/noise/payload"
	"github.com/perlin-network/noise/transport"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"testing"
)

type testMsg struct {
	Text string
}

func (testMsg) Read(reader payload.Reader) (Message, error) {
	text, err := reader.ReadString()
	if err != nil {
		return nil, errors.Wrap(err, "failed to read test message")
	}

	return testMsg{Text: text}, nil
}

func (m testMsg) Write() []byte {
	return payload.NewWriter(nil).WriteString(m.Text).Bytes()
}

// What this test does:
// 1. Check send message
// 2. Check receive message
// 3. Check the callbacks must be called in sequence
// 4. Check the callbacks must be called exactly once
func TestPeerFlow(t *testing.T) {
	log.Disable()
	defer log.Enable()

	resetOpcodes()
	opcodeTest := RegisterMessage(NextAvailableOpcode(), (*testMsg)(nil))

	var text = "hello"
	var port uint16 = 8888
	var err error

	var wgListen sync.WaitGroup
	wgListen.Add(1)

	var wgAccept sync.WaitGroup
	wgAccept.Add(1)

	layer := transport.NewBuffered()

	go func() {
		listener, err := layer.Listen("127.0.0.1", port)
		assert.Nil(t, err)

		wgListen.Done()

		conn, err := listener.Accept()
		assert.NoError(t, err)

		wgAccept.Done()

		peer := newPeer(nil, nil)

		var buf []byte
		reader := bufio.NewReader(conn)

		// Read message size.
		size, err := binary.ReadUvarint(reader)
		assert.NoError(t, err)

		// Read message.
		buf = make([]byte, size)

		_, err = io.ReadFull(reader, buf)
		assert.NoError(t, err)

		_, msg, err := peer.DecodeMessage(buf)
		assert.Equal(t, text, msg.(testMsg).Text)

		// Create a new message.
		payload, err := peer.EncodeMessage(testMsg{Text: text})
		assert.NoError(t, err)

		buf = make([]byte, binary.MaxVarintLen64)
		prepended := binary.PutUvarint(buf[:], uint64(len(payload)))
		buf = append(buf[:prepended], payload[:]...)

		// Send the message.
		_, err = conn.Write(buf)
		assert.NoError(t, err)
	}()

	wgListen.Wait()

	conn, err := layer.Dial(fmt.Sprintf("%s:%d", "127.0.0.1", port))
	assert.NoError(t, err)

	wgAccept.Wait()

	var state int32 = 0

	p := peer(t, layer, conn, port)

	p.OnEncodeHeader(func(node *Node, peer *Peer, header, msg []byte) (bytes []byte, e error) {
		check(t, &state, 0)
		return nil, nil
	})

	p.OnEncodeFooter(func(node *Node, peer *Peer, header, msg []byte) (bytes []byte, e error) {
		check(t, &state, 1)
		return nil, nil
	})

	p.BeforeMessageSent(func(node *Node, peer *Peer, msg []byte) (bytes []byte, e error) {
		check(t, &state, 2)
		return msg, nil
	})

	p.AfterMessageSent(func(node *Node, peer *Peer) error {
		check(t, &state, 3)
		return nil
	})

	p.BeforeMessageReceived(func(node *Node, peer *Peer, msg []byte) (bytes []byte, e error) {
		check(t, &state, 4)
		return msg, nil
	})

	p.OnDecodeHeader(func(node *Node, peer *Peer, reader payload.Reader) error {
		check(t, &state, 5)
		return nil
	})

	p.OnDecodeFooter(func(node *Node, peer *Peer, msg []byte, reader payload.Reader) error {
		check(t, &state, 6)
		return nil
	})

	p.AfterMessageReceived(func(node *Node, peer *Peer) error {
		check(t, &state, 7)
		return nil
	})

	p.OnConnError(func(node *Node, peer *Peer, err error) error {
		check(t, &state, 10)
		return nil
	})

	var wgDisconnect sync.WaitGroup
	wgDisconnect.Add(1)

	p.OnDisconnect(func(node *Node, peer *Peer) error {
		defer wgDisconnect.Done()
		check(t, &state, 9)
		return nil
	})

	p.init()

	// Send a message.
	err = p.SendMessage(testMsg{Text: text})
	assert.NoError(t, err)

	// Read a message.
	msg := <-p.Receive(opcodeTest)
	assert.Equal(t, text, msg.(testMsg).Text)

	check(t, &state, 8)

	p.Disconnect()

	wgDisconnect.Wait()

	check(t, &state, 11)
}

func TestPeer(t *testing.T) {
	log.Disable()
	defer log.Enable()

	var port uint16 = 8888
	var err error

	var wgListen sync.WaitGroup
	wgListen.Add(1)

	layer := transport.NewBuffered()

	go func() {
		listener, err := layer.Listen("127.0.0.1", port)
		assert.Nil(t, err)

		wgListen.Done()

		_, err = listener.Accept()
		assert.NoError(t, err)
	}()

	wgListen.Wait()

	conn, err := layer.Dial(fmt.Sprintf("%s:%d", "127.0.0.1", port))
	assert.NoError(t, err)

	p := peer(t, layer, conn, port)
	p.init()

	// check net
	assert.Equal(t, net.IPv4(127, 0, 0, 1), p.LocalIP(), "found invalid local IP")
	assert.Equal(t, port, p.LocalPort(), "found invalid local port")
	assert.Equal(t, net.IPv4(127, 0, 0, 1), p.RemoteIP(), "found invalid remote IP")
	assert.Equal(t, port, p.RemotePort(), "found invalid remote port")

	// check store

	assert.Nil(t, p.Get("key"))
	assert.False(t, p.Has("key"))

	assert.Equal(t, "value", p.LoadOrStore("key", "value"))
	p.Delete("key")
	assert.Nil(t, p.Get("key"))
	assert.False(t, p.Has("key"))

	p.Set("key", "value")
	assert.Equal(t, "value", p.Get("key"))
	assert.True(t, p.Has("key"))

	p.Disconnect()
}

// check the state equal to the expected state, and then increment it by 1
func check(t *testing.T, currentState *int32, expectedState int32) {
	assert.Equal(t, expectedState, atomic.LoadInt32(currentState))
	atomic.AddInt32(currentState, 1)
}

func peer(t *testing.T, layer transport.Layer, conn net.Conn, port uint16) *Peer {
	params := DefaultParams()
	params.Keys = ed25519.RandomKeys()
	params.Port = port
	params.Transport = layer

	node, err := NewNode(params)
	assert.Nil(t, err, "failed to create node")

	p := newPeer(node, conn)

	return p
}
