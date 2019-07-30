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
		assert.True(t, addressMatches(hostMatchTest.bind, hostMatchTest.subject) == hostMatchTest.equal, hostMatchTest.bind+" compared to "+hostMatchTest.subject)
	}
}

func TestQuickCheckAddressMatchesIPV4(t *testing.T) {
	f := func(ip [net.IPv4len]byte, port uint16) bool {
		return addressMatches(net.IP(ip[:]).String()+":"+strconv.FormatUint(uint64(port), 10), net.IP(ip[:]).String()+":"+strconv.FormatUint(uint64(port), 10))
	}
	assert.NoError(t, quick.Check(f, nil))
}

func TestQuickCheckAddressMatchesIPV6(t *testing.T) {
	f := func(ip [net.IPv6len]byte, port uint16) bool {
		return addressMatches("["+net.IP(ip[:]).String()+"]"+":"+strconv.FormatUint(uint64(port), 10), "["+net.IP(ip[:]).String()+"]"+":"+strconv.FormatUint(uint64(port), 10))

	}
	assert.NoError(t, quick.Check(f, nil))
}

func TestProtocol(t *testing.T) {
	c, cl := getClient(t, 1, 1)
	defer cl.Close()
	s, sl := getClient(t, 1, 1)
	defer sl.Close()

	sinfo := noise.Info{}
	accept := make(chan struct{})
	go func() {
		defer close(accept)
		serverHandle(t, s.protocol, sinfo, sl)
	}()

	cinfo := noise.Info{}
	cconn := clientHandle(t, c.protocol, cinfo, sl.Addr().String())
	defer cconn.Close()

	<-accept

	// Check ID
	assert.NotNil(t, cinfo.Get(KeyID))
	assert.NotNil(t, sinfo.Get(KeyID))

	sid := sinfo.Get(KeyID).(*ID)
	cid := cinfo.Get(KeyID).(*ID)
	assert.Equal(t, c.id.checksum, sid.checksum)
	assert.Equal(t, s.id.checksum, cid.checksum)

	// Test FindNode
	{
		res, err := s.protocol.FindNode(context.Background(), &FindNodeRequest{
			Id: cid.Marshal(),
		})
		assert.NoError(t, err)

		id, err := UnmarshalID(bytes.NewReader(res.Ids[0]))
		assert.NoError(t, err)
		assert.Equal(t, c.id.id, id.id)
	}
}

func TestProtocolEviction(t *testing.T) {
	s, sl := getClient(t, 1, 1)
	s.table.setBucketSize(1)
	defer sl.Close()

	accept := make(chan struct{})
	go func() {
		for {
			serverHandle(t, s.protocol, noise.Info{}, sl)
			accept <- struct{}{}
		}
	}()

	var clients []*Client
	go func() {
		for i := 0; i < 5; i++ {
			c, _ := getClient(t, 1, 1)

			_ = clientHandle(t, c.protocol, noise.Info{}, sl.Addr().String())
			clients = append(clients, c)
		}

	}()

	for i := 0; i < 5; i++ {
		<-accept
	}

	assert.Len(t, s.ClosestPeerIDs(), 1)
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

				conn.Close()
			},
		},
		{
			name: "disconnected during read signature",
			node: func(addr string) {
				conn, err := net.Dial("tcp", addr)
				if err != nil {
					t.Fatal(err)
				}

				c, _ := getClient(t, 1, 1)
				_, err = conn.Write(c.id.Marshal())
				assert.NoError(t, err)

				conn.Close()
			},
		},
		{
			name: "bad signature",
			node: func(addr string) {
				conn, err := net.Dial("tcp", addr)
				if err != nil {
					t.Fatal(err)
				}

				c, _ := getClient(t, 1, 1)
				_, err = conn.Write(c.id.Marshal())
				assert.NoError(t, err)

				var signature [64]byte
				edwards25519.Sign(c.keys.privateKey, []byte("invalid_message"))

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

				c, _ := getClient(t, 1, 10)
				buf := c.id.Marshal()
				signature := edwards25519.Sign(c.keys.privateKey, buf)
				handshake := append(buf, signature[:]...)

				_, err = conn.Write(handshake[:])
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

				c, _ := getClient(t, 10, 1)
				buf := c.id.Marshal()
				signature := edwards25519.Sign(c.keys.privateKey, buf)
				handshake := append(buf, signature[:]...)

				_, err = conn.Write(handshake[:])
				assert.NoError(t, err)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Test Server
			{
				c, lis := getClient(t, 10, 10)

				go tc.node(lis.Addr().String())

				conn, err := lis.Accept()
				if err != nil {
					t.Fatal(err)
				}

				_, err = c.protocol.Server(noise.Info{}, conn)
				assert.Error(t, err)
			}

			// Test Client
			{
				c, lis := getClient(t, 10, 10)

				go tc.node(lis.Addr().String())

				conn, err := lis.Accept()
				if err != nil {
					t.Fatal(err)
				}

				_, err = c.protocol.Client(noise.Info{}, context.Background(), "", conn)
				assert.Error(t, err)
			}
		})
	}
}

