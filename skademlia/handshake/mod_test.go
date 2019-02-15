package handshake_test

import (
	"encoding/hex"
	"testing"
	"testing/quick"

	"github.com/perlin-network/noise"
	"github.com/perlin-network/noise/log"
	"github.com/perlin-network/noise/payload"
	"github.com/perlin-network/noise/protocol"
	"github.com/perlin-network/noise/skademlia"
	"github.com/perlin-network/noise/skademlia/handshake"
	"github.com/perlin-network/noise/transport"
	"github.com/stretchr/testify/assert"
)

var (
	transportLayer = transport.NewBuffered()
	port           = 3000
)

func node(t *testing.T) *noise.Node {
	params := noise.DefaultParams()
	params.Transport = transportLayer
	params.Port = uint16(port)
	params.ID = skademlia.NewIdentityRandom()

	port++

	node, err := noise.NewNode(params)
	assert.NoError(t, err)

	go node.Listen()

	return node
}

type msg struct{ noise.EmptyMessage }

var _ protocol.Block = (*receiverBlock)(nil)

type receiverBlock struct {
	opcodeMsg noise.Opcode
	receiver  chan interface{}
	peer      *noise.Peer
}

func (b *receiverBlock) OnRegister(p *protocol.Protocol, node *noise.Node) {
	b.opcodeMsg = noise.RegisterMessage(noise.NextAvailableOpcode(), (*msg)(nil))
	b.receiver = make(chan interface{}, 1)
}

func (b *receiverBlock) OnBegin(p *protocol.Protocol, peer *noise.Peer) error {
	err := peer.SendMessage(msg{})
	if err != nil {
		return err
	}

	b.peer = peer
	b.receiver <- <-peer.Receive(b.opcodeMsg)
	return nil
}

func (b *receiverBlock) OnEnd(p *protocol.Protocol, peer *noise.Peer) error {
	return nil
}

func TestBlock_OnBeginSuccessful(t *testing.T) {
	t.Parallel()
	log.Disable()
	defer log.Enable()

	alice, bob := node(t), node(t)

	defer alice.Kill()
	defer bob.Kill()

	aliceReceiver, bobReceiver := new(receiverBlock), new(receiverBlock)

	p := protocol.New()
	p.Register(handshake.New())
	p.Register(aliceReceiver)
	p.Enforce(alice)

	p = protocol.New()
	p.Register(handshake.New())
	p.Register(bobReceiver)
	p.Enforce(bob)

	_, err := alice.Dial(bob.ExternalAddress())
	assert.NoError(t, err)

	<-aliceReceiver.receiver
	<-bobReceiver.receiver

	assert.Equal(t, hex.EncodeToString(alice.ID.PublicID()), bobReceiver.peer.Get(handshake.KeySkademliaHandshake))
	assert.Equal(t, hex.EncodeToString(bob.ID.PublicID()), aliceReceiver.peer.Get(handshake.KeySkademliaHandshake))
}

func TestHandshakeMessage(t *testing.T) {
	t.Parallel()

	checkHandshake := func(Msg string,
		ID []byte,
		PublicKey []byte,
		Nonce []byte,
		C1 uint16,
		C2 uint16) bool {
		msg := &handshake.Handshake{
			Msg:       Msg,
			ID:        ID,
			PublicKey: PublicKey,
			Nonce:     Nonce,
			C1:        C1,
			C2:        C2,
		}
		wrote := msg.Write()
		assert.True(t, len(wrote) > 6, "should be at least 6 fields, so 6 bytes")

		var result handshake.Handshake
		read, err := result.Read(payload.NewReader(wrote))
		assert.Nil(t, err)
		actual, ok := read.(handshake.Handshake)
		assert.True(t, ok)

		assert.Equal(t, Msg, actual.Msg)
		assert.Equal(t, ID, actual.ID)
		assert.Equal(t, PublicKey, actual.PublicKey)
		assert.Equal(t, Nonce, actual.Nonce)
		assert.Equal(t, C1, actual.C1)
		assert.Equal(t, C2, actual.C2)

		return true
	}
	// quick test all the parameter types
	assert.Nil(t, quick.Check(checkHandshake, nil))
}

func Test_block_VerifyHandshake(t *testing.T) {
	t.Parallel()

	idm := skademlia.NewIdentityRandom()
	b := handshake.New()
	params := noise.DefaultParams()
	params.ID = idm
	params.Transport = transportLayer
	node, err := noise.NewNode(params)
	assert.Nil(t, err)

	b.OnRegister(nil, node)

	tests := []struct {
		name    string
		args    handshake.Handshake
		wantErr bool
	}{
		{
			name: "good case",
			args: func() handshake.Handshake {
				return handshake.Handshake{
					PublicKey: idm.PublicID(),
					ID:        idm.NodeID,
					Nonce:     idm.Nonce,
					C1:        uint16(idm.C1),
					C2:        uint16(idm.C2),
				}
			}(),
			wantErr: false,
		},
		{
			name: "bad public key",
			args: func() handshake.Handshake {
				changed := make([]byte, len(idm.PublicID()))
				copy(changed, idm.PublicID())
				changed[0] = 0
				changed[1] = 0
				return handshake.Handshake{
					PublicKey: changed,
					ID:        idm.NodeID,
					Nonce:     idm.Nonce,
					C1:        uint16(idm.C1),
					C2:        uint16(idm.C2),
				}
			}(),
			wantErr: true,
		},
		{
			name: "bad node id",
			args: func() handshake.Handshake {
				changed := make([]byte, len(idm.NodeID))
				copy(changed, idm.NodeID)
				changed[0] = 0
				changed[1] = 0
				return handshake.Handshake{
					PublicKey: idm.PublicID(),
					ID:        changed,
					Nonce:     idm.Nonce,
					C1:        uint16(idm.C1),
					C2:        uint16(idm.C2),
				}
			}(),
			wantErr: true,
		},
		{
			name: "bad nonce",
			args: func() handshake.Handshake {
				changed := make([]byte, len(idm.Nonce))
				copy(changed, idm.Nonce)
				changed[0] = 0
				changed[1] = 0
				return handshake.Handshake{
					PublicKey: idm.PublicID(),
					ID:        idm.NodeID,
					Nonce:     changed,
					C1:        uint16(idm.C1),
					C2:        uint16(idm.C2),
				}
			}(),
			wantErr: true,
		},
		{
			name: "bad c1",
			args: func() handshake.Handshake {
				return handshake.Handshake{
					PublicKey: idm.PublicID(),
					ID:        idm.NodeID,
					Nonce:     idm.Nonce,
					C1:        uint16(idm.C1 - 8),
					C2:        uint16(idm.C2),
				}
			}(),
			wantErr: true,
		},
		{
			name: "bad c2",
			args: func() handshake.Handshake {
				return handshake.Handshake{
					PublicKey: idm.PublicID(),
					ID:        idm.NodeID,
					Nonce:     idm.Nonce,
					C1:        uint16(idm.C1),
					C2:        uint16(idm.C2 - 8),
				}
			}(),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err = b.VerifyHandshake(tt.args)
			assert.Equal(t, tt.wantErr, (err != nil), err)
		})
	}
}
