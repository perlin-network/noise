package rpc

import (
	"github.com/perlin-network/noise"
	"github.com/perlin-network/noise/callbacks"
	"github.com/perlin-network/noise/log"
	"github.com/perlin-network/noise/payload"
	"github.com/pkg/errors"
	"sync"
	"sync/atomic"
	"time"
)

const KeyRequestNonce = "rpc.nonce"
const KeyRequestCallbacks = "rpc.callbacks"

var (
	opcodeRequest       noise.Opcode
	opcodeResponse      noise.Opcode
	registerOpcodesOnce sync.Once

	_ noise.Message = (*messageRequest)(nil)
	_ noise.Message = (*messageResponse)(nil)
)

type messageRequest struct {
	opcode noise.Opcode
	body   noise.Message

	nonce uint32
}

func (m messageRequest) Read(reader payload.Reader) (noise.Message, error) {
	var err error

	// Read message nonce.
	m.nonce, err = reader.ReadUint32()
	if err != nil {
		return nil, errors.New("rpc: failed to decode request nonce")
	}

	// Read opcode.
	opcode, err := reader.ReadUint16()
	if err != nil {
		return nil, errors.New("rpc: failed to decode request opcode")
	}

	m.opcode = noise.Opcode(opcode)

	body, err := noise.MessageFromOpcode(m.opcode)

	// Read message body given the opcode.
	m.body, err = body.Read(reader)
	if err != nil {
		return nil, errors.New("rpc: failed to decode request message body")
	}

	return m, nil
}

func (m messageRequest) Write() []byte {
	return append(payload.NewWriter(nil).WriteUint32(m.nonce).WriteUint16(uint16(m.opcode)).Bytes(), m.body.Write()...)
}

func Request(peer *noise.Peer, timeout time.Duration, opcode noise.Opcode, request noise.Message) (noise.Message, error) {
	nonce := atomic.AddUint32(peer.Get(KeyRequestNonce).(*uint32), 1)

	stream := make(chan messageResponse, 1)

	peer.OnMessageReceived(opcodeResponse, func(node *noise.Node, opcode noise.Opcode, peer *noise.Peer, message noise.Message) error {
		req := message.(messageResponse)

		if req.nonce == nonce {
			stream <- req
			return callbacks.DeregisterCallback
		}

		return nil
	})

	err := peer.SendMessage(opcodeRequest, messageRequest{opcode: opcode, body: request, nonce: nonce})

	if err != nil {
		return nil, err
	}

	select {
	case res := <-stream:
		close(stream)
		return res.body, nil
	case <-time.After(timeout):
		return nil, errors.New("rpc: request timed out")
	}
}

type messageResponse struct {
	opcode noise.Opcode
	body   noise.Message

	nonce uint32
}

func (m messageResponse) Read(reader payload.Reader) (noise.Message, error) {
	var err error

	// Read message nonce.
	m.nonce, err = reader.ReadUint32()
	if err != nil {
		return nil, errors.New("rpc: failed to decode response nonce")
	}

	// Read opcode.
	opcode, err := reader.ReadUint16()
	if err != nil {
		return nil, errors.New("rpc: failed to decode response opcode")
	}

	m.opcode = noise.Opcode(opcode)

	body, err := noise.MessageFromOpcode(m.opcode)

	// Read message body given the opcode.
	m.body, err = body.Read(reader)
	if err != nil {
		return nil, errors.New("rpc: failed to decode response message body")
	}

	return m, nil
}

func (m messageResponse) Write() []byte {
	return append(payload.NewWriter(nil).WriteUint32(m.nonce).WriteUint16(uint16(m.opcode)).Bytes(), m.body.Write()...)
}

func OnRequestReceived(peer *noise.Peer, opcode noise.Opcode, c func(peer *noise.Peer, req noise.Message) (noise.Message, error)) {
	if !peer.Has(KeyRequestCallbacks) {
		panic("rpc: not registered to node")
	}

	manager := peer.Get(KeyRequestCallbacks).(*callbacks.OpcodeCallbackManager)

	manager.RegisterCallback(byte(opcode), func(params ...interface{}) error {
		if len(params) != 2 {
			panic(errors.Errorf("rpc: got invalid params for OnRequestReceived: %v", params))
		}

		peer, ok := params[0].(*noise.Peer)
		if !ok {
			panic(errors.Errorf("rpc: got invalid params for OnRequestReceived: %v", params))
		}

		req, ok := params[1].(messageRequest)
		if !ok {
			panic(errors.Errorf("rpc: got invalid params for OnRequestReceived: %v", params))
		}

		res, err := c(peer, req.body)
		if err != nil {
			return errors.Wrap(err, "rpc: error handling request")
		}

		opcode, err := noise.OpcodeFromMessage(res)
		if err != nil {
			return errors.Wrap(err, "rpc: failed to get opcode of response msg")
		}

		err = peer.SendMessage(opcodeResponse, messageResponse{opcode: opcode, body: res, nonce: req.nonce})
		if err != nil {
			return errors.Wrap(err, "rpc: failed to reply to peer")
		}

		return nil
	})
}

func Register(node *noise.Node) {
	registerOpcodesOnce.Do(func() {
		opcodeRequest = noise.RegisterMessage(noise.NextAvailableOpcode(), (*messageRequest)(nil))
		opcodeResponse = noise.RegisterMessage(noise.NextAvailableOpcode(), (*messageResponse)(nil))
	})

	node.OnPeerInit(func(node *noise.Node, peer *noise.Peer) error {
		manager := callbacks.NewOpcodeCallbackManager()

		zero := uint32(0)

		peer.Set(KeyRequestCallbacks, manager)
		peer.Set(KeyRequestNonce, &zero)

		peer.OnMessageReceived(opcodeRequest, func(node *noise.Node, opcode noise.Opcode, peer *noise.Peer, message noise.Message) error {
			req := message.(messageRequest)

			errs := manager.RunCallbacks(byte(req.opcode), peer, req)

			if len(errs) > 0 {
				log.Warn().Errs("errors", errs).Msg("Got errors processing RPC.")
			}

			return nil
		})

		return nil
	})
}
