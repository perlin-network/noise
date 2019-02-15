package handshake

import (
	"github.com/perlin-network/noise"
	"github.com/perlin-network/noise/payload"
	"github.com/pkg/errors"
)

var _ noise.Message = (*Handshake)(nil)

type Handshake struct {
	Msg       string
	ID        []byte
	PublicKey []byte
	Nonce     []byte
	C1        uint16
	C2        uint16
}

func (Handshake) Read(reader payload.Reader) (noise.Message, error) {
	msg, err := reader.ReadString()
	if err != nil {
		return nil, errors.Wrap(err, "failed to read msg")
	}

	nodeID, err := reader.ReadBytes()
	if err != nil {
		return nil, errors.Wrap(err, "failed to read nodeID")
	}

	publicKey, err := reader.ReadBytes()
	if err != nil {
		return nil, errors.Wrap(err, "failed to read public key")
	}

	nonce, err := reader.ReadBytes()
	if err != nil {
		return nil, errors.Wrap(err, "failed to read nonce")
	}

	c1, err := reader.ReadUint16()
	if err != nil {
		return nil, errors.Wrap(err, "failed to read c1")
	}

	c2, err := reader.ReadUint16()
	if err != nil {
		return nil, errors.Wrap(err, "failed to read c2")
	}

	return Handshake{
		Msg:       msg,
		ID:        nodeID,
		PublicKey: publicKey,
		Nonce:     nonce,
		C1:        c1,
		C2:        c2,
	}, nil
}

func (m Handshake) Write() []byte {
	return payload.NewWriter(nil).
		WriteString(m.Msg).
		WriteBytes(m.ID).
		WriteBytes(m.PublicKey).
		WriteBytes(m.Nonce).
		WriteUint16(m.C1).
		WriteUint16(m.C2).
		Bytes()
}
