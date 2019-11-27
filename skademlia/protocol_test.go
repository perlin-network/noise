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
	"context"
	"fmt"
	"github.com/perlin-network/noise"
	"github.com/perlin-network/noise/edwards25519"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"net"
	"strconv"
	"testing"
	"testing/quick"
	"time"
)

var addressMatchesTests = []struct {
	bind    string
	subject string
	equal   bool
}{
	{"[7a8d:e2e:9b16:e1cc:f9c4:ce95:d96a:a32c]:100", "[7a8d:e2e:9b16:e1cc:f9c4:ce95:d96a:a32c]:100", true},
	{"127.0.0.1:100", "127.0.0.1:100", true},
	{"[::]:100", "127.0.0.1:100", true},
	{"127.0.0.1:100", "127.0.0.2:100", false},
	{"127.0.0.1:100", "127.0.0.1:101", false},
	{"0.0:0", "127.0.0.1:100", false},
	{"invalid_address", "127.0.0.1:100", false},
}

func TestAddressMatches(t *testing.T) {
	for _, hostMatchTest := range addressMatchesTests {
		am := addressMatches(hostMatchTest.bind, hostMatchTest.subject)
		assert.True(t, am == hostMatchTest.equal, hostMatchTest.bind+" compared to "+hostMatchTest.subject)
	}
}

func TestQuickCheckAddressMatchesIPV4(t *testing.T) {
	f := func(ip [net.IPv4len]byte, port uint16) bool {
		return addressMatches(
			net.IP(ip[:]).String()+":"+strconv.FormatUint(uint64(port), 10),
			net.IP(ip[:]).String()+":"+strconv.FormatUint(uint64(port), 10),
		)
	}
	assert.NoError(t, quick.Check(f, nil))
}

func TestQuickCheckAddressMatchesIPV6(t *testing.T) {
	f := func(ip [net.IPv6len]byte, port uint16) bool {
		return addressMatches(
			"["+net.IP(ip[:]).String()+"]"+":"+strconv.FormatUint(uint64(port), 10),
			"["+net.IP(ip[:]).String()+"]"+":"+strconv.FormatUint(uint64(port), 10),
		)
	}
	assert.NoError(t, quick.Check(f, nil))
}

func TestProtocol(t *testing.T) {
	c := newClientTestContainer(t, 1, 1)
	c.serve()

	defer c.cleanup()

	s := newClientTestContainer(t, 1, 1)
	s.serve()

	defer s.cleanup()

	var sinfo noise.Info

	s.onServer = func(info noise.Info) {
		sinfo = info
	}

	var cinfo noise.Info

	c.onClient = func(info noise.Info) {
		cinfo = info
	}

	accept := make(chan struct{})

	c.client.OnPeerJoin(func(conn *grpc.ClientConn, id *ID) {
		close(accept)
	})

	go func() {
		if _, err := c.client.Dial(s.lis.Addr().String()); err != nil {
			assert.FailNow(t, "failed to dial")
		}
	}()

	<-accept

	time.Sleep(1 * time.Second)

	// Check ID
	assert.NotNil(t, cinfo.Get(KeyID))
	assert.NotNil(t, sinfo.Get(KeyID))

	sid := sinfo.Get(KeyID).(*ID)
	cid := cinfo.Get(KeyID).(*ID)

	assert.Equal(t, c.client.id.checksum, sid.checksum)
	assert.Equal(t, s.client.id.checksum, cid.checksum)

	// Test FindNode
	{
		res, err := s.client.protocol.FindNode(context.Background(), &FindNodeRequest{
			Id: cid.Marshal(),
		})

		assert.NoError(t, err)

		id, err := UnmarshalID(bytes.NewReader(res.Ids[0]))

		assert.NoError(t, err)
		assert.Equal(t, c.client.id.id, id.id)
	}
}

