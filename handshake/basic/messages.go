package basic

import (
	"github.com/perlin-network/noise"
	"github.com/perlin-network/noise/payload"
	"github.com/pkg/errors"
)

var _ noise.Message = (*messageHandshake)(nil)

type messageHandshake struct {
	publicKey []byte
}

func (messageHandshake) Read(reader payload.Reader) (noise.Message, error) {
	publicKey, err := reader.ReadBytes()
	if err != nil {
		return nil, errors.Wrap(err, "failed to read public key")
	}

	return messageHandshake{publicKey: publicKey}, nil
}

func (m messageHandshake) Write() []byte {
	return payload.NewWriter(nil).WriteBytes(m.publicKey).Bytes()
}
