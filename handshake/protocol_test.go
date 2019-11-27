package handshake

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"github.com/perlin-network/noise"
	"github.com/stretchr/testify/assert"
	"net"
	"testing"
	"time"
)

func TestProtocol(t *testing.T) {
	ecdh := NewECDH()

	lis, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatal(err)
	}

	accept := make(chan noise.Info)

	go func() {
		serverRawConn, err := lis.Accept()
		if !assert.NoError(t, err) {
			close(accept)
			return
		}

		// Initiate server handshake

		info := noise.Info{}
		if _, err := ecdh.Server(info, serverRawConn); !assert.NoError(t, err) {
			_ = serverRawConn.Close()
			close(accept)
			return
		}

		accept <- info
	}()

	// Dial the server

	conn, err := net.Dial("tcp", lis.Addr().String())
	if err != nil {
		t.Fatal(err)
	}

	// Initiate client handshake

	clientInfo := noise.Info{}
	if _, err := ecdh.Client(clientInfo, context.Background(), "", conn); err != nil {
		t.Fatalf("Error protocol.Client(): %v", err)
	}

	serverInfo := <-accept

	if !bytes.Equal(serverInfo.Bytes(SharedKey), clientInfo.Bytes(SharedKey)) {
		t.Fatalf("Key is different: %x vs %x ", serverInfo.Bytes(SharedKey), clientInfo.Bytes(SharedKey))
	}
}

func TestProtocolBadHandshake(t *testing.T) {
	lis, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		node func()
	}{
		{
			name: "bad message",
			node: func() {
				conn, err := net.Dial("tcp", lis.Addr().String())
				if err != nil {
					t.Fatal(err)
				}

				defer func() {
					_ = conn.Close()
				}()

				// Write random message

				msg := make([]byte, 96)
				if _, err := rand.Read(msg); err != nil {
					t.Fatal(err)
				}

				_, err = conn.Write(msg)
				assert.NoError(t, err)
			},
		},
		{
			name: "disconnected",
			node: func() {
				conn, err := net.Dial("tcp", lis.Addr().String())
				if err != nil {
					t.Fatal(err)
				}

				buf := make([]byte, 1024)
				n, err := conn.Read(buf)
				assert.NoError(t, err)
				assert.Equal(t, 96, n)

				if err := conn.Close(); err != nil {
					t.Fatal(err)
				}
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// Test Server
			{
				go tc.node()

				conn, err := lis.Accept()
				if err != nil {
					t.Fatal(err)
				}

				ecdh := NewECDH()
				_, err = ecdh.Server(noise.Info{}, conn)
				assert.Error(t, err)
			}

			// Test Client
			{
				go tc.node()

				conn, err := lis.Accept()
				if err != nil {
					t.Fatal(err)
				}

				ecdh := NewECDH()
				_, err = ecdh.Client(noise.Info{}, context.Background(), "", conn)
				assert.Error(t, err)
			}
		})
	}
}

func TestProtocolConnWriteError(t *testing.T) {
	ecdh := NewECDH()

	// Test the Write is incomplete
	_, err := ecdh.Server(noise.Info{}, ErrorConn{isShortWrite: true})
	assert.Error(t, err)

	// Test the Write returns an error
	_, err = ecdh.Server(noise.Info{}, ErrorConn{isWriteError: true})
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
