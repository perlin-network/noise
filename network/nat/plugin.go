package nat

import (
	"github.com/golang/glog"
	"github.com/perlin-network/noise/network"
	"github.com/perlin-network/noise/peer"
)

type plugin struct {
	*network.Plugin

	mapping *LocalPortMappingInfo
}

func (state *plugin) Startup(net *network.Network) {
	glog.Info("Setting up UPnP...")

	info, err := network.ParseAddress(net.Address)
	if err != nil {
		return
	}

	mapping, err := ForwardPort(info.Port)
	if err == nil {
		defer mapping.Close()

		info.Host = mapping.ExternalIP
		info.Port = mapping.ExternalPort

		// Set peer information base off of port mapping info.
		net.Address = info.String()
		net.ID = peer.CreateID(net.Address, net.Keys.PublicKey)

		// Keep reference to port mapping.
		state.mapping = mapping
	} else {
		glog.Warning("Cannot setup UPnP mapping: ", err)
	}
}

func (state *plugin) Cleanup(net *network.Network) {
	if state.mapping != nil {
		glog.Info("Removing UPnP port binding...")

		err := state.mapping.Close()
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
