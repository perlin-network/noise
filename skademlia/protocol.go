package skademlia

import (
	"bytes"
	"github.com/perlin-network/noise"
	"github.com/perlin-network/noise/edwards25519"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
	"io"
	"net"
	"time"
)

const (
	KeyID = "skademlia.id"
)

type Protocol struct {
	client *Client
}

func (p Protocol) handshake(info noise.Info, conn net.Conn) (*ID, error) {
	buf := p.client.id.Marshal()
	signature := edwards25519.Sign(p.client.keys.privateKey, buf)

	handshake := append(buf, signature[:]...)

	n, err := conn.Write(handshake[:])
	if err != nil {
		return nil, err
	}

	if n != len(handshake) {
		return nil, errors.New("short write")
	}

	id, err := UnmarshalID(conn)
	if err != nil {
		return nil, err
	}

	if _, err = io.ReadFull(conn, signature[:]); err != nil {
		return nil, errors.Wrap(err, "failed to read signature")
	}

	if !edwards25519.Verify(id.publicKey, id.Marshal(), signature) {
		return nil, errors.New("failed to verify signature")
	}

	if err := verifyPuzzle(id.checksum, id.nonce, p.client.c1, p.client.c2); err != nil {
		return nil, errors.Wrap(err, "skademlia: peer connected with invalid id")
	}

	if prefixDiff(p.client.id.checksum[:], id.checksum[:], p.client.prefixDiffLen) < p.client.prefixDiffMin {
		return nil, errors.New("skademlia: peer id is too similar to ours")
	}

	ptr := &id

	info.Put(KeyID, ptr)

	for p.client.table.Update(ptr) == ErrBucketFull {
		bucket := p.client.table.buckets[getBucketID(p.client.table.self.checksum, id.checksum)]

		bucket.Lock()
		last := bucket.Back()
		lastID := last.Value.(*ID)
		bucket.Unlock()

		p.client.peersLock.RLock()
		lastConn, exists := p.client.peers[lastID.address]
		p.client.peersLock.RUnlock()

		if !exists {
			p.client.table.Delete(bucket, lastID)
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		if _, err = NewOverlayClient(lastConn).DoPing(ctx, &Ping{}); err != nil {
			p.client.table.Delete(bucket, lastID)
			lastConn.Close()
			cancel()
			continue
		}
		cancel()

		p.client.logger.Printf("Routing table is full; evicting peer %s.\n", id)

		// Ping was successful; disallow the current peer from connecting.

		conn.Close()

		p.client.peersLock.Lock()
		if conn, exists := p.client.peers[id.address]; exists {
			conn.Close()
			delete(p.client.peers, id.address)
		}
		delete(p.client.peers, id.address)
		p.client.peersLock.Unlock()

		return nil, errors.New("skademlia: cannot evict any peers to make room for new peer")
	}

	return ptr, nil
}

func (p Protocol) Client(info noise.Info, ctx context.Context, authority string, conn net.Conn) (net.Conn, error) {
	id, err := p.handshake(info, conn)
	if err != nil {
		return nil, err
	}

	p.client.logger.Printf("Connected to server %s.\n", id)

	return conn, nil
}

func (p Protocol) Server(info noise.Info, conn net.Conn) (net.Conn, error) {
	id, err := p.handshake(info, conn)
	if err != nil {
		return nil, err
	}

	p.client.logger.Printf("Client %s has connected to you.\n", id)

	go func() {
		if _, err = p.client.Dial(id.address); err != nil {
			_ = conn.Close()
		}
	}()

	return conn, nil
}

func (p Protocol) DoPing(context.Context, *Ping) (*Ping, error) {
	return &Ping{}, nil
}

func (p Protocol) FindNode(ctx context.Context, req *FindNodeRequest) (*FindNodeResponse, error) {
	target, err := UnmarshalID(bytes.NewReader(req.Id))
	if err != nil {
		return nil, err
	}

	ids := p.client.table.FindClosest(&target, p.client.table.getBucketSize())

	res := &FindNodeResponse{Ids: make([][]byte, len(ids))}

	for i := range ids {
		res.Ids[i] = ids[i].Marshal()
	}

	return res, nil
}
