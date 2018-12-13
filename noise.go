package noise

import (
	"context"
	"fmt"
	"github.com/perlin-network/noise/base"
	"github.com/perlin-network/noise/crypto"
	"github.com/perlin-network/noise/crypto/ed25519"
	"github.com/perlin-network/noise/protocol"
	"github.com/perlin-network/noise/skademlia"
	"github.com/perlin-network/noise/skademlia/peer"
	"net"
	"time"
)

type CallbackID int
type NodeID []byte
type OpCode uint32
type StartupCallback func(id NodeID)
type ReceiveCallback func(ctx context.Context, request *protocol.Message) (*protocol.MessageBody, error)
type CleanupCallback func(id NodeID)
type PeerConnectCallback func(id NodeID)
type PeerDisconnectCallback func(id NodeID)
type PeerID peer.ID

type Noise struct {
	protocol.Service
	config           *Config
	node             *protocol.Node
	onStartup        []StartupCallback
	onReceive        map[OpCode][]ReceiveCallback
	onCleanup        []CleanupCallback
	onPeerConnect    []PeerConnectCallback
	onPeerDisconnect []PeerDisconnectCallback
}

func dialTCP(addr string) (net.Conn, error) {
	return net.DialTimeout("tcp", addr, 10*time.Second)
}

func CreatePeerID(publicKey []byte, addr string) PeerID {
	return PeerID(peer.CreateID(addr, publicKey))
}

func NewNoise(config *Config) (*Noise, error) {

	var idAdapter protocol.IdentityAdapter
	if len(config.PrivateKeyHex) == 0 {
		// generate a new identity
		if config.EnableSKademlia {
			idAdapter = skademlia.NewIdentityAdapterDefault()
		} else {
			idAdapter = base.NewIdentityAdapter()
		}
	} else {
		// if you're reusing a key, then get the keypair
		kp, err := crypto.FromPrivateKey(ed25519.New(), config.PrivateKeyHex)
		if err != nil {
			return nil, err
		}
		if config.EnableSKademlia {
			idAdapter, err = skademlia.NewIdentityFromKeypair(kp, skademlia.DefaultC1, skademlia.DefaultC2)
			if err != nil {
				return nil, err
			}
		} else {
			idAdapter = base.NewIdentityAdapterFromKeypair(kp)
		}
	}

	node := protocol.NewNode(
		protocol.NewController(),
		idAdapter,
	)

	addr := fmt.Sprintf("%s:%d", config.Host, config.Port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	if config.EnableSKademlia {
		if _, err := skademlia.NewConnectionAdapter(listener, dialTCP, node, addr); err != nil {
			return nil, err
		}
	} else {
		if _, err := base.NewConnectionAdapter(listener, dialTCP, node); err != nil {
			return nil, err
		}
	}

	n := &Noise{
		config:           config,
		node:             node,
		onStartup:        []StartupCallback{},
		onReceive:        map[OpCode][]ReceiveCallback{},
		onCleanup:        []CleanupCallback{},
		onPeerConnect:    []PeerConnectCallback{},
		onPeerDisconnect: []PeerDisconnectCallback{},
	}

	node.AddService(n)

	node.Start()

	return n, nil
}

// Callback for when the network starts listening for peers.
func (n *Noise) OnStartup(cb StartupCallback) {
	n.onStartup = append(n.onStartup, cb)
}

// Callback for when an incoming message is received.
// Returns a message body to reply or whether there was an error.
func (n *Noise) OnReceive(opCode OpCode, cb ReceiveCallback) {
	if len(n.onReceive[opCode]) == 0 {
		n.onReceive[opCode] = []ReceiveCallback{}
	}
	n.onReceive[opCode] = append(n.onReceive[opCode], cb)
}

// Callback for when the network stops listening for peers.
func (n *Noise) OnCleanup(cb CleanupCallback) {
	n.onCleanup = append(n.onCleanup, cb)
}

// Callback for when a peer connects to the node
func (n *Noise) OnPeerConnect(cb PeerConnectCallback) {
	n.onPeerConnect = append(n.onPeerConnect, cb)
}

// Callback for when a peer disconnects from the node.
func (n *Noise) OnPeerDisconnect(cb PeerDisconnectCallback) {
	n.onPeerDisconnect = append(n.onPeerDisconnect, cb)
}

func (n *Noise) Shutdown() {
	n.node.Stop()
}

func (n *Noise) Self() PeerID {
	return CreatePeerID(
		n.node.GetIdentityAdapter().MyIdentity(),
		fmt.Sprintf("%s:%d", n.config.Host, n.config.Port))
}

func (n *Noise) Bootstrap(peers ...PeerID) error {
	if n.config.EnableSKademlia {
		var skPeers []peer.ID
		for _, p := range peers {
			skPeers = append(skPeers, peer.ID(p))
		}
		return n.node.GetConnectionAdapter().(*skademlia.ConnectionAdapter).Bootstrap(skPeers...)
	} else {
		for _, p := range peers {
			if err := n.node.GetConnectionAdapter().AddRemoteID(p.PublicKey, p.Address); err != nil {
				return err
			}
		}
	}
	return nil
}

func (n *Noise) Messenger() protocol.SendAdapter {
	return n.node
}
