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

package cipher

import (
	"bytes"
	"crypto/sha256"
	"net"
	"reflect"
	"testing"
)

type testConn struct {
	net.Conn
	in  *bytes.Buffer
	out *bytes.Buffer
}

func (c *testConn) Read(b []byte) (n int, err error) {
	return c.in.Read(b)
}

func (c *testConn) Write(b []byte) (n int, err error) {
	return c.out.Write(b)
}

func (c *testConn) Close() error {
	return nil
}

func newTestConn(in, out *bytes.Buffer) *connAEAD {
	key := []byte{0x1f, 0x8b, 0x08, 0x00, 0x00, 0x09, 0x6e, 0x88, 0x02, 0xff, 0xe2, 0xd2, 0x4c, 0xce, 0x4f, 0x49, 0x1f,
		0x8b, 0x08, 0x00, 0x00, 0x09, 0x6e, 0x88, 0x02, 0xff, 0xe2, 0xd2, 0x4c, 0xce, 0x4f, 0x49}

	tc := testConn{
		in:  in,
		out: out,
	}

	suite, _, _ := DeriveAEAD(Aes256GCM(), sha256.New, key, nil)

	return newConnAEAD(suite, &tc)
}

func newConnPair() (client, server *connAEAD) {
	clientBuf := new(bytes.Buffer)
	serverBuf := new(bytes.Buffer)
	clientConn := newTestConn(clientBuf, serverBuf)
	serverConn := newTestConn(serverBuf, clientBuf)

	return clientConn, serverConn
}

func TestPingPong(t *testing.T) {
	clientConn, serverConn := newConnPair()
	clientMsg := []byte("Client Message")

	if n, err := clientConn.Write(clientMsg); n != len(clientMsg) || err != nil {
		t.Fatalf("Client Write() = %v, %v; want %v, <nil>", n, err, len(clientMsg))
	}

	rcvClientMsg := make([]byte, len(clientMsg))

	if n, err := serverConn.Read(rcvClientMsg); n != len(rcvClientMsg) || err != nil {
		t.Fatalf("Server Read() = %v, %v; want %v, <nil>", n, err, len(rcvClientMsg))
	}

	if !reflect.DeepEqual(clientMsg, rcvClientMsg) {
		t.Fatalf("Client Write()/Server Read() = %v, want %v", rcvClientMsg, clientMsg)
	}

	serverMsg := []byte("Server Message")

	if n, err := serverConn.Write(serverMsg); n != len(serverMsg) || err != nil {
		t.Fatalf("Server Write() = %v, %v; want %v, <nil>", n, err, len(serverMsg))
	}

	rcvServerMsg := make([]byte, len(serverMsg))

	if n, err := clientConn.Read(rcvServerMsg); n != len(rcvServerMsg) || err != nil {
		t.Fatalf("Client Read() = %v, %v; want %v, <nil>", n, err, len(rcvServerMsg))
	}

	if !reflect.DeepEqual(serverMsg, rcvServerMsg) {
		t.Fatalf("Server Write()/Client Read() = %v, want %v", rcvServerMsg, serverMsg)
	}
}

func TestSmallReadBuffer(t *testing.T) {
	clientConn, serverConn := newConnPair()
	msg := []byte("Very Important Message")

	if n, err := clientConn.Write(msg); err != nil {
		t.Fatalf("Write() = %v, %v; want %v, <nil>", n, err, len(msg))
	}

	rcvMsg := make([]byte, len(msg))
	n := 2 // Arbitrary index to break rcvMsg in two.
	rcvMsg1 := rcvMsg[:n]
	rcvMsg2 := rcvMsg[n:]

	if n, err := serverConn.Read(rcvMsg1); n != len(rcvMsg1) || err != nil {
		t.Fatalf("Read() = %v, %v; want %v, <nil>", n, err, len(rcvMsg1))
	}

	if n, err := serverConn.Read(rcvMsg2); n != len(rcvMsg2) || err != nil {
		t.Fatalf("Read() = %v, %v; want %v, <nil>", n, err, len(rcvMsg2))
	}

	if !reflect.DeepEqual(msg, rcvMsg) {
		t.Fatalf("Write()/Read() = %v, want %v", rcvMsg, msg)
	}
}
