package noise

import (
	"encoding/binary"
	"github.com/perlin-network/noise/wire"
	"time"
)

const (
	WireKeyOpcode byte = iota
	WireKeyMuxID
)

var DefaultProtocol = wire.Codec{
	PrefixSize: true,
	Read: func(wire *wire.Reader, state *wire.State) {
		state.SetByte(WireKeyOpcode, wire.ReadByte())
		state.SetUint64(WireKeyMuxID, wire.ReadUint64(binary.BigEndian))
		state.SetMessage(wire.ReadBytes(wire.BytesLeft()))
	},
	Write: func(wire *wire.Writer, state *wire.State) {
		wire.WriteByte(state.Byte(WireKeyOpcode))
		wire.WriteUint64(binary.BigEndian, state.Uint64(WireKeyMuxID))
		wire.WriteBytes(state.Message())
	},
}

type Wire struct {
	m Mux
	o byte
	b []byte
}

func (c Wire) Peer() *Peer {
	return c.m.peer
}

func (c Wire) Mux() Mux {
	c.m.peer.initMuxQueue(c.m.id)

	return c.m
}

func (c Wire) Send(opcode byte, msg []byte) error {
	return c.m.Send(opcode, msg)
}

func (c Wire) SendWithTimeout(opcode byte, msg []byte, timeout time.Duration) error {
	return c.m.SendWithTimeout(opcode, msg, timeout)
}

func (c Wire) Bytes() []byte {
	return c.b
}

func (c Wire) Opcode() byte {
	return c.o
}
