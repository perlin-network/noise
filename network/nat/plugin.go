package nat

import (
	"github.com/fd/go-nat"
	"github.com/golang/glog"
	"github.com/perlin-network/noise/network"
	"github.com/perlin-network/noise/peer"
	"net"
	"time"
)

type plugin struct {
	*network.Plugin

	gateway nat.NAT

	internalIP net.IP
	externalIP net.IP

	internalPort int
	externalPort int
}

func (state *plugin) Startup(net *network.Network) {
	glog.Info("Setting up NAT traversal...")

	info, err := network.ParseAddress(net.Address)
	if err != nil {
		return
	}

	state.internalPort = int(info.Port)

	gateway, err := nat.DiscoverGateway()
	if err != nil {
		glog.Warning("Unable to discover gateway: ", err)
		return
	}

	state.internalIP, err = gateway.GetInternalAddress()
	if err != nil {
		glog.Warning("Unable to fetch internal IP: ", err)
		return
	}

	state.externalIP, err = gateway.GetExternalAddress()
	if err != nil {
		glog.Warning("Unable to fetch external IP: ", err)
		return
	}

	glog.Infof("Discovered gateway following the protocol %s.", gateway.Type())

	glog.Info("Internal IP: ", state.internalIP.String())
	glog.Info("External IP: ", state.externalIP.String())

	state.externalPort, err = gateway.AddPortMapping("tcp", state.internalPort, "noise", 1*time.Second)

	if err != nil {
		glog.Warning("Cannot setup port mapping: ", err)
		return
	}

	glog.Infof("External port %d now forwards to your local port %d.", state.externalPort, state.internalPort)

	state.gateway = gateway

	info.Host = state.externalIP.String()
	info.Port = uint16(state.externalPort)

	// Set peer information based off of port mapping info.
	net.Address = info.String()
	net.ID = peer.CreateID(net.Address, net.Keys.PublicKey)

	glog.Infof("Other peers may connect to you through the address %s.", net.Address)
}

func (state *plugin) Cleanup(net *network.Network) {
	if state.gateway != nil {
		glog.Info("Removing port binding...")

		err := state.gateway.DeletePortMapping("tcp", state.internalPort)
		if err != nil {
			glog.Error(err)
		}
	}
}

// RegisterPlugin registers a plugin that automates port-forwarding of this nodes
// listening socket through any available UPnP interface.
//
// The plugin is registered with a priority of -999999, and thus is executed first.
func RegisterPlugin(builder *network.Builder) {
	builder.AddPluginWithPriority(-99999, new(plugin))
}
