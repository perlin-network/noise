package kademlia

import (
	"fmt"
	"github.com/perlin-network/noise"
	"io"
)

// Ping represents an empty ping message.
type Ping struct{}

func (r Ping) Marshal() []byte { return nil }

func UnmarshalPing([]byte) (Ping, error) { return Ping{}, nil }

// Pong represents an empty pong message.
type Pong struct{}

func (r Pong) Marshal() []byte { return nil }

func UnmarshalPong([]byte) (Pong, error) { return Pong{}, nil }

// FindNodeRequest represents a FIND_NODE RPC call in the Kademlia specification. It contains a target ID to which
// a peer is supposed to respond with a slice of IDs that neighbor the target ID w.r.t. XOR distance.
type FindNodeRequest struct {
	Target noise.PublicKey
}

func (r FindNodeRequest) Marshal() []byte {
	return r.Target[:]
}

func UnmarshalFindNodeRequest(buf []byte) (FindNodeRequest, error) {
	if len(buf) != noise.SizePublicKey {
		return FindNodeRequest{}, fmt.Errorf("expected buf to be %d bytes, but got %d bytes: %w", noise.SizePublicKey, len(buf), io.ErrUnexpectedEOF)
	}

	var req FindNodeRequest
	copy(req.Target[:], buf)

	return req, nil
}

type FindNodeResponse struct {
	Results []noise.ID
}

func (r FindNodeResponse) Marshal() []byte {
	buf := []byte{byte(len(r.Results))}

	for _, result := range r.Results {
		buf = append(buf, result.Marshal()...)
	}

	return buf
}

func UnmarshalFindNodeResponse(buf []byte) (FindNodeResponse, error) {
	var res FindNodeResponse

	if len(buf) < 1 {
		return res, io.ErrUnexpectedEOF
	}

	size := buf[0]
	buf = buf[1:]

	results := make([]noise.ID, 0, size)

	for i := 0; i < cap(results); i++ {
		id, err := noise.UnmarshalID(buf)
		if err != nil {
			return res, io.ErrUnexpectedEOF
		}

		results = append(results, id)
		buf = buf[id.Size():]
	}

	res.Results = results

	return res, nil
}
