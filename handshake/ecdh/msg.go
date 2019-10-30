package ecdh

import (
	"github.com/Yayg/noise"
	"github.com/Yayg/noise/payload"
	"github.com/pkg/errors"
)

var _ noise.Message = (*Handshake)(nil)

type Handshake struct {
	publicKey []byte
	signature []byte
}

func (Handshake) Read(reader payload.Reader) (noise.Message, error) {
	publicKey, err := reader.ReadBytes()
	if err != nil {
		return nil, errors.Wrap(err, "failed to read public key")
	}

	signature, err := reader.ReadBytes()
	if err != nil {
		return nil, errors.Wrap(err, "failed to read signature")
	}

	return Handshake{publicKey: publicKey, signature: signature}, nil
}

func (m Handshake) Write() []byte {
	return payload.NewWriter(nil).WriteBytes(m.publicKey).WriteBytes(m.signature).Bytes()
}
