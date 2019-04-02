package skademlia

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/perlin-network/noise/edwards25519"
	"github.com/pkg/errors"
	"golang.org/x/crypto/blake2b"
	"io"
)

type ID struct {
	address   string
	publicKey edwards25519.PublicKey

	id, checksum, nonce [blake2b.Size256]byte
}

func NewID(address string, publicKey edwards25519.PublicKey, nonce [blake2b.Size256]byte) *ID {
	id := blake2b.Sum256(publicKey[:])
	checksum := blake2b.Sum256(id[:])

	return &ID{
		address:   address,
		publicKey: publicKey,

		id:       id,
		checksum: checksum,
		nonce:    nonce,
	}
}

func (m ID) Address() string {
	return m.address
}

func (m ID) PublicKey() edwards25519.PublicKey {
	return m.publicKey
}

func (m ID) Checksum() [blake2b.Size256]byte {
	return m.checksum
}

func (m ID) Nonce() [blake2b.Size256]byte {
	return m.nonce
}

func (m ID) String() string {
	return fmt.Sprintf("%s[%x]", m.address, m.publicKey)
}

func (m ID) Marshal() []byte {
	b := bytes.NewBuffer(make([]byte, 0, 2+len(m.address)+edwards25519.SizePublicKey+blake2b.Size256))

	_ = binary.Write(b, binary.BigEndian, uint16(len(m.address)))
	_, _ = b.WriteString(m.address)

	_, _ = b.Write(m.publicKey[:])
	_, _ = b.Write(m.nonce[:])

	return b.Bytes()
}

func UnmarshalID(b io.Reader) (m ID, err error) {
	var length uint16

	if err = binary.Read(b, binary.BigEndian, &length); err != nil {
		return
	}

	address := make([]byte, length)

	if err = binary.Read(b, binary.BigEndian, &address); err != nil {
		return
	}

	m.address = string(address)

	if err = binary.Read(b, binary.BigEndian, &m.publicKey); err != nil {
		return
	}

	m.id = blake2b.Sum256(m.publicKey[:])
	m.checksum = blake2b.Sum256(m.id[:])

	if err = binary.Read(b, binary.BigEndian, &m.nonce); err != nil {
		return
	}

	return
}

type IDs []*ID

func (ids IDs) Marshal() []byte {
	b := bytes.NewBuffer(make([]byte, 0, edwards25519.SizePublicKey+edwards25519.SizeSignature))

	_ = binary.Write(b, binary.BigEndian, uint8(len(ids)))

	for _, id := range ids {
		_, _ = b.Write(id.Marshal())
	}

	return b.Bytes()
}

func UnmarshalIDs(b io.Reader) (ids IDs, err error) {
	var size uint8

	if err := binary.Read(b, binary.BigEndian, &size); err != nil {
		return nil, errors.Wrap(err, "failed to read id array size")
	}

	ids = make(IDs, size)

	for i := range ids {
		id, err := UnmarshalID(b)

		if err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal one of the ids")
		}

		ids[i] = &id
	}

	return
}

type Keypair struct {
	self *ID

	privateKey edwards25519.PrivateKey
	publicKey  edwards25519.PublicKey

	id, checksum, nonce [blake2b.Size256]byte
	c1, c2              int
}

func (k *Keypair) ID(address string) *ID {
	if k.self == nil || k.self.address != address {
		k.self = NewID(address, k.publicKey, k.nonce)
	}

	return k.self
}

func NewKeys(c1, c2 int) (*Keypair, error) {
	publicKey, privateKey, id, checksum, err := generateKeys(c1)

	if err != nil {
		return nil, err
	}

	nonce, err := generateNonce(checksum, c2)

	if err != nil {
		return nil, errors.Wrap(err, "failed to generate valid puzzle nonce")
	}

	keys := &Keypair{
		privateKey: privateKey,
		publicKey:  publicKey,

		id:       id,
		checksum: checksum,
		nonce:    nonce,

		c1: c1,
		c2: c2,
	}

	return keys, nil
}

func LoadKeys(privateKey edwards25519.PrivateKey, nonce [blake2b.Size256]byte, c1, c2 int) (*Keypair, error) {
	publicKey := privateKey.Public()

	id := blake2b.Sum256(publicKey[:])
	checksum := blake2b.Sum256(id[:])

	if err := verifyPuzzle(checksum, nonce, c1, c2); err != nil {
		return nil, errors.Wrap(err, "keys are invalid")
	}

	keys := &Keypair{
		privateKey: privateKey,
		publicKey:  publicKey,

		id:       id,
		checksum: checksum,
		nonce:    nonce,

		c1: c1,
		c2: c2,
	}

	return keys, nil
}
