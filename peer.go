package noise

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"github.com/perlin-network/noise/callbacks"
	"github.com/perlin-network/noise/log"
	"github.com/perlin-network/noise/payload"
	"github.com/pkg/errors"
	"io"
	"net"
	"strings"
	"sync"
	"time"
)

type receiveHandle struct {
	hub  chan Message
	lock chan struct{}
}

func (r *receiveHandle) Unlock() {
	<-r.lock
}

type sendHandle struct {
	payload []byte
	result  chan error
}

type Peer struct {
	node *Node
	conn net.Conn

	onConnErrorCallbacks  *callbacks.SequentialCallbackManager
	onDisconnectCallbacks *callbacks.SequentialCallbackManager

	onEncodeHeaderCallbacks *callbacks.ReduceCallbackManager
	onEncodeFooterCallbacks *callbacks.ReduceCallbackManager

	onDecodeHeaderCallbacks *callbacks.SequentialCallbackManager
	onDecodeFooterCallbacks *callbacks.SequentialCallbackManager

	beforeMessageSentCallbacks     *callbacks.ReduceCallbackManager
	beforeMessageReceivedCallbacks *callbacks.ReduceCallbackManager

	afterMessageSentCallbacks     *callbacks.SequentialCallbackManager
	afterMessageReceivedCallbacks *callbacks.SequentialCallbackManager

	sendQueue     chan sendHandle
	receiveQueues sync.Map // map[Opcode]chan receiveHandle

	kill     chan struct{}
	killOnce sync.Once

	metadata sync.Map
}

func newPeer(node *Node, conn net.Conn) *Peer {
	return &Peer{
		node: node,
		conn: conn,

		onConnErrorCallbacks:  callbacks.NewSequentialCallbackManager(),
		onDisconnectCallbacks: callbacks.NewSequentialCallbackManager(),

		onEncodeHeaderCallbacks: callbacks.NewReduceCallbackManager(),
		onEncodeFooterCallbacks: callbacks.NewReduceCallbackManager(),

		onDecodeHeaderCallbacks: callbacks.NewSequentialCallbackManager(),
		onDecodeFooterCallbacks: callbacks.NewSequentialCallbackManager(),

		beforeMessageReceivedCallbacks: callbacks.NewReduceCallbackManager().UnsafelySetReverse(),
		beforeMessageSentCallbacks:     callbacks.NewReduceCallbackManager(),

		afterMessageReceivedCallbacks: callbacks.NewSequentialCallbackManager(),
		afterMessageSentCallbacks:     callbacks.NewSequentialCallbackManager(),

		kill:      make(chan struct{}, 1),
		sendQueue: make(chan sendHandle, 128),
	}
}

func (p *Peer) init() {
	go p.spawnSendWorker()
	go p.spawnReceiveWorker()
}

func (p *Peer) spawnSendWorker() {
	for {
		var cmd sendHandle

		select {
		//case <-p.kill: // TODO(kenta): kill switch is broken
		//	return
		case cmd = <-p.sendQueue:
		}

		payload := cmd.payload

		payload = p.beforeMessageSentCallbacks.MustRunCallbacks(payload, p.node).([]byte)

		size := len(payload)

		// Prepend message length to packet.
		buf := make([]byte, binary.MaxVarintLen64)
		prepended := binary.PutUvarint(buf[:], uint64(size))

		buf = append(buf[:prepended], payload[:]...)

		copied, err := io.Copy(p.conn, bytes.NewReader(buf))

		if copied != int64(size+prepended) {
			if cmd.result != nil {
				cmd.result <- errors.Errorf("only written %d bytes when expected to write %d bytes to peer", copied, size+prepended)
			}
			continue
		}

		if err != nil {
			if cmd.result != nil {
				cmd.result <- errors.Wrap(err, "failed to send message to peer")
			}
			continue
		}

		if errs := p.afterMessageSentCallbacks.RunCallbacks(p.node); len(errs) > 0 {
			if cmd.result != nil {
				var err = errs[0]

				if len(errs) > 1 {
					for _, e := range errs[1:] {
						err = errors.Wrap(err, e.Error())
					}
				}

				cmd.result <- errors.Wrap(err, "got errors running AfterMessageSent callbacks")
			}
			continue
		}

		if cmd.result != nil {
			cmd.result <- nil
		}
	}
}

