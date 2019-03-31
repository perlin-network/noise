package handshake

import (
	"bytes"
	"encoding/binary"
	"github.com/perlin-network/noise/edwards25519"
)

type Handshake struct {
	publicKey edwards25519.PublicKey
	signature edwards25519.Signature
}

func (m Handshake) Marshal() []byte {
	b := bytes.NewBuffer(make([]byte, 0, edwards25519.SizePublicKey+edwards25519.SizeSignature))

	_, _ = b.Write(m.publicKey[:])
	_, _ = b.Write(m.signature[:])

	return b.Bytes()
}

func UnmarshalHandshake(buf []byte) (m Handshake, err error) {
	b := bytes.NewReader(buf)

	if err = binary.Read(b, binary.BigEndian, &m.publicKey); err != nil {
		return
	}

	if err = binary.Read(b, binary.BigEndian, &m.signature); err != nil {
		return
	}

	return
}
