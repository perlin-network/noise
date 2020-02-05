package noise

import (
	"encoding/binary"
	"encoding/hex"
	"io"
	"net"
	"strconv"
	"strings"
)

// ID represents a peer ID. It comprises of a cryptographic public key, and a public, reachable network address
// specified by a IPv4/IPv6 host and 16-bit port number. The size of an ID in terms of its byte representation
// is static, with its contents being deterministic.
type ID struct {
	// The Ed25519 public key of the bearer of this ID.
	ID PublicKey `json:"public_key"`

	// Public host of the bearer of this ID.
	Host net.IP `json:"address"`

	// Public port of the bearer of this ID.
	Port uint16

	// 'host:port'
	Address string
}

// NewID instantiates a new, immutable cryptographic user ID.
func NewID(id PublicKey, host net.IP, port uint16) ID {
	addr := net.JoinHostPort(normalizeIP(host), strconv.FormatUint(uint64(port), 10))
	return ID{ID: id, Host: host, Port: port, Address: addr}
}

// Size returns the number of bytes this ID comprises of.
func (e ID) Size() int {
	return len(e.ID) + net.IPv6len + 2
}

// String returns a JSON representation of this ID.
func (e ID) String() string {
	var builder strings.Builder
	builder.WriteString(`{"public_key": "`)
	builder.WriteString(hex.EncodeToString(e.ID[:]))
	builder.WriteString(`", "address": "`)
	builder.WriteString(e.Address)
	builder.WriteString(`"}`)
	return builder.String()
}

// Marshal serializes this ID into its byte representation.
func (e ID) Marshal() []byte {
	buf := make([]byte, e.Size())

	copy(buf[:len(e.ID)], e.ID[:])
	copy(buf[len(e.ID):len(e.ID)+net.IPv6len], e.Host)
	binary.BigEndian.PutUint16(buf[len(e.ID)+net.IPv6len:len(e.ID)+net.IPv6len+2], e.Port)

	return buf
}

// UnmarshalID deserializes buf, representing a slice of bytes, ID instance. It throws io.ErrUnexpectedEOF if the
// contents of buf is malformed.
func UnmarshalID(buf []byte) (ID, error) {
	if len(buf) < SizePublicKey {
		return ID{}, io.ErrUnexpectedEOF
	}

	var id PublicKey

	copy(id[:], buf[:SizePublicKey])
	buf = buf[SizePublicKey:]

	if len(buf) < net.IPv6len {
		return ID{}, io.ErrUnexpectedEOF
	}

	host := make([]byte, net.IPv6len)
	copy(host, buf[:net.IPv6len])

	buf = buf[net.IPv6len:]

	if len(buf) < 2 {
		return ID{}, io.ErrUnexpectedEOF
	}

	port := binary.BigEndian.Uint16(buf[:2])

	return NewID(id, host, port), nil
}
