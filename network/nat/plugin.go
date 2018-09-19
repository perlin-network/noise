package nat

import (
	"net"
	"time"

	"github.com/perlin-network/noise/log"
	"github.com/perlin-network/noise/network"
	"github.com/perlin-network/noise/peer"

	"github.com/fd/go-nat"
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
	log.Info().
		Str("address", n.Address).
		Msg("setting up NAT traversal")

	info, err := network.ParseAddress(n.Address)
	if err != nil {
		return
	}

	p.internalPort = int(info.Port)

	gateway, err := nat.DiscoverGateway()
	if err != nil {
		log.Warn().
			Err(err).
			Msg("unable to discover gateway")
		return
	}

	p.internalIP, err = gateway.GetInternalAddress()
	if err != nil {
		log.Warn().
			Err(err).
			Msg("unable to fetch internal IP")
		return
	}

	p.externalIP, err = gateway.GetExternalAddress()
	if err != nil {
		log.Warn().
			Err(err).
			Msg("unable to fetch external IP")
		return
	}

	log.Info().
		Str("protocol", gateway.Type()).
		Msg("discovered gateway")

	log.Info().
		Str("internal_ip", p.internalIP.String()).
		Str("external_ip", p.externalIP.String()).
		Msg("")

	p.externalPort, err = gateway.AddPortMapping("tcp", p.internalPort, "noise", 1*time.Second)

	if err != nil {
		log.Warn().
			Err(err).
			Msg("cannot setup port mapping")
		return
	}

	log.Info().
		Int("internal_port", p.internalPort).
		Int("external_port", p.externalPort).
		Msgf("external port now forwards to your local port")

	p.gateway = gateway

	info.Host = p.externalIP.String()
	info.Port = uint16(p.externalPort)

	// Set peer information based off of port mapping info.
	n.Address = info.String()
	n.ID = peer.CreateID(n.Address, n.GetKeys().PublicKey)

	log.Info().Msgf("other peers may connect to you through the address %s.", n.Address)
}

func (p *plugin) Cleanup(n *network.Network) {
	if p.gateway != nil {
		log.Info().Msg("removing port binding...")

		err := p.gateway.DeletePortMapping("tcp", p.internalPort)
		if err != nil {
			log.Error().Err(err).Msg("")
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
