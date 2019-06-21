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
	"github.com/perlin-network/noise"
	"net"
	"strconv"
	"testing"
	"testing/quick"

	"github.com/stretchr/testify/assert"
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

func getClient(t *testing.T) (*Client, net.Listener) {
	lis, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatal(err)
	}

	c1 := 1
	c2 := 1
	keys, err := NewKeys(c1, c2)
	if err != nil {
		t.Fatalf("error NewKeys(): %v", err)
	}

	c := NewClient(lis.Addr().String(), keys, WithC1(c1), WithC2(c2))
	c.SetCredentials(noise.NewCredentials(lis.Addr().String(), c.Protocol()))

	return c, lis
}

func TestProtocol(t *testing.T) {
	c, cl := getClient(t)
	defer cl.Close()
	s, sl := getClient(t)
	defer sl.Close()

	sinfo := noise.Info{}
	accept := make(chan struct{})
	go serverHandle(t, s.protocol, sinfo, accept, sl)

	cinfo := noise.Info{}
	clientHandle(t, c.protocol, cinfo, sl.Addr().String())

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

func serverHandle(t *testing.T, protocol Protocol, info noise.Info, accept chan struct{}, lis net.Listener) {
	defer close(accept)

	serverRawConn, err := lis.Accept()
	if err != nil {
		t.Fatal(err)
	}

	if _, err := protocol.Server(info, serverRawConn); err != nil {
		_ = serverRawConn.Close()
		t.Fatalf("Error server: %v", err)
	}
}

func clientHandle(t *testing.T, protocol Protocol, info noise.Info, lisAddr string) {
	conn, err := net.Dial("tcp", lisAddr)
	if err != nil {
		t.Fatalf("Error client: %v", err)
	}
	defer conn.Close()

	_, err = protocol.Client(info, context.Background(), "", conn)
	if err != nil {
		t.Fatalf("Error client: %v", err)
	}
}
