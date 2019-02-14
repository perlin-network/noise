package noise

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"github.com/perlin-network/noise/identity/ed25519"
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
	text string
}

func (testMsg) Read(reader payload.Reader) (Message, error) {
	text, err := reader.ReadString()
	if err != nil {
		return nil, errors.Wrap(err, "failed to read test message")
	}

	return testMsg{text: text}, nil
}

func (m testMsg) Write() []byte {
	return payload.NewWriter(nil).WriteString(m.text).Bytes()
}

func TestEncodeMessage(t *testing.T) {
	resetOpcodes()
	o := RegisterMessage(Opcode(123), (*testMsg)(nil))

	msg := testMsg{
		text: "hello",
	}

	p := newPeer(nil, nil)

	bytes, err := p.EncodeMessage(msg)
	assert.Nil(t, err)
	assert.Equal(t, append([]byte{byte(o)}, msg.Write()...), bytes)
}

func TestDecodeMessage(t *testing.T) {
	resetOpcodes()
	o := RegisterMessage(Opcode(45), (*testMsg)(nil))

	msg := testMsg{
		text: "world",
	}
	assert.Equal(t, o, RegisterMessage(o, (*testMsg)(nil)))

	p := newPeer(nil, nil)

	resultO, resultM, err := p.DecodeMessage(append([]byte{byte(o)}, msg.Write()...))
	assert.Nil(t, err)
	assert.Equal(t, o, resultO)
	assert.Equal(t, msg, resultM)
}

// What this test does:
// 1. Check send message
// 2. Check receive message
// 3. Check the callbacks must be called in sequence
// 4. Check the callbacks must be called exactly once
func TestPeer(t *testing.T) {
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
		lis, err := layer.Listen(port)
		assert.Nil(t, err)
		wgListen.Done()

		lisConn, err := lis.Accept()
		assert.Nil(t, err, "failed to accept")

		wgAccept.Done()

		p := newPeer(nil, nil)
		var buf []byte
		reader := bufio.NewReader(lisConn)

		// read the size
		size, err := binary.ReadUvarint(reader)
		assert.Nil(t, err, "failed to read size")

		// read the message
		buf = make([]byte, size)
		_, err = io.ReadFull(reader, buf)
		assert.Nil(t, err, "failed to read message")
		_, msg, err := p.DecodeMessage(buf)
		assert.Equal(t, text, msg.(testMsg).text, "invalid text. expected %s, actual %s", text, msg.(testMsg).text)

		// create a new message
		sendMessage, err := p.EncodeMessage(testMsg{text: text})
		assert.Nil(t, err, "failed to encode message")
		msgSize := len(sendMessage)
		buf = make([]byte, binary.MaxVarintLen64)
		prepended := binary.PutUvarint(buf[:], uint64(msgSize))
		buf = append(buf[:prepended], sendMessage[:]...)

		// send the message
		_, err = lisConn.Write(buf)
		assert.Nil(t, err, "failed to write message")
	}()

	wgListen.Wait()

	dialConn, err := layer.Dial(fmt.Sprintf(":%d", port))
	assert.Nilf(t, err, "Dial error: %v", err)

	wgAccept.Wait()

	var state int32 = 0

	p := getPeer(t, layer, dialConn, port)

	p.OnEncodeHeader(func(node *Node, peer *Peer, header, msg []byte) (bytes []byte, e error) {
		checkState(t, &state, 0)
		return nil, nil
	})

	p.OnEncodeFooter(func(node *Node, peer *Peer, header, msg []byte) (bytes []byte, e error) {
		checkState(t, &state, 1)
		return nil, nil
	})

	p.BeforeMessageSent(func(node *Node, peer *Peer, msg []byte) (bytes []byte, e error) {
		checkState(t, &state, 2)
		return msg, nil
	})

	p.AfterMessageSent(func(node *Node, peer *Peer) error {
		checkState(t, &state, 3)
		return nil
	})

	p.BeforeMessageReceived(func(node *Node, peer *Peer, msg []byte) (bytes []byte, e error) {
		checkState(t, &state, 4)
		return msg, nil
	})

	p.OnDecodeHeader(func(node *Node, peer *Peer, reader payload.Reader) error {
		checkState(t, &state, 5)
		return nil
	})

	p.OnDecodeFooter(func(node *Node, peer *Peer, msg []byte, reader payload.Reader) error {
		checkState(t, &state, 6)
		return nil
	})

	p.AfterMessageReceived(func(node *Node, peer *Peer) error {
		checkState(t, &state, 7)
		return nil
	})

	p.OnConnError(func(node *Node, peer *Peer, err error) error {
		checkState(t, &state, 9)
		return nil
	})

	var wgDisconnect sync.WaitGroup
	wgDisconnect.Add(1)
	p.OnDisconnect(func(node *Node, peer *Peer) error {
		defer wgDisconnect.Done()

		checkState(t, &state, 10)
		return nil
	})

	p.init()

	// send a message
	err = p.SendMessage(testMsg{text: text})
	assert.Nil(t, err, "Failed to send message")

	// read a message
	msg := <-p.Receive(opcodeTest)
	assert.Equal(t, text, msg.(testMsg).text, "invalid text. expected %s, actual %s", text, msg.(testMsg).text)

	checkState(t, &state, 8)

	p.Disconnect()

	wgDisconnect.Wait()

	checkState(t, &state, 11)
}

// check the state equal to the expected state, and then increment it by 1
func checkState(t *testing.T, state *int32, expectedState int32) {
	s := atomic.LoadInt32(state)
	assert.True(t, s == expectedState, "invalid state in OnEncodeHeader: expected %d, actual %d", expectedState, s)

	atomic.AddInt32(state, 1)
}

func getPeer(t *testing.T, layer transport.Layer, conn net.Conn, port uint16) *Peer {
	params := DefaultParams()
	params.ID = ed25519.Random()
	params.Port = port
	params.Transport = layer

	node, err := NewNode(params)
	assert.Nil(t, err, "failed to create node")

	p := newPeer(node, conn)

	return p
}