func (p *Peer) spawnReceiveWorker() {
	reader := bufio.NewReader(p.conn)

	for {
		select {
		case <-p.kill:
			return
		default:
		}

		size, err := binary.ReadUvarint(reader)
		if err != nil {
			if errors.Cause(err) != io.EOF && !strings.Contains(errors.Cause(err).Error(), "use of closed network connection") && !strings.Contains(errors.Cause(err).Error(), "read: connection reset by peer") {
				p.onConnErrorCallbacks.RunCallbacks(p.node, errors.Wrap(err, "failed to read message size"))
			}

			p.Disconnect()
			continue
		}

		if size > p.node.maxMessageSize {
			p.onConnErrorCallbacks.RunCallbacks(p.node, errors.Errorf("exceeded max message size; got size %d", size))

			p.Disconnect()
			continue
		}

		buf := make([]byte, int(size))

		seen, err := io.ReadFull(reader, buf)
		if err != nil {
			p.onConnErrorCallbacks.RunCallbacks(p.node, errors.Wrap(err, "failed to read remaining message contents"))

			p.Disconnect()
			continue
		}

		if seen < int(size) {
			p.onConnErrorCallbacks.RunCallbacks(p.node, errors.Errorf("only read %d bytes when expected to read %d from peer", seen, size))

			p.Disconnect()
			continue
		}

		buf = p.beforeMessageReceivedCallbacks.MustRunCallbacks(buf, p.node).([]byte)

		opcode, msg, err := p.DecodeMessage(buf)

		if opcode == OpcodeNil || err != nil {
			p.onConnErrorCallbacks.RunCallbacks(p.node, errors.Wrap(err, "failed to decode message"))

			p.Disconnect()
			continue
		}

		c, _ := p.receiveQueues.LoadOrStore(opcode, receiveHandle{hub: make(chan Message), lock: make(chan struct{}, 1)})
		recv := c.(receiveHandle)

		select {
		case recv.hub <- msg:
			recv.lock <- struct{}{}
			<-recv.lock
		case <-time.After(3 * time.Second):
			p.Disconnect()
			continue
		}

		if errs := p.afterMessageReceivedCallbacks.RunCallbacks(p.node); len(errs) > 0 {
			log.Warn().Errs("errors", errs).Msg("Got errors running AfterMessageReceived callbacks.")
		}

	}
}

// SendMessage sends a message whose type is registered with Noise to a specified peer. Calling
// this function will block the current goroutine until the message is successfully sent. In
// order to not block, refer to `SendMessageAsync(message Message) error`.
//
// It is guaranteed that all messages are sent in a linearized order.
//
// It returns an error should it take too long to send a message, the message is not registered
// with Noise, or there are message that are blocking the peers send worker.
func (p *Peer) SendMessage(message Message) error {
	payload, err := p.EncodeMessage(message)
	if err != nil {
		return errors.Wrap(err, "failed to serialize message contents to be sent to a peer")
	}

	cmd := sendHandle{payload: payload, result: make(chan error, 1)}
	defer close(cmd.result)

	select {
	case <-time.After(3 * time.Second):
		return errors.New("noise: send message queue is full and not being processed")
	case p.sendQueue <- cmd:
	}

	select {
	case <-time.After(3 * time.Second):
		return errors.New("noise: timed out attempting to send a message")
	case err = <-cmd.result:
		return err
	}
}

// SendMessageAsync sends a message whose type is registered with Noise to a specified peer. Calling
// this function will not block the current goroutine until the message is successfully sent. In
// order to block, refer to `SendMessage(message Message) error`.
//
// It is guaranteed that all messages are sent in a linearized order.
//
// It returns an error should the message not be registered with Noise, or there are message that are
// blocking the peers send worker.
func (p *Peer) SendMessageAsync(message Message) error {
	payload, err := p.EncodeMessage(message)
	if err != nil {
		return errors.Wrap(err, "failed to serialize message contents to be sent to a peer")
	}

	cmd := sendHandle{payload: payload, result: nil}

	select {
	case <-time.After(3 * time.Second):
		return errors.New("noise: send message queue is full and not being processed")
	case p.sendQueue <- cmd:
	}

	return nil
}

