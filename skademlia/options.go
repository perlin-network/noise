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

package skademlia

import "google.golang.org/grpc"

const (
	DefaultPrefixDiffLen = 128
	DefaultPrefixDiffMin = 32

	DefaultC1 = 16
	DefaultC2 = 16
)

type Option func(c *Client)

func WithC1(c1 int) Option {
	return func(c *Client) {
		c.c1 = c1
	}
}

func WithC2(c2 int) Option {
	return func(c *Client) {
		c.c2 = c2
	}
}

func WithDialOptions(opts ...grpc.DialOption) Option {
	return func(c *Client) {
		c.dopts = append(opts)
	}
}

func WithPrefixDiffLen(prefixDiffLen int) Option {
	return func(c *Client) {
		c.prefixDiffLen = prefixDiffLen
	}
}

func WithPrefixDiffMin(prefixDiffMin int) Option {
	return func(c *Client) {
		c.prefixDiffMin = prefixDiffMin
	}
}
