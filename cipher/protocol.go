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
	"crypto/sha256"
	"github.com/perlin-network/noise"
	"github.com/perlin-network/noise/handshake"
	"golang.org/x/net/context"
	"net"
)

type ProtocolAEAD struct{}

func NewAEAD() ProtocolAEAD {
	return ProtocolAEAD{}
}

func (ProtocolAEAD) Client(info noise.Info, ctx context.Context, auth string, conn net.Conn) (net.Conn, error) {
	suite, _, err := DeriveAEAD(Aes256GCM(), sha256.New, info.Bytes(handshake.SharedKey), nil)
	if err != nil {
		return nil, err
	}

	return newConnAEAD(suite, conn), nil
}

func (ProtocolAEAD) Server(info noise.Info, conn net.Conn) (net.Conn, error) {
	suite, _, err := DeriveAEAD(Aes256GCM(), sha256.New, info.Bytes(handshake.SharedKey), nil)
	if err != nil {
		return nil, err
	}

	return newConnAEAD(suite, conn), nil
}
