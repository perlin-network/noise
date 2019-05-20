package cipher

import (
	"crypto/sha256"
	"github.com/perlin-network/noise"
	"github.com/perlin-network/noise/handshake"
	"golang.org/x/net/context"
	"net"
)

type ProtocolAEAD struct{}

func NewAEAD() ProtocolAEAD {
	return ProtocolAEAD{}
}

func (ProtocolAEAD) ClientHandshake(info noise.Info, ctx context.Context, auth string, conn net.Conn) (net.Conn, error) {
	suite, _, err := DeriveAEAD(Aes256GCM(), sha256.New, info.Bytes(handshake.SharedKey), nil)
	if err != nil {
		return nil, err
	}

	return newConnAEAD(suite, conn), nil
}

func (ProtocolAEAD) ServerHandshake(info noise.Info, conn net.Conn) (net.Conn, error) {
	suite, _, err := DeriveAEAD(Aes256GCM(), sha256.New, info.Bytes(handshake.SharedKey), nil)
	if err != nil {
		return nil, err
	}

	return newConnAEAD(suite, conn), nil
}
