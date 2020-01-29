// noise is an opinionated, easy-to-use P2P network stack for decentralized applications, and
// cryptographic protocols written in Go.
//
// noise is made to be minimal, robust, developer-friendly, performant, secure, and cross-platform across
// multitudes of devices by making use of a small amount of well-tested, production-grade dependencies.
package noise

type Handler func(ctx HandlerContext) error

type Binder interface {
	Bind(node *Node) error
	OnPeerJoin(client *Client)
	OnPeerLeave(client *Client)
	OnMessageSent(client *Client)
	OnMessageRecv(client *Client)
}
