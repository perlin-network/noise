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
	w := bytes.NewBuffer(make([]byte, 0, edwards25519.SizePublicKey+edwards25519.SizeSignature))

	_, _ = w.Write(m.publicKey[:])
	_, _ = w.Write(m.signature[:])

	return w.Bytes()
}

func UnmarshalHandshake(buf []byte) (msg Handshake, err error) {
	b := bytes.NewReader(buf)

	if err = binary.Read(b, binary.BigEndian, &msg.publicKey); err != nil {
		return
	}

	if err = binary.Read(b, binary.BigEndian, &msg.signature); err != nil {
		return
	}

	return
}
