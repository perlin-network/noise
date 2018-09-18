package nat

import (
	"net"
	"time"

	"github.com/fd/go-nat"
	"github.com/perlin-network/noise/network"
	"github.com/perlin-network/noise/peer"
	"github.com/rs/zerolog/log"
)

type plugin struct {
	*network.Plugin

	gateway nat.NAT

	internalIP net.IP
	externalIP net.IP

	internalPort int
	externalPort int
}

var (
	// PluginID to reference NAT plugin
	PluginID                         = (*plugin)(nil)
	_        network.PluginInterface = (*plugin)(nil)
)

func (p *plugin) Startup(n *network.Network) {
	log.Info().Msgf("Setting up NAT traversal for address: %s", n.Address)

	info, err := network.ParseAddress(n.Address)
	if err != nil {
		return
	}

	p.internalPort = int(info.Port)

	gateway, err := nat.DiscoverGateway()
	if err != nil {
		log.Warn().Err(err).Msg("Unable to discover gateway")
		return
	}

	p.internalIP, err = gateway.GetInternalAddress()
	if err != nil {
		log.Warn().Err(err).Msg("Unable to fetch internal IP")
		return
	}

	p.externalIP, err = gateway.GetExternalAddress()
	if err != nil {
		log.Warn().Err(err).Msg("Unable to fetch external IP")
		return
	}

	log.Info().Msgf("Discovered gateway following the protocol %s.", gateway.Type())

	log.Info().Msgf("Internal IP: %s", p.internalIP.String())
	log.Info().Msgf("External IP: %s", p.externalIP.String())

	p.externalPort, err = gateway.AddPortMapping("tcp", p.internalPort, "noise", 1*time.Second)

	if err != nil {
		log.Warn().Err(err).Msg("Cannot setup port mapping")
		return
	}

	log.Info().Msgf("External port %d now forwards to your local port %d.", p.externalPort, p.internalPort)

	p.gateway = gateway

	info.Host = p.externalIP.String()
	info.Port = uint16(p.externalPort)

	// Set peer information based off of port mapping info.
	n.Address = info.String()
	n.ID = peer.CreateID(n.Address, n.GetKeys().PublicKey)

	log.Info().Msgf("Other peers may connect to you through the address %s.", n.Address)
}

func (p *plugin) Cleanup(n *network.Network) {
	if p.gateway != nil {
		log.Info().Msg("Removing port binding...")

		err := p.gateway.DeletePortMapping("tcp", p.internalPort)
		if err != nil {
			log.Error().Err(err)
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