func (p *Peer) BeforeMessageSent(c BeforeMessageSentCallback) {
	p.beforeMessageSentCallbacks.RegisterCallback(func(in interface{}, params ...interface{}) (i interface{}, e error) {
		if len(params) != 1 {
			panic(errors.Errorf("noise: BeforeMessageSent received unexpected args %v", params))
		}

		node, ok := params[0].(*Node)
		if !ok {
			return in.([]byte), nil
		}

		return c(node, p, in.([]byte))
	})
}

func (p *Peer) BeforeMessageReceived(c BeforeMessageReceivedCallback) {
	p.beforeMessageReceivedCallbacks.RegisterCallback(func(in interface{}, params ...interface{}) (i interface{}, e error) {
		if len(params) != 1 {
			panic(errors.Errorf("noise: BeforeMessageReceived received unexpected args %v", params))
		}

		node, ok := params[0].(*Node)
		if !ok {
			return in.([]byte), nil
		}

		return c(node, p, in.([]byte))
	})
}

func (p *Peer) AfterMessageSent(c AfterMessageSentCallback) {
	p.afterMessageSentCallbacks.RegisterCallback(func(params ...interface{}) error {
		if len(params) != 1 {
			panic(errors.Errorf("noise: AfterMessageSent received unexpected args %v", params))
		}

		node, ok := params[0].(*Node)
		if !ok {
			return nil
		}

		return c(node, p)
	})
}

func (p *Peer) AfterMessageReceived(c AfterMessageReceivedCallback) {
	p.afterMessageReceivedCallbacks.RegisterCallback(func(params ...interface{}) error {
		if len(params) != 1 {
			panic(errors.Errorf("noise: AfterMessageReceived received unexpected args %v", params))
		}

		node, ok := params[0].(*Node)
		if !ok {
			return nil
		}

		return c(node, p)
	})
}

func (p *Peer) OnDecodeHeader(c OnPeerDecodeHeaderCallback) {
	p.onDecodeHeaderCallbacks.RegisterCallback(func(params ...interface{}) error {
		if len(params) != 2 {
			panic(errors.Errorf("noise: OnDecodeHeader received unexpected args %v", params))
		}

		node, ok := params[0].(*Node)
		if !ok {
			return nil
		}

		reader, ok := params[1].(payload.Reader)

		if !ok {
			return nil
		}

		return c(node, p, reader)
	})
}

func (p *Peer) OnDecodeFooter(c OnPeerDecodeFooterCallback) {
	p.onDecodeFooterCallbacks.RegisterCallback(func(params ...interface{}) error {
		if len(params) != 3 {
			panic(errors.Errorf("noise: OnDecodeFooter received unexpected args %v", params))
		}

		node, ok := params[0].(*Node)
		if !ok {
			return nil
		}

		msg, ok := params[1].([]byte)

		if !ok {
			return nil
		}

		reader, ok := params[2].(payload.Reader)

		if !ok {
			return nil
		}

		return c(node, p, msg, reader)
	})
}

func (p *Peer) OnEncodeHeader(c AfterMessageEncodedCallback) {
	p.onEncodeHeaderCallbacks.RegisterCallback(func(header interface{}, params ...interface{}) (i interface{}, e error) {
		if len(params) != 2 {
			panic(errors.Errorf("noise: OnEncodeHeader received unexpected args %v", params))
		}

		node, ok := params[0].(*Node)
		if !ok {
			return header.([]byte), errors.New("noise: OnEncodeHeader did not receive 1st param (node *noise.Node)")
		}

		msg, ok := params[1].([]byte)

		if !ok {
			return header.([]byte), errors.New("noise: OnEncodeHeader did not receive 2nd param (msg []byte)")
		}

		return c(node, p, header.([]byte), msg)
	})
}

