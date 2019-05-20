package noise

import (
	"golang.org/x/net/context"
	"google.golang.org/grpc/credentials"
	"net"
)

type Credentials struct {
	Host      string
	Protocols []Protocol
}

func NewCredentials(host string, protocols ...Protocol) *Credentials {
	return &Credentials{Host: host, Protocols: protocols}
}

func (c *Credentials) ClientHandshake(ctx context.Context, authority string, conn net.Conn) (net.Conn, credentials.AuthInfo, error) {
	info := make(Info)
	var err error

	for _, protocol := range c.Protocols {
		conn, err = protocol.ClientHandshake(info, ctx, authority, conn)
		if err != nil {
			return nil, nil, err
		}
	}

	return conn, info, err
}

func (c *Credentials) ServerHandshake(conn net.Conn) (net.Conn, credentials.AuthInfo, error) {
	info := make(Info)
	var err error

	for _, protocol := range c.Protocols {
		conn, err = protocol.ServerHandshake(info, conn)
		if err != nil {
			return nil, nil, err
		}
	}

	return conn, info, err
}

func (c *Credentials) Info() credentials.ProtocolInfo {
	return credentials.ProtocolInfo{
		SecurityProtocol: "noise",
		SecurityVersion:  "0.0.1",
		ServerName:       c.Host,
	}
}

func (c *Credentials) Clone() credentials.TransportCredentials {
	return &Credentials{Host: c.Host}
}

func (c *Credentials) OverrideServerName(host string) error {
	c.Host = host
	return nil
}
