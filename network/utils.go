package network

import (
	"encoding/binary"
	"github.com/perlin-network/noise/protobuf"
)

func serializeMessage(id *protobuf.ID, message []byte) []byte {
	const UINT32_SIZE = 4

	serialized := make([]byte, UINT32_SIZE+len(id.Address)+UINT32_SIZE+len(id.PublicKey)+len(message))
	pos := 0

	binary.LittleEndian.PutUint32(serialized[pos:], uint32(len(id.Address)))
	pos += UINT32_SIZE

	copy(serialized[pos:], []byte(id.Address))
	pos += len(id.Address)

	binary.LittleEndian.PutUint32(serialized[pos:], uint32(len(id.PublicKey)))
	pos += UINT32_SIZE

	copy(serialized[pos:], id.PublicKey)
	pos += len(id.PublicKey)

	copy(serialized[pos:], message)
	pos += len(message)

	if pos != len(serialized) {
		panic("internal error: invalid serialization output")
	}

	return serialized
}

// FilterPeers filters out duplicate/empty addresses.
func FilterPeers(address string, peers []string) (filtered []string) {
	visited := make(map[string]struct{})
	visited[address] = struct{}{}

	for _, peerAddress := range peers {
		if len(peerAddress) == 0 {
			continue
		}

		resolved, err := ToUnifiedAddress(peerAddress)
		if err != nil {
			continue
		}
		if _, exists := visited[resolved]; !exists {
			filtered = append(filtered, resolved)
			visited[resolved] = struct{}{}
		}
	}
	return filtered
}
