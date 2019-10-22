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
	"google.golang.org/grpc/peer"
	"net"
	"sync"
)

type Protocol interface {
	Client(*Info, context.Context, string, net.Conn) (net.Conn, error)
	Server(*Info, net.Conn) (net.Conn, error)
}

func InfoFromPeer(peer *peer.Peer) *Info {
	if peer.AuthInfo == nil {
		return nil
	}

	return peer.AuthInfo.(*Info)
}

// Info is thread safe.
type Info struct {
	data map[string]interface{}
	mu   sync.RWMutex
}

func NewInfo() *Info {
	return &Info{
		data: make(map[string]interface{}),
	}
}

func (*Info) AuthType() string {
	return "noise"
}

func (i *Info) Put(key string, val interface{}) {
	i.mu.Lock()
	defer i.mu.Unlock()

	i.data[key] = val
}

func (i *Info) Get(key string) interface{} {
	i.mu.RLock()
	defer i.mu.Unlock()

	return i.data[key]
}

func (i *Info) PutString(key, val string) {
	i.mu.Lock()
	defer i.mu.Unlock()

	i.data[key] = val
}

func (i *Info) String(key string) string {
	i.mu.RLock()
	defer i.mu.RUnlock()

	return i.data[key].(string)
}

func (i *Info) PutBytes(key string, val []byte) {
	i.mu.Lock()
	defer i.mu.Unlock()

	i.data[key] = val
}

func (i *Info) Bytes(key string) []byte {
	i.mu.RLock()
	defer i.mu.RUnlock()

	return i.data[key].([]byte)
}