func TestProtocolBadHandshake(t *testing.T) {
	tests := []struct {
		name string
		node func(addr string)
	}{
		{
			name: "disconnected during read ID",
			node: func(addr string) {
				conn, err := net.Dial("tcp", addr)
				if err != nil {
					t.Fatal(err)
				}

				if err := conn.Close(); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "disconnected during read signature",
			node: func(addr string) {
				conn, err := net.Dial("tcp", addr)
				if err != nil {
					t.Fatal(err)
				}

				c := newClientTestContainer(t, 1, 1)
				_, err = conn.Write(c.client.id.Marshal())
				assert.NoError(t, err)

				if err := conn.Close(); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "bad signature",
			node: func(addr string) {
				conn, err := net.Dial("tcp", addr)
				if err != nil {
					t.Fatal(err)
				}

				c := newClientTestContainer(t, 1, 1)
				_, err = conn.Write(c.client.id.Marshal())
				assert.NoError(t, err)

				var signature [64]byte
				edwards25519.Sign(c.client.keys.privateKey, []byte("invalid_message"))

				_, err = conn.Write(signature[:])
				assert.NoError(t, err)
			},
		},
		{
			name: "bad puzzle c1",
			node: func(addr string) {
				conn, err := net.Dial("tcp", addr)
				if err != nil {
					t.Fatal(err)
				}

				c := newClientTestContainer(t, 1, 10)
				buf := c.client.id.Marshal()
				signature := edwards25519.Sign(c.client.keys.privateKey, buf)
				handshake := append(buf, signature[:]...)

				_, err = conn.Write(handshake)
				assert.NoError(t, err)
			},
		},
		{
			name: "bad puzzle c2",
			node: func(addr string) {
				conn, err := net.Dial("tcp", addr)
				if err != nil {
					t.Fatal(err)
				}

				c := newClientTestContainer(t, 10, 1)
				buf := c.client.id.Marshal()
				signature := edwards25519.Sign(c.client.keys.privateKey, buf)
				handshake := append(buf, signature[:]...)

				_, err = conn.Write(handshake)
				assert.NoError(t, err)
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// Test Server
			{
				c := newClientTestContainer(t, 10, 10)

				go tc.node(c.lis.Addr().String())

				conn, err := c.lis.Accept()
				if err != nil {
					t.Fatal(err)
				}

				_, err = c.client.protocol.Server(noise.Info{}, conn)
				assert.Error(t, err)
			}

			// Test Client
			{
				c := newClientTestContainer(t, 10, 10)

				go tc.node(c.lis.Addr().String())

				conn, err := c.lis.Accept()
				if err != nil {
					t.Fatal(err)
				}

				_, err = c.client.protocol.Client(noise.Info{}, context.Background(), "", conn)
				assert.Error(t, err)
			}
		})
	}
}

// Test if the peer ID's address is different than connections's address
func TestProtocolHandshakeClientBadAddress(t *testing.T) {
	node := func(addr string) {
		conn, err := net.Dial("tcp", addr)
		if !assert.NoError(t, err) {
			return
		}

		c := newClientTestContainer(t, 1, 1)
		c.client.id.address = ":"
		buf := c.client.id.Marshal()
		signature := edwards25519.Sign(c.client.keys.privateKey, buf)

		handshake := append(buf, signature[:]...)
		_, err = conn.Write(handshake)
		assert.NoError(t, err)
	}

	c := newClientTestContainer(t, 1, 1)

	go node(c.lis.Addr().String())

	conn, err := c.lis.Accept()
	if err != nil {
		t.Fatal(err)
	}

	_, err = c.client.protocol.Client(noise.Info{}, context.Background(), "", conn)
	assert.Error(t, err)
}

// Test both nodes have the same ID
func TestProtocolHandshakeSamePeerID(t *testing.T) {
	c := newClientTestContainer(t, 1, 1)

	node := func(addr string) {
		conn, err := net.Dial("tcp", addr)
		if !assert.NoError(t, err) {
			return
		}

		buf := c.client.id.Marshal()
		signature := edwards25519.Sign(c.client.keys.privateKey, buf)
		handshake := append(buf, signature[:]...)

		_, err = conn.Write(handshake)
		assert.NoError(t, err)
	}

	go node(c.lis.Addr().String())

	conn, err := c.lis.Accept()
	if err != nil {
		t.Fatal(err)
	}

	_, err = c.client.protocol.handshake(conn)
	assert.Error(t, err)
}

func TestProtocolConnWriteError(t *testing.T) {
	c := newClientTestContainer(t, 1, 1)

	// Test the Write is incomplete
	_, err := c.client.protocol.handshake(ErrorConn{isShortWrite: true})
	assert.Error(t, err)

	// Test the Write returns an error
	_, err = c.client.protocol.handshake(ErrorConn{isWriteError: true})
	assert.Error(t, err)
}

type ErrorConn struct {
	isWriteError bool
	isShortWrite bool
}

func (c ErrorConn) Write(b []byte) (n int, err error) {
	if c.isWriteError {
		return 0, fmt.Errorf("write error")
	}

	if c.isShortWrite {
		return len(b) / 2, nil
	}

	return 0, nil
}

func (c ErrorConn) Read(b []byte) (n int, err error)   { return 0, nil }
func (c ErrorConn) Close() error                       { return nil }
func (c ErrorConn) LocalAddr() net.Addr                { return nil }
func (c ErrorConn) RemoteAddr() net.Addr               { return nil }
func (c ErrorConn) SetDeadline(t time.Time) error      { return nil }
func (c ErrorConn) SetReadDeadline(t time.Time) error  { return nil }
func (c ErrorConn) SetWriteDeadline(t time.Time) error { return nil }
