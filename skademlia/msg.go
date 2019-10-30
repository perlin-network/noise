package skademlia

import (
	"github.com/Yayg/noise"
	"github.com/Yayg/noise/payload"
	"github.com/pkg/errors"
)

const maxNumPeersToLookup = 64

var (
	_ noise.Message = (*Ping)(nil)
	_ noise.Message = (*Evict)(nil)
	_ noise.Message = (*LookupRequest)(nil)
	_ noise.Message = (*LookupResponse)(nil)
)

type Ping struct{ ID }

func (m Ping) Read(reader payload.Reader) (noise.Message, error) {
	id, err := ID{}.Read(reader)
	if err != nil {
		return nil, errors.Wrap(err, "skademlia: failed to read id")
	}

	m.ID = id.(ID)

	return m, nil
}

type Evict struct{ noise.EmptyMessage }

type LookupRequest struct{ ID }

func (m LookupRequest) Read(reader payload.Reader) (noise.Message, error) {
	id, err := ID{}.Read(reader)
	if err != nil {
		return nil, errors.Wrap(err, "skademlia: failed to read id")
	}

	m.ID = id.(ID)

	return m, nil
}

type LookupResponse struct {
	peers []ID
}

func (l LookupResponse) Read(reader payload.Reader) (noise.Message, error) {
	numPeers, err := reader.ReadUint32()
	if err != nil {
		return nil, errors.Wrap(err, "failed to read number of peers")
	}

	if numPeers > maxNumPeersToLookup {
		return nil, errors.Errorf("received too many peers on lookup response; got %d peer IDs when at most we can only handle %d peer IDs", numPeers, maxNumPeersToLookup)
	}

	l.peers = make([]ID, numPeers)

	for i := 0; i < int(numPeers); i++ {
		id, err := ID{}.Read(reader)
		if err != nil {
			return nil, errors.Wrap(err, "failed to decode peer ID")
		}

		l.peers[i] = id.(ID)
	}

	return l, nil
}

func (l LookupResponse) Write() []byte {
	writer := payload.NewWriter(nil)

	writer.WriteUint32(uint32(len(l.peers)))

	for _, id := range l.peers {
		writer.Write(id.Write())
	}

	return writer.Bytes()
}
