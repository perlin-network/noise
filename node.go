package noise

import (
	"bufio"
	"io"
	"math"
	"net"
	"sync"
)

type Node struct {
	l net.Listener
	p func() Protocol

	peers     map[string]*Peer
	peersLock sync.RWMutex

	opcodes      map[string]byte
	opcodesIndex map[byte]string
	opcodesLock  sync.RWMutex
}

func NewNode(l net.Listener) *Node {
	return &Node{
		l: l,

		peers: make(map[string]*Peer),

		opcodes:      make(map[string]byte),
		opcodesIndex: make(map[byte]string),
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
	return n.NewPeer(conn.RemoteAddr(), conn, bufio.NewReader(conn), conn)
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

// RegisterOpcode reserves an opcode under a human-readable name.
// Note that opcodes are one-indexed, and that the zero opcode is
// reserved.
//
// If an opcode is already registered under a designated name or
// byte, then RegisterOpcode no-ops.
//
// It is safe to call RegisterOpcode concurrently.
func (n *Node) RegisterOpcode(name string, opcode byte) {
	if opcode == 0 {
		return
	}

	n.opcodesLock.Lock()
	_, registered1 := n.opcodes[name]
	_, registered2 := n.opcodesIndex[opcode]

	if !registered1 && !registered2 {
		n.opcodes[name] = opcode
		n.opcodesIndex[opcode] = name
	}
	n.opcodesLock.Unlock()
}

// NextAvailableOpcode returns the next unreserved opcode that
// may be registered by a node.
//
// In total, 255 opcodes may be registered per node. Opcodes
// are one-indexed, and the zero opcode is reserved.
func (n *Node) NextAvailableOpcode() (next byte) {
	n.opcodesLock.RLock()
	for i := byte(0x01); i < math.MaxUint8; i++ {
		if _, exists := n.opcodesIndex[i]; !exists {
			next = i
			break
		}
	}
	n.opcodesLock.RUnlock()

	return
}

// Opcode returns the opcode assigned to a given human-readable
// name. In total, 255 opcodes may be assigned to a single node.
//
// Opcodes are one-indexed, and if it returns the zero opcode,
// then the opcode has not been assigned as of yet.
func (n *Node) Opcode(name string) byte {
	n.opcodesLock.RLock()
	opcode := n.opcodes[name]
	n.opcodesLock.RUnlock()

	return opcode
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
