package discovery

import (
	"encoding/binary"
	"time"

	"github.com/perlin-network/noise/peer"
)

// serializeMessage compactly packs all bytes of a message together for cryptographic signing purposes.
func serializeMessage(id *peer.ID, message []byte) []byte {
	const uint32Size = 4

	serialized := make([]byte, uint32Size+len(id.Address)+uint32Size+len(id.Id)+len(message))
	pos := 0

	binary.LittleEndian.PutUint32(serialized[pos:], uint32(len(id.Address)))
	pos += uint32Size

	copy(serialized[pos:], []byte(id.Address))
	pos += len(id.Address)

	binary.LittleEndian.PutUint32(serialized[pos:], uint32(len(id.Id)))
	pos += uint32Size

	copy(serialized[pos:], id.Id)
	pos += len(id.Id)

	copy(serialized[pos:], message)
	pos += len(message)

	if pos != len(serialized) {
		panic("internal error: invalid serialization output")
	}

	return serialized
}

// serializePeerIDAndExpiration compacts the peer ID's address, port and signature expiration
// for cryptographic signing purposes. Weak signatures are used where the integrity of the entire
// message can be disregarded (e.g., ping/pong and node lookup messages)
func serializePeerIDAndExpiration(id *peer.ID, expiration *time.Time) []byte {
	const uint32Size = 4
	const uint64Size = 8

	serialized := make([]byte, uint32Size+len(id.Address)+uint64Size)
	pos := 0

	binary.LittleEndian.PutUint32(serialized[pos:], uint32(len(id.Address)))
	pos += uint32Size

	copy(serialized[pos:], []byte(id.Address))
	pos += len(id.Address)

	binary.LittleEndian.PutUint64(serialized[pos:], uint64(expiration.UnixNano()))
	pos += uint64Size

	if pos != len(serialized) {
		panic("internal error: invalid serialization output")
	}

	return serialized
}
