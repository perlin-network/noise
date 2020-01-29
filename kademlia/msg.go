package kademlia

import (
	"fmt"
	"github.com/perlin-network/noise"
	"io"
)

// Ping represents an empty ping message.
type Ping struct{}

// Marshal implements noise.Serializable and returns a nil byte slice.
func (r Ping) Marshal() []byte { return nil }

// UnmarshalPing returns a Ping instance and never throws an error.
func UnmarshalPing([]byte) (Ping, error) { return Ping{}, nil }

// Pong represents an empty pong message.
type Pong struct{}

// Marshal implements noise.Serializable and returns a nil byte slice.
func (r Pong) Marshal() []byte { return nil }

// UnmarshalPong returns a Pong instance and never throws an error.
func UnmarshalPong([]byte) (Pong, error) { return Pong{}, nil }

// FindNodeRequest represents a FIND_NODE RPC call in the Kademlia specification. It contains a target public key to
// which a peer is supposed to respond with a slice of IDs that neighbor the target ID w.r.t. XOR distance.
type FindNodeRequest struct {
	Target noise.PublicKey
}

// Marshal implements noise.Serializable and returns the public key of the target for this search request as a
// byte slice.
func (r FindNodeRequest) Marshal() []byte {
	return r.Target[:]
}

// UnmarshalFindNodeRequest decodes buf, which must be the exact size of a public key, into a FindNodeRequest. It
// throws an io.ErrUnexpectedEOF if buf is malformed.
func UnmarshalFindNodeRequest(buf []byte) (FindNodeRequest, error) {
	if len(buf) != noise.SizePublicKey {
		return FindNodeRequest{}, fmt.Errorf("expected buf to be %d bytes, but got %d bytes: %w",
			noise.SizePublicKey, len(buf), io.ErrUnexpectedEOF,
		)
	}

	var req FindNodeRequest
	copy(req.Target[:], buf)

	return req, nil
}

// FindNodeResponse returns the results of a FIND_NODE RPC call which comprises of the IDs of peers closest to a
// target public key specified in a FindNodeRequest.
type FindNodeResponse struct {
	Results []noise.ID
}

// Marshal implements noise.Serializable and encodes the list of closest peer ID results into a byte representative
// of the length of the list, concatenated with the serialized byte representation of the peer IDs themselves.
func (r FindNodeResponse) Marshal() []byte {
	buf := []byte{byte(len(r.Results))}

	for _, result := range r.Results {
		buf = append(buf, result.Marshal()...)
	}

	return buf
}

// UnmarshalFindNodeResponse decodes buf, which is expected to encode a list of closest peer ID results, into a
// FindNodeResponse. It throws an io.ErrUnexpectedEOF if buf is malformed.
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
