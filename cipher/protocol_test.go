package cipher

import (
	"context"
	"github.com/perlin-network/noise"
	"github.com/perlin-network/noise/handshake"
	"github.com/stretchr/testify/assert"
	"net"
	"testing"
)

func TestProtocol(t *testing.T) {
	sharedKey := []byte("secret_key")
	accept := make(chan *connAEAD, 1)

	ecdh := NewAEAD()
	lis := launchServer(t, ecdh, sharedKey, accept)

	defer func() {
		_ = lis.Close()
	}()

	clientInfo := noise.Info{}
	clientInfo.PutBytes(handshake.SharedKey, sharedKey)
	clientConn := clientHandle(t, ecdh, clientInfo, lis.Addr().String())

	serverConn := <-accept

	msg := []byte("secret_message")

	go func() {
		_, err := serverConn.Write(msg)
		assert.NoError(t, err)
	}()

	buf := make([]byte, 1024)
	n, err := clientConn.Read(buf)
	assert.NoError(t, err)
	assert.Equal(t, msg, buf[:n])
}

func TestProtocolKeyMismatch(t *testing.T) {
	accept := make(chan *connAEAD, 1)

	ecdh := NewAEAD()
	lis := launchServer(t, ecdh, []byte("server_secret_key"), accept)

	defer func() {
		_ = lis.Close()
	}()

	clientInfo := noise.Info{}
	clientInfo.PutBytes(handshake.SharedKey, []byte("client_secret_key"))
	clientConn := clientHandle(t, ecdh, clientInfo, lis.Addr().String())

	serverConn := <-accept

	go func() {
		_, err := serverConn.Write([]byte("secret_message"))
		assert.NoError(t, err)
	}()

	buf := make([]byte, 1024)
	_, err := clientConn.Read(buf)
	assert.Error(t, err)
}

func launchServer(t *testing.T, protocol ProtocolAEAD, sharedKey []byte, accept chan *connAEAD) net.Listener {
	lis, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatal(err)
	}

	go serverHandle(t, protocol, sharedKey, accept, lis)

	return lis
}

func serverHandle(t *testing.T, protocol ProtocolAEAD, sharedKey []byte, accept chan *connAEAD, lis net.Listener) {
	rawConn, err := lis.Accept()
	if !assert.NoError(t, err) {
		close(accept)
		return
	}

	info := noise.Info{}
	info.PutBytes(handshake.SharedKey, sharedKey)

	conn, err := protocol.Server(info, rawConn)
	if !assert.NoError(t, err) {
		close(accept)
		return
	}

	accept <- conn.(*connAEAD)
}

func clientHandle(t *testing.T, protocol ProtocolAEAD, info noise.Info, lisAddr string) *connAEAD {
	rawConn, err := net.Dial("tcp", lisAddr)
	if err != nil {
		t.Fatal(err)
	}

	conn, err := protocol.Client(info, context.Background(), "", rawConn)
	if err != nil {
		t.Fatalf("Error protocol.Client(): %v", err)
	}

	return conn.(*connAEAD)
}
