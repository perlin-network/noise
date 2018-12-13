package noise

import (
	"github.com/perlin-network/noise/base"
	"github.com/perlin-network/noise/protocol"
	"github.com/perlin-network/noise/skademlia"
	"github.com/perlin-network/noise/skademlia/peer"
)

type CallbackID int
type NodeID []byte
type Opcode uint32
type StartupCallback func(id NodeID)
type ReceiveCallback func(ctx context.Context, request *protocol.Message) (*protocol.MessageBody, error)
type CleanupCallback func(id NodeID)
type PeerConnectCallback func(id NodeID)
type PeerDisconnectCallback func(id NodeID)
type PeerID peer.ID

type Noise struct {
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
			idAdapter = base.NewIdentityFromKeypair(kp)
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

	node.Start()

	return &Noise{
		config:           config,
		node:             node,
		onStartup:        []StartupCallback{},
		onReceive:        map[OpCode][]ReceiveCallback{},
		onCleanup:        []CleanupCallback{},
		onPeerConnect:    []PeerConnectCallback{},
		onPeerDisconnect: []PeerDisconnectCallback{},
	}, nil
}

// Callback for when the network starts listening for peers.
func (n *Noise) OnStartup(cb StartupCallback) {
	n.onStartup = append(n.onStartup, cb)
}

// Callback for when an incoming message is received.
// Returns a message body to reply or whether there was an error.
func (n *Noise) OnReceive(opCode Opcode, cb ReceiveCallback) {
	if len(n.onReceive[opCode]) == 0 {
		n.onReceive[opCode] = make([]ReceiveCallback)
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
	return PeerID(peer.CreateID(fmt.Sprintf("%s:%d", n.config.Host, n.config.Port), n.node.GetIdentityAdapter().MyIdentity()))
}

func (n *Noise) Bootstrap(peers []PeerID) error {
	if n.config.EnableSKademlia {
		var skPeers []peer.ID
		for _, p := range peers {
			skPeers = append(skPeers peer.ID(p))
		}
		n.node.GetConnectionAdapter().(*skademlia.ConnectionAddapter).Bootstrap(skPeers...)
	} else {
		for _, p := range peers {
			n.node.GetConnectionAdapter().AddRemoteID(p.Id, p.Address)
		}
	}
}

func (n *Noise) Messenger() protocol.SendAdapter {
	return n.node
}
