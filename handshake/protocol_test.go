package handshake

import (
	"bytes"
	"context"
	"github.com/perlin-network/noise"
	"net"
	"testing"
)

func TestProtocol(t *testing.T) {
	ecdh := NewECDH()

	accept := make(chan noise.Info, 1)
	lis := launchServer(t, ecdh, accept)
	defer lis.Close()

	clientInfo := noise.Info{}
	clientHandle(t, ecdh, clientInfo, lis.Addr().String())

	serverInfo := <-accept

	if !bytes.Equal(serverInfo.Bytes(SharedKey), clientInfo.Bytes(SharedKey)) {
		t.Fatalf("Key is different: %x vs %x ", serverInfo.Bytes(SharedKey), clientInfo.Bytes(SharedKey))
	}
}

func launchServer(t *testing.T, protocol ProtocolECDH, accept chan noise.Info) net.Listener {
	lis, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatal(err)
	}
	go serverHandle(t, protocol, accept, lis)

	return lis
}

func serverHandle(t *testing.T, protocol ProtocolECDH, accept chan noise.Info, lis net.Listener) {
	serverRawConn, err := lis.Accept()
	if err != nil {
		close(accept)
		t.Fatal(err)
	}

	info := noise.Info{}
	if _, err := protocol.Server(info, serverRawConn); err != nil {
		_ = serverRawConn.Close()
		close(accept)
		t.Fatalf("Error protocol.Server(): %v", err)
	}

	accept <- info
}

func clientHandle(t *testing.T, protocol ProtocolECDH, info noise.Info, lisAddr string) {
	conn, err := net.Dial("tcp", lisAddr)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	if _, err := protocol.Client(info, context.Background(), "", conn); err != nil {
		t.Fatalf("Error protocol.Client(): %v", err)

	}
}