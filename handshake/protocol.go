package handshake

import (
	"crypto"
	"github.com/perlin-network/noise"
	"github.com/perlin-network/noise/edwards25519"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
	"io"
	"net"
)

const (
	SharedKey = "ecdh.shared_key"
)

type ProtocolECDH struct{}

func NewECDH() ProtocolECDH {
	return ProtocolECDH{}
}

func (ProtocolECDH) Client(info noise.Info, ctx context.Context, auth string, conn net.Conn) (net.Conn, error) {
	if err := handshakeECDH(info, conn); err != nil {
		return nil, err
	}

	return conn, nil
}

func (ProtocolECDH) Server(info noise.Info, conn net.Conn) (net.Conn, error) {
	if err := handshakeECDH(info, conn); err != nil {
		return nil, err
	}

	return conn, nil
}

func handshakeECDH(info noise.Info, conn net.Conn) error {
	ephemeralPublicKey, ephemeralPrivateKey, err := edwards25519.GenerateKey(nil)
	if err != nil {
		return errors.New("ecdh: failed to generate ephemeral keypair")
	}

	var signature edwards25519.Signature

	if signature, err = ephemeralPrivateKey.Sign([]byte(".__noise_handshake"), crypto.Hash(0)); err != nil {
		return errors.Wrap(err, "ecdh: failed to sign handshake message")
	}

	handshake := append(ephemeralPublicKey[:], signature[:]...)

	n, err := conn.Write(handshake)
	if err != nil {
		return errors.Wrap(err, "ecdh: failed to send handshake message")
	}
	if n != len(handshake) {
		return errors.New("short write sending handshake message")
	}

	if _, err = io.ReadFull(conn, handshake); err != nil {
		return errors.Wrap(err, "ecdh: failed to receive handshake from server")
	}

	var remotePublicKey edwards25519.PublicKey
	var remoteSignature edwards25519.Signature

	copy(remotePublicKey[:], handshake[:edwards25519.SizePublicKey])
	copy(remoteSignature[:], handshake[edwards25519.SizePublicKey:edwards25519.SizePublicKey+edwards25519.SizeSignature])

	if !edwards25519.Verify(remotePublicKey, []byte(".__noise_handshake"), remoteSignature) {
		return errors.New("ecdh: handshake signature is malformed")
	}

	info.PutBytes(SharedKey, computeSharedKey(ephemeralPrivateKey, remotePublicKey))
	return nil
}
