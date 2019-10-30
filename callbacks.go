package noise

import "github.com/Yayg/noise/payload"

type OnErrorCallback func(node *Node, err error) error
type OnPeerErrorCallback func(node *Node, peer *Peer, err error) error
type OnPeerDisconnectCallback func(node *Node, peer *Peer) error
type OnPeerInitCallback func(node *Node, peer *Peer) error

type BeforeMessageSentCallback func(node *Node, peer *Peer, msg []byte) ([]byte, error)
type BeforeMessageReceivedCallback func(node *Node, peer *Peer, msg []byte) ([]byte, error)

type AfterMessageSentCallback func(node *Node, peer *Peer) error
type AfterMessageReceivedCallback func(node *Node, peer *Peer) error

type AfterMessageEncodedCallback func(node *Node, peer *Peer, header, msg []byte) ([]byte, error)

type OnPeerDecodeHeaderCallback func(node *Node, peer *Peer, reader payload.Reader) error
type OnPeerDecodeFooterCallback func(node *Node, peer *Peer, msg []byte, reader payload.Reader) error

type OnMessageReceivedCallback func(node *Node, opcode Opcode, peer *Peer, message Message) error