func (p *Peer) OnEncodeFooter(c AfterMessageEncodedCallback) {
	p.onEncodeFooterCallbacks.RegisterCallback(func(footer interface{}, params ...interface{}) (i interface{}, e error) {
		if len(params) != 2 {
			panic(errors.Errorf("noise: OnEncodeFooter received unexpected args %v", params))
		}

		node, ok := params[0].(*Node)
		if !ok {
			return footer.([]byte), errors.New("noise: OnEncodeHeader did not receive 1st param (node *noise.Node)")
		}

		msg, ok := params[1].([]byte)

		if !ok {
			return footer.([]byte), errors.New("noise: OnEncodeHeader did not receive (msg []byte)")
		}

		return c(node, p, footer.([]byte), msg)
	})
}

// OnConnError registers a callback for whenever somethings wrong with our peers connection
func (p *Peer) OnConnError(c OnPeerErrorCallback) {
	p.onConnErrorCallbacks.RegisterCallback(func(params ...interface{}) error {
		if len(params) != 2 {
			panic(errors.Errorf("noise: OnConnError received unexpected args %v", params))
		}

		node, ok := params[0].(*Node)
		if !ok {
			return nil
		}

		err, ok := params[1].(error)

		if !ok {
			return nil
		}

		return c(node, p, errors.Wrap(err, "peer conn reported error"))
	})
}

// OnDisconnect registers a callback for whenever the peer disconnects.
func (p *Peer) OnDisconnect(srcCallbacks ...OnPeerDisconnectCallback) {
	targetCallbacks := make([]callbacks.Callback, 0, len(srcCallbacks))

	for _, c := range srcCallbacks {
		c := c
		targetCallbacks = append(targetCallbacks, func(params ...interface{}) error {
			node, ok := params[0].(*Node)
			if !ok {
				panic("params[0] is not a Node")
			}

			return c(node, p)
		})
	}

	p.onDisconnectCallbacks.RegisterCallback(targetCallbacks...)
}

func (p *Peer) Receive(o Opcode) <-chan Message {
	c, _ := p.receiveQueues.LoadOrStore(o, receiveHandle{hub: make(chan Message), lock: make(chan struct{}, 1)})
	return c.(receiveHandle).hub
}

func (p *Peer) Disconnect() {
	//_, file, no, ok := runtime.Caller(1)
	//if ok {
	//	log.Debug().Msgf("Disconnect() called from %s#%d.", file, no)
	//}

	p.killOnce.Do(func() {
		if err := p.conn.Close(); err != nil {
			p.onConnErrorCallbacks.RunCallbacks(p.node, errors.Wrapf(err, "got errors closing peer connection"))
		}

		p.onDisconnectCallbacks.RunCallbacks(p.node)

		p.kill <- struct{}{}
		close(p.kill)
	})
}

func (p *Peer) LocalIP() net.IP {
	return p.node.transport.IP(p.conn.LocalAddr())
}

func (p *Peer) LocalPort() uint16 {
	return p.node.transport.Port(p.conn.LocalAddr())
}

func (p *Peer) RemoteIP() net.IP {
	return p.node.transport.IP(p.conn.RemoteAddr())
}

func (p *Peer) RemotePort() uint16 {
	return p.node.transport.Port(p.conn.RemoteAddr())
}

// Set sets a metadata entry given a key-value pair on our node.
func (p *Peer) Set(key string, val interface{}) {
	p.metadata.Store(key, val)
}

// Get returns the value to a metadata key from our node, or otherwise returns nil should
// there be no corresponding value to a provided key.
func (p *Peer) Get(key string) interface{} {
	val, _ := p.metadata.Load(key)
	return val
}

func (p *Peer) LoadOrStore(key string, val interface{}) interface{} {
	val, _ = p.metadata.LoadOrStore(key, val)
	return val
}

func (p *Peer) Has(key string) bool {
	_, exists := p.metadata.Load(key)
	return exists
}

func (p *Peer) Delete(key string) {
	p.metadata.Delete(key)
}

func (p *Peer) Node() *Node {
	return p.node
}

func (p *Peer) LockOnReceive(opcode Opcode) receiveHandle {
	c, _ := p.receiveQueues.LoadOrStore(opcode, receiveHandle{hub: make(chan Message), lock: make(chan struct{}, 1)})
	recv := c.(receiveHandle)

	recv.lock <- struct{}{}

	return recv
}

func (p *Peer) SetNode(node *Node) {
	p.node = node
}