// Test if the peer ID's address is different than connections's address
func TestProtocolHandshakeClientBadAddress(t *testing.T) {
	node := func(addr string) {
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			t.Fatal(err)
		}

		c, _ := getClient(t, 1, 1)
		c.id.address = ":"
		buf := c.id.Marshal()
		signature := edwards25519.Sign(c.keys.privateKey, buf)

		handshake := append(buf, signature[:]...)
		_, err = conn.Write(handshake[:])
		assert.NoError(t, err)
	}
	c, lis := getClient(t, 1, 1)

	go node(lis.Addr().String())

	conn, err := lis.Accept()
	if err != nil {
		t.Fatal(err)
	}

	_, err = c.protocol.Client(noise.Info{}, context.Background(), "", conn)
	assert.Error(t, err)
}

// Test both nodes have the same ID
func TestProtocolHandshakeSamePeerID(t *testing.T) {
	c, lis := getClient(t, 1, 1)

	node := func(addr string) {
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			t.Fatal(err)
		}

		buf := c.id.Marshal()
		signature := edwards25519.Sign(c.keys.privateKey, buf)
		handshake := append(buf, signature[:]...)

		_, err = conn.Write(handshake[:])
		assert.NoError(t, err)
	}

	go node(lis.Addr().String())

	conn, err := lis.Accept()
	if err != nil {
		t.Fatal(err)
	}

	_, err = c.protocol.handshake(noise.Info{}, conn)
	assert.Error(t, err)
}

func TestProtocolConnWriteError(t *testing.T) {
	c, _ := getClient(t, 1, 1)

	// Test the Write is incomplete
	_, err := c.protocol.handshake(noise.Info{}, ErrorConn{isShortWrite: true})
	assert.Error(t, err)

	// Test the Write returns an error
	_, err = c.protocol.handshake(noise.Info{}, ErrorConn{isWriteError: true})
	assert.Error(t, err)
}

func serverHandle(t *testing.T, protocol Protocol, info noise.Info, lis net.Listener) {
	serverRawConn, err := lis.Accept()
	if err != nil {
		return
	}

	if _, err := protocol.Server(info, serverRawConn); err != nil {
		_ = serverRawConn.Close()
		t.Fatalf("Error server: %v", err)
	}
}

func clientHandle(t *testing.T, protocol Protocol, info noise.Info, lisAddr string) net.Conn {
	conn, err := net.Dial("tcp", lisAddr)
	if err != nil {
		t.Fatalf("Error client: %v", err)
	}
	_, err = protocol.Client(info, context.Background(), "", conn)
	if err != nil {
		t.Fatalf("Error client: %v", err)
	}

	return conn
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
