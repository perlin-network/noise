package noise

import (
	"io"
	"math"
	"net"
	"sync"
	"sync/atomic"
)

type Handler func(Context, []byte) ([]byte, error)

type Node struct {
	l net.Listener

	p atomic.Value // noise.Protocol

	peers     map[string]*Peer
	peersLock sync.RWMutex

	opcodes     map[byte]Handler
	opcodesLock sync.RWMutex
}

func NewNode(l net.Listener) *Node {
	return &Node{
		l: l,

		peers:   make(map[string]*Peer),
		opcodes: make(map[byte]Handler),
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

// FollowProtocol enforces all peers to follow a specified protocol, which is
// representative of an implicitly defined finite state machine (FSM).
//
// The protocol is enforced upon calling (*Peer).Start(). At any other moment,
// FollowProtocol will no-nop.
//
// It is NOT safe to call FollowProtocol concurrently.
func (n *Node) FollowProtocol(p Protocol) {
	n.p.Store(p)
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

// Handle reserves an opcode under a human-readable name.
// Note that opcodes are one-indexed, and that the zero opcode is
// reserved.
//
// If an opcode is already registered under a designated name or
// byte, then Handle no-ops.
//
// It is safe to call Handle concurrently.
func (n *Node) Handle(opcode byte, fn Handler) byte {
	n.opcodesLock.Lock()
	if _, registered := n.opcodes[opcode]; !registered {
		n.opcodes[opcode] = fn
	}
	n.opcodesLock.Unlock()

	return opcode
}

// NextAvailableOpcode atomically returns the next unreserved opcode for
// a node.
//
// In total, 255 opcodes may be registered per node. Opcodes are one-
// indexed, and thus the zero opcode is never returned
//
// It ideally should only be called before a node connects to or
// listens for its first peer. Doing otherwise could cause race
// conditions on different PCs running the same software, where the
// opcodes registered for one PC differs from another PC.
//
// Ideally, opcodes should be specifically set by a user before a node
// connects to or listens for its first peer using Handle.
//
// It is safe to call NextAvailableOpcode concurrently, though heed
// the warnings above.
func (n *Node) NextAvailableOpcode() byte {
	n.opcodesLock.RLock()
	defer n.opcodesLock.RUnlock()

	for next := byte(0); next < math.MaxUint8; next++ {
		if _, exists := n.opcodes[next]; !exists {
			return next
		}
	}

	panic("no opcodes available")
}

// Addr returns the underlying address of the nodes listener.
//
// It is safe to call Addr concurrently.
func (n *Node) Addr() net.Addr {
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
