package noise

import "github.com/perlin-network/noise/payload"

type onErrorCallback func(node *Node, err error) error
type onPeerErrorCallback func(node *Node, peer *Peer, err error) error
type onPeerDisconnectCallback func(node *Node, peer *Peer) error
type onPeerInitCallback func(node *Node, peer *Peer) error

type beforeMessageSentCallback func(node *Node, peer *Peer, msg []byte) ([]byte, error)
type beforeMessageReceivedCallback func(node *Node, peer *Peer, msg []byte) ([]byte, error)

type afterMessageSentCallback func(node *Node, peer *Peer) error
type afterMessageReceivedCallback func(node *Node, peer *Peer) error

type afterMessageEncodedCallback func(node *Node, peer *Peer, header, msg []byte) ([]byte, error)

type onPeerDecodeHeaderCallback func(node *Node, peer *Peer, reader payload.Reader) error
type onPeerDecodeFooterCallback func(node *Node, peer *Peer, msg []byte, reader payload.Reader) error

type onMessageReceivedCallback func(node *Node, opcode Opcode, peer *Peer, message Message) error
