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
		conn, err = protocol.Client(info, ctx, authority, conn)
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
		conn, err = protocol.Server(info, conn)
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
