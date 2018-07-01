package nat

import (
	"github.com/NebulousLabs/go-upnp"
)

type LocalPortMappingInfo struct {
	LocalPort      uint16
	ExternalPort   uint16
	ExternalIP     string
	RouterLocation string
}

func (m *LocalPortMappingInfo) Close() {
	gateway, err := upnp.Load(m.RouterLocation)
	if err == nil {
		gateway.Clear(m.ExternalPort)
	}
}

func PortForward(localPort uint16) (*LocalPortMappingInfo, error) {
	gateway, err := upnp.Discover()
	if err != nil {
		return nil, err
	}

	ip, err := gateway.ExternalIP()
	if err != nil {
		return nil, err
	}

	err = gateway.Forward(localPort, "Noise")
	if err != nil {
		return nil, err
	}

	return &LocalPortMappingInfo{
		LocalPort:      localPort,
		ExternalPort:   localPort, // TODO: Allow mapping to a different port
		ExternalIP:     ip,
		RouterLocation: gateway.Location(),
	}, nil
}
