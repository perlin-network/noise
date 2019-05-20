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
