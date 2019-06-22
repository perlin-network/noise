package cipher

import (
	"context"
	"github.com/perlin-network/noise"
	"github.com/perlin-network/noise/handshake"
	"net"
	"testing"
)

func TestProtocol(t *testing.T) {
	ecdh := NewAEAD()

	accept := make(chan noise.Info, 1)
	lis := launchServer(t, ecdh, accept)
	defer lis.Close()

	clientInfo := noise.Info{}
	clientInfo.PutBytes(handshake.SharedKey, []byte("sharedkey"))
	clientHandle(t, ecdh, clientInfo, lis.Addr().String())

	<-accept
}

func launchServer(t *testing.T, protocol ProtocolAEAD, accept chan noise.Info) net.Listener {
	lis, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatal(err)
	}
	go serverHandle(t, protocol, accept, lis)

	return lis
}

func serverHandle(t *testing.T, protocol ProtocolAEAD, accept chan noise.Info, lis net.Listener) {
	serverRawConn, err := lis.Accept()
	if err != nil {
		close(accept)
		t.Fatal(err)
	}

	info := noise.Info{}
	info.PutBytes(handshake.SharedKey, []byte("sharedkey"))
	if _, err := protocol.Server(info, serverRawConn); err != nil {
		_ = serverRawConn.Close()
		close(accept)
		t.Fatalf("Error protocol.Server(): %v", err)
	}

	accept <- info
}

func clientHandle(t *testing.T, protocol ProtocolAEAD, info noise.Info, lisAddr string) {
	conn, err := net.Dial("tcp", lisAddr)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	if _, err := protocol.Client(info, context.Background(), "", conn); err != nil {
		t.Fatalf("Error protocol.Client(): %v", err)

	}
}
