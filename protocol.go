package noise

import (
	"golang.org/x/net/context"
	"google.golang.org/grpc/peer"
	"net"
)

type Protocol interface {
	ClientHandshake(Info, context.Context, string, net.Conn) (net.Conn, error)
	ServerHandshake(Info, net.Conn) (net.Conn, error)
}

func InfoFromPeer(peer *peer.Peer) Info {
	if peer.AuthInfo == nil {
		return nil
	}

	return peer.AuthInfo.(Info)
}

type Info map[string]interface{}

func (Info) AuthType() string {
	return "noise"
}

func (i Info) Put(key string, val interface{}) {
	i[key] = val
}

func (i Info) Get(key string) interface{} {
	return i[key]
}

func (i Info) PutString(key, val string) {
	i[key] = val
}

func (i Info) String(key string) string {
	return i[key].(string)
}

func (i Info) PutBytes(key string, val []byte) {
	i[key] = val
}

func (i Info) Bytes(key string) []byte {
	return i[key].([]byte)
}
