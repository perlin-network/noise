package nat

import (
	"github.com/golang/glog"
	"github.com/perlin-network/noise/network"
	"github.com/perlin-network/noise/peer"
)

const PluginID = "upnp"

// Plugin for automating port-forwarding of this nodes listening socket through
// the UPnP interface.
//
// NOTE: This must be the first plugin registered onto the network!!!
// TODO: Enforce priority on which plugin gets executed first.
type Plugin struct {
	*network.Plugin

	mapping *LocalPortMappingInfo
}

func (state *Plugin) Startup(net *network.Network) {
	glog.Info("Setting up UPnP...")

	mapping, err := ForwardPort(net.GetPort())
	if err == nil {
		defer mapping.Close()

		addressInfo, err := network.ExtractAddressInfo(net.Address)
		if err != nil {
			glog.Fatal(err)
		}

		addressInfo.Host = mapping.ExternalIP
		addressInfo.Port = mapping.ExternalPort

		// Set peer information base off of port mapping info.
		net.Address = addressInfo.String()
		net.ID = peer.CreateID(net.Address, net.Keys.PublicKey)

		// Keep reference to port mapping.
		state.mapping = mapping
	} else {
		glog.Warning("Cannot setup UPnP mapping: ", err)
	}
}

func (state *Plugin) Cleanup(net *network.Network) {
	if state.mapping != nil {
		glog.Info("Removing UPnP port binding...")

		err := state.mapping.Close()
		if err != nil {
			glog.Error(err)
		}
	}
}
