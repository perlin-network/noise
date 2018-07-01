package network

import "github.com/perlin-network/noise/peer"

type PluginInterface interface {
	// Callback for when the network starts listening for peers.
	Startup(net *Network)

	// Callback for when an incoming message is received. Return true
	// if the plugin will intercept messages to be processed.
	Receive(ctx *MessageContext) error

	// Callback for when the network stops listening for peers.
	Cleanup(net *Network)

	// Callback for when a peer disconnects from the network.
	PeerDisconnect(id *peer.ID)
}

type Plugin struct{}
func (*Plugin) Startup(net *Network)             {}
func (*Plugin) Receive(ctx *MessageContext) error { return nil }
func (*Plugin) Cleanup(net *Network)             {}
func (*Plugin) PeerDisconnect(id *peer.ID) {}