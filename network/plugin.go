package network

// PluginInterface is used to proxy callbacks to a particular Plugin instance.
type PluginInterface interface {
	// Callback for when the network starts listening for peers.
	Startup(net *Network)

	// Callback for when an incoming message is received. Return true
	// if the plugin will intercept messages to be processed.
	Receive(ctx *MessageContext) error

	// Callback for when the network stops listening for peers.
	Cleanup(net *Network)

	// Callback for when a peer connects to the network.
	PeerConnect(client *PeerClient)

	// Callback for when a peer disconnects from the network.
	PeerDisconnect(client *PeerClient)
}

// Plugin is an abstract class which all plugins extend.
type Plugin struct{}

func (*Plugin) Startup(net *Network)              {}
func (*Plugin) Receive(ctx *MessageContext) error { return nil }
func (*Plugin) Cleanup(net *Network)              {}
func (*Plugin) PeerConnect(client *PeerClient)    {}
func (*Plugin) PeerDisconnect(client *PeerClient) {}
