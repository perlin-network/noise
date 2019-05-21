package skademlia

import (
	"bytes"
	"context"
	"github.com/perlin-network/noise"
	"github.com/phf/go-queue/queue"
	"github.com/pkg/errors"
	"golang.org/x/crypto/blake2b"
	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"
	"sort"
	"sync"
	"time"
)

type Client struct {
	c1, c2, prefixDiffLen, prefixDiffMin int

	creds *noise.Credentials
	dopts []grpc.DialOption

	id    *ID
	keys  *Keypair
	table *Table

	peers     map[string]*grpc.ClientConn
	peersLock sync.RWMutex

	protocol Protocol
}

func NewClient(addr string, keys *Keypair, opts ...Option) *Client {
	id := keys.ID(addr)
	table := NewTable(id)

	c := &Client{
		c1:            DefaultC1,
		c2:            DefaultC2,
		prefixDiffLen: DefaultPrefixDiffLen,
		prefixDiffMin: DefaultPrefixDiffMin,

		id:    id,
		keys:  keys,
		table: table,

		peers: make(map[string]*grpc.ClientConn),
	}

	c.protocol = Protocol{client: c}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

func (c *Client) SetCredentials(creds *noise.Credentials) {
	c.creds = creds
}

func (c *Client) Protocol() Protocol {
	return c.protocol
}

func (c *Client) BucketSize() int {
	return c.table.getBucketSize()
}

func (c *Client) Keys() *Keypair {
	return c.keys
}

func (c *Client) ID() *ID {
	return c.id
}

func (c *Client) AllPeers() []*grpc.ClientConn {
	c.peersLock.RLock()
	defer c.peersLock.RUnlock()

	conns := make([]*grpc.ClientConn, len(c.peers))

	for _, conn := range c.peers {
		conns = append(conns, conn)
	}

	return conns
}

func (c *Client) ClosestPeers() []*grpc.ClientConn {
	ids := c.table.FindClosest(c.table.self, c.table.getBucketSize())

	conns := make([]*grpc.ClientConn, len(ids))

	for i := range ids {
		if conn, err := c.Dial(ids[i].address); err == nil {
			conns[i] = conn
		}
	}

	return conns
}

func (c *Client) ClosestPeerIDs() []*ID {
	return c.table.FindClosest(c.table.self, c.table.getBucketSize())
}

func (c *Client) Listen(opts ...grpc.ServerOption) *grpc.Server {
	server := grpc.NewServer(
		append(
			opts,
			grpc.Creds(c.creds),
			grpc.UnaryInterceptor(c.serverUnaryInterceptor),
			grpc.StreamInterceptor(c.serverStreamInterceptor),
		)...,
	)

	RegisterOverlayServer(server, c.protocol)

	return server
}

func (c *Client) Dial(addr string) (*grpc.ClientConn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	return c.DialContext(ctx, addr)
}

func (c *Client) DialContext(ctx context.Context, addr string) (*grpc.ClientConn, error) {
	if addr == c.table.self.address {
		return nil, errors.New("attempted to dial self")
	}

	c.peersLock.RLock()
	if conn, exists := c.peers[addr]; exists {
		c.peersLock.RUnlock()
		return conn, nil
	}
	c.peersLock.RUnlock()

	conn, err := grpc.DialContext(ctx, addr,
		append(
			c.dopts,
			grpc.WithTransportCredentials(c.creds),
		)...,
	)

	if err != nil {
		return nil, errors.Wrap(err, "failed to dial peer")
	}

	c.peersLock.Lock()
	c.peers[addr] = conn
	c.peersLock.Unlock()

	defer func() {
		if err != nil {
			c.peersLock.Lock()
			delete(c.peers, addr)
			c.peersLock.Unlock()

			conn.Close()
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err = NewOverlayClient(conn).DoPing(ctx, &Ping{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to ping peer")
	}

	return conn, nil
}

// RefreshPeriodically periodically refreshes the list of peers for a node given a time period.
func (c *Client) RefreshPeriodically(stop chan struct{}, duration time.Duration) {
	timer := time.NewTicker(duration)
	defer timer.Stop()

	for {
		select {
		case <-stop:
			return
		case <-timer.C:
			c.Bootstrap()
		}
	}
}

func (c *Client) Bootstrap() (results []*ID) {
	return c.FindNode(c.table.self, c.table.getBucketSize(), 3, 8)
}

func (c *Client) FindNode(target *ID, k int, a int, d int) (results []*ID) {
	type request ID

	type response struct {
		requester *request
		ids       []*ID
	}

	var mu sync.Mutex

	visited := map[[blake2b.Size256]byte]struct{}{
		c.table.self.checksum: {},
		target.checksum:       {},
	}

	lookups := make([]queue.Queue, d)

	for i, id := range c.table.FindClosest(target, k) {
		visited[id.checksum] = struct{}{}
		lookups[i%d].PushBack(id)
	}

	var wg sync.WaitGroup
	wg.Add(d)

	for _, lookup := range lookups { // Perform d parallel disjoint lookups.
		go func(lookup queue.Queue) {
			requests := make(chan *request, a)
			responses := make(chan *response, a)

			for i := 0; i < a; i++ { // Perform α queries in parallel per disjoint lookup.
				go func() {
					for id := range requests {
						f := func() error {
							conn, err := c.Dial(id.address)

							if err != nil {
								responses <- nil
								return err
							}

							ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
							defer cancel()

							res, err := NewOverlayClient(conn).FindNode(ctx, &FindNodeRequest{Id: (ID)(*id).Marshal()})

							if err != nil {
								responses <- nil
								return err
							}

							ids := make([]*ID, len(res.Ids))

							for i := range res.Ids {
								id, err := UnmarshalID(bytes.NewReader(res.Ids[i]))
								if err != nil {
									responses <- nil
									return err
								}

								ids[i] = &id
							}

							responses <- &response{requester: id, ids: ids}
							return nil
						}

						if err := f(); err != nil {
							continue
						}
					}
				}()
			}

			pending := 0

			for lookup.Len() > 0 || pending > 0 {
				for lookup.Len() > 0 && len(requests) < cap(requests) {
					requests <- (*request)(lookup.PopFront().(*ID))
					pending++
				}

				if pending > 0 {
					res := <-responses

					if res != nil {
						for _, id := range res.ids {
							mu.Lock()
							if _, seen := visited[id.checksum]; !seen {
								visited[id.checksum] = struct{}{}
								lookup.PushBack(id)
							}
							mu.Unlock()
						}

						mu.Lock()
						results = append(results, (*ID)(res.requester))
						mu.Unlock()
					}

					pending--
				}
			}

			close(requests)

			wg.Done()
		}(lookup)
	}

	wg.Wait() // Wait until all d parallel disjoint lookups are complete.

	sort.Slice(results, func(i, j int) bool {
		return bytes.Compare(xor(results[i].checksum[:], target.checksum[:]), xor(results[j].checksum[:], target.checksum[:])) == -1
	})

	if len(results) > k {
		results = results[:k]
	}

	return
}

func (c *Client) serverUnaryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	res, err := handler(ctx, req)

	p, ok := peer.FromContext(ctx)
	if !ok {
		return res, errors.New("could not load peer")
	}

	id := noise.InfoFromPeer(p).Get(KeyID)

	if id != nil {
		id := id.(*ID)

		if err := c.table.Update(id); err != nil {
			return res, err
		}
	}

	return res, err
}

func (c *Client) serverStreamInterceptor(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	p, ok := peer.FromContext(ss.Context())
	if !ok {
		return errors.New("could not load peer")
	}

	id := noise.InfoFromPeer(p).Get(KeyID)

	if id == nil {
		return errors.New("peer does not have id available")
	}

	return handler(srv, InterceptedServerStream{ServerStream: ss, client: c, id: id.(*ID)})
}

type InterceptedClientStream struct {
	grpc.ClientStream

	client *Client
	id     *ID
}

func (s InterceptedClientStream) SendMsg(m interface{}) error {
	if err := s.ClientStream.SendMsg(m); err != nil {
		return err
	}

	if err := s.client.table.Update(s.id); err != nil {
		return err
	}

	return nil
}

func (s InterceptedClientStream) RecvMsg(m interface{}) error {
	if err := s.ClientStream.RecvMsg(m); err != nil {
		return err
	}

	if err := s.client.table.Update(s.id); err != nil {
		return err
	}

	return nil
}

type InterceptedServerStream struct {
	grpc.ServerStream

	client *Client
	id     *ID
}

func (s InterceptedServerStream) SendMsg(m interface{}) error {
	if err := s.ServerStream.SendMsg(m); err != nil {
		return err
	}

	if err := s.client.table.Update(s.id); err != nil {
		return err
	}

	return nil
}

func (s InterceptedServerStream) RecvMsg(m interface{}) error {
	if err := s.ServerStream.RecvMsg(m); err != nil {
		return err
	}

	if err := s.client.table.Update(s.id); err != nil {
		return err
	}

	return nil
}
