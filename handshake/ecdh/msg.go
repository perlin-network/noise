package ecdh

import (
	"github.com/perlin-network/noise"
	"github.com/perlin-network/noise/payload"
	"github.com/pkg/errors"
)

var _ noise.Message = (*messageHandshake)(nil)

type messageHandshake struct {
	publicKey []byte
	signature []byte
}

func (messageHandshake) Read(reader payload.Reader) (noise.Message, error) {
	publicKey, err := reader.ReadBytes()
	if err != nil {
		return nil, errors.Wrap(err, "failed to read public key")
	}

	signature, err := reader.ReadBytes()
	if err != nil {
		return nil, errors.Wrap(err, "failed to read signature")
	}

	return messageHandshake{publicKey: publicKey, signature: signature}, nil
}

func (m messageHandshake) Write() []byte {
	return payload.NewWriter(nil).WriteBytes(m.publicKey).WriteBytes(m.signature).Bytes()
}
