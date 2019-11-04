// Copyright (c) 2019 Perlin
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

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

func (p Protocol) registerPeerID(id *ID) error {
	for p.client.table.Update(id) == ErrBucketFull {
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
		if _, err := NewOverlayClient(lastConn).DoPing(ctx, &Ping{}); err != nil {
			p.client.table.Delete(bucket, lastID)
			lastConn.Close()
			cancel()
			continue
		}
		cancel()

		p.client.logger.Printf("Routing table is full; evicting peer %s.\n", id)

		// Ping was successful; disallow the current peer from connecting.

		p.client.peersLock.Lock()
		delete(p.client.peers, id.address)
		p.client.peersLock.Unlock()

		return errors.New("skademlia: cannot evict any peers to make room for new peer")
	}

	return nil
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

	return ptr, nil
}

func (p Protocol) Client(info noise.Info, ctx context.Context, authority string, conn net.Conn) (net.Conn, error) {
	var cancel context.CancelFunc
	ctx, cancel = context.WithCancel(ctx)

	id, err := p.handshake(info, conn)
	if err != nil {
		cancel()
		if cerr := conn.Close(); cerr != nil {
			err = errors.Wrap(cerr, err.Error())
		}
		return nil, err
	}

	if !addressMatches(id.Address(), conn.RemoteAddr().String()) {
		err := errors.Errorf("connected to peer with addr %s, but their id writes addr %s", conn.RemoteAddr().String(), id.Address())
		if cerr := conn.Close(); cerr != nil {
			err = errors.Wrap(cerr, err.Error())
		}
		return nil, err
	}

	p.client.logger.Printf("Connected to server %s.\n", id)

	info.Put(KeyID, id)

	/* We verified that the server is valid, add them to the routing table */
	_ = p.registerPeerID(id)

	return conn, nil
}

func addressMatches(bind string, subject string) bool {
	bindHost, bindPort, err := net.SplitHostPort(bind)
	if err != nil {
		return false
	}

	subjectHost, subjectPort, err := net.SplitHostPort(subject)
	if err != nil {
		return false
	}

	if bindPort != subjectPort {
		return false
	}

	subjectAddrs, err := net.LookupIP(subjectHost)
	if err != nil {
		return false
	}

	bindAddrs, err := net.LookupIP(bindHost)
	if err != nil {
		return false
	}

	for _, bindAddr := range bindAddrs {
		if bindAddr.IsUnspecified() {
			return true
		}

		for _, subjectAddr := range subjectAddrs {
			if bindAddr.Equal(subjectAddr) {
				return true
			}
		}
	}

	return false
}

func (p Protocol) Server(info noise.Info, conn net.Conn) (net.Conn, error) {
	id, err := p.handshake(info, conn)
	if err != nil {
		if cerr := conn.Close(); cerr != nil {
			err = errors.Wrap(cerr, err.Error())
		}
		return nil, err
	}

	p.client.logger.Printf("Client %s has connected to you", id)

	info.Put(KeyID, id)

	go func() {
		if _, err = p.client.Dial(id.address, WithTimeout(3*time.Second)); err != nil {
			p.client.logger.Printf("Client %s was not able to be dialed back, closing connection", id)
		} else {
			/* We were able to dial the peer, add them to our table */
			p.client.logger.Printf("Client %s was successfully dialed back, adding it as a peer", id)

			_ = p.registerPeerID(id)
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
