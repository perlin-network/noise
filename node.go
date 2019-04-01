package noise

import (
	"io"
	"net"
	"sync"
)

type Node struct {
	l net.Listener
	d Dialer

	p func() Protocol

	peers     map[string]*Peer
	peersLock sync.RWMutex
}

func NewNode(l net.Listener) *Node {
	return &Node{
		l: l,
		d: defaultDialer,

		peers: make(map[string]*Peer),
	}
}

// NewPeer instantiates a new peer instance, listening for incoming data through
// a provided io.Reader, and writing new data to a provided io.Writer, and
// enforcing timeouts through a provided noise.Conn.
//
// It registers the newly instantiated peer to the node, prefixed by a provided
// net.Addr instance.
//
// It is safe to call NewPeer concurrently.
func (n *Node) NewPeer(addr net.Addr, w io.Writer, r io.Reader, c Conn) *Peer {
	n.peersLock.Lock()
	defer n.peersLock.Unlock()

	if addr != nil {
		p, existed := n.peers[addr.String()]

		if existed {
			return p
		}
	}

	p := newPeer(n, addr, w, r, c)

	if addr != nil {
		n.peers[addr.String()] = p
	}

	return p
}

// Wrap instantiates a new peer instance from an existing network connection.
//
// It is safe to call Wrap concurrently.
func (n *Node) Wrap(conn net.Conn) *Peer {
	return n.NewPeer(conn.RemoteAddr(), conn, conn, conn)
}

// SetDialer sets the dialer used to establish connections to new peers.
// By default, the nodes default dialer dials nodes through TCP.
//
// It is NOT safe to call SetDialer concurrently.
func (n *Node) SetDialer(d Dialer) {
	n.d = d
}

// Dial dials the address, and returns the peer instance representative
// of the established underlying connection.
//
// In order to redefine how addresses are dialed, or how peer instances
// are instantiated, refer to (*Node).SetDialer(noise.Dialer).
//
// It is safe to call Dial concurrently.
func (n *Node) Dial(address string) (*Peer, error) {
	if p := n.PeerByAddr(address); p != nil {
		return p, nil
	}

	return n.d(n, address)
}

// FollowProtocol enforces all peers to follow a specified protocol, which is
// representative of an implicitly defined finite state machine (FSM).
//
// The protocol is enforced upon calling (*Peer).Start(). At any other moment,
// FollowProtocol will no-nop.
//
// It is NOT safe to call FollowProtocol concurrently.
func (n *Node) FollowProtocol(p func() Protocol) {
	n.p = p
}

// Peers returns a slice of presently-connected peer instances to the node.
//
// It is safe to call Peers concurrently.
func (n *Node) Peers() []*Peer {
	n.peersLock.RLock()

	peers := make([]*Peer, 0, len(n.peers))
	for _, peer := range n.peers {
		peers = append(peers, peer)
	}

	n.peersLock.RUnlock()

	return peers
}

// PeerByAddr returns a peer instance if available given its address.
//
// It is safe to call PeerByAddr concurrently.
func (n *Node) PeerByAddr(address string) *Peer {
	n.peersLock.RLock()
	peer := n.peers[address]
	n.peersLock.RUnlock()

	return peer
}

// Addr returns the underlying address of the nodes listener.
//
// It is safe to call Addr concurrently.
func (n Node) Addr() net.Addr {
	return n.l.Addr()
}

// Shutdown closes the underlying nodes peer acceptor socket, and gracefully
// kills all peer instances connected to the node.
//
// It is safe to call Shutdown concurrently.
func (n *Node) Shutdown() {
	if n.l != nil {
		_ = n.l.Close()
	}

	for _, peer := range n.Peers() {
		peer.Disconnect(nil)
	}
}
